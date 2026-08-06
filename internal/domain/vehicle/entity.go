package vehicle

import (
	"fmt"
	"time"

	"transport-app/internal/domain/types"
)

// Vehicle represents a transport vehicle.
type Vehicle struct {
	ID                 types.VehicleID
	RegistrationNumber string
	VehicleNumber      string
	VehicleType        VehicleType
	Capacity           int64
	FuelType           FuelType
	InsuranceExpiry    time.Time
	FitnessExpiry      time.Time
	PermitExpiry       time.Time
	Status             VehicleStatus
	CurrentMileage     *float64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// VehicleType represents the type of a vehicle.
type VehicleType string

const (
	VehicleTypeTruck     VehicleType = "truck"
	VehicleTypeMiniTruck VehicleType = "mini_truck"
	VehicleTypeBus       VehicleType = "bus"
	VehicleTypeVan       VehicleType = "van"
	VehicleTypePickup    VehicleType = "pickup"
	VehicleTypeTempo     VehicleType = "tempo"
)

// FuelType represents the fuel type of a vehicle.
type FuelType string

const (
	FuelTypeDiesel   FuelType = "diesel"
	FuelTypePetrol   FuelType = "petrol"
	FuelTypeGas      FuelType = "gas"
	FuelTypeElectric FuelType = "electric"
	FuelTypeCNG      FuelType = "cng"
)

// VehicleStatus represents the operational status of a vehicle.
type VehicleStatus string

const (
	VehicleAvailable   VehicleStatus = "available"
	VehicleRunning     VehicleStatus = "running"
	VehicleMaintenance VehicleStatus = "maintenance"
	VehicleInactive    VehicleStatus = "inactive"
)

// CanAssign validates that a vehicle can be assigned to a trip (must be available).
func (v Vehicle) CanAssign() error {
	if v.Status != VehicleAvailable {
		return fmt.Errorf("vehicle must be available to be assigned; current status: %s", v.Status)
	}
	return nil
}
