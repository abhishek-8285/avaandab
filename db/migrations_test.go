package db

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// openMigratedDB opens a fresh in-memory DB at the given goose version.
func openMigratedDB(t *testing.T, to int64) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("test_migrate_%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	require.NoError(t, goose.SetDialect("sqlite"))
	require.NoError(t, goose.UpTo(db, "migrations", to))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n))
	return n == 1
}

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

// Test00042GeofenceEngine_UpDownRoundTrip verifies the full 00042 migration
// applies cleanly and rolls back cleanly (Master Directive §4).
func Test00042GeofenceEngine_UpDownRoundTrip(t *testing.T) {
	// Up: apply everything up to and including 00042.
	db := openMigratedDB(t, 42)

	require.True(t, tableExists(t, db, "geofences"))
	require.True(t, tableExists(t, db, "vehicle_geofences"))
	require.True(t, tableExists(t, db, "geofence_events"))
	require.True(t, tableExists(t, db, "engine_state"))
	require.True(t, tableExists(t, db, "trip_detentions"))
	require.True(t, tableExists(t, db, "invoice_line_items"))
	require.True(t, tableExists(t, db, "company_config"))
	require.True(t, columnExists(t, db, "vehicles", "tank_capacity_litres"))
	require.True(t, columnExists(t, db, "vehicles", "fuel_sensor_fitted"))
	require.True(t, columnExists(t, db, "vehicles", "maintenance_due"))
	var idxN int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_telemetry_snapshots_vehicle_timestamp'`).Scan(&idxN))
	require.Equal(t, 1, idxN, "polling index must exist after up")

	// RBAC seeds.
	var n int
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM permissions WHERE name LIKE 'geofences:%'`).Scan(&n))
	require.Equal(t, 4, n)

	// Down: revert 00042 (and anything above it — none at version 42).
	require.NoError(t, goose.Down(db, "migrations"))
	require.False(t, tableExists(t, db, "geofences"))
	require.False(t, tableExists(t, db, "company_config"))
	require.False(t, tableExists(t, db, "trip_detentions"))
	require.False(t, columnExists(t, db, "vehicles", "tank_capacity_litres"))
	require.False(t, columnExists(t, db, "vehicles", "fuel_sensor_fitted"))
	require.False(t, columnExists(t, db, "vehicles", "maintenance_due"))
	require.NoError(t, db.QueryRow(
		`SELECT count(*) FROM permissions WHERE name LIKE 'geofences:%'`).Scan(&n))
	require.Equal(t, 0, n)

	// Re-up: the full chain must apply again after the rollback.
	require.NoError(t, goose.Up(db, "migrations"))
	require.True(t, tableExists(t, db, "geofences"))
	require.True(t, tableExists(t, db, "company_config"))
}
