package db_test

import (
	"context"
	"database/sql"
	"io/fs"
	"testing"

	_ "modernc.org/sqlite"

	dbmigr "transport-app/db"

	"github.com/pressly/goose/v3"
)

func TestMigrationsUpAndDown(t *testing.T) {
	database, err := sql.Open("sqlite", "file:migrations_cycle_test.db?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	migrations, err := fs.Sub(dbmigr.Migrations, "migrations")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, database, migrations)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	ctx := context.Background()

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	if err := database.Ping(); err != nil {
		t.Fatalf("db unreachable after up: %v", err)
	}

	// Take the database all the way back down.
	if _, err := provider.DownTo(ctx, 0); err != nil {
		t.Fatalf("migrate down: %v", err)
	}

	// The schema should be fully gone after DownTo(0); only goose's own
	// bookkeeping table may remain.
	var count int
	if err := database.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'goose_db_version'`).Scan(&count); err != nil {
		t.Fatalf("query tables after down: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 tables after full down, got %d", count)
	}

	// And the cycle should repeat cleanly.
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("migrate up again: %v", err)
	}
}
