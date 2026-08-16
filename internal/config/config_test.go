package config_test

import (
	"os"
	"strings"
	"testing"

	"transport-app/internal/config"
)

func TestLoad_DefaultsAndEnv(t *testing.T) {
	_ = os.Setenv("APP_ENV", "production")
	_ = os.Setenv("PORT", "9090")
	_ = os.Setenv("MAX_UPLOAD_SIZE", "20")
	_ = os.Setenv("SESSION_MAX_AGE", "48")

	cfg := config.Load()

	if !cfg.IsProduction() || cfg.IsDevelopment() {
		t.Fatalf("expected production environment")
	}

	if cfg.Port != "9090" {
		t.Fatalf("expected port 9090, got %s", cfg.Port)
	}

	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("MAX_UPLOAD_SIZE")
	_ = os.Unsetenv("SESSION_MAX_AGE")

	devCfg := config.Load()
	if !devCfg.IsDevelopment() || devCfg.IsProduction() {
		t.Fatalf("expected development environment default")
	}
}

func TestValidate_DatabaseURL(t *testing.T) {
	valid := []string{
		"file:transport.db?mode=rwc&cache=shared&_foreign_keys=on&_journal_mode=WAL",
		"file:/abs/path/data.sqlite",
		"file:path.db",
		"transport.db",
		"./uploads/transport.sqlite3",
		"data/transport.db",
		"UPPER.DB",
	}
	for _, raw := range valid {
		cfg := &config.Config{DatabaseURL: raw}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", raw, err)
		}
	}

	invalid := []string{
		"http://example.com/db.sqlite",
		"postgres://user:pass@host/db",
		"mysql://host/db",
		"notes.txt",
	}
	for _, raw := range invalid {
		cfg := &config.Config{DatabaseURL: raw}
		err := cfg.Validate()
		if err == nil {
			t.Errorf("Validate(%q) = nil, want error", raw)
			continue
		}
		if !strings.Contains(err.Error(), "DATABASE_URL") {
			t.Errorf("Validate(%q) error %q missing key name DATABASE_URL", raw, err)
		}
	}
}

func TestLoad_MaxUploadSizeMalformedKeepsDefault(t *testing.T) {
	_ = os.Setenv("MAX_UPLOAD_SIZE", "not-a-number")
	defer os.Unsetenv("MAX_UPLOAD_SIZE")

	cfg := config.Load()
	if cfg.MaxUploadSize != int64(10<<20) {
		t.Fatalf("expected default max upload size %d, got %d", int64(10<<20), cfg.MaxUploadSize)
	}
}
