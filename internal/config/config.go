package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	AppEnv            string
	Port              string
	DatabaseURL       string
	CookieSecret      string
	APITokenSecret    string
	SessionMaxAge     time.Duration
	CookieSecure      bool
	LogLevel          string
	UploadDir         string
	StaticDir         string
	MaxUploadSize     int64
	RazorpayKeyID     string
	RazorpayKeySecret string
	BootstrapAdmin    BootstrapAdminConfig
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
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
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

	return &Config{
		AppEnv:            env,
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "file:transport.db?mode=rwc&cache=shared&_foreign_keys=on&_journal_mode=WAL"),
		CookieSecret:      getEnv("COOKIE_SECRET", "dev-secret-key-change-in-production-32b!"),
		APITokenSecret:    getEnv("API_SECRET", ""),
		SessionMaxAge:     sessionMaxAge,
		CookieSecure:      cookieSecure,
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		StaticDir:         getEnv("STATIC_DIR", "internal/static"),
		MaxUploadSize:     maxUpload,
		RazorpayKeyID:     getEnv("RAZORPAY_KEY_ID", "rzp_test_TMdP3QXQq2L67c"),
		RazorpayKeySecret: getEnv("RAZORPAY_KEY_SECRET", "Fv17NyJHioQluynfHY59F0da"),
		BootstrapAdmin: BootstrapAdminConfig{
			Email:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
			Name:     getEnv("BOOTSTRAP_ADMIN_NAME", "Admin"),
			Password: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
	if c.APITokenSecret == "" && c.CookieSecret != "" {
		return true
	}
	if c.RazorpayKeyID == "rzp_test_TMdP3QXQq2L67c" && c.RazorpayKeySecret == "Fv17NyJHioQluynfHY59F0da" {
		return true
	}
	return false
}
