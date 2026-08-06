package vehicle

import (
	"context"

	"transport-app/internal/domain/types"
)

// VehicleService defines the interface for vehicle business operations.
type VehicleService interface {
	CreateVehicle(ctx context.Context, regNumber, vehicleNumber string, vehicleType VehicleType, capacity int64, fuelType FuelType, insuranceExpiry, fitnessExpiry, permitExpiry, mileage string) (Vehicle, error)
	GetVehicle(ctx context.Context, id types.VehicleID) (Vehicle, error)
	ListVehicles(ctx context.Context, query, status string, limit, offset int) ([]Vehicle, int64, error)
	UpdateVehicle(ctx context.Context, id types.VehicleID, regNumber, vehicleNumber string, vehicleType VehicleType, capacity int64, fuelType FuelType, insuranceExpiry, fitnessExpiry, permitExpiry, mileage string, status VehicleStatus) (Vehicle, error)
	DeleteVehicle(ctx context.Context, id types.VehicleID) error
	SetVehicleStatus(ctx context.Context, id types.VehicleID, status VehicleStatus) (Vehicle, error)
	GetAvailableVehicles(ctx context.Context) ([]Vehicle, error)
}
