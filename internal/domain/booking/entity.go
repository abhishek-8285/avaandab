package booking

import (
	"errors"
	"fmt"
	"time"

	"transport-app/internal/domain/types"
	"transport-app/internal/domain/vehicle"
)

// ErrInvalidState is returned when a state transition is not allowed.
var ErrInvalidState = errors.New("invalid state transition")

// Booking represents a customer's booking request.
type Booking struct {
	ID            types.BookingID
	BookingNumber string
	CustomerID    types.CustomerID
	PickupDate    time.Time
	RouteID       types.RouteID
	VehicleType   vehicle.VehicleType
	Passengers    int64
	CargoWeight   *float64
	Price         float64
	Notes         *string
	Status        BookingStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// BookingStatus represents the status of a booking.
type BookingStatus string

const (
	BookingDraft     BookingStatus = "draft"
	BookingPending   BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingCancelled BookingStatus = "cancelled"
	BookingCompleted BookingStatus = "completed"
)

// CanConfirm validates that a booking can be confirmed (must be pending).
func (b Booking) CanConfirm() error {
	if b.Status == BookingCancelled {
		return fmt.Errorf("cancelled bookings cannot be confirmed")
	}
	if b.Status != BookingPending {
		return ErrInvalidState
	}
	return nil
}

// CanCancel validates that a booking can be cancelled (must not be completed).
func (b Booking) CanCancel() error {
	if b.Status == BookingCompleted {
		return fmt.Errorf("completed bookings cannot be cancelled")
	}
	return nil
}

// CanDelete validates that a booking can be deleted (must be pending or confirmed).
func (b Booking) CanDelete() error {
	if b.Status != BookingPending && b.Status != BookingConfirmed {
		return fmt.Errorf("only pending or confirmed bookings can be deleted; current status: %s", b.Status)
	}
	return nil
}
