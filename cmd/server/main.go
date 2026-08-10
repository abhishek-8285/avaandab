package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"

	dbmigr "transport-app/db"
	"transport-app/internal/auth"
	"transport-app/internal/config"
	"transport-app/internal/graphqlservice"
	"transport-app/internal/grpcservice"
	"transport-app/internal/handlers"
	"transport-app/internal/logging"
	"transport-app/internal/middleware"
	"transport-app/internal/mqttservice"
	"transport-app/internal/repository/sqlite"
	"transport-app/internal/service"

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
	"transport-app/internal/shared/clock"
	"transport-app/internal/shared/id"
	"transport-app/internal/shared/uow"

	"github.com/pressly/goose/v3"
)

func main() {
	cfg := config.Load()
	logging.Setup(cfg.LogLevel, cfg.AppEnv)

	logger := slog.Default()
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
	database.SetConnMaxLifetime(5 * time.Minute)

	database.SetConnMaxLifetime(15 * time.Minute)

	// Execute WAL mode & performance pragmas for extreme throughput
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=OFF;",
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
	authStore := auth.NewSessionStore(cfg.CookieSecret)

	// Initialize Casbin authorization service
	authSvc, err := auth.NewCasbinAuthorizationService(database)
	if err != nil {
		logger.Error("Failed to initialize Casbin authorization service", "error", err)
		os.Exit(1)
	}

	// Initialize handlers app
	app := handlers.NewApp(services, cfg, authStore, database, authSvc)

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

	// Sprint 4 – Payment use cases
	recordPayment := paymentApp.NewRecordPaymentUseCase(sqlUoW, idGen, realClock)
	getPayment := paymentApp.NewGetPaymentUseCase(sqlUoW)
	listPayments := paymentApp.NewListPaymentsUseCase(sqlUoW)

	// ── API handlers ──────────────────────────────────────────────────────
	bookingAPIHandler := bookingHandlers.NewAPIBookingHandler(
		createBookingUC, confirmBookingUC, cancelBookingUC, updateBookingUC, completeBookingUC, deleteBookingUC, getBookingUC, listBookingsUC,
		authSvc,
	)
	tripAPIHandler := tripHandlers.NewAPITripHandler(
		createTrip, assignDriver, assignVehicle, scheduleTrip, startTrip, reachPickup, startTransit, deliver, completeTrip, cancelTrip, getTrip, listTrips)
	invoiceAPIHandler := invoiceHandlers.NewAPIInvoiceHandler(generateInvoice, getInvoice, listInvoices)
	paymentAPIHandler := paymentHandlers.NewAPIPaymentHandler(recordPayment, getPayment, listPayments)

	// Setup router
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.Recoverer)
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

	// ── REST API v1 ───────────────────────────────────────────────────────
	apiSecret := []byte(cfg.CookieSecret) // same secret; rotate independently in prod
	authAPIHandler := authAPIHandlers.NewAPIAuthHandler(services.Auth, services.Users, apiSecret)

	// ── High-Performance Architecture Protocols ──────────────────────
	// 1. MQTT Broker Client Setup
	_ = mqttservice.NewMQTTBroker("tcp://localhost:1883")

	// 2. gRPC Dispatch Microservice
	grpcservice.StartGRPCServer("50051")

	// 3. GraphQL Query Endpoint
	r.Post("/query", graphqlservice.GraphQLHandler)
	r.Get("/graphql", graphqlservice.GraphQLHandler)

	// 4. Telemetry Batch Sync Endpoint
	r.Post("/api/v1/telemetry/sync", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		
		logs, _ := body["logs"].([]interface{})
		syncedIDs := make([]interface{}, 0)
		if logs != nil {
			for _, item := range logs {
				if m, ok := item.(map[string]interface{}); ok {
					if id, exists := m["id"]; exists {
						syncedIDs = append(syncedIDs, id)
					}
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":      true,
			"synced_count": len(syncedIDs),
			"synced_ids":   syncedIDs,
			"server_time":  time.Now().Format(time.RFC3339),
		})
	})

	// Public: token endpoint (no auth required)
	authAPIHandler.Register(r)

	// Protected: all other /api/v1/* routes require a valid session or Bearer token
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAPIAuth(authStore, apiSecret))
		bookingAPIHandler.Register(r)
		tripAPIHandler.Register(r)
		invoiceAPIHandler.Register(r)
		paymentAPIHandler.Register(r)
	})

	// Static files with Cache-Control headers
	fileServer := http.FileServer(http.Dir("internal/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})))

	// Uploaded files (logos, documents)
	uploadsServer := http.FileServer(http.Dir(cfg.UploadDir))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
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
		r.Post("/login", app.Auth.Login)
		r.Get("/register", app.Auth.RegisterPage)
		r.Post("/register", app.Auth.Register)
		r.Get("/forgot-password", app.Auth.ForgotPasswordPage)
		r.Post("/forgot-password", app.Auth.SubmitForgotPassword)
		r.Post("/logout", app.Auth.Logout)

		// Public Contact & Status Tracking
		r.Route("/contact-us", app.Contact.Routes)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(authStore))

			// Dashboard
			r.Get("/dashboard", app.Dashboard.Index)
			r.Get("/files/{id}", app.DownloadFile)

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

			// Profile (auth)
			r.Get("/profile", app.Auth.ProfilePage)
			r.Post("/profile", app.Auth.UpdateProfile)
			r.Get("/change-password", app.Auth.ChangePasswordPage)
			r.Post("/change-password", app.Auth.ChangePassword)
		})
	})

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("Server listening", "address", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}
