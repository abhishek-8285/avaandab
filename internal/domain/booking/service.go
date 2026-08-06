package booking

import (
	"context"

	"transport-app/internal/domain/types"
)

// BookingService defines the interface for booking business operations.
type BookingService interface {
	CreateBooking(ctx context.Context, req CreateBookingRequest) (Booking, error)
	GetBooking(ctx context.Context, id types.BookingID) (BookingWithJoins, error)
	GetBookingByNumber(ctx context.Context, number string) (BookingWithJoins, error)
	ListBookings(ctx context.Context, query, status string, limit, offset int) ([]BookingWithJoins, int64, error)
	UpdateBooking(ctx context.Context, id types.BookingID, req CreateBookingRequest, notes string) (Booking, error)
	ConfirmBooking(ctx context.Context, id types.BookingID) (Booking, error)
	CancelBooking(ctx context.Context, id types.BookingID) (Booking, error)
	CompleteBooking(ctx context.Context, id types.BookingID) (Booking, error)
	DeleteBooking(ctx context.Context, id types.BookingID) error
}

// CreateBookingRequest contains fields needed to create or update a booking.
type CreateBookingRequest struct {
	CustomerID types.CustomerID
	RouteID    types.RouteID
	PickupDate string
	VehicleType string
	Passengers  int64
	CargoWeight *float64
	Price       float64
	Notes       string
}
