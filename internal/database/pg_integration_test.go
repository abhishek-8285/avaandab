//go:build pg_integration

// Postgres engine integration test — verifies the full migration set applies
// cleanly on a real PostgreSQL server via the appdb.Open factory + goose
// provider, mirroring cmd/server startup. Not part of default `go test ./...`:
//
//	DATABASE_URL="postgres://user:pass@localhost:5432/mvtms_it?sslmode=disable" \
//	    go test -tags pg_integration ./internal/database/ -run TestPostgresMigrations -v
//
// The database must exist; migrations create all tables. The test drops and
// recreates nothing — use a disposable scratch DB.
package database_test

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	dbmigr "transport-app/db"
	appdb "transport-app/internal/database"
)

func TestPostgresMigrations(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping postgres integration test")
	}

	s := &testSettings{
		driver:  "postgres",
		url:     url,
		maxOpen: 4,
		maxIdle: 2,
	}
	db, err := appdb.Open(context.Background(), s, slog.Default())
	if err != nil {
		t.Fatalf("Open(postgres) = %v, want nil", err)
	}
	defer func() { _ = db.Close() }()

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("Ping = %v, want nil (is the server reachable?)", err)
	}

	migrations, err := fsSub()
	if err != nil {
		t.Fatalf("embed fs: %v", err)
	}
	provider, err := goose.NewProvider(appdb.GooseDialect("postgres"), db, migrations)
	if err != nil {
		t.Fatalf("goose.NewProvider = %v", err)
	}
	if _, err := provider.Up(pingCtx); err != nil {
		t.Fatalf("migrations up on postgres = %v", err)
	}

	// Spot-check core tables exist.
	for _, tbl := range []string{"users", "drivers", "vehicles", "bookings", "trips", "files", "worker_leases"} {
		var n int
		if err := db.QueryRowContext(pingCtx,
			`SELECT count(*) FROM information_schema.tables WHERE table_name = $1`, tbl).Scan(&n); err != nil {
			t.Fatalf("table check %s: %v", tbl, err)
		}
		if n != 1 {
			t.Errorf("table %s missing after migrations", tbl)
		}
	}

	var version int
	if err := db.QueryRowContext(pingCtx,
		`SELECT max(version_id) FROM goose_db_version`).Scan(&version); err == nil && version < 80 {
		t.Logf("postgres migrated to version %d", version)
	}
}

func fsSub() (fs.FS, error) {
	return fs.Sub(dbmigr.Migrations, "migrations")
}
