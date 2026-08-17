package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"gopkg.in/telebot.v3"
	_ "modernc.org/sqlite"
	"transport-app/internal/apiversion"

	dbmigr "transport-app/db"
	"transport-app/internal/auth"
	"transport-app/internal/config"
	"transport-app/internal/domain"
	"transport-app/internal/graphqlservice"
	"transport-app/internal/grpcservice"
	"transport-app/internal/handlers"
	"transport-app/internal/integration"
	"transport-app/internal/logging"
	"transport-app/internal/middleware"
	"transport-app/internal/mqttservice"
	"transport-app/internal/openapispec"
	"transport-app/internal/operations/audit"
	"transport-app/internal/operations/dashboard"
	opserrors "transport-app/internal/operations/errors"
	"transport-app/internal/operations/health"
	"transport-app/internal/operations/notifications"
	"transport-app/internal/pnl"
	"transport-app/internal/repository/sqlite"
	"transport-app/internal/service"
	"transport-app/internal/telemetry"

	// Vertical-slice use cases
	bookingApp "transport-app/internal/booking/application"
	bookingHandlers "transport-app/internal/booking/presentation/api/handlers"

	authAPIHandlers "transport-app/internal/auth/presentation/api/handlers"

	invoiceApp "transport-app/internal/invoice/application"
	invoiceHandlers "transport-app/internal/invoice/presentation/api/handlers"

	paymentApp "transport-app/internal/payment/application"
	paymentHandlers "transport-app/internal/payment/presentation/api/handlers"

	tripApp "transport-app/internal/trip/application"
	tripHandlers "transport-app/internal/trip/presentation/api/handlers"

	// Shared infrastructure
	"transport-app/internal/events"
	founder "transport-app/internal/founder"
	founderAlerts "transport-app/internal/founder/alerts"
	"transport-app/internal/founder/digest"
	"transport-app/internal/shared/clock"
	"transport-app/internal/shared/id"
	"transport-app/internal/shared/outbox"
	"transport-app/internal/shared/uow"

	"github.com/pressly/goose/v3"
)

// Version is set via ldflags during build
var Version string

func main() {
	if Version != "" {
		_ = os.Setenv("APP_VERSION", Version)
		handlers.AppVersion = Version
	}
	cfg := config.Load()
	logging.Setup(cfg.LogLevel, cfg.AppEnv)

	logger := slog.Default()
	if cfg.IsProduction() && cfg.UsingKnownDefaultSecret() {
		logger.Error("Refusing to start in production with known default secrets. Set strong, unique COOKIE_SECRET, API_SECRET and RAZORPAY_* values in the environment.")
		os.Exit(1)
	}
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	logger.Info("Starting MVTMS server", "env", cfg.AppEnv, "port", cfg.Port)

	// Open database with optimized PRAGMAs for high concurrency
	database, err := sql.Open("sqlite", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer func() { _ = database.Close() }()

	database.SetMaxOpenConns(64)
	database.SetMaxIdleConns(32)
	database.SetConnMaxLifetime(15 * time.Minute)

	// Execute WAL mode & performance pragmas. synchronous=NORMAL keeps
	// WAL durability while avoiding the FULL sync penalty on every commit.
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=10000;",
		"PRAGMA cache_size=-131072;",  // 128MB cache
		"PRAGMA mmap_size=536870912;", // 512MB memory-mapped file I/O
		"PRAGMA locking_mode=NORMAL;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA temp_store=MEMORY;",
	}
	for _, p := range pragmas {
		if _, err := database.Exec(p); err != nil {
			logger.Warn("Failed to execute pragma", "pragma", p, "error", err)
		}
	}

	// Run migrations from embedded filesystem
	ctx := context.Background()
	migrations, err := fs.Sub(dbmigr.Migrations, "migrations")
	if err != nil {
		logger.Error("Failed to read embedded migrations", "error", err)
		os.Exit(1)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, database, migrations)
	if err != nil {
		logger.Error("Failed to create migration provider", "error", err)
		os.Exit(1)
	}

	if _, err := provider.Up(ctx); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	logger.Info("Database migrated successfully")

	// Initialize repository
	repo := sqlite.NewRepository(database)

	// Initialize services
	services := service.NewServices(repo, cfg, logger)

	// Initialize auth store
	authStore := auth.NewSessionStore(cfg.CookieSecret, cfg.CookieSecure)
	authStore.SetValidator(services.Auth)

	// Initialize Casbin authorization service
	authSvc, err := auth.NewCasbinAuthorizationService(database)
	if err != nil {
		logger.Error("Failed to initialize Casbin authorization service", "error", err)
		os.Exit(1)
	}

	// Create the initial admin account from env vars (optional; skipped when
	// an admin already exists or the vars are unset).
	bootstrapAdmin(ctx, services, authSvc, cfg, logger)

	// Initialize handlers app
	app := handlers.NewApp(services, cfg, authStore, database, authSvc)

	// ── Ops: error reporting, login audit, dashboard ─────────────────────
	notifSvc := notifications.NewService()
	reporter := opserrors.NewReporter(notifSvc, cfg.AppEnv, Version)
	loginAuditSvc := audit.NewLoginAuditService(notifSvc, audit.SecurityPolicy{
		NotifyOnNewDevice: true,
		NotifyOnNewIP:     true,
	})
	dashboardHandler := dashboard.NewDashboardHandler(reporter, loginAuditSvc)
	healthChecker := health.NewChecker(database)

	// ── Vertical-slice infrastructure ────────────────────────────────────
	sqlUoW := uow.NewSQLUnitOfWork(database)
	idGen := id.NewUUIDGenerator()
	realClock := clock.NewRealClock()

	// Sprint 1 – Booking use cases
	createBookingUC := bookingApp.NewCreateBookingUseCase(sqlUoW, idGen, realClock)
	confirmBookingUC := bookingApp.NewConfirmBookingUseCase(sqlUoW, realClock)
	cancelBookingUC := bookingApp.NewCancelBookingUseCase(sqlUoW, realClock)
	updateBookingUC := bookingApp.NewUpdateBookingUseCase(sqlUoW)
	completeBookingUC := bookingApp.NewCompleteBookingUseCase(sqlUoW, realClock)
	deleteBookingUC := bookingApp.NewDeleteBookingUseCase(sqlUoW)
	getBookingUC := bookingApp.NewGetBookingUseCase(sqlUoW)
	listBookingsUC := bookingApp.NewListBookingsUseCase(sqlUoW)

	// Sprint 2 – Trip use cases
	createTrip := tripApp.NewCreateTripUseCase(sqlUoW, idGen, realClock)
	assignDriver := tripApp.NewAssignDriverUseCase(sqlUoW, realClock)
	assignVehicle := tripApp.NewAssignVehicleUseCase(sqlUoW, realClock)
	scheduleTrip := tripApp.NewScheduleTripUseCase(sqlUoW, realClock)
	startTrip := tripApp.NewStartTripUseCase(sqlUoW, realClock)
	reachPickup := tripApp.NewReachPickupUseCase(sqlUoW, realClock)
	startTransit := tripApp.NewStartTransitUseCase(sqlUoW, realClock)
	deliver := tripApp.NewDeliverUseCase(sqlUoW, realClock)
	completeTrip := tripApp.NewCompleteTripUseCase(sqlUoW, realClock)
	cancelTrip := tripApp.NewCancelTripUseCase(sqlUoW, realClock)
	getTrip := tripApp.NewGetTripUseCase(sqlUoW)
	listTrips := tripApp.NewListTripsUseCase(sqlUoW)

	// Sprint 3 – Invoice use cases
	generateInvoice := invoiceApp.NewGenerateInvoiceUseCase(sqlUoW, idGen, realClock)
	getInvoice := invoiceApp.NewGetInvoiceUseCase(sqlUoW)
	listInvoices := invoiceApp.NewListInvoicesUseCase(sqlUoW)
	voidInvoice := invoiceApp.NewVoidInvoiceUseCase(sqlUoW, realClock)

	// Sprint 4 – Payment use cases
	recordPayment := paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock)
	getPayment := paymentApp.NewGetPaymentUseCase(sqlUoW)
	listPayments := paymentApp.NewListPaymentsUseCase(sqlUoW)
	reversePayment := paymentApp.NewReversePaymentUseCase(sqlUoW, idGen, realClock)
	listPaymentsByInvoice := paymentApp.NewListPaymentsByInvoiceUseCase(sqlUoW)

	// ── API handlers ──────────────────────────────────────────────────────
	bookingAPIHandler := bookingHandlers.NewAPIBookingHandler(
		createBookingUC, confirmBookingUC, cancelBookingUC, updateBookingUC, completeBookingUC, deleteBookingUC, getBookingUC, listBookingsUC,
		authSvc,
	)
	tripAPIHandler := tripHandlers.NewAPITripHandler(
		createTrip, assignDriver, assignVehicle, scheduleTrip, startTrip, reachPickup, startTransit, deliver, completeTrip, cancelTrip, getTrip, listTrips,
		authSvc,
	)
	invoiceAPIHandler := invoiceHandlers.NewAPIInvoiceHandler(generateInvoice, getInvoice, listInvoices, voidInvoice, authSvc)
	razorpayWebhookUC := paymentApp.NewRazorpayWebhookUseCase(recordPayment, sqlUoW, cfg.RazorpayWebhook, realClock)
	paymentAPIHandler := paymentHandlers.NewAPIPaymentHandler(recordPayment, getPayment, listPayments, reversePayment, listPaymentsByInvoice, razorpayWebhookUC, authSvc)
	integrationHandler := integration.NewHandler(integration.LoadConfig(), authSvc)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.SecurityHeaders)
	r.Use(apiversion.Middleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer(reporter))
	r.Use(chiMiddleware.Compress(5))
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(middleware.SPAMiddleware)

	// Global HTTP middleware: Limit request body to 32MB in RAM (prevents disk spooling)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(w, req.Body, 32<<20)
			next.ServeHTTP(w, req)
		})
	})

	// CSRF defense-in-depth: reject cross-site state-changing requests that
	// carry a session cookie (complements SameSite=Lax). Bearer-token API
	// requests and cookie-less requests are unaffected. Strict mode rejects
	// browser requests that omit both Origin and Referer.
	r.Use(middleware.CSRFProtectStrict(authStore))

	// Ops: liveness, health, readiness (no auth — probe endpoints)
	r.Get("/healthz", healthChecker.LivenessHandler)
	r.Get("/health", healthChecker.HealthHandler)
	r.Get("/readyz", healthChecker.ReadinessHandler)

	// Direct SEO Endpoints
	r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\nSitemap: https://avandab.com/sitemap.xml\n"))
	})
	r.Get("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://avandab.com/</loc>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://avandab.com/login</loc>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://avandab.com/register</loc>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>`
		_, _ = w.Write([]byte(sitemap))
	})

	// API discovery and OpenAPI spec (public)
	r.Get("/api/versions", apiversion.VersionsHandler)
	openapispec.RegisterRoutes(r)

	// ── REST API v1 ───────────────────────────────────────────────────────
	// Require dedicated API token secret; fall back to cookie secret only in non-production.
	apiSecret := []byte(cfg.APITokenSecret)
	if len(apiSecret) == 0 {
		if cfg.IsProduction() {
			slog.Error("API_SECRET must be explicitly configured in production")
			os.Exit(1)
		}
		apiSecret = []byte(cfg.CookieSecret)
	}
	authAPIHandler := authAPIHandlers.NewAPIAuthHandler(services.Auth, services.Users, apiSecret)

	// ── High-Performance Architecture Protocols ──────────────────────
	// 1. MQTT Broker Client Setup
	mqttURL := os.Getenv("MQTT_URL")
	if mqttURL == "" {
		mqttURL = "tcp://localhost:1883"
	}
	_ = mqttservice.NewMQTTBroker(mqttURL)

	// 2. gRPC Dispatch Microservice
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	grpcservice.StartGRPCServer(grpcPort)

	// 3. GraphQL Query Endpoint
	graphqlH := graphqlservice.NewGraphQLHandler(listTrips)

	// Public: token endpoint (no auth required) — rate-limited against brute force
	authAPIHandler.Register(r.With(middleware.RateLimit(10)))

	// Public: Razorpay webhook — signature-verified, rate-limited against flood attacks
	r.With(middleware.RateLimit(30)).Post("/api/v1/payments/razorpay-webhook", paymentAPIHandler.RazorpayWebhook)

	// Protected: GraphQL, Telemetry, and all /api/v1/* routes require a valid session or Bearer token
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAPIAuth(authStore, apiSecret))
		r.Post("/query", graphqlH.ServeHTTP)
		r.Get("/graphql", graphqlH.ServeHTTP)
		telemetry.RegisterTelemetryRoutes(r, database)
		pnl.RegisterRoutes(r, pnl.NewService(database), authSvc)
		bookingAPIHandler.Register(r)
		tripAPIHandler.Register(r)
		invoiceAPIHandler.Register(r)
		paymentAPIHandler.Register(r)
		integrationHandler.Register(r)
	})

	// Deprecated v2 alias routes (rewrite to v1) plus /api/v2/health.
	// Aliased routes require the same API auth as v1; the public health check
	// is mounted separately so probes stay unauthenticated.
	apiversion.MountV2(r, middleware.RequireAPIAuth(authStore, apiSecret), http.HandlerFunc(healthChecker.HealthHandler), bookingAPIHandler, tripAPIHandler, invoiceAPIHandler, paymentAPIHandler)

	// Static files with Cache-Control headers
	fileServer := http.FileServer(http.Dir(cfg.StaticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If request has a version query param (?v=...), cache immutably since URL changes on deploy.
		// Otherwise, use short max-age with revalidation so updates take effect immediately.
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
		}
		fileServer.ServeHTTP(w, r)
	})))

	// Uploaded files (logos, documents) - require authentication
	uploadsServer := http.FileServer(http.Dir(cfg.UploadDir))
	r.With(middleware.RequireAuth(authStore)).Handle("/uploads/*", http.StripPrefix("/uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		uploadsServer.ServeHTTP(w, r)
	})))

	// All application routes
	r.Group(func(r chi.Router) {

		// Public routes
		r.Get("/", app.Marketing)
		r.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\nSitemap: https://avandab.com/sitemap.xml\n"))
		})
		r.Get("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			sitemap := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://avandab.com/</loc>
    <changefreq>daily</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>https://avandab.com/login</loc>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
  <url>
    <loc>https://avandab.com/register</loc>
    <changefreq>monthly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>`
			_, _ = w.Write([]byte(sitemap))
		})
		r.Get("/login", app.Auth.LoginPage)
		r.With(middleware.RateLimit(10)).Post("/login", app.Auth.Login)
		r.Get("/register", app.Auth.RegisterPage)
		r.With(middleware.RateLimit(10)).Post("/register", app.Auth.Register)
		r.Get("/forgot-password", app.Auth.ForgotPasswordPage)
		r.Post("/forgot-password", app.Auth.SubmitForgotPassword)
		r.Post("/logout", app.Auth.Logout)

		// Public Contact & Status Tracking
		r.Route("/contact-us", app.Contact.Routes)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(authStore))

			// User Setup & Onboarding
			r.Get("/user/onboard", app.Auth.UserOnboardingPage)

			// Dashboard
			r.Get("/dashboard", app.Dashboard.Index)
			r.Get("/files/{id}", app.DownloadFile)

			// Ops dashboard (errors & incidents, login audit) - Admin only
			r.With(middleware.RoleRequired(domain.DefaultRoleID(domain.RoleAdmin))).Get("/ops/dashboard", dashboardHandler.ServeHTTP)

			// Users (Admin only)
			r.Route("/users", app.Users.Routes)

			// Drivers
			r.Route("/drivers", app.Drivers.Routes)

			// Vehicles
			r.Route("/vehicles", app.Vehicles.Routes)

			// Customers
			r.Route("/customers", app.Customers.Routes)

			// Routes
			r.Route("/routes", app.Routes.Routes)

			// Bookings
			r.Route("/bookings", app.Bookings.Routes)

			// Trips
			r.Route("/trips", app.Trips.Routes)

			// Invoices
			r.Route("/invoices", app.Invoices.Routes)

			// Payments
			r.Route("/payments", app.Payments.Routes)

			// Reports
			r.Route("/reports", app.Reports.Routes)

			// Settings & Company Onboarding
			r.Route("/settings", app.SettingsH.Routes)
			r.Route("/company", app.SettingsH.Routes)

			// Audit Logs
			r.Route("/audit-logs", app.AuditLogs.Routes)

			// Kharcha Ledger (driver expense approvals)
			r.Route("/kharcha", app.Kharcha.Routes)

			// e-POD delivery from driver mobile
			r.Post("/trips/{id}/deliver-pod", app.Kharcha.DeliverWithPOD)

			// Profile (auth)
			r.Get("/profile", app.Auth.ProfilePage)
			r.Post("/profile", app.Auth.UpdateProfile)
			r.Get("/change-password", app.Auth.ChangePasswordPage)
			r.Post("/change-password", app.Auth.ChangePassword)
		})
	})

	// Start server with graceful shutdown
	addr := fmt.Sprintf(":%s", port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Outbox relay & founder notifications ──────────────────────────
	eventBus := events.NewInMemoryBus()
	founderSvc := founder.NewFounderService(newFounderNotifier(logger))
	founderSvc.RegisterEventHandlers(eventBus)
	if founderConfigured() {
		go runDailyDigest(ctx, founderSvc, logger)
	}
	outboxRelay := outbox.NewRelay(database, eventBus, logger)
	go outboxRelay.Run(ctx)

	go func() {
		logger.Info("Server listening", "address", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Graceful shutdown failed", "error", err)
	}
}

// bootstrapAdmin creates the initial admin account from environment config.
// It runs only when BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD are
// set, no admin account exists yet, and the password meets the policy.
// Admin provisioning otherwise happens through the authenticated user
// management interface.
func bootstrapAdmin(ctx context.Context, services *service.Services, authSvc auth.AuthorizationService, cfg *config.Config, logger *slog.Logger) {
	ba := cfg.BootstrapAdmin
	if ba.Email == "" || ba.Password == "" {
		logger.Info("bootstrap admin skipped: BOOTSTRAP_ADMIN_EMAIL / BOOTSTRAP_ADMIN_PASSWORD not set")
		return
	}

	users, _, err := services.Users.ListUsers(ctx, "", "", 100, 0)
	if err != nil {
		logger.Error("bootstrap admin failed: cannot list users", "error", err)
		return
	}
	for _, u := range users {
		if u.RoleID == 1 {
			logger.Info("bootstrap admin skipped: an admin account already exists")
			return
		}
	}

	user, err := services.Users.CreateUserWithPassword(ctx, ba.Email, ba.Name, "", ba.Password, 1, domain.UserStatusActive)
	if err != nil {
		logger.Error("bootstrap admin failed", "error", err)
		return
	}
	if err := authSvc.AddRoleForUser(user.ID.String(), "admin"); err != nil {
		logger.Warn("bootstrap admin created but RBAC role assignment failed", "error", err)
	}
	logger.Info("bootstrap admin created", "email", ba.Email)
}

// noopNotifier drops alerts; used when Telegram is not configured.
type noopNotifier struct{}

func (noopNotifier) SendAlert(founderAlerts.AlertEvent) error { return nil }

// founderConfigured reports whether Telegram founder alerting is configured.
func founderConfigured() bool {
	return os.Getenv("FOUNDER_TELEGRAM_BOT_TOKEN") != "" && os.Getenv("FOUNDER_TELEGRAM_CHAT_ID") != ""
}

// newFounderNotifier builds the Telegram notifier from env config,
// falling back to a noop notifier when Telegram is not configured.
func newFounderNotifier(logger *slog.Logger) founder.Notifier {
	token := os.Getenv("FOUNDER_TELEGRAM_BOT_TOKEN")
	chatID, err := strconv.ParseInt(os.Getenv("FOUNDER_TELEGRAM_CHAT_ID"), 10, 64)
	if token == "" || err != nil || chatID == 0 {
		if token != "" {
			logger.Warn("founder telegram notifier disabled: invalid FOUNDER_TELEGRAM_CHAT_ID")
		}
		return noopNotifier{}
	}
	bot, err := telebot.NewBot(telebot.Settings{Token: token})
	if err != nil {
		logger.Warn("founder telegram notifier unavailable", "error", err)
		return noopNotifier{}
	}
	logger.Info("founder telegram notifier enabled")
	return founderAlerts.NewTelegramBotNotifier(bot, chatID)
}

// runDailyDigest sends a daily founder report at FOUNDER_DIGEST_HOUR
// (default 9, UTC) until ctx is cancelled. Report metrics are zero-valued
// until a data source exists; the notification service is wired for
// future population.
func runDailyDigest(ctx context.Context, svc *founder.FounderService, logger *slog.Logger) {
	hour := 9
	if v := os.Getenv("FOUNDER_DIGEST_HOUR"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h >= 0 && h < 24 {
			hour = h
		}
	}
	logger.Info("daily founder digest scheduled", "hour", hour)
	for {
		next := time.Now().Truncate(24 * time.Hour).Add(time.Duration(hour) * time.Hour)
		if !next.After(time.Now()) {
			next = next.Add(24 * time.Hour)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := svc.SendDailyDigest(digest.DailyDigestReport{Date: time.Now()}); err != nil {
				logger.Error("daily founder digest failed", "error", err)
			}
		}
	}
}
