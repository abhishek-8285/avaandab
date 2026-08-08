package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"

	"transport-app/internal/trip/domain/aggregate"
)

func TestTripRepository_SaveAndFind(t *testing.T) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	defer dbConn.Close()

	// Set up simple sqlite schema for testing
	_, err = dbConn.Exec(`
		CREATE TABLE drivers (
			id TEXT PRIMARY KEY,
			driver_id TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT
		);
		CREATE TABLE vehicles (
			id TEXT PRIMARY KEY,
			registration_number TEXT,
			vehicle_number TEXT
		);
		CREATE TABLE routes (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			destination TEXT NOT NULL
		);
		CREATE TABLE trips (
			id TEXT PRIMARY KEY,
			trip_number TEXT NOT NULL UNIQUE,
			booking_id TEXT,
			driver_id TEXT,
			vehicle_id TEXT,
			route_id TEXT NOT NULL,
			departure_time DATETIME NOT NULL,
			arrival_time DATETIME,
			status TEXT NOT NULL,
			remarks TEXT,
			started_at DATETIME,
			reached_pickup_at DATETIME,
			in_transit_at DATETIME,
			delivered_at DATETIME,
			completed_at DATETIME,
			tenant_id TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (driver_id) REFERENCES drivers(id),
			FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
			FOREIGN KEY (route_id) REFERENCES routes(id)
		);
		CREATE TABLE outbox_events (
			id TEXT PRIMARY KEY,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			event_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			published_at DATETIME
		);
	`)
	assert.NoError(t, err)

	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination) VALUES ('route-1', 'A', 'B')`)
	assert.NoError(t, err)

	repo := NewTripRepository(dbConn)
	ctx := context.Background()

	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	agg := aggregate.NewTripAggregate(
		"tr-1",
		"1",
		"TR-0001",
		nil,
		"route-1",
		now.Add(2*time.Hour),
		"Testing remarks",
		now,
	)

	err = repo.Save(ctx, agg)
	assert.NoError(t, err)

	found, err := repo.Find(ctx, "tr-1", "1")
	assert.NoError(t, err)
	assert.Equal(t, agg.TripNumber, found.TripNumber)
	assert.Equal(t, agg.Remarks, found.Remarks)
	assert.Equal(t, agg.Status, found.Status)
}

func TestTripRepository_SearchReadModels_NullFields(t *testing.T) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		CREATE TABLE drivers (
			id TEXT PRIMARY KEY,
			driver_id TEXT NOT NULL,
			first_name TEXT,
			last_name TEXT
		);
		CREATE TABLE vehicles (
			id TEXT PRIMARY KEY,
			registration_number TEXT,
			vehicle_number TEXT
		);
		CREATE TABLE routes (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			destination TEXT NOT NULL
		);
		CREATE TABLE trips (
			id TEXT PRIMARY KEY,
			trip_number TEXT NOT NULL UNIQUE,
			booking_id TEXT,
			driver_id TEXT,
			vehicle_id TEXT,
			route_id TEXT NOT NULL,
			departure_time DATETIME NOT NULL,
			arrival_time DATETIME,
			status TEXT NOT NULL,
			remarks TEXT,
			started_at DATETIME,
			reached_pickup_at DATETIME,
			in_transit_at DATETIME,
			delivered_at DATETIME,
			completed_at DATETIME,
			tenant_id TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (driver_id) REFERENCES drivers(id),
			FOREIGN KEY (vehicle_id) REFERENCES vehicles(id),
			FOREIGN KEY (route_id) REFERENCES routes(id)
		);
		CREATE TABLE outbox_events (
			id TEXT PRIMARY KEY,
			aggregate_id TEXT NOT NULL,
			aggregate_type TEXT NOT NULL,
			event_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			published_at DATETIME
		);
	`)
	assert.NoError(t, err)

	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination) VALUES ('route-1', 'Origin', 'Destination')`)
	assert.NoError(t, err)

	// Insert trip with NULL vehicle_id, NULL driver_id, and NULL booking_id
	_, err = dbConn.Exec(`
		INSERT INTO trips (id, trip_number, booking_id, driver_id, vehicle_id, route_id, departure_time, status, tenant_id, version)
		VALUES ('trip-unassigned', 'TR-999', NULL, NULL, NULL, 'route-1', CURRENT_TIMESTAMP, 'scheduled', '1', 1)
	`)
	assert.NoError(t, err)

	repo := NewTripRepository(dbConn)
	ctx := context.Background()

	models, total, err := repo.SearchReadModels(ctx, "1", "", "", 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, models, 1)
	assert.Nil(t, models[0].VehicleID)
	assert.Nil(t, models[0].DriverID)
	assert.Nil(t, models[0].BookingID)
}

