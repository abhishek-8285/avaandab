package fuel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"transport-app/internal/shared/uow"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SQLite DB with all migrations applied.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("test_fuel_%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)
	_ = goose.SetDialect("sqlite")
	require.NoError(t, goose.Up(db, "../../db/migrations"))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedVehicle inserts a vehicle with fuel-sensor columns (00042).
func seedVehicle(t *testing.T, db *sql.DB, sensorFitted bool, capacity float64) {
	t.Helper()
	sensor := 0
	if sensorFitted {
		sensor = 1
	}
	_, err := db.Exec(`INSERT INTO vehicles
		(id, registration_number, vehicle_number, vehicle_type, capacity, fuel_type,
		 insurance_expiry, fitness_expiry, permit_expiry, status, tank_capacity_litres, fuel_sensor_fitted)
		VALUES ('v1', 'KA01AB1234', 'KA01AB1234', 'truck', 2000, 'diesel',
		        '2027-01-01', '2027-01-01', '2027-01-01', 'available', ?, ?)`, capacity, sensor)
	require.NoError(t, err)
}

// seedDriver inserts a driver so behaviour-event FKs resolve.
func seedDriver(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO drivers
		(id, driver_id, first_name, last_name, phone, license_number, license_expiry, status)
		VALUES ('d1', 'D-001', 'Ravi', 'Kumar', '9988776655', 'KA-12345', '2028-01-01', 'available')`)
	require.NoError(t, err)
}

// seedTrip inserts a trip with the given status.
func seedTrip(t *testing.T, db *sql.DB, status string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO trips (id, trip_number, route_id, departure_time, status, driver_id)
		VALUES ('t1', 'TRIP-001', 'r1', '2026-08-19 09:00:00', ?, 'd1')`, status)
	require.NoError(t, err)
}

// insertSnapshots feeds synthetic telemetry_snapshots rows.
func insertSnapshots(t *testing.T, db *sql.DB, snaps []snapshotRow) {
	t.Helper()
	for _, s := range snaps {
		_, err := db.Exec(`INSERT INTO telemetry_snapshots
			(id, trip_id, vehicle_id, timestamp, speed, fuel_level, odometer, driver_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			s.id, orNil(s.tripID), s.vehicleID, timeStr(s.ts), s.speed, s.fuelLevel, s.odometer, orNil(s.driverID))
		require.NoError(t, err)
	}
}

// buildEngine wires a FuelEngine over the real DB with a fixed clock so the
// active-vehicle window (now - gap_tolerance) always includes the fixtures.
func buildEngine(t *testing.T, db *sql.DB, t0 time.Time) *FuelEngine {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	e := NewEngine(db, uow.NewSQLUnitOfWork(db), NewConfigReader(db), logger)
	// now is set just past the first fixture so now-30min stays behind every
	// snapshot while the 30-minute gap reset inside the pipeline never trips.
	e.now = func() time.Time { return t0.Add(time.Minute) }
	return e
}

func orNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// feedScenario inserts the snapshots, runs one tick and returns consumed count.
func feedScenario(t *testing.T, db *sql.DB, e *FuelEngine, snaps []snapshotRow) int {
	t.Helper()
	insertSnapshots(t, db, snaps)
	handled, err := e.Tick(context.Background())
	require.NoError(t, err)
	return handled
}

// fuelEventsOf returns the event_types recorded for the vehicle.
func fuelEventsOf(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	rows, err := db.Query(`SELECT event_type, count(*) FROM fuel_events WHERE vehicle_id = 'v1' GROUP BY event_type`)
	require.NoError(t, err)
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var et string
		var n int
		require.NoError(t, rows.Scan(&et, &n))
		out[et] = n
	}
	require.NoError(t, rows.Err())
	return out
}

// outboxAlerts returns the decoded FUEL alert payloads in the outbox.
func outboxAlerts(t *testing.T, db *sql.DB) []map[string]interface{} {
	t.Helper()
	rows, err := db.Query(`SELECT payload FROM outbox_events WHERE event_type = 'AlertEvent'`)
	require.NoError(t, err)
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var raw string
		require.NoError(t, rows.Scan(&raw))
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &m))
		out = append(out, m)
	}
	require.NoError(t, rows.Err())
	return out
}

func TestEngine_RefillDetected(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// 25L jump (40% → 65%), sustained across readings 2-3.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 40, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 65, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 65, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 1, events["refill_detected"])

	var est float64
	require.NoError(t, db.QueryRow(`SELECT estimated_litres FROM fuel_events WHERE event_type = 'refill_detected'`).Scan(&est))
	assert.InDelta(t, 25.0, est, 0.01)

	alerts := outboxAlerts(t, db)
	require.Len(t, alerts, 1)
	assert.Equal(t, "FUEL", alerts[0]["category"])
	assert.Equal(t, "Fuel refill detected", alerts[0]["title"])
}

func TestEngine_TheftSuspected(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// 12L drop (50% → 38%) while stationary on an active trip.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 1, events["drain_theft_suspected"])

	var sev string
	var w float64
	require.NoError(t, db.QueryRow(
		`SELECT severity, weight FROM driver_behaviour_events WHERE event_type = 'fuel_theft_suspicion'`).Scan(&sev, &w))
	assert.Equal(t, "high", sev)
	assert.Equal(t, 25.0, w)

	alerts := outboxAlerts(t, db)
	require.Len(t, alerts, 1)
	assert.Equal(t, "HIGH", alerts[0]["priority"])
}

func TestEngine_SiphonConfirmed(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// 16L drop (50% → 34%) during a 21-minute stationary stop.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(10 * time.Minute), speed: 0, fuelLevel: 34, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(21 * time.Minute), speed: 0, fuelLevel: 34, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 1, events["siphon_confirmed"])

	alerts := outboxAlerts(t, db)
	require.Len(t, alerts, 1)
	assert.Equal(t, "CRITICAL", alerts[0]["priority"])
	assert.Equal(t, "Fuel siphon confirmed", alerts[0]["title"])
}

func TestEngine_AbnormalDrainWhileMoving(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// Driving 10 km (odo 100→110) at 60 km/h while losing 8L — expected burn
	// = 5 km × 0.6 L/km × 1.3 margin = 3.9L, so 8L is abnormal.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 60, fuelLevel: 50, odometer: 100},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Minute), speed: 60, fuelLevel: 42, odometer: 105},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Minute), speed: 60, fuelLevel: 40, odometer: 110},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 1, events["abnormal_drain"])
}

func TestEngine_OdometerRollbackResetsBaseline(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 50, odometer: 990},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 1, events["odometer_rollback"])

	var beW float64
	require.NoError(t, db.QueryRow(
		`SELECT weight FROM driver_behaviour_events WHERE event_type = 'odometer_rollback'`).Scan(&beW))
	assert.Equal(t, 20.0, beW)

	// Baseline reset: a subsequent reading at 995 is forward movement, so no
	// second rollback fires.
	insertSnapshots(t, db, []snapshotRow{
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 50, odometer: 995},
	})
	handled, err := e.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, handled)
	assert.Equal(t, 1, fuelEventsOf(t, db)["odometer_rollback"])
}

func TestEngine_SensorMissingSkipsLevelChecks(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, false, 100) // no fuel sensor
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// 12L drop would be theft IF the sensor were fitted — it must be ignored.
	// Odometer rollback (1000 → 990) must still trigger.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 38, odometer: 990},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	events := fuelEventsOf(t, db)
	assert.Equal(t, 0, events["drain_theft_suspected"], "level checks must be skipped without a sensor")
	assert.Equal(t, 1, events["odometer_rollback"], "odometer rollback must still be detected")
}

// failingSaver forces the outbox insert to fail inside the UnitOfWork.
type failingSaver struct{}

func (failingSaver) SaveEvents(ctx context.Context, _, _ string, _ []any) error {
	return errors.New("forced outbox failure")
}

func TestEngine_UoWAtomicity_RollsBackFuelEvents(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	e.WithAlertSaver(failingSaver{})
	insertSnapshots(t, db, snaps)

	_, err := e.Tick(context.Background())
	require.Error(t, err, "outbox failure must surface from the sweep")

	var fuelCount, behaviourCount int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM fuel_events`).Scan(&fuelCount))
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM driver_behaviour_events`).Scan(&behaviourCount))
	assert.Equal(t, 0, fuelCount, "fuel_events insert must roll back with the outbox failure")
	assert.Equal(t, 0, behaviourCount, "behaviour events insert must roll back with the outbox failure")
}

func TestEngine_NoiseFloorIgnoresSmallDeltas(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100) // noise floor = 1.5% of 100 = 1.5L
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 49.5, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 49.8, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)

	assert.Len(t, fuelEventsOf(t, db), 0, "sub-noise-floor deltas must be ignored")
}

func TestEngine_TripEndGatesTheftAndWatermarkAdvance(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 50, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)
	assert.Equal(t, 1, fuelEventsOf(t, db)["drain_theft_suspected"])

	// Trip ends; the same drop pattern after delivery must NOT fire theft
	// (Spec 03 §2.5 — theft requires an active trip).
	_, err := db.Exec(`UPDATE trips SET status = 'delivered' WHERE id = 't1'`)
	require.NoError(t, err)
	insertSnapshots(t, db, []snapshotRow{
		{id: "s4", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(3 * time.Second), speed: 0, fuelLevel: 38, odometer: 1000},
		{id: "s5", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(4 * time.Second), speed: 0, fuelLevel: 26, odometer: 1000},
		{id: "s6", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(5 * time.Second), speed: 0, fuelLevel: 26, odometer: 1000},
	})
	handled, err := e.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, handled)
	assert.Equal(t, 1, fuelEventsOf(t, db)["drain_theft_suspected"], "no theft after trip end")

	// Watermark advanced: a repeat tick consumes nothing.
	handled, err = e.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, handled, "already-consumed snapshots must not re-process")
}

// TestEngine_LevelUnitLitres verifies fuel.level_unit=litres is honoured
// (Spec 03 §2.3 / §9 — percent is the default; litres treats raw levels as
// litres).
func TestEngine_LevelUnitLitres(t *testing.T) {
	db := newTestDB(t)
	seedVehicle(t, db, true, 100)
	seedDriver(t, db)
	seedTrip(t, db, "in_transit")
	_, err := db.Exec(`UPDATE company_config SET value = 'litres' WHERE key = 'fuel.level_unit'`)
	require.NoError(t, err)

	t0 := time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)
	// Raw Δ = 15L (40 → 55) — under the 20L refill threshold in litres mode.
	snaps := []snapshotRow{
		{id: "s1", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0, speed: 0, fuelLevel: 40, odometer: 1000},
		{id: "s2", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(1 * time.Second), speed: 0, fuelLevel: 55, odometer: 1000},
		{id: "s3", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(2 * time.Second), speed: 0, fuelLevel: 55, odometer: 1000},
	}
	e := buildEngine(t, db, t0)
	feedScenario(t, db, e, snaps)
	assert.Equal(t, 0, fuelEventsOf(t, db)["refill_detected"])

	// A 25L raw jump (55 → 80) crossed and sustained across enough readings
	// for the median window to follow it (median of 7 needs the elevated
	// level to dominate; spike-hold confirms before acting).
	insertSnapshots(t, db, []snapshotRow{
		{id: "s4", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(3 * time.Second), speed: 0, fuelLevel: 55, odometer: 1000},
		{id: "s5", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(4 * time.Second), speed: 0, fuelLevel: 80, odometer: 1000},
		{id: "s6", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(5 * time.Second), speed: 0, fuelLevel: 80, odometer: 1000},
		{id: "s7", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(6 * time.Second), speed: 0, fuelLevel: 80, odometer: 1000},
		{id: "s8", tripID: "t1", vehicleID: "v1", driverID: "d1", ts: t0.Add(7 * time.Second), speed: 0, fuelLevel: 80, odometer: 1000},
	})
	_, err = e.Tick(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, fuelEventsOf(t, db)["refill_detected"])
}
