package driver

import (
	"context"

	"transport-app/internal/domain/types"
)

// DriverService defines the interface for driver business operations.
type DriverService interface {
	CreateDriver(ctx context.Context, firstName, lastName, phone, email, address, licenseNumber string, licenseExpiry string, experience int64, emergencyContactName, emergencyContactPhone, notes *string) (Driver, error)
	GetDriver(ctx context.Context, id types.DriverID) (Driver, error)
	GetDriverByNumber(ctx context.Context, driverID string) (Driver, error)
	ListDrivers(ctx context.Context, query, status string, limit, offset int) ([]Driver, int64, error)
	UpdateDriver(ctx context.Context, id types.DriverID, firstName, lastName, phone, email, address, licenseNumber string, licenseExpiry string, experience int64, status DriverStatus, emergencyContactName, emergencyContactPhone, notes *string) (Driver, error)
	DeleteDriver(ctx context.Context, id types.DriverID) error
	SetDriverStatus(ctx context.Context, id types.DriverID, status DriverStatus) (Driver, error)
	GetAvailableDrivers(ctx context.Context) ([]Driver, error)
}
