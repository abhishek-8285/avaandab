package db

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"io/fs"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestMigration00083ErrorReportsUpAndDown(t *testing.T) {
	content, err := Migrations.ReadFile("migrations/00083_error_reports_incidents.sql")
	require.NoError(t, err)

	mapFS := fstest.MapFS{
		"00083_error_reports_incidents.sql": &fstest.MapFile{Data: content},
	}
	var fsys fs.FS = mapFS

	dbPath := filepath.Join(t.TempDir(), "mig.db")
	database, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer database.Close()

	ctx := context.Background()

	_, err = database.Exec(`
CREATE TABLE roles (id INTEGER PRIMARY KEY, name TEXT);
CREATE TABLE permissions (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT
);
CREATE TABLE role_permissions (
    role_id       INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);`)
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectSQLite3, database, fsys)
	require.NoError(t, err)

	_, err = provider.Up(ctx)
	require.NoError(t, err)

	for _, q := range []string{
		`SELECT COUNT(*) FROM error_reports`,
		`SELECT COUNT(*) FROM incidents`,
		`SELECT COUNT(*) FROM permissions WHERE name IN ('errors:read','errors:update')`,
	} {
		var n int
		require.NoError(t, database.QueryRow(q).Scan(&n), q)
	}

	_, err = provider.DownTo(ctx, 0)
	require.NoError(t, err)

	for _, table := range []string{"error_reports", "incidents"} {
		var name string
		err := database.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		require.ErrorIs(t, err, sql.ErrNoRows, "table %s should be dropped", table)
	}

	var perms int
	require.NoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM permissions WHERE name LIKE 'errors:%'`,
	).Scan(&perms))
	require.Zero(t, perms, "seeded permissions should be removed on down")
}
