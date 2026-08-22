package db

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestMigration00081TripStatusCheckRealign proves the trips CHECK rebuild both
// applies AND rolls back (Prove-It #4). The whole point of 00081: a trip in
// 'delivered' status must be storable — the pre-00081 CHECK rejected it.
func TestMigration00081TripStatusCheckRealign(t *testing.T) {
	content, err := Migrations.ReadFile("migrations/00081_trip_status_check_realign.sql")
	require.NoError(t, err)

	mapFS := fstest.MapFS{
		"00081_trip_status_check_realign.sql": &fstest.MapFile{Data: content},
	}
	var fsys fs.FS = mapFS

	dbPath := filepath.Join(t.TempDir(), "mig.db")
	database, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer database.Close()

	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectSQLite3, database, fsys)
	require.NoError(t, err)

	_, err = provider.Up(ctx)
	require.NoError(t, err)

	// Seed minimal parents for FKs.
	for _, q := range []string{
		`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare) VALUES ('r1', 'A', 'B', 10.0, 1.0, 100.0)`,
		`INSERT INTO trips (id, trip_number, route_id, status) VALUES ('t1', 'TR-00081', 'r1', 'assigned')`,
	} {
		_, err = database.Exec(q)
		require.NoError(t, err, q)
	}

	// The regression this migration fixes: 'delivered' must be writable.
	_, err = database.Exec(`UPDATE trips SET status = 'delivered' WHERE id = 't1'`)
	require.NoError(t, err, "'delivered' must pass the realigned CHECK")

	var status string
	require.NoError(t, database.QueryRow(`SELECT status FROM trips WHERE id = 't1'`).Scan(&status))
	require.Equal(t, "delivered", status)

	// Down restores the legacy CHECK; newer statuses are mapped to 'completed'.
	_, err = provider.DownTo(ctx, 0)
	require.NoError(t, err)

	require.NoError(t, database.QueryRow(`SELECT status FROM trips WHERE id = 't1'`).Scan(&status))
	require.Equal(t, "completed", status, "down should map delivered → completed")

	var n int
	require.NoError(t, database.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='trips_rebuild_00081'`,
	).Scan(&n))
	require.Zero(t, n, "rebuild table should be dropped")
}
