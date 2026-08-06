package driver

import (
	"context"
	"time"

	"transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
)

// CreateDriverCommand contains standard parameters to register a driver.
type CreateDriverCommand struct {
	TenantID              shared.TenantID
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
}

// UpdateDriverCommand contains parameters to update driver details.
type UpdateDriverCommand struct {
	ID                    aggregate.DriverID
	TenantID              shared.TenantID
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	Status                aggregate.DriverStatus
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
}

// DriverFacade is the module entrypoint contract.
type DriverFacade interface {
	CreateDriver(ctx context.Context, cmd CreateDriverCommand) (aggregate.DriverID, error)
	UpdateDriver(ctx context.Context, cmd UpdateDriverCommand) error
}
