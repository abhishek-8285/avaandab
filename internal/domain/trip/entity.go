package trip

import (
	"fmt"
	"time"

	"transport-app/internal/domain/types"
)

// Trip represents a scheduled transport trip.
type Trip struct {
	ID            types.TripID
	TripNumber    string
	BookingID     *types.BookingID
	DriverID      *types.DriverID
	VehicleID     *types.VehicleID
	RouteID       types.RouteID
	DepartureTime time.Time
	ArrivalTime   *time.Time
	Status        TripStatus
	Remarks       *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TripStatus represents the lifecycle status of a trip.
type TripStatus string

const (
	TripDraft     TripStatus = "draft"
	TripScheduled TripStatus = "scheduled"
	TripAssigned  TripStatus = "assigned"
	TripStarted   TripStatus = "started"
	TripCompleted TripStatus = "completed"
	TripCancelled TripStatus = "cancelled"
)

// ActiveTripStatuses are statuses that indicate a trip is currently in progress.
var ActiveTripStatuses = []TripStatus{
	TripScheduled,
	TripAssigned,
	TripStarted,
}

// CanSchedule validates that a trip can be scheduled (must be in draft status).
func (t Trip) CanSchedule() error {
	if t.Status != TripDraft {
		return fmt.Errorf("only draft trips can be scheduled; current status: %s", t.Status)
	}
	return nil
}

// CanStart validates that a trip can be started (must be assigned).
func (t Trip) CanStart() error {
	if t.Status != TripAssigned {
		return fmt.Errorf("only assigned trips can be started; current status: %s", t.Status)
	}
	return nil
}

// CanComplete validates that a trip can be completed (must have been started).
func (t Trip) CanComplete() error {
	if t.Status != TripStarted {
		return fmt.Errorf("only started trips can be completed; current status: %s", t.Status)
	}
	return nil
}

// CanCancel validates that a trip can be cancelled (must not be completed).
func (t Trip) CanCancel() error {
	if t.Status == TripCompleted {
		return ErrCompletedTripImmutable
	}
	return nil
}
