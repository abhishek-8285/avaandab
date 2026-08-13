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
	RCExpiry           time.Time
	Status             VehicleStatus
	Blocked            bool
	BlockedReason      *string
	Odometer           float64
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
	VehicleBlocked     VehicleStatus = "blocked"
)

// CanAssign validates that a vehicle can be assigned to a trip (must be available).
// Rule 1: Compliance Hard-Block - If RC/Fitness/Insurance is expired or vehicle is blocked, prevent dispatch.
func (v Vehicle) CanAssign() error {
	if v.Blocked || v.Status == VehicleBlocked {
		reason := "vehicle is compliance blocked"
		if v.BlockedReason != nil && *v.BlockedReason != "" {
			reason = *v.BlockedReason
		}
		return fmt.Errorf("compliance hard-block: %s", reason)
	}
	now := time.Now()
	if !v.RCExpiry.IsZero() && v.RCExpiry.Before(now) {
		return fmt.Errorf("compliance hard-block: vehicle RC expired on %s", v.RCExpiry.Format("2006-01-02"))
	}
	if !v.FitnessExpiry.IsZero() && v.FitnessExpiry.Before(now) {
		return fmt.Errorf("compliance hard-block: vehicle fitness cert expired on %s", v.FitnessExpiry.Format("2006-01-02"))
	}
	if !v.InsuranceExpiry.IsZero() && v.InsuranceExpiry.Before(now) {
		return fmt.Errorf("compliance hard-block: vehicle insurance expired on %s", v.InsuranceExpiry.Format("2006-01-02"))
	}
	if v.Status != VehicleAvailable {
		return fmt.Errorf("vehicle must be available to be assigned; current status: %s", v.Status)
	}
	return nil
}
