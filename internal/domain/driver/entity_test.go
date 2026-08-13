package driver_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/driver"
	"transport-app/internal/domain/types"
)

func TestDriver_FullName(t *testing.T) {
	drv := driver.Driver{
		FirstName: "Ramesh",
		LastName:  "Kumar",
	}
	if drv.FullName() != "Ramesh Kumar" {
		t.Fatalf("expected 'Ramesh Kumar', got '%s'", drv.FullName())
	}
}

func TestDriver_CanAcceptTrip_Valid(t *testing.T) {
	drv := driver.Driver{
		ID:            types.DriverID("drv-1"),
		LicenseNumber: "DL-100200",
		LicenseExpiry: time.Now().Add(30 * 24 * time.Hour),
		Status:        driver.DriverAvailable,
	}

	if err := drv.CanAcceptTrip(); err != nil {
		t.Fatalf("expected valid driver to accept trip, got error: %v", err)
	}
}

func TestDriver_CanAcceptTrip_Blocked(t *testing.T) {
	reason := "Medical certification pending"
	drv := driver.Driver{
		ID:            types.DriverID("drv-2"),
		LicenseNumber: "DL-100201",
		LicenseExpiry: time.Now().Add(30 * 24 * time.Hour),
		Status:        driver.DriverBlocked,
		Blocked:       true,
		BlockedReason: &reason,
	}

	err := drv.CanAcceptTrip()
	if err == nil {
		t.Fatalf("expected compliance hard-block error for blocked driver, got nil")
	}
}

func TestDriver_CanAcceptTrip_LicenseExpired(t *testing.T) {
	drv := driver.Driver{
		ID:            types.DriverID("drv-3"),
		LicenseNumber: "DL-100202",
		LicenseExpiry: time.Now().Add(-24 * time.Hour),
		Status:        driver.DriverAvailable,
	}

	err := drv.CanAcceptTrip()
	if err == nil {
		t.Fatalf("expected compliance hard-block error for expired license, got nil")
	}
}

func TestDriver_CanAcceptTrip_UnavailableStatus(t *testing.T) {
	drv := driver.Driver{
		ID:            types.DriverID("drv-4"),
		LicenseNumber: "DL-100203",
		LicenseExpiry: time.Now().Add(30 * 24 * time.Hour),
		Status:        driver.DriverOnTrip,
	}

	err := drv.CanAcceptTrip()
	if err == nil {
		t.Fatalf("expected error for on_trip status driver, got nil")
	}
}
