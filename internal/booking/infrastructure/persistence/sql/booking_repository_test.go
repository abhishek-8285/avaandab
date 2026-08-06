package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"

	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
)

func TestBookingRepository_SaveAndFind(t *testing.T) {
	dbConn, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	defer dbConn.Close()

	// Set up simple sqlite schema for testing
	_, err = dbConn.Exec(`
		CREATE TABLE customers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			company TEXT,
			phone TEXT,
			email TEXT,
			address TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE routes (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			destination TEXT NOT NULL,
			distance REAL,
			duration REAL,
			standard_fare REAL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE bookings (
			id TEXT PRIMARY KEY,
			booking_number TEXT NOT NULL UNIQUE,
			customer_id TEXT NOT NULL,
			pickup_date DATETIME NOT NULL,
			route_id TEXT NOT NULL,
			vehicle_type TEXT NOT NULL,
			passengers INTEGER NOT NULL DEFAULT 1,
			cargo_weight REAL,
			price REAL NOT NULL,
			notes TEXT,
			status TEXT NOT NULL,
			tenant_id TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (customer_id) REFERENCES customers(id),
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

	_, err = dbConn.Exec(`INSERT INTO customers (id, name, company) VALUES ('cust-1', 'Alice', 'ACME')`)
	assert.NoError(t, err)
	_, err = dbConn.Exec(`INSERT INTO routes (id, source, destination) VALUES ('route-1', 'A', 'B')`)
	assert.NoError(t, err)

	repo := NewBookingRepository(dbConn)
	ctx := context.Background()

	price := shared.FloatToMoney(150.0, "USD")
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	agg := aggregate.NewBookingAggregate(
		"bk-1",
		"1",
		"BK-0001",
		"cust-1",
		"route-1",
		now.Add(24*time.Hour),
		"Van",
		4,
		nil,
		price,
		"Testing",
		now,
	)

	err = repo.Save(ctx, agg)
	assert.NoError(t, err)

	found, err := repo.Find(ctx, "bk-1", "1")
	assert.NoError(t, err)
	assert.Equal(t, agg.BookingNumber, found.BookingNumber)
	assert.Equal(t, agg.Notes, found.Notes)
	assert.Equal(t, agg.Status, found.Status)

	// Test read model
	readModel, err := repo.GetReadModel(ctx, "bk-1", "1")
	assert.NoError(t, err)
	assert.Equal(t, "Alice", readModel.CustomerName)
	assert.Equal(t, "A", readModel.RouteSource)
	assert.Equal(t, "B", readModel.RouteDestination)
}
