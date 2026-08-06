package test

import (
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"transport-app/internal/config"
	"transport-app/internal/repository/sqlite"
	"transport-app/internal/service"
)

// NewTestDB creates an in-memory SQLite database with migrations applied.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?cache=shared&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	goose.SetDialect("sqlite")
	if err := goose.Up(db, "../db/migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

// NewTestServices creates service instances backed by a real SQLite test database.
func NewTestServices(t *testing.T, db *sql.DB) *service.Services {
	t.Helper()

	repo := sqlite.NewRepository(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return service.NewServices(repo, loadTestConfig(), logger)
}

func loadTestConfig() *config.Config {
	return &config.Config{
		AppEnv:        "testing",
		Port:          "8080",
		DatabaseURL:   "file::memory:?cache=shared",
		CookieSecret:  "test-secret-32bytes-long-enough!",
		SessionMaxAge: 24 * 3600 * 1000000000,
		LogLevel:      "error",
		UploadDir:     "./uploads",
		MaxUploadSize: 10 << 20,
	}
}

// NewTestRepo creates a real repository backed by a test database.
func NewTestRepo(t *testing.T, db *sql.DB) *sqlite.SQLRepository {
	t.Helper()
	return sqlite.NewRepository(db)
}
