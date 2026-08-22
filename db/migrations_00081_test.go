package db

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestMigration00081TripStatusCheckRealign proves the trips CHECK rebuild both
// applies AND rolls back (Prove-It #4) against the full migration chain.
// The point of 00081: a trip in 'delivered' status must be storable — the
// pre-00081 CHECK rejected it.
func TestMigration00081TripStatusCheckRealign(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mig.db")
	database, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer database.Close()

	ctx := context.Background()
	migFS, err := fs.Sub(Migrations, "migrations")
	require.NoError(t, err)
	provider, err := goose.NewProvider(goose.DialectSQLite3, database, migFS)
	require.NoError(t, err)

	_, err = provider.Up(ctx)
	require.NoError(t, err)

	checkSQL := ""
	require.NoError(t, database.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='trips'`,
	).Scan(&checkSQL))
	for _, want := range []string{"reached_pickup", "in_transit", "delivered"} {
		require.True(t, strings.Contains(checkSQL, want), "CHECK should allow %q after up", want)
	}

	// The regression this migration fixes: 'delivered' must be writable.
	_, err = database.Exec(
		`INSERT INTO trips (id, trip_number, route_id, status, departure_time) VALUES ('t1', 'TR-00081', 'r1', 'assigned', datetime('now'))`,
	)
	require.NoError(t, err)
	_, err = database.Exec(`UPDATE trips SET status = 'delivered' WHERE id = 't1'`)
	require.NoError(t, err, "'delivered' must pass the realigned CHECK")

	// Roll back ONLY 00081 (DownTo 80) and verify the legacy behaviour returns.
	_, err = provider.DownTo(ctx, 80)
	require.NoError(t, err)

	var status string
	require.NoError(t, database.QueryRow(`SELECT status FROM trips WHERE id = 't1'`).Scan(&status))
	require.Equal(t, "completed", status, "down should map delivered → completed")

	require.NoError(t, database.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='trips'`,
	).Scan(&checkSQL))
	require.False(t, strings.Contains(checkSQL, `'in_transit'`), "down should restore legacy CHECK")
}
