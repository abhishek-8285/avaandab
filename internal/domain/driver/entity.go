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
	Blocked               bool
	BlockedReason         *string
	Aadhaar               *string
	PAN                   *string
	BankDetails           *string
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
	DriverBlocked   DriverStatus = "blocked"
)

// FullName returns the driver's full name.
func (d Driver) FullName() string {
	return d.FirstName + " " + d.LastName
}

// CanAcceptTrip validates that a driver is available to accept a trip.
// Rule 1: Compliance Hard-Block - If license is expired or driver is blocked, prevent dispatch.
func (d Driver) CanAcceptTrip() error {
	if d.Blocked || d.Status == DriverBlocked {
		reason := "driver is compliance blocked"
		if d.BlockedReason != nil && *d.BlockedReason != "" {
			reason = *d.BlockedReason
		}
		return fmt.Errorf("compliance hard-block: %s", reason)
	}
	if !d.LicenseExpiry.IsZero() && d.LicenseExpiry.Before(time.Now()) {
		return fmt.Errorf("compliance hard-block: driver license %s expired on %s", d.LicenseNumber, d.LicenseExpiry.Format("2006-01-02"))
	}
	if d.Status != DriverAvailable {
		return fmt.Errorf("driver must be available to accept a trip; current status: %s", d.Status)
	}
	return nil
}
