package service_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/config"
	"transport-app/internal/domain"
	"transport-app/internal/domain/types"
	"transport-app/internal/service"
)

func testConfig() *config.Config {
	return &config.Config{
		AppEnv:        "testing",
		Port:          "8080",
		DatabaseURL:   "file::memory:?cache=shared",
		CookieSecret:  "test-secret-32bytes-long-enough!",
		SessionMaxAge: 24 * 3600 * 1000000000,
		LogLevel:      "error",
		UploadDir:     "./uploads",
		MaxUploadSize: 10 << 20,
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServices(t *testing.T) *service.Services {
	t.Helper()
	return service.NewServices(nil, testConfig(), testLogger())
}

func TestComplianceService_ValidateDriverCompliance_WithoutStore(t *testing.T) {
	svc := newTestServices(t)
	ctx := context.Background()

	res, err := svc.Compliance.ValidateDriverCompliance(ctx, types.DriverID("drv-1"))
	require.Error(t, err)
	assert.False(t, res.Valid)
	assert.Contains(t, err.Error(), "store uninitialized")
}

func TestComplianceService_ValidateVehicleCompliance_WithoutStore(t *testing.T) {
	svc := newTestServices(t)
	ctx := context.Background()

	res, err := svc.Compliance.ValidateVehicleCompliance(ctx, types.VehicleID("vh-1"))
	require.Error(t, err)
	assert.False(t, res.Valid)
	assert.Contains(t, err.Error(), "store uninitialized")
}

func TestTelemetryService_ProcessTelemetryStream(t *testing.T) {
	svc := newTestServices(t)
	ctx := context.Background()

	// Normal stream within planned route and no fuel theft
	dpNormal := service.TelemetryDataPoint{
		VehicleID:       types.VehicleID("vh-normal"),
		Latitude:        19.0760,
		Longitude:       72.8777,
		PlannedRouteLat: 19.0761,
		PlannedRouteLng: 72.8778,
		FuelLevel:       45.0,
		IgnitionOn:      true,
		Timestamp:       time.Now(),
	}

	alerts, err := svc.Telemetry.ProcessTelemetryStream(ctx, dpNormal, 45.5)
	require.NoError(t, err)
	assert.Empty(t, alerts)

	// GPS deviation > 5km triggers alert
	dpDeviation := service.TelemetryDataPoint{
		VehicleID:       types.VehicleID("vh-deviation"),
		Latitude:        19.0760,
		Longitude:       72.8777,
		PlannedRouteLat: 19.2000,
		PlannedRouteLng: 73.0000,
		FuelLevel:       45.0,
		IgnitionOn:      true,
		Timestamp:       time.Now(),
	}

	alerts, err = svc.Telemetry.ProcessTelemetryStream(ctx, dpDeviation, 45.5)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "gps_deviation", alerts[0].AlertType)

	// Fuel drop > 10L while ignition OFF triggers alert
	dpFuelTheft := service.TelemetryDataPoint{
		VehicleID:  types.VehicleID("vh-fuel"),
		Latitude:   19.0760,
		Longitude:  72.8777,
		FuelLevel:  30.0,
		IgnitionOn: false,
		Timestamp:  time.Now(),
	}

	alerts, err = svc.Telemetry.ProcessTelemetryStream(ctx, dpFuelTheft, 45.0)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "fuel_theft", alerts[0].AlertType)
}

func TestDriverSettlementService_CreateSettlementForTrip(t *testing.T) {
	svc := newTestServices(t)
	ctx := context.Background()

	tripID := types.TripID("trp-test-1")
	settlement, err := svc.Settlements.CreateSettlementForTrip(ctx, tripID, 2000.0, 300.0, 100.0)
	require.NoError(t, err)

	assert.Equal(t, tripID, settlement.TripID)
	assert.Equal(t, 1600.0, settlement.NetPayout)
	assert.Equal(t, domain.DriverID("drv-default"), settlement.DriverID)
	assert.Equal(t, "pending", settlement.Status)
}
