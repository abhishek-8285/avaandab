package aggregate

import (
	"errors"
	"time"

	"transport-app/internal/shared"
)

type VehicleID string
type VehicleType string
type FuelType string
type VehicleStatus string

const (
	VehicleTypeTruck     VehicleType = "truck"
	VehicleTypeMiniTruck VehicleType = "mini_truck"
	VehicleTypeBus       VehicleType = "bus"
	VehicleTypeVan       VehicleType = "van"
	VehicleTypePickup    VehicleType = "pickup"
	VehicleTypeTempo     VehicleType = "tempo"

	FuelTypeDiesel   FuelType = "diesel"
	FuelTypePetrol   FuelType = "petrol"
	FuelTypeGas      FuelType = "gas"
	FuelTypeElectric FuelType = "electric"
	FuelTypeCNG      FuelType = "cng"

	VehicleAvailable   VehicleStatus = "available"
	VehicleRunning     VehicleStatus = "running"
	VehicleMaintenance VehicleStatus = "maintenance"
	VehicleInactive    VehicleStatus = "inactive"
)

// VehicleAggregate is the aggregate root representing a vehicle.
type VehicleAggregate struct {
	ID                 VehicleID
	TenantID           shared.TenantID
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
	Version            int64
	events             []any
}

// NewVehicleAggregate constructs a new VehicleAggregate and records a created event.
func NewVehicleAggregate(
	id VehicleID,
	tenantID shared.TenantID,
	registrationNumber string,
	vehicleNumber string,
	vehicleType VehicleType,
	capacity int64,
	fuelType FuelType,
	insuranceExpiry time.Time,
	fitnessExpiry time.Time,
	permitExpiry time.Time,
	status VehicleStatus,
	currentMileage *float64,
	now time.Time,
) *VehicleAggregate {
	v := &VehicleAggregate{
		ID:                 id,
		TenantID:           tenantID,
		RegistrationNumber: registrationNumber,
		VehicleNumber:      vehicleNumber,
		VehicleType:        vehicleType,
		Capacity:           capacity,
		FuelType:           fuelType,
		InsuranceExpiry:    insuranceExpiry,
		FitnessExpiry:      fitnessExpiry,
		PermitExpiry:       permitExpiry,
		Status:             status,
		CurrentMileage:     currentMileage,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	v.events = append(v.events, VehicleCreatedEvent{
		ID:                 id,
		TenantID:           tenantID,
		RegistrationNumber: registrationNumber,
		VehicleNumber:      vehicleNumber,
		CreatedAt:          now,
	})

	return v
}

// Events returns recorded domain events.
func (a *VehicleAggregate) Events() []any {
	return a.events
}

// ClearEvents clears the recorded events list.
func (a *VehicleAggregate) ClearEvents() {
	a.events = nil
}

// UpdateDetails updates vehicle properties and records an updated event.
func (a *VehicleAggregate) UpdateDetails(
	registrationNumber string,
	vehicleNumber string,
	vehicleType VehicleType,
	capacity int64,
	fuelType FuelType,
	insuranceExpiry time.Time,
	fitnessExpiry time.Time,
	permitExpiry time.Time,
	status VehicleStatus,
	currentMileage *float64,
	now time.Time,
) error {
	if registrationNumber == "" || vehicleNumber == "" {
		return errors.New("registration and vehicle number are required")
	}

	a.RegistrationNumber = registrationNumber
	a.VehicleNumber = vehicleNumber
	a.VehicleType = vehicleType
	a.Capacity = capacity
	a.FuelType = fuelType
	a.InsuranceExpiry = insuranceExpiry
	a.FitnessExpiry = fitnessExpiry
	a.PermitExpiry = permitExpiry
	a.Status = status
	a.CurrentMileage = currentMileage
	a.UpdatedAt = now

	a.events = append(a.events, VehicleUpdatedEvent{
		ID:                 a.ID,
		TenantID:           a.TenantID,
		Status:             status,
		RegistrationNumber: registrationNumber,
		UpdatedAt:          now,
	})

	return nil
}

// VehicleCreatedEvent emitted when a vehicle is registered.
type VehicleCreatedEvent struct {
	ID                 VehicleID
	TenantID           shared.TenantID
	RegistrationNumber string
	VehicleNumber      string
	CreatedAt          time.Time
}

// VehicleUpdatedEvent emitted when vehicle details change.
type VehicleUpdatedEvent struct {
	ID                 VehicleID
	TenantID           shared.TenantID
	Status             VehicleStatus
	RegistrationNumber string
	UpdatedAt          time.Time
}
