package route

import (
	"context"

	"transport-app/internal/domain/types"
)

// CreateRouteRequest carries all fields needed to create a route.
type CreateRouteRequest struct {
	Source              string
	Destination         string
	Distance            float64
	EstimatedHours      float64
	StandardFare        float64
	ReverseDistance     *float64
	ReverseStandardFare *float64
	Direction           string
	Remarks             string
}

// UpdateRouteRequest carries all fields needed to update a route.
type UpdateRouteRequest struct {
	Source              string
	Destination         string
	Distance            float64
	EstimatedHours      float64
	StandardFare        float64
	ReverseDistance     *float64
	ReverseStandardFare *float64
	Direction           string
	IsActive            bool
	Remarks             string
}

// RouteService defines the interface for route business operations.
type RouteService interface {
	CreateRoute(ctx context.Context, source, destination string, distance, estHours, standardFare float64, remarks string) (Route, error)
	CreateRouteFull(ctx context.Context, req CreateRouteRequest) (Route, error)
	GetRoute(ctx context.Context, id types.RouteID) (Route, error)
	ListRoutes(ctx context.Context, query string, limit, offset int) ([]Route, int64, error)
	UpdateRoute(ctx context.Context, id types.RouteID, source, destination string, distance, estHours, standardFare float64, remarks string) (Route, error)
	UpdateRouteFull(ctx context.Context, id types.RouteID, req UpdateRouteRequest) (Route, error)
	DeleteRoute(ctx context.Context, id types.RouteID) error
}
