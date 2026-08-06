package report_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/report"
)

func TestReport_CompletionAndUtilizationCalculations(t *testing.T) {
	now := time.Now()
	period := report.DateRange{
		From: now.Add(-30 * 24 * time.Hour),
		To:   now,
	}

	trips := report.TripsReport{
		Period:         period,
		TotalTrips:     100,
		CompletedTrips: 85,
		CancelledTrips: 15,
	}
	trips.CompletionRatePct = float64(trips.CompletedTrips) / float64(trips.TotalTrips) * 100.0

	if trips.CompletionRatePct != 85.0 {
		t.Errorf("expected 85%% completion rate, got %f", trips.CompletionRatePct)
	}

	fleet := report.FleetUtilizationReport{
		Period:         period,
		TotalVehicles:  50,
		ActiveVehicles: 40,
		IdleVehicles:   10,
	}
	fleet.UtilizationRatePct = float64(fleet.ActiveVehicles) / float64(fleet.TotalVehicles) * 100.0

	if fleet.UtilizationRatePct != 80.0 {
		t.Errorf("expected 80%% fleet utilization, got %f", fleet.UtilizationRatePct)
	}

	driver := report.DriverUtilizationReport{
		Period:           period,
		TotalDrivers:     40,
		ActiveOnTrip:     30,
		AvailableDrivers: 10,
	}
	driver.UtilizationRatePct = float64(driver.ActiveOnTrip) / float64(driver.TotalDrivers) * 100.0

	if driver.UtilizationRatePct != 75.0 {
		t.Errorf("expected 75%% driver utilization, got %f", driver.UtilizationRatePct)
	}
}
