package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"transport-app/internal/domain"
	"transport-app/internal/events"
	sqliterepo "transport-app/internal/repository/sqlite"
)

func newTelemetryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("test_telem_svc_%d", time.Now().UnixNano())
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared&_pragma=journal_mode(WAL)")
	require.NoError(t, err)

	cwd, _ := os.Getwd()
	migrationsDir := "../../db/migrations"
	if filepath.Base(cwd) == "basic" {
		migrationsDir = "db/migrations"
	}

	_ = goose.SetDialect("sqlite")
	require.NoError(t, goose.Up(db, migrationsDir))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestTelemetryService_ProcessTelemetryStream_PersistsAlerts(t *testing.T) {
	db := newTelemetryTestDB(t)
	repo := sqliterepo.NewRepository(db)
	bus := events.NewInMemoryBus()

	var receivedAlertEvents []events.Event
	bus.Subscribe("AlertEvent", func(ctx context.Context, e events.Event) error {
		receivedAlertEvents = append(receivedAlertEvents, e)
		return nil
	})

	svc := NewServices(repo, nil, nil, bus)

	ctx := context.Background()
	dp := TelemetryDataPoint{
		VehicleID:       domain.VehicleID("veh-telem-1"),
		Latitude:        19.0,
		Longitude:       74.0,
		PlannedRouteLat: 18.5,
		PlannedRouteLng: 73.8, // > 5 km deviation
		FuelLevel:       40.0,
		IgnitionOn:      false,
		Timestamp:       time.Now(),
	}

	// Last fuel level 60 -> drop 20L while ignition is OFF (theft_suspicion)
	alerts, err := svc.Telemetry.ProcessTelemetryStream(ctx, dp, 60.0)
	require.NoError(t, err)
	assert.Len(t, alerts, 2, "must generate both gps_deviation and theft_suspicion alerts")

	// Verify alert types
	alertTypes := []string{alerts[0].AlertType, alerts[1].AlertType}
	assert.Contains(t, alertTypes, "gps_deviation")
	assert.Contains(t, alertTypes, "theft_suspicion")

	// Verify persistence in telemetry_alerts table
	rows, err := db.Query("SELECT alert_type, severity FROM telemetry_alerts WHERE vehicle_id = 'veh-telem-1' ORDER BY created_at ASC")
	require.NoError(t, err)
	defer rows.Close()

	var savedTypes []string
	for rows.Next() {
		var at, sev string
		require.NoError(t, rows.Scan(&at, &sev))
		savedTypes = append(savedTypes, at)
	}
	assert.Len(t, savedTypes, 2)
	assert.Contains(t, savedTypes, "gps_deviation")
	assert.Contains(t, savedTypes, "theft_suspicion")

	// Verify AlertEvent published on bus
	assert.Len(t, receivedAlertEvents, 2)
}
