package aggregate

import (
	"errors"
	"time"

	"transport-app/internal/shared"
)

type TripID string
type TripStatus string

const (
	TripDraft     TripStatus = "draft"
	TripScheduled TripStatus = "scheduled"
	TripAssigned  TripStatus = "assigned"
	TripStarted   TripStatus = "started"
	TripCompleted TripStatus = "completed"
	TripCancelled TripStatus = "cancelled"
)

// TripAggregate represents the consistency boundary for a single transport Trip.
type TripAggregate struct {
	ID            TripID
	TenantID      shared.TenantID
	TripNumber    string
	BookingID     *string
	DriverID      *string
	VehicleID     *string
	RouteID       string
	DepartureTime time.Time
	ArrivalTime   *time.Time
	Status        TripStatus
	Remarks       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Version       int64

	events []any
}

// NewTripAggregate creates a new TripAggregate in 'draft' status.
func NewTripAggregate(
	id TripID,
	tenantID shared.TenantID,
	tripNumber string,
	bookingID *string,
	routeID string,
	departureTime time.Time,
	remarks string,
	now time.Time,
) *TripAggregate {
	t := &TripAggregate{
		ID:            id,
		TenantID:      tenantID,
		TripNumber:    tripNumber,
		BookingID:     bookingID,
		RouteID:       routeID,
		DepartureTime: departureTime,
		Status:        TripDraft,
		Remarks:       remarks,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.RecordEvent(TripCreatedEvent{
		TripID:     id,
		TenantID:   tenantID,
		TripNumber: tripNumber,
		OccurredAt: now,
	})

	return t
}

// Schedule updates status to scheduled.
func (t *TripAggregate) Schedule(now time.Time) error {
	if t.Status != TripDraft {
		return errors.New("only draft trips can be scheduled")
	}
	t.Status = TripScheduled
	t.UpdatedAt = now
	t.RecordEvent(TripScheduledEvent{
		TripID:     t.ID,
		TenantID:   t.TenantID,
		OccurredAt: now,
	})
	return nil
}

// AssignDriver associates a driver.
func (t *TripAggregate) AssignDriver(driverID string, now time.Time) error {
	if t.Status == TripCompleted || t.Status == TripCancelled {
		return errors.New("cannot assign driver to completed or cancelled trip")
	}
	t.DriverID = &driverID
	t.Status = TripAssigned
	t.UpdatedAt = now
	t.RecordEvent(TripAssignedEvent{
		TripID:     t.ID,
		TenantID:   t.TenantID,
		DriverID:   driverID,
		OccurredAt: now,
	})
	return nil
}

// AssignVehicle associates a vehicle.
func (t *TripAggregate) AssignVehicle(vehicleID string, now time.Time) error {
	if t.Status == TripCompleted || t.Status == TripCancelled {
		return errors.New("cannot assign vehicle to completed or cancelled trip")
	}
	t.VehicleID = &vehicleID
	t.UpdatedAt = now
	return nil
}

// Start moves status to started.
func (t *TripAggregate) Start(now time.Time) error {
	if t.Status != TripAssigned && t.Status != TripScheduled {
		return errors.New("trip must be scheduled or assigned to start")
	}
	t.Status = TripStarted
	t.UpdatedAt = now
	t.RecordEvent(TripStartedEvent{
		TripID:     t.ID,
		TenantID:   t.TenantID,
		OccurredAt: now,
	})
	return nil
}

// Complete completes the trip execution.
func (t *TripAggregate) Complete(now time.Time) error {
	if t.Status == TripCompleted {
		return nil
	}
	if t.Status != TripStarted {
		return errors.New("only started trips can be completed")
	}
	t.Status = TripCompleted
	t.ArrivalTime = &now
	t.UpdatedAt = now
	t.RecordEvent(TripCompletedEvent{
		TripID:     t.ID,
		TenantID:   t.TenantID,
		OccurredAt: now,
	})
	return nil
}

// Cancel cancels the trip.
func (t *TripAggregate) Cancel(now time.Time) error {
	if t.Status == TripCompleted {
		return errors.New("completed trips cannot be cancelled")
	}
	t.Status = TripCancelled
	t.UpdatedAt = now
	t.RecordEvent(TripCancelledEvent{
		TripID:     t.ID,
		TenantID:   t.TenantID,
		OccurredAt: now,
	})
	return nil
}

// Events returns collected events.
func (t *TripAggregate) Events() []any {
	return t.events
}

// ClearEvents clears the events.
func (t *TripAggregate) ClearEvents() {
	t.events = nil
}

// RecordEvent records a domain event.
func (t *TripAggregate) RecordEvent(event any) {
	t.events = append(t.events, event)
}

// Event Definitions
type TripCreatedEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	TripNumber string
	OccurredAt time.Time
}

type TripScheduledEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	OccurredAt time.Time
}

type TripAssignedEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	DriverID   string
	OccurredAt time.Time
}

type TripStartedEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	OccurredAt time.Time
}

type TripCompletedEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	OccurredAt time.Time
}

type TripCancelledEvent struct {
	TripID     TripID
	TenantID   shared.TenantID
	OccurredAt time.Time
}
