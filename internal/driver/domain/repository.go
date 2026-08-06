package domain

import (
	"context"
	"time"

	"transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
)

// DriverReadModel represents drivers optimized for read performance.
type DriverReadModel struct {
	ID                    string
	DriverDisplayID       string
	FirstName             string
	LastName              string
	Phone                 string
	Email                 *string
	Address               *string
	LicenseNumber         string
	LicenseExpiry         time.Time
	ExperienceYears       int64
	Status                string
	EmergencyContactName  *string
	EmergencyContactPhone *string
	Notes                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// DriverRepository defines the DDD-compliant repository interface for drivers.
type DriverRepository interface {
	Save(ctx context.Context, d *aggregate.DriverAggregate) error
	Find(ctx context.Context, id aggregate.DriverID, tenantID shared.TenantID) (*aggregate.DriverAggregate, error)
	GetReadModel(ctx context.Context, id aggregate.DriverID, tenantID shared.TenantID) (DriverReadModel, error)
	SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]DriverReadModel, int64, error)
}
