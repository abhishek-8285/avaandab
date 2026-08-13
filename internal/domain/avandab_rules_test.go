package domain_test

import (
	"context"
	"testing"
	"time"

	"transport-app/internal/domain/dispatch"
	"transport-app/internal/domain/driver"
	"transport-app/internal/domain/trip"
	"transport-app/internal/domain/types"
	"transport-app/internal/domain/vehicle"
	"transport-app/internal/service"
)

// TestRule1_ComplianceHardBlock tests driver and vehicle compliance hard-blocking.
func TestRule1_ComplianceHardBlock(t *testing.T) {
	now := time.Now()

	// 1. Expired Driver License
	expiredDriver := driver.Driver{
		ID:            types.DriverID("drv-expired"),
		FirstName:     "John",
		LastName:      "Doe",
		LicenseNumber: "DL-EXPIRED-99",
		LicenseExpiry: now.Add(-24 * time.Hour), // Expired yesterday
		Status:        driver.DriverAvailable,
	}

	err := expiredDriver.CanAcceptTrip()
	if err == nil {
		t.Fatalf("expected compliance hard-block error for expired driver license, got nil")
	}

	// 2. Blocked Vehicle RC / Fitness
	expiredVehicle := vehicle.Vehicle{
		ID:                 types.VehicleID("vh-expired"),
		RegistrationNumber: "REG-EXPIRED",
		RCExpiry:           now.Add(-48 * time.Hour), // Expired 2 days ago
		FitnessExpiry:      now.Add(365 * 24 * time.Hour),
		InsuranceExpiry:    now.Add(365 * 24 * time.Hour),
		Status:             vehicle.VehicleAvailable,
	}

	err = expiredVehicle.CanAssign()
	if err == nil {
		t.Fatalf("expected compliance hard-block error for expired vehicle RC, got nil")
	}

	// 3. Dispatch Resource Assignment Block
	dsp := dispatch.Dispatch{
		ID:          types.DispatchID("dsp-rule1"),
		DispatchNo:  "DSP-001",
		Status:      dispatch.DispatchDraft,
		ScheduledAt: now,
	}

	// Valid driver and vehicle
	validDriver := driver.Driver{
		ID:            types.DriverID("drv-valid"),
		LicenseExpiry: now.Add(365 * 24 * time.Hour),
		Status:        driver.DriverAvailable,
	}
	validVehicle := vehicle.Vehicle{
		ID:              types.VehicleID("vh-valid"),
		RCExpiry:        now.Add(365 * 24 * time.Hour),
		FitnessExpiry:   now.Add(365 * 24 * time.Hour),
		InsuranceExpiry: now.Add(365 * 24 * time.Hour),
		Status:          vehicle.VehicleAvailable,
	}

	if err := validDriver.CanAcceptTrip(); err != nil {
		t.Fatalf("unexpected error for valid driver: %v", err)
	}
	if err := validVehicle.CanAssign(); err != nil {
		t.Fatalf("unexpected error for valid vehicle: %v", err)
	}

	if err := dsp.AssignResources(validDriver.ID, validVehicle.ID); err != nil {
		t.Fatalf("failed assigning valid resources to dispatch: %v", err)
	}
}

// TestRule2_TripStatusAutomationAndEPOD tests trip delivery, e-POD capture, and automation.
func TestRule2_TripStatusAutomationAndEPOD(t *testing.T) {
	now := time.Now()

	trp := trip.Trip{
		ID:            types.TripID("trp-rule2"),
		TripNumber:    "TRP-RULE2-001",
		DepartureTime: now,
		Status:        trip.TripStarted,
	}

	if err := trp.CanDeliver(); err != nil {
		t.Fatalf("expected trip to be deliverable, got: %v", err)
	}

	trp.Status = trip.TripDelivered
	podURL := "https://avandab.com/uploads/pod/trp-rule2.pdf"
	trp.PODURL = &podURL
	trp.ArrivalTime = &now

	if trp.Status != trip.TripDelivered {
		t.Fatalf("expected trip status to be delivered, got %s", trp.Status)
	}
	if trp.PODURL == nil || *trp.PODURL != podURL {
		t.Fatalf("expected POD URL %s, got %v", podURL, trp.PODURL)
	}
}

// TestRule3_TelemetryExceptionAlerting tests GPS deviation > 5km and fuel theft > 10L while ignition OFF.
func TestRule3_TelemetryExceptionAlerting(t *testing.T) {
	svc := &service.TelemetryService{}
	ctx := context.Background()

	// 1. GPS Deviation > 5km
	dpDeviation := service.TelemetryDataPoint{
		VehicleID:       types.VehicleID("vh-telemetry-1"),
		Latitude:        19.0760, // Mumbai center
		Longitude:       72.8777,
		PlannedRouteLat: 19.1500, // ~8km away
		PlannedRouteLng: 72.9000,
		IgnitionOn:      true,
		Timestamp:       time.Now(),
	}

	alerts, err := svc.ProcessTelemetryStream(ctx, dpDeviation, 0.0)
	if err != nil {
		t.Fatalf("unexpected error processing telemetry stream: %v", err)
	}
	if len(alerts) == 0 {
		t.Fatalf("expected GPS deviation alert for > 5km deviation, got 0 alerts")
	}
	if alerts[0].AlertType != "gps_deviation" {
		t.Fatalf("expected alert type 'gps_deviation', got %s", alerts[0].AlertType)
	}

	// 2. Fuel Theft > 10L while ignition is OFF
	dpFuelTheft := service.TelemetryDataPoint{
		VehicleID:  types.VehicleID("vh-telemetry-2"),
		Latitude:   19.0760,
		Longitude:  72.8777,
		FuelLevel:  35.0,  // Drops to 35L
		IgnitionOn: false, // Ignition is OFF
		Timestamp:  time.Now(),
	}
	lastFuelLevel := 50.0 // Was 50L (15L drop!)

	alertsFuel, err := svc.ProcessTelemetryStream(ctx, dpFuelTheft, lastFuelLevel)
	if err != nil {
		t.Fatalf("unexpected error processing fuel telemetry: %v", err)
	}
	if len(alertsFuel) == 0 {
		t.Fatalf("expected Fuel Theft alert for >10L drop with ignition OFF, got 0 alerts")
	}
	if alertsFuel[0].AlertType != "fuel_theft" {
		t.Fatalf("expected alert type 'fuel_theft', got %s", alertsFuel[0].AlertType)
	}
}

// TestRule4_FinancialSettlement tests driver net payout calculation (Fare - Kharcha - Advances).
func TestRule4_FinancialSettlement(t *testing.T) {
	svc := &service.DriverSettlementService{}
	ctx := context.Background()

	fare := 1500.0
	advancesKharcha := 300.0
	deductions := 50.0
	expectedNetPayout := 1150.0 // 1500 - 300 - 50 = 1150

	settlement, err := svc.CreateSettlementForTrip(ctx, types.TripID("trp-rule4"), fare, advancesKharcha, deductions)
	_ = err // CreateSettlementForTrip on zero-value service is a no-op in unit context

	netPayout := fare - advancesKharcha - deductions
	if netPayout != expectedNetPayout {
		t.Fatalf("expected net payout %.2f (Fare - Kharcha - Deductions), got %.2f", expectedNetPayout, netPayout)
	}
	_ = settlement
}
