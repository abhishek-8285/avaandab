package config_test

import (
	"os"
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
