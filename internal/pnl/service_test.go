package pnl

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
	"transport-app/internal/shared"
)

func TestCalculateUsesTelemetryAndApprovedCosts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, schema := range []string{
		`CREATE TABLE trips (id TEXT PRIMARY KEY, booking_id TEXT, vehicle_id TEXT, route_id TEXT, tenant_id TEXT, estimated_margin REAL, fuel_consumed_liters REAL, toll_costs REAL, last_pnl_update DATETIME, fuel_cost_low REAL, fuel_cost_high REAL, margin_low REAL, margin_high REAL, pnl_confidence TEXT, fuel_cost_status TEXT)`,
		`CREATE TABLE bookings (id TEXT PRIMARY KEY, price REAL)`,
		`CREATE TABLE vehicles (id TEXT PRIMARY KEY, fuel_type TEXT, current_mileage REAL)`,
		`CREATE TABLE routes (id TEXT PRIMARY KEY, source TEXT)`,
		`CREATE TABLE telemetry_snapshots (trip_id TEXT, odometer REAL)`,
		`CREATE TABLE fuel_prices (tenant_id TEXT, diesel_price REAL, petrol_price REAL, updated_at DATETIME)`,
		`CREATE TABLE driver_expenses (trip_id TEXT, expense_type TEXT, category TEXT, amount REAL, status TEXT, approved INTEGER)`,
	} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = db.Exec(`INSERT INTO trips (id, booking_id, vehicle_id, route_id, tenant_id, estimated_margin, fuel_consumed_liters, toll_costs, last_pnl_update) VALUES ('trip-1','booking-1','vehicle-1','route-1','tenant-1',0,0,0,NULL)`)
	_, _ = db.Exec(`INSERT INTO bookings VALUES ('booking-1',10000)`)
	_, _ = db.Exec(`INSERT INTO vehicles VALUES ('vehicle-1','diesel',5)`)
	_, _ = db.Exec(`INSERT INTO routes VALUES ('route-1','Mumbai')`)
	_, _ = db.Exec(`INSERT INTO telemetry_snapshots VALUES ('trip-1',1000),('trip-1',1100)`)
	_, _ = db.Exec(`INSERT INTO fuel_prices VALUES ('tenant-1',90,105,CURRENT_TIMESTAMP)`)
	_, _ = db.Exec(`INSERT INTO driver_expenses VALUES ('trip-1','toll','toll',500,'approved',0),('trip-1','food','food',300,'approved',0)`)

	ctx := shared.ContextWithTenantID(context.Background(), "tenant-1")
	got, err := NewService(db).Calculate(ctx, "trip-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.FuelConsumedLiters != 20 || got.FuelCost != 1800 || got.TollCost != 500 || got.KharchaApproved != 300 || got.EstimatedMargin != 7400 || got.MarginLow != 7130 || got.MarginHigh != 7670 {
		t.Fatalf("unexpected P&L: %+v", got)
	}
	if got.MarginPercentage != 74 || got.LowMargin {
		t.Fatalf("unexpected margin state: %+v", got)
	}
}
