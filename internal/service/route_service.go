package service

import (
	"context"
	"fmt"

	"transport-app/internal/domain"
)

// RouteService handles route management.
type RouteService struct {
	baseService
}

// CreateRoute creates a new route.
func (s *RouteService) CreateRoute(ctx context.Context, source, destination string, distance, estHours, standardFare float64, remarks string) (domain.Route, error) {
	if source == "" || destination == "" {
		return domain.Route{}, fmt.Errorf("source and destination are required")
	}
	if source == destination {
		return domain.Route{}, fmt.Errorf("source and destination must be different")
	}
	if distance <= 0 {
		return domain.Route{}, fmt.Errorf("distance must be greater than zero")
	}
	if estHours <= 0 {
		return domain.Route{}, fmt.Errorf("estimated hours must be greater than zero")
	}
	if standardFare <= 0 {
		return domain.Route{}, fmt.Errorf("base fare must be greater than zero")
	}

	// Check uniqueness
	if _, err := s.store.GetRouteBySourceAndDestination(ctx, source, destination); err == nil {
		return domain.Route{}, fmt.Errorf("route from %s to %s already exists", source, destination)
	}

	route := domain.Route{
		ID:             domain.RouteID(generateID()),
		Source:         source,
		Destination:    destination,
		Distance:       distance,
		EstimatedHours: estHours,
		StandardFare:   standardFare,
		Remarks:        strPtr(remarks),
	}

	created, err := s.store.CreateRoute(ctx, route)
	if err != nil {
		return domain.Route{}, err
	}

	s.log.Info("route created", "route_id", created.ID)
	return created, nil
}

// GetRoute retrieves a route by ID.
func (s *RouteService) GetRoute(ctx context.Context, id domain.RouteID) (domain.Route, error) {
	return s.store.GetRouteByID(ctx, id)
}

// ListRoutes retrieves routes with search and pagination.
func (s *RouteService) ListRoutes(ctx context.Context, query string, limit, offset int) ([]domain.Route, int64, error) {
	routes, err := s.store.SearchRoutes(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountRoutes(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return routes, total, nil
}

// UpdateRoute updates an existing route.
func (s *RouteService) UpdateRoute(ctx context.Context, id domain.RouteID, source, destination string, distance, estHours, standardFare float64, remarks string) (domain.Route, error) {
	route, err := s.store.GetRouteByID(ctx, id)
	if err != nil {
		return domain.Route{}, domain.ErrRouteNotFound
	}

	if source == "" || destination == "" {
		return domain.Route{}, fmt.Errorf("source and destination are required")
	}
	if source == destination {
		return domain.Route{}, fmt.Errorf("source and destination must be different")
	}
	if distance <= 0 {
		return domain.Route{}, fmt.Errorf("distance must be greater than zero")
	}
	if estHours <= 0 {
		return domain.Route{}, fmt.Errorf("estimated hours must be greater than zero")
	}
	if standardFare <= 0 {
		return domain.Route{}, fmt.Errorf("base fare must be greater than zero")
	}

	// Check uniqueness for other routes
	if existing, err := s.store.GetRouteBySourceAndDestination(ctx, source, destination); err == nil && existing.ID != id {
		return domain.Route{}, fmt.Errorf("route from %s to %s already exists", source, destination)
	}

	route.Source = source
	route.Destination = destination
	route.Distance = distance
	route.EstimatedHours = estHours
	route.StandardFare = standardFare
	route.Remarks = strPtr(remarks)

	updated, err := s.store.UpdateRoute(ctx, route)
	if err != nil {
		return domain.Route{}, err
	}

	s.log.Info("route updated", "route_id", id)
	return updated, nil
}

// DeleteRoute deletes a route.
func (s *RouteService) DeleteRoute(ctx context.Context, id domain.RouteID) error {
	if err := s.store.DeleteRoute(ctx, id); err != nil {
		return err
	}
	s.log.Info("route deleted", "route_id", id)
	return nil
}
