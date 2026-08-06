package trip

import (
	"context"

	"transport-app/internal/domain/types"
)

// TripRepository defines the interface for trip persistence.
type TripRepository interface {
	CreateTrip(ctx context.Context, trip Trip) (Trip, error)
	GetTripByID(ctx context.Context, id types.TripID) (TripWithJoins, error)
	GetTripByNumber(ctx context.Context, number string) (TripWithJoins, error)
	GetTripByBookingID(ctx context.Context, bookingID types.BookingID) (TripWithJoins, error)
	UpdateTrip(ctx context.Context, trip Trip) (Trip, error)
	UpdateTripStatus(ctx context.Context, id types.TripID, status TripStatus) (Trip, error)
	AssignDriver(ctx context.Context, tripID types.TripID, driverID types.DriverID) (Trip, error)
	AssignVehicle(ctx context.Context, tripID types.TripID, vehicleID types.VehicleID) (Trip, error)
	DeleteTrip(ctx context.Context, id types.TripID) error
	SearchTrips(ctx context.Context, query string, status string, limit, offset int) ([]TripWithJoins, error)
	CountTrips(ctx context.Context, query string, status string) (int64, error)
	CheckVehicleConflict(ctx context.Context, vehicleID types.VehicleID, excludeTripID *types.TripID) ([]Trip, error)
	CheckDriverConflict(ctx context.Context, driverID types.DriverID, excludeTripID *types.TripID) ([]Trip, error)
	GetTripsByDate(ctx context.Context, date string) ([]TripWithJoins, error)
	CountTripsByStatusForDate(ctx context.Context, date string) (map[TripStatus]int64, error)
}

// TripWithJoins includes driver, vehicle, and route details.
type TripWithJoins struct {
	Trip
	DriverDisplayID     *string
	DriverFirstName     *string
	DriverLastName      *string
	VehicleRegistration *string
	VehicleNumber       *string
	RouteSource         string
	RouteDestination    string
}
