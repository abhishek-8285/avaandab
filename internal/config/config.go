package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// RoutingConfig holds route optimization provider settings (Spec 18 Wave A).
type RoutingConfig struct {
	Provider string // mock | osrm-public | osrm-selfhost | http://...
	OSRMURL  string // override for self-host
}

// Config holds all application configuration.
type Config struct {
	AppEnv               string
	Port                 string
	DatabaseURL          string
	CookieSecret         string
	APITokenSecret       string
	SessionMaxAge        time.Duration
	CookieSecure         bool
	LogLevel             string
	UploadDir            string
	StaticDir            string
	MaxUploadSize        int64
	ExportMaxRows        int
	DashboardSSEEnabled  bool
	DashboardSSEInterval time.Duration
	PWAEnabled           bool
	RazorpayKeyID        string
	RazorpayKeySecret    string
	RazorpayWebhook      string
	BootstrapAdmin       BootstrapAdminConfig
	RAG                  RAGConfig
	Agent                AgentConfig
	Experiment           ExperimentConfig
	Telemetry            TelemetryConfig
	LiveMap              LiveMapConfig
	Alerts               AlertConfig
	EWayBill             EWayBillConfig
	GSTN                 GSTNConfig
	FASTag               FASTagConfig
	Routing              RoutingConfig
}

// GSTNConfig holds configuration for GSTN / GSP / E-Invoicing (Spec 07).
type GSTNConfig struct {
	UseMock      bool
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
}

// FASTagConfig holds configuration for FASTag NETC (Spec 21 §6).
type FASTagConfig struct {
	UseMock  bool
	APIKey   string
	Enabled  bool
	Endpoint string
}

// EWayBillConfig holds configuration for the E-Way Bill lifecycle worker (Spec 05 §7, Spec 07).
type EWayBillConfig struct {
	WorkerEnabled        bool
	WorkerInterval       time.Duration
	ExtensionKM          float64
	ExtensionLeadSeconds int
	MinInvoiceValue      float64
}

// AlertConfig holds configuration for operational alerts (Spec 05 §14).
type AlertConfig struct {
	TelegramBotToken string
	TelegramChatID   string
}

// ExperimentConfig configures the server-side A/B experiment framework.
type ExperimentConfig struct {
	// Rollout is the percentage (0-100) of users assigned the treatment
	// variant of an experiment. 0 = control only, 100 = treatment only.
	Rollout int
	// ForceVariant overrides assignment for every request (QA/testing).
	// Empty means no override.
	ForceVariant string
}

// AgentConfig holds configuration for the AI operations assistant.
type AgentConfig struct {
	Enabled         bool
	APIKey          string
	BaseURL         string
	Model           string
	MaxTurns        int
	SystemPrompt    string
	RLEnabled       bool
	RLDBPath        string
	RequireApproval bool
}

// RAGConfig holds configuration for the codebase RAG system.
type RAGConfig struct {
	Enabled          bool
	EmbeddingAPIKey  string
	EmbeddingBaseURL string
	EmbeddingModel   string
	ChunkSize        int
	ChunkOverlap     int
	IndexDirs        []string
	VectorDBPath     string
}

// TelemetryConfig holds configuration for the telemetry ingestion pipeline
// (Phase 1: Specs 01 §8, 17 §3).
type TelemetryConfig struct {
	Enabled                 bool
	WebhookSecretLocoNav    string
	WebhookSecretWheelsEye  string
	WheelsEyeAccessToken    string
	WheelsEyePollInterval   time.Duration
	DeviceSecretPepper      string
	WebhookRateLimit        int
	RawRetentionDays        int
	BatchSize               int
	FlushInterval           time.Duration
	OdometerMaxRegressionKM float64
	FuelClampDeltaPct       float64
}

// LiveMapConfig holds configuration for the live map + share links + ETA +
// preventive-maintenance stack (Spec 04 §9).
type LiveMapConfig struct {
	MapTileProvider       string // google | osm | auto (google → OSM on tileerror)
	MapGoogleStyle        string // m=roadmap, s=satellite, y=hybrid, p=terrain
	MapGL                 string // Google tile country bias
	MapOSMURL             string // OSM fallback tile template
	NominatimURL          string // Geocoding base
	MapPollSec            int    // Tracking page REST poll interval
	CSPEnabled            bool
	ShareLinkTTLHours     int
	ShareLinkMaxTTLHours  int
	ShareLinkMaxActive    int
	EtaStaleMin           int
	EtaWindowMin          int
	EtaGuardMaxRegressMin int
	TelemetryStaleMin     int
	SSEEnabled            bool
	SSEKeepaliveSec       int
	PMEnabled             bool
	PMCheckIntervalMin    int
	PMCriticalDTCs        string
}

// BootstrapAdminConfig configures the initial admin account created at
// startup when no admin exists yet. Intentionally not set by default.
type BootstrapAdminConfig struct {
	Email    string
	Name     string
	Password string
}

// Load reads configuration from environment variables.
func Load() *Config {
	maxUpload := int64(10 << 20) // 10 MB default
	if v := os.Getenv("MAX_UPLOAD_SIZE"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			slog.Error("invalid MAX_UPLOAD_SIZE", "value", v, "error", err)
		} else {
			maxUpload = parsed << 20
		}
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	sessionMaxAge := 24 * time.Hour
	if v := os.Getenv("SESSION_MAX_AGE"); v != "" {
		if hours, err := strconv.Atoi(v); err == nil {
			sessionMaxAge = time.Duration(hours) * time.Hour
		}
	}

	cookieSecure := env == "production"
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		cookieSecure = v == "true" || v == "1"
	}

	cfg := &Config{
		AppEnv:               env,
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "file:transport.db?mode=rwc&cache=shared&_foreign_keys=on&_journal_mode=WAL"),
		CookieSecret:         getEnv("COOKIE_SECRET", "dev-secret-key-change-in-production-32b!"),
		APITokenSecret:       getEnv("API_SECRET", ""),
		SessionMaxAge:        sessionMaxAge,
		CookieSecure:         cookieSecure,
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		UploadDir:            getEnv("UPLOAD_DIR", "./uploads"),
		StaticDir:            getEnv("STATIC_DIR", "internal/static"),
		MaxUploadSize:        maxUpload,
		ExportMaxRows:        getEnvInt("EXPORT_MAX_ROWS", 50000),
		DashboardSSEEnabled:  getEnvBool("DASHBOARD_SSE_ENABLED", true),
		DashboardSSEInterval: time.Duration(getEnvInt("DASHBOARD_SSE_INTERVAL_SEC", 5)) * time.Second,
		PWAEnabled:           getEnvBool("PWA_ENABLED", false),
		RazorpayKeyID:        getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:    getEnv("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhook:      os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		BootstrapAdmin: BootstrapAdminConfig{
			Email:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
			Name:     getEnv("BOOTSTRAP_ADMIN_NAME", "Admin"),
			Password: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		},
		RAG: RAGConfig{
			Enabled:          getEnv("RAG_ENABLED", "false") == "true",
			EmbeddingAPIKey:  os.Getenv("RAG_EMBEDDING_API_KEY"),
			EmbeddingBaseURL: getEnv("RAG_EMBEDDING_BASE_URL", "https://api.openai.com/v1"),
			EmbeddingModel:   getEnv("RAG_EMBEDDING_MODEL", "text-embedding-3-small"),
			ChunkSize:        getEnvInt("RAG_CHUNK_SIZE", 512),
			ChunkOverlap:     getEnvInt("RAG_CHUNK_OVERLAP", 50),
			VectorDBPath:     getEnv("RAG_VECTOR_DB_PATH", "./rag_vectors.db"),
		},
	}

	// Parse RAG index directories from comma-separated env var
	if dirs := os.Getenv("RAG_INDEX_DIRS"); dirs != "" {
		cfg.RAG.IndexDirs = strings.Split(dirs, ",")
		for i := range cfg.RAG.IndexDirs {
			cfg.RAG.IndexDirs[i] = strings.TrimSpace(cfg.RAG.IndexDirs[i])
		}
	}

	cfg.Agent = AgentConfig{
		Enabled:         getEnv("AGENT_ENABLED", "false") == "true",
		APIKey:          os.Getenv("AGENT_API_KEY"),
		BaseURL:         getEnv("AGENT_BASE_URL", "https://api.openai.com/v1"),
		Model:           getEnv("AGENT_MODEL", "gpt-4o-mini"),
		MaxTurns:        getEnvInt("AGENT_MAX_TURNS", 10),
		RLEnabled:       getEnv("AGENT_RL_ENABLED", "true") == "true",
		RLDBPath:        getEnv("AGENT_RL_DB_PATH", "agent_rl.db"),
		RequireApproval: getEnv("AGENT_REQUIRE_APPROVAL", "true") == "true",
	}

	cfg.Experiment = ExperimentConfig{
		Rollout:      getEnvInt("EXPERIMENT_ROLLOUT", 0),
		ForceVariant: getEnv("EXPERIMENT_FORCE_VARIANT", ""),
	}

	cfg.Telemetry = TelemetryConfig{
		Enabled:                 getEnv("TELEMETRY_ENABLED", "true") == "true",
		WebhookSecretLocoNav:    os.Getenv("TELEMETRY_WEBHOOK_SECRET_LOCONAV"),
		WebhookSecretWheelsEye:  os.Getenv("TELEMETRY_WEBHOOK_SECRET_WHEELSEYE"),
		WheelsEyeAccessToken:    os.Getenv("TELEMETRY_WHEELSEYE_ACCESS_TOKEN"),
		WheelsEyePollInterval:   getEnvDuration("TELEMETRY_WHEELSEYE_POLL_INTERVAL", 5*time.Minute),
		DeviceSecretPepper:      os.Getenv("TELEMETRY_DEVICE_SECRET_PEPPER"),
		WebhookRateLimit:        getEnvInt("TELEMETRY_WEBHOOK_RATE_LIMIT", 30),
		RawRetentionDays:        getEnvInt("TELEMETRY_RAW_RETENTION_DAYS", 30),
		BatchSize:               getEnvInt("TELEMETRY_BATCH_SIZE", 500),
		FlushInterval:           getEnvDuration("TELEMETRY_FLUSH_INTERVAL", 2*time.Second),
		OdometerMaxRegressionKM: getEnvFloat("TELEMETRY_ODOMETER_MAX_REGRESSION_KM", 1.0),
		FuelClampDeltaPct:       getEnvFloat("TELEMETRY_FUEL_CLAMP_DELTA_PCT", 5.0),
	}

	// Spec 04 §9 — live map + share links + ETA + preventive maintenance.
	cfg.LiveMap = LiveMapConfig{
		MapTileProvider:       getEnv("MAP_TILE_PROVIDER", "auto"),
		MapGoogleStyle:        getEnv("MAP_GOOGLE_STYLE", "m"),
		MapGL:                 getEnv("MAP_GL", "IN"),
		MapOSMURL:             getEnv("MAP_OSM_URL", "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"),
		NominatimURL:          getEnv("NOMINATIM_URL", "https://nominatim.openstreetmap.org"),
		MapPollSec:            getEnvInt("MAP_POLL_SEC", 10),
		CSPEnabled:            getEnv("CSP_ENABLED", "false") == "true",
		ShareLinkTTLHours:     getEnvInt("SHARE_LINK_TTL_HOURS", 24),
		ShareLinkMaxTTLHours:  getEnvInt("SHARE_LINK_MAX_TTL_HOURS", 168),
		ShareLinkMaxActive:    getEnvInt("SHARE_LINK_MAX_ACTIVE", 20),
		EtaStaleMin:           getEnvInt("ETA_STALE_MIN", 15),
		EtaWindowMin:          getEnvInt("ETA_WINDOW_MIN", 30),
		EtaGuardMaxRegressMin: getEnvInt("ETA_GUARD_MAX_REGRESS_MIN", 5),
		TelemetryStaleMin:     getEnvInt("TELEMETRY_STALE_MIN", 15),
		SSEEnabled:            getEnv("SSE_ENABLED", "true") == "true",
		SSEKeepaliveSec:       getEnvInt("SSE_KEEPALIVE_SEC", 15),
		PMEnabled:             getEnv("PM_ENABLED", "true") == "true",
		PMCheckIntervalMin:    getEnvInt("PM_CHECK_INTERVAL_MIN", 15),
		PMCriticalDTCs:        getEnv("PM_CRITICAL_DTCS", "P0A0F,P1602"),
	}

	// Spec 05 §14 — Operational alerts Telegram configuration.
	cfg.Alerts = AlertConfig{
		TelegramBotToken: getEnv("ALERT_TELEGRAM_BOT_TOKEN", os.Getenv("FOUNDER_TELEGRAM_BOT_TOKEN")),
		TelegramChatID:   getEnv("ALERT_TELEGRAM_CHAT_ID", os.Getenv("FOUNDER_TELEGRAM_CHAT_ID")),
	}

	// Spec 05 §7, Spec 07 — E-Way Bill lifecycle worker configuration.
	cfg.EWayBill = EWayBillConfig{
		WorkerEnabled:        getEnvBool("EWAYBILL_WORKER_ENABLED", true),
		WorkerInterval:       getEnvDuration("EWAYBILL_WORKER_INTERVAL", 60*time.Second),
		ExtensionKM:          getEnvFloat("EWAYBILL_EXTENSION_KM", 5.0),
		ExtensionLeadSeconds: getEnvInt("EWAYBILL_EXTENSION_LEAD_SECONDS", 14400),
		MinInvoiceValue:      getEnvFloat("EWAYBILL_MIN_INVOICE_VALUE", 50000.0),
	}

	// Spec 07 — GST E-Invoicing / GSTN configuration.
	cfg.GSTN = GSTNConfig{
		UseMock:      getEnvBool("INTEGRATION_GSTN_USE_MOCK", true),
		Username:     os.Getenv("INTEGRATION_GSTN_USERNAME"),
		Password:     os.Getenv("INTEGRATION_GSTN_PASSWORD"),
		ClientID:     os.Getenv("INTEGRATION_GSTN_CLIENT_ID"),
		ClientSecret: os.Getenv("INTEGRATION_GSTN_CLIENT_SECRET"),
	}

	// Spec 21 §6 — FASTag NETC configuration (fix: load INTEGRATION_FASTAG_USE_MOCK).
	cfg.FASTag = FASTagConfig{
		UseMock:  getEnvBool("INTEGRATION_FASTAG_USE_MOCK", true),
		APIKey:   getEnv("INTEGRATION_FASTAG_API_KEY", os.Getenv("FASTAG_API_KEY")),
		Enabled:  getEnvBool("INTEGRATION_FASTAG_ENABLED", false),
		Endpoint: getEnv("INTEGRATION_FASTAG_ENDPOINT", "https://api.fastag.org"),
	}

	// Spec 18 — Route optimization (Wave A)
	cfg.Routing = RoutingConfig{
		Provider: getEnv("ROUTING_PROVIDER", "mock"),
		OSRMURL:  getEnv("OSRM_URL", getEnv("ROUTING_OSRM_URL", "http://osrm.internal:5000")),
	}

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "error", err)
	}

	return cfg
}

func validateDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: invalid value %q: %w", raw, err)
	}
	if u.Scheme == "" {
		if !strings.HasSuffix(strings.ToLower(raw), ".db") &&
			!strings.HasSuffix(strings.ToLower(raw), ".sqlite") &&
			!strings.HasSuffix(strings.ToLower(raw), ".sqlite3") {
			return fmt.Errorf("DATABASE_URL: plain file path %q must end in .db, .sqlite or .sqlite3", raw)
		}
		return nil
	}
	if u.Scheme != "file" {
		return fmt.Errorf("DATABASE_URL: unsupported scheme %q in %q; use a plain file path or a file: URI", u.Scheme, raw)
	}
	return nil
}

// Validate checks the configuration for invalid values.
func (c *Config) Validate() error {
	return validateDatabaseURL(c.DatabaseURL)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return fallback
}

// IsProduction returns true if the app is running in production.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true if the app is running in development.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// UsingKnownDefaultSecret returns true when production would rely on known,
// committed default values for secrets instead of environment-provided ones.
func (c *Config) UsingKnownDefaultSecret() bool {
	if c.CookieSecret == "dev-secret-key-change-in-production-32b!" {
		return true
	}
	if c.CookieSecret == "dev-secret-32bytes-for-cookie-signing!" {
		return true
	}
	if c.APITokenSecret == "" {
		return true
	}
	return false
}
