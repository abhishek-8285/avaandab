package service_test

import (
	"context"
	"testing"
	"time"

	"transport-app/internal/domain/types"
	"transport-app/internal/service"
)

func TestComplianceService_ValidateDriverCompliance(t *testing.T) {
	svc := &service.ComplianceService{}
	ctx := context.Background()

	// Uninitialized store returns error gracefully
	res, err := svc.ValidateDriverCompliance(ctx, types.DriverID("drv-1"))
	if err == nil {
		t.Fatalf("expected error without store, got nil")
	}
	if res.Valid {
		t.Fatalf("expected valid=false without store")
	}
}

func TestComplianceService_ValidateVehicleCompliance(t *testing.T) {
	svc := &service.ComplianceService{}
	ctx := context.Background()

	res, err := svc.ValidateVehicleCompliance(ctx, types.VehicleID("vh-1"))
	if err == nil {
		t.Fatalf("expected error without store, got nil")
	}
	if res.Valid {
		t.Fatalf("expected valid=false without store")
	}
}

func TestTelemetryService_ProcessTelemetryStream(t *testing.T) {
	svc := &service.TelemetryService{}
	ctx := context.Background()

	// Test normal stream without alerts
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

	alerts, err := svc.ProcessTelemetryStream(ctx, dpNormal, 45.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts for normal point, got %d", len(alerts))
	}
}

func TestDriverSettlementService_CreateSettlementForTrip(t *testing.T) {
	svc := &service.DriverSettlementService{}
	ctx := context.Background()

	tripID := types.TripID("trp-test-1")
	settlement, err := svc.CreateSettlementForTrip(ctx, tripID, 2000.0, 300.0, 100.0)
	if err != nil {
		t.Fatalf("unexpected error creating settlement: %v", err)
	}

	if settlement.NetPayout != 1600.0 {
		t.Fatalf("expected NetPayout 1600.0 (2000-300-100), got %.2f", settlement.NetPayout)
	}
	if settlement.Status != "pending" {
		t.Fatalf("expected status 'pending', got '%s'", settlement.Status)
	}
}
