package driver

import (
	"context"

	"transport-app/internal/domain/types"
)

// DriverRepository defines the interface for driver persistence.
type DriverRepository interface {
	CreateDriver(ctx context.Context, driver Driver) (Driver, error)
	GetDriverByID(ctx context.Context, id types.DriverID) (Driver, error)
	GetDriverByDriverID(ctx context.Context, driverID string) (Driver, error)
	GetDriverByPhone(ctx context.Context, phone string) (Driver, error)
	UpdateDriver(ctx context.Context, driver Driver) (Driver, error)
	DeleteDriver(ctx context.Context, id types.DriverID) error
	SearchDrivers(ctx context.Context, query string, status string, limit, offset int) ([]Driver, error)
	CountDrivers(ctx context.Context, query string, status string) (int64, error)
	GetAvailableDrivers(ctx context.Context) ([]Driver, error)
}
