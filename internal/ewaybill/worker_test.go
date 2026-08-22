package ewaybill

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"transport-app/internal/events"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE routes (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			destination TEXT NOT NULL,
			distance REAL NOT NULL,
			estimated_hours REAL NOT NULL,
			standard_fare REAL NOT NULL,
			tenant_id TEXT NOT NULL DEFAULT 'tenant-1'
		);
		CREATE TABLE vehicles (
			id TEXT PRIMARY KEY,
			registration_number TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'available',
			tenant_id TEXT NOT NULL DEFAULT 'tenant-1'
		);
		CREATE TABLE trips (
			id TEXT PRIMARY KEY,
			trip_number TEXT NOT NULL,
			route_id TEXT NOT NULL,
			vehicle_id TEXT,
			driver_id TEXT,
			status TEXT NOT NULL DEFAULT 'scheduled',
			eway_bill_ref TEXT,
			tenant_id TEXT NOT NULL DEFAULT 'tenant-1'
		);
		CREATE TABLE eway_bills (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			ewb_number TEXT NOT NULL,
			generation_date DATETIME NOT NULL,
			valid_until DATETIME NOT NULL,
			vehicle_number TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			cancelled_at DATETIME,
			cancellation_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE telemetry_snapshots (
			vehicle_id TEXT PRIMARY KEY,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			speed REAL NOT NULL,
			fuel_level REAL NOT NULL,
			odometer REAL NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE geofences (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			fence_type TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			radius_meters REAL NOT NULL,
			tenant_id TEXT NOT NULL DEFAULT 'tenant-1'
		);
	`)
	require.NoError(t, err)
	return db
}

func TestEWayBillWorker_SchemaTolerance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bus := events.NewInMemoryBus()
	worker := NewWorker(db, bus, nil, nil, Config{
		Interval: 100 * time.Millisecond,
	})

	ctx := context.Background()

	// 1. Without 00047 column 'part_b_updated_at', schemaReady returns false
	assert.False(t, worker.SchemaReady(ctx))

	// Worker tick must not panic or error
	worker.Tick(ctx)

	// 2. Add 00047 column
	_, err := db.Exec(`ALTER TABLE eway_bills ADD COLUMN part_b_updated_at DATETIME;`)
	require.NoError(t, err)

	assert.True(t, worker.SchemaReady(ctx))
}

func TestEWayBillWorker_FullLifecycleWith00047(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Add 00047 columns and events table
	_, err := db.Exec(`
		ALTER TABLE eway_bills ADD COLUMN part_b_updated_at DATETIME;
		CREATE TABLE eway_bill_events (
			id TEXT PRIMARY KEY,
			eway_bill_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			details TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	bus := events.NewInMemoryBus()
	worker := NewWorker(db, bus, nil, nil, Config{
		Interval:    100 * time.Millisecond,
		ExtensionKM: 5.0,
	})

	ctx := context.Background()
	require.True(t, worker.SchemaReady(ctx))

	// Seed route, vehicle, trip
	_, err = db.Exec(`INSERT INTO routes (id, source, destination, distance, estimated_hours, standard_fare) VALUES ('r-1', 'Mumbai', 'Pune', 150.0, 4.0, 5000.0)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO vehicles (id, registration_number) VALUES ('v-1', 'MH12AB1234')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO trips (id, trip_number, route_id, vehicle_id, status) VALUES ('t-1', 'TRP-101', 'r-1', 'v-1', 'started')`)
	require.NoError(t, err)

	// 1. Generate Missing
	worker.Tick(ctx)

	var ewbID, ewbNum, vehNum, status string
	var validUntil time.Time
	err = db.QueryRow(`SELECT id, ewb_number, vehicle_number, status, valid_until FROM eway_bills WHERE trip_id = 't-1'`).Scan(&ewbID, &ewbNum, &vehNum, &status, &validUntil)
	require.NoError(t, err)
	assert.NotEmpty(t, ewbNum)
	assert.Equal(t, "MH12AB1234", vehNum)
	assert.Equal(t, "active", status)

	// 2. Extension Gate: Vehicle position not near destination -> Extension denied
	_, err = db.Exec(`INSERT INTO geofences (id, name, entity_id, fence_type, latitude, longitude, radius_meters) VALUES ('g-1', 'Pune Hub', 'r-1', 'destination', 18.5204, 73.8567, 500)`)
	require.NoError(t, err)
	// Vehicle is in Mumbai (far away)
	_, err = db.Exec(`INSERT INTO telemetry_snapshots (vehicle_id, latitude, longitude, speed, fuel_level, odometer, updated_at) VALUES ('v-1', 19.0760, 72.8777, 60.0, 80.0, 1000.0, datetime('now'))`)
	require.NoError(t, err)

	msg, err := worker.ExtendForTrip(ctx, "t-1")
	require.NoError(t, err)
	assert.Contains(t, msg, "Extension denied")

	// Now move vehicle to within 2 km of destination (Pune)
	_, err = db.Exec(`UPDATE telemetry_snapshots SET latitude = 18.5250, longitude = 73.8580 WHERE vehicle_id = 'v-1'`)
	require.NoError(t, err)

	msg, err = worker.ExtendForTrip(ctx, "t-1")
	require.NoError(t, err)
	assert.Contains(t, msg, "Successfully extended")

	var newValidUntil time.Time
	err = db.QueryRow(`SELECT valid_until FROM eway_bills WHERE id = ?`, ewbID).Scan(&newValidUntil)
	require.NoError(t, err)
	assert.True(t, newValidUntil.After(validUntil))

	// 3. Cancel EWB on trip cancellation
	_, err = db.Exec(`UPDATE trips SET status = 'cancelled' WHERE id = 't-1'`)
	require.NoError(t, err)

	worker.Tick(ctx)

	var cancelStatus, cancelReason string
	err = db.QueryRow(`SELECT status, cancellation_reason FROM eway_bills WHERE id = ?`, ewbID).Scan(&cancelStatus, &cancelReason)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelStatus)
	assert.Equal(t, "Trip Cancelled", cancelReason)
}

func TestIsAutoGenerateEnabled_ReadsCompanyConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := db.Exec(`CREATE TABLE company_config (
		tenant_id TEXT NOT NULL DEFAULT '1',
		key TEXT NOT NULL,
		value TEXT,
		updated_at DATETIME,
		PRIMARY KEY (tenant_id, key))`)
	require.NoError(t, err)

	svc := NewEWayBillService(db, nil, nil, nil, Config{})
	ctx := context.Background()

	// No row → spec default: enabled
	assert.True(t, svc.isAutoGenerateEnabled(ctx), "missing config row must default to true")

	// Explicit false must be honoured (regression: old query read wrong column
	// names and always fell through to the default)
	_, err = db.Exec(`INSERT INTO company_config (tenant_id, key, value) VALUES ('1', 'ewaybill_auto_generate', 'false')`)
	require.NoError(t, err)
	assert.False(t, svc.isAutoGenerateEnabled(ctx), "config value 'false' must disable auto-generate")

	_, err = db.Exec(`UPDATE company_config SET value = 'true' WHERE key = 'ewaybill_auto_generate'`)
	require.NoError(t, err)
	assert.True(t, svc.isAutoGenerateEnabled(ctx))
}
