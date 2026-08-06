package driver

import (
	"fmt"
	"time"

	"transport-app/internal/domain/types"
)

// Driver represents a transport driver.
type Driver struct {
	ID                    types.DriverID
	DriverID              string
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	Status                DriverStatus
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// DriverStatus represents the availability status of a driver.
type DriverStatus string

const (
	DriverAvailable DriverStatus = "available"
	DriverOnTrip    DriverStatus = "on_trip"
	DriverLeave     DriverStatus = "leave"
	DriverInactive  DriverStatus = "inactive"
)

// FullName returns the driver's full name.
func (d Driver) FullName() string {
	return d.FirstName + " " + d.LastName
}

// CanAcceptTrip validates that a driver is available to accept a trip.
func (d Driver) CanAcceptTrip() error {
	if d.Status != DriverAvailable {
		return fmt.Errorf("driver must be available to accept a trip; current status: %s", d.Status)
	}
	return nil
}
