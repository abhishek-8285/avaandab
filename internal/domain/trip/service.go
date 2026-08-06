package trip

import (
	"context"

	"transport-app/internal/domain/types"
)

// TripService defines the interface for trip business operations.
type TripService interface {
	CreateTrip(ctx context.Context, req CreateTripRequest) (Trip, error)
	GetTrip(ctx context.Context, id types.TripID) (TripWithJoins, error)
	GetTripByNumber(ctx context.Context, number string) (TripWithJoins, error)
	ListTrips(ctx context.Context, query, status string, limit, offset int) ([]TripWithJoins, int64, error)
	GetTripByBooking(ctx context.Context, bookingID types.BookingID) (*Trip, error)
	UpdateTrip(ctx context.Context, id types.TripID, req CreateTripRequest) (Trip, error)
	ScheduleTrip(ctx context.Context, id types.TripID) (Trip, error)
	AssignDriver(ctx context.Context, tripID types.TripID, driverID types.DriverID) (Trip, error)
	AssignVehicle(ctx context.Context, tripID types.TripID, vehicleID types.VehicleID) (Trip, error)
	StartTrip(ctx context.Context, id types.TripID) (Trip, error)
	CompleteTrip(ctx context.Context, id types.TripID) (Trip, error)
	CancelTrip(ctx context.Context, id types.TripID) (Trip, error)
	DeleteTrip(ctx context.Context, id types.TripID) error
}

// CreateTripRequest contains fields needed to create or update a trip.
type CreateTripRequest struct {
	BookingID     *types.BookingID
	RouteID       types.RouteID
	DriverID      *types.DriverID
	VehicleID     *types.VehicleID
	DepartureTime string
	ArrivalTime   string
	Remarks       string
}
