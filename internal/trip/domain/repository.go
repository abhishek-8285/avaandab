package domain

import (
	"context"
	"time"

	"transport-app/internal/shared"
	"transport-app/internal/trip/domain/aggregate"
)

// TripReadModel represents an optimized read model combining trip, driver, vehicle, and route details.
type TripReadModel struct {
	ID                        string
	TripNumber                string
	BookingID                 *string
	DriverID                  *string
	DriverDisplayID           string
	DriverFirstName           string
	DriverLastName            string
	VehicleID                 *string
	VehicleRegistrationNumber string
	VehicleNumber             string
	RouteID                   string
	RouteSource               string
	RouteDestination          string
	DepartureTime             time.Time
	ArrivalTime               *time.Time
	Status                    string
	Remarks                   string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	StartedAt                 *time.Time
	ReachedPickupAt           *time.Time
	InTransitAt               *time.Time
	DeliveredAt               *time.Time
	CompletedAt               *time.Time
}

// ConflictInfo describes a trip that conflicts with a proposed assignment.
type ConflictInfo struct {
	ID            string
	TripNumber    string
	Status        string
	DepartureTime time.Time
	ArrivalTime   *time.Time
}

// TripRepository defines the contract for persisting and retrieving TripAggregates and Read Models.
type TripRepository interface {
	Save(ctx context.Context, t *aggregate.TripAggregate) error
	Find(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (*aggregate.TripAggregate, error)
	FindByNumber(ctx context.Context, number string, tenantID shared.TenantID) (*aggregate.TripAggregate, error)
	Exists(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (bool, error)

	// Read Model Queries
	GetReadModel(ctx context.Context, id aggregate.TripID, tenantID shared.TenantID) (TripReadModel, error)
	SearchReadModels(ctx context.Context, tenantID shared.TenantID, query string, status string, limit int, offset int) ([]TripReadModel, int64, error)
	SearchReadModelsByDriver(ctx context.Context, tenantID shared.TenantID, driverIDs []string, query string, status string, limit int, offset int) ([]TripReadModel, int64, error)

	// Conflict Checks
	CheckDriverConflict(ctx context.Context, driverID string, tenantID shared.TenantID, excludeTripID string) ([]ConflictInfo, error)
	CheckVehicleConflict(ctx context.Context, vehicleID string, tenantID shared.TenantID, excludeTripID string) ([]ConflictInfo, error)
}
