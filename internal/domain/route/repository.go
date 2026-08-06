package route

import (
	"context"

	"transport-app/internal/domain/types"
)

// RouteRepository defines the interface for route persistence.
type RouteRepository interface {
	CreateRoute(ctx context.Context, route Route) (Route, error)
	GetRouteByID(ctx context.Context, id types.RouteID) (Route, error)
	GetRouteBySourceAndDestination(ctx context.Context, source, destination string) (Route, error)
	UpdateRoute(ctx context.Context, route Route) (Route, error)
	DeleteRoute(ctx context.Context, id types.RouteID) error
	SearchRoutes(ctx context.Context, query string, limit, offset int) ([]Route, error)
	CountRoutes(ctx context.Context, query string) (int64, error)
}
