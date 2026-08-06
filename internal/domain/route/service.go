package route

import (
	"context"

	"transport-app/internal/domain/types"
)

// RouteService defines the interface for route business operations.
type RouteService interface {
	CreateRoute(ctx context.Context, source, destination string, distance, estHours, standardFare float64, remarks string) (Route, error)
	GetRoute(ctx context.Context, id types.RouteID) (Route, error)
	ListRoutes(ctx context.Context, query string, limit, offset int) ([]Route, int64, error)
	UpdateRoute(ctx context.Context, id types.RouteID, source, destination string, distance, estHours, standardFare float64, remarks string) (Route, error)
	DeleteRoute(ctx context.Context, id types.RouteID) error
}
