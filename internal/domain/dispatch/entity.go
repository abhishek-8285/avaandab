package dispatch

import (
	"errors"
	"fmt"
	"time"

	"transport-app/internal/domain/types"
)

var (
	ErrDispatchNotFound  = errors.New("dispatch order not found")
	ErrInvalidDispatch   = errors.New("invalid dispatch transition")
	ErrUnassignedResource = errors.New("driver and vehicle must be assigned before trip creation")
)

type DispatchStatus string

const (
	DispatchDraft     DispatchStatus = "draft"
	DispatchAssigned  DispatchStatus = "assigned"
	DispatchConverted DispatchStatus = "converted"
	DispatchCancelled DispatchStatus = "cancelled"
)

// Dispatch represents the operational planning record owned by Dispatchers.
type Dispatch struct {
	ID           types.DispatchID
	DispatchNo   string
	DispatcherID types.UserID
	BookingID    types.BookingID
	DriverID     *types.DriverID
	VehicleID    *types.VehicleID
	ScheduledAt  time.Time
	Status       DispatchStatus
	TripID       *types.TripID
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AssignResources assigns driver and vehicle to dispatch.
func (d *Dispatch) AssignResources(driverID types.DriverID, vehicleID types.VehicleID) error {
	if d.Status != DispatchDraft && d.Status != DispatchAssigned {
		return fmt.Errorf("cannot assign resources to dispatch in status: %s", d.Status)
	}
	d.DriverID = &driverID
	d.VehicleID = &vehicleID
	d.Status = DispatchAssigned
	d.UpdatedAt = time.Now()
	return nil
}

// ConvertToTrip validates and marks dispatch as converted.
func (d *Dispatch) ConvertToTrip(tripID types.TripID) error {
	if d.Status != DispatchAssigned {
		return ErrUnassignedResource
	}
	d.TripID = &tripID
	d.Status = DispatchConverted
	d.UpdatedAt = time.Now()
	return nil
}

// Cancel cancels the dispatch planning order.
func (d *Dispatch) Cancel() error {
	if d.Status == DispatchConverted {
		return fmt.Errorf("converted dispatches cannot be cancelled")
	}
	d.Status = DispatchCancelled
	d.UpdatedAt = time.Now()
	return nil
}
