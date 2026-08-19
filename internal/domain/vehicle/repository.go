package vehicle

import (
	"context"

	"transport-app/internal/domain/types"
)

// VehicleRepository defines the interface for vehicle persistence.
type VehicleRepository interface {
	CreateVehicle(ctx context.Context, vehicle Vehicle) (Vehicle, error)
	GetVehicleByID(ctx context.Context, id types.VehicleID) (Vehicle, error)
	GetVehicleByRegistration(ctx context.Context, regNum string) (Vehicle, error)
	UpdateVehicle(ctx context.Context, vehicle Vehicle) (Vehicle, error)
	DeleteVehicle(ctx context.Context, id types.VehicleID) error
	SearchVehicles(ctx context.Context, query string, status string, limit, offset int) ([]Vehicle, error)
	CountVehicles(ctx context.Context, query string, status string) (int64, error)
	GetAvailableVehicles(ctx context.Context) ([]Vehicle, error)
	GetIdleVehicles(ctx context.Context) ([]Vehicle, error)
	IsMaintenanceBlocked(ctx context.Context, vehicleID string) (bool, string, error)
}
