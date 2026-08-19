package service

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/domain"
)

// VehicleService handles vehicle management.
type VehicleService struct {
	baseService
}

// CreateVehicle creates a new vehicle.
func (s *VehicleService) CreateVehicle(ctx context.Context, regNumber, vehicleNumber string, vehicleType domain.VehicleType, capacity int64, fuelType domain.FuelType, insuranceExpiry, fitnessExpiry, permitExpiry, mileage string) (domain.Vehicle, error) {
	if regNumber == "" || vehicleNumber == "" {
		return domain.Vehicle{}, fmt.Errorf("registration number and vehicle number are required")
	}

	// Check registration uniqueness
	if _, err := s.store.GetVehicleByRegistration(ctx, regNumber); err == nil {
		return domain.Vehicle{}, fmt.Errorf("vehicle with registration number %s already exists", regNumber)
	}

	insExp, err := time.Parse("2006-01-02", insuranceExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid insurance expiry date")
	}
	ftExp, err := time.Parse("2006-01-02", fitnessExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid fitness expiry date")
	}
	permExp, err := time.Parse("2006-01-02", permitExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid permit expiry date")
	}

	var mileageVal *float64
	if mileage != "" {
		// parse mileage as float
		var m float64
		_, err := fmt.Sscanf(mileage, "%f", &m)
		if err == nil {
			mileageVal = &m
		}
	}

	vehicle := domain.Vehicle{
		ID:                 domain.VehicleID(generateID()),
		RegistrationNumber: regNumber,
		VehicleNumber:      vehicleNumber,
		VehicleType:        vehicleType,
		Capacity:           capacity,
		FuelType:           fuelType,
		InsuranceExpiry:    insExp,
		FitnessExpiry:      ftExp,
		PermitExpiry:       permExp,
		Status:             domain.VehicleAvailable,
		CurrentMileage:     mileageVal,
	}

	created, err := s.store.CreateVehicle(ctx, vehicle)
	if err != nil {
		return domain.Vehicle{}, err
	}

	s.log.Info("vehicle created", "vehicle_id", created.ID)
	s.logAudit(ctx, nil, "create", "vehicles", string(created.ID), nil, nil)
	return created, nil
}

// GetVehicle retrieves a vehicle by ID.
func (s *VehicleService) GetVehicle(ctx context.Context, id domain.VehicleID) (domain.Vehicle, error) {
	return s.store.GetVehicleByID(ctx, id)
}

// ListVehicles retrieves vehicles with search and pagination.
func (s *VehicleService) ListVehicles(ctx context.Context, query, status string, limit, offset int) ([]domain.Vehicle, int64, error) {
	vehicles, err := s.store.SearchVehicles(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountVehicles(ctx, query, status)
	if err != nil {
		return nil, 0, err
	}
	return vehicles, total, nil
}

// UpdateVehicle updates an existing vehicle.
func (s *VehicleService) UpdateVehicle(ctx context.Context, id domain.VehicleID, regNumber, vehicleNumber string, vehicleType domain.VehicleType, capacity int64, fuelType domain.FuelType, insuranceExpiry, fitnessExpiry, permitExpiry, mileage string, status domain.VehicleStatus) (domain.Vehicle, error) {
	vehicle, err := s.store.GetVehicleByID(ctx, id)
	if err != nil {
		return domain.Vehicle{}, domain.ErrVehicleNotFound
	}

	// Check registration uniqueness for other vehicles
	if existing, err := s.store.GetVehicleByRegistration(ctx, regNumber); err == nil && existing.ID != id {
		return domain.Vehicle{}, fmt.Errorf("vehicle with registration number %s already exists", regNumber)
	}

	insExp, err := time.Parse("2006-01-02", insuranceExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid insurance expiry date")
	}
	ftExp, err := time.Parse("2006-01-02", fitnessExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid fitness expiry date")
	}
	permExp, err := time.Parse("2006-01-02", permitExpiry)
	if err != nil {
		return domain.Vehicle{}, fmt.Errorf("invalid permit expiry date")
	}

	vehicle.RegistrationNumber = regNumber
	vehicle.VehicleNumber = vehicleNumber
	vehicle.VehicleType = vehicleType
	vehicle.Capacity = capacity
	vehicle.FuelType = fuelType
	vehicle.InsuranceExpiry = insExp
	vehicle.FitnessExpiry = ftExp
	vehicle.PermitExpiry = permExp
	vehicle.Status = status

	var mileageVal *float64
	if mileage != "" {
		var m float64
		_, err := fmt.Sscanf(mileage, "%f", &m)
		if err == nil {
			mileageVal = &m
		}
	}
	vehicle.CurrentMileage = mileageVal

	updated, err := s.store.UpdateVehicle(ctx, vehicle)
	if err != nil {
		return domain.Vehicle{}, err
	}

	s.log.Info("vehicle updated", "vehicle_id", id)
	s.logAudit(ctx, nil, "update", "vehicles", string(updated.ID), nil, nil)
	return updated, nil
}

// DeleteVehicle deletes a vehicle.
func (s *VehicleService) DeleteVehicle(ctx context.Context, id domain.VehicleID) error {
	// Block deletion if assigned to active/upcoming trips
	conflicts, err := s.store.CheckVehicleConflict(ctx, id, nil)
	if err == nil && len(conflicts) > 0 {
		return fmt.Errorf("cannot delete vehicle because it is assigned to trip %s", conflicts[0].TripNumber)
	}

	if err := s.store.DeleteVehicle(ctx, id); err != nil {
		return err
	}
	s.log.Info("vehicle deleted", "vehicle_id", id)
	s.logAudit(ctx, nil, "delete", "vehicles", string(id), nil, nil)
	return nil
}

// SetVehicleStatus updates a vehicle's status.
func (s *VehicleService) SetVehicleStatus(ctx context.Context, id domain.VehicleID, status domain.VehicleStatus) (domain.Vehicle, error) {
	vehicle, err := s.store.GetVehicleByID(ctx, id)
	if err != nil {
		return domain.Vehicle{}, domain.ErrVehicleNotFound
	}
	vehicle.Status = status
	updated, err := s.store.UpdateVehicle(ctx, vehicle)
	if err != nil {
		return domain.Vehicle{}, err
	}

	s.log.Info("vehicle status updated", "vehicle_id", id)
	return updated, nil
}

// GetAvailableVehicles returns vehicles available for assignment.
func (s *VehicleService) GetAvailableVehicles(ctx context.Context) ([]domain.Vehicle, error) {
	return s.store.GetAvailableVehicles(ctx)
}

// IsMaintenanceBlocked checks if a vehicle is blocked for maintenance (Spec 04 §6, §12).
func (s *VehicleService) IsMaintenanceBlocked(ctx context.Context, vehicleID string) (bool, string, error) {
	return s.store.IsMaintenanceBlocked(ctx, vehicleID)
}
