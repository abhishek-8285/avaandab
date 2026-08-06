package aggregate

import (
	"errors"
	"time"

	"transport-app/internal/shared"
)

type DriverID string
type DriverStatus string

const (
	DriverAvailable DriverStatus = "available"
	DriverOnTrip    DriverStatus = "on_trip"
	DriverLeave     DriverStatus = "leave"
	DriverInactive  DriverStatus = "inactive"
)

// DriverAggregate is the aggregate root representing a driver.
type DriverAggregate struct {
	ID                    DriverID
	TenantID              shared.TenantID
	DriverDisplayID       string
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
	Version               int64
	events                []any
}

// NewDriverAggregate constructs a new DriverAggregate and registers a created event.
func NewDriverAggregate(
	id DriverID,
	tenantID shared.TenantID,
	driverDisplayID string,
	firstName string,
	lastName string,
	phone string,
	email *string,
	address *string,
	licenseNumber string,
	licenseExpiry time.Time,
	experienceYears int64,
	status DriverStatus,
	emergencyContactName *string,
	emergencyContactPhone *string,
	notes *string,
	now time.Time,
) *DriverAggregate {
	d := &DriverAggregate{
		ID:                    id,
		TenantID:              tenantID,
		DriverDisplayID:       driverDisplayID,
		FirstName:             firstName,
		LastName:              lastName,
		Phone:                 phone,
		Email:                 email,
		Address:               address,
		LicenseNumber:         licenseNumber,
		LicenseExpiry:         licenseExpiry,
		ExperienceYears:       experienceYears,
		Status:                status,
		EmergencyContactName:  emergencyContactName,
		EmergencyContactPhone: emergencyContactPhone,
		Notes:                 notes,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	d.events = append(d.events, DriverCreatedEvent{
		ID:              id,
		TenantID:        tenantID,
		DriverDisplayID: driverDisplayID,
		Phone:           phone,
		LicenseNumber:   licenseNumber,
		CreatedAt:       now,
	})

	return d
}

// Events returns recorded domain events.
func (a *DriverAggregate) Events() []any {
	return a.events
}

// ClearEvents clears the recorded events list.
func (a *DriverAggregate) ClearEvents() {
	a.events = nil
}

// UpdateDetails updates standard driver details and records an updated event.
func (a *DriverAggregate) UpdateDetails(
	firstName string,
	lastName string,
	phone string,
	email *string,
	address *string,
	licenseNumber string,
	licenseExpiry time.Time,
	experienceYears int64,
	status DriverStatus,
	emergencyContactName *string,
	emergencyContactPhone *string,
	notes *string,
	now time.Time,
) error {
	if firstName == "" || lastName == "" {
		return errors.New("first and last name are required")
	}
	if phone == "" {
		return errors.New("phone number is required")
	}

	a.FirstName = firstName
	a.LastName = lastName
	a.Phone = phone
	a.Email = email
	a.Address = address
	a.LicenseNumber = licenseNumber
	a.LicenseExpiry = licenseExpiry
	a.ExperienceYears = experienceYears
	a.Status = status
	a.EmergencyContactName = emergencyContactName
	a.EmergencyContactPhone = emergencyContactPhone
	a.Notes = notes
	a.UpdatedAt = now

	a.events = append(a.events, DriverUpdatedEvent{
		ID:        a.ID,
		TenantID:  a.TenantID,
		Status:    status,
		UpdatedAt: now,
	})

	return nil
}

// DriverCreatedEvent emitted when a driver is successfully registered.
type DriverCreatedEvent struct {
	ID              DriverID
	TenantID        shared.TenantID
	DriverDisplayID string
	Phone           string
	LicenseNumber   string
	CreatedAt       time.Time
}

// DriverUpdatedEvent emitted when driver details or status changes.
type DriverUpdatedEvent struct {
	ID        DriverID
	TenantID  shared.TenantID
	Status    DriverStatus
	UpdatedAt time.Time
}
