// Zone CRUD use cases for the geofence registry (Spec 02 §8).
package application

import (
	"context"
	"errors"
	"fmt"

	"transport-app/internal/geofence/domain"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// CreateZoneCommand contains the fields for a new geofence definition.
type CreateZoneCommand struct {
	TenantID  shared.TenantID
	Name      string
	Kind      string // pickup | drop | depot | restricted | no_entry
	Shape     string // circle | polygon
	CenterLat float64
	CenterLng float64
	RadiusM   float64
	Polygon   []domain.Point
	RouteName string
	Priority  int
	CreatedBy *string
}

// UpdateZoneCommand carries the mutable fields for an existing geofence.
type UpdateZoneCommand struct {
	TenantID  shared.TenantID
	ID        string
	Name      string
	Kind      string
	Shape     string
	CenterLat float64
	CenterLng float64
	RadiusM   float64
	Polygon   []domain.Point
	RouteName string
	Priority  int
}

// ZoneCRUDUseCase creates, updates and soft-deletes geofence definitions.
type ZoneCRUDUseCase struct {
	uow   ports.UnitOfWork
	repo  domain.GeofenceAdminRepository
	idGen ports.IDGenerator
}

// NewZoneCRUDUseCase constructs a ZoneCRUDUseCase.
func NewZoneCRUDUseCase(uow ports.UnitOfWork, repo domain.GeofenceAdminRepository, idGen ports.IDGenerator) *ZoneCRUDUseCase {
	return &ZoneCRUDUseCase{uow: uow, repo: repo, idGen: idGen}
}

// Create validates and persists a new geofence.
func (uc *ZoneCRUDUseCase) Create(ctx context.Context, cmd CreateZoneCommand) (string, error) {
	if err := validateZoneShape(cmd.Shape, cmd.CenterLat, cmd.CenterLng, cmd.RadiusM, cmd.Polygon); err != nil {
		return "", err
	}
	if cmd.Name == "" {
		return "", errors.New("zone name is required")
	}
	if !validKind(cmd.Kind) {
		return "", errors.New("invalid zone kind")
	}

	id := uc.idGen.GenerateUUID()
	zone := domain.Geofence{
		ID:        id,
		TenantID:  string(cmd.TenantID),
		Name:      cmd.Name,
		Kind:      cmd.Kind,
		Shape:     cmd.Shape,
		CenterLat: cmd.CenterLat,
		CenterLng: cmd.CenterLng,
		RadiusM:   cmd.RadiusM,
		Polygon:   cmd.Polygon,
		RouteName: cmd.RouteName,
		Priority:  cmd.Priority,
		IsActive:  true,
		CreatedBy: cmd.CreatedBy,
	}

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		return uc.repo.Create(txCtx, zone)
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// Update validates and persists changes to an existing geofence.
func (uc *ZoneCRUDUseCase) Update(ctx context.Context, cmd UpdateZoneCommand) error {
	if cmd.ID == "" {
		return errors.New("zone ID is required")
	}
	if err := validateZoneShape(cmd.Shape, cmd.CenterLat, cmd.CenterLng, cmd.RadiusM, cmd.Polygon); err != nil {
		return err
	}
	if cmd.Name == "" {
		return errors.New("zone name is required")
	}
	if !validKind(cmd.Kind) {
		return errors.New("invalid zone kind")
	}

	zone := domain.Geofence{
		ID:        cmd.ID,
		TenantID:  string(cmd.TenantID),
		Name:      cmd.Name,
		Kind:      cmd.Kind,
		Shape:     cmd.Shape,
		CenterLat: cmd.CenterLat,
		CenterLng: cmd.CenterLng,
		RadiusM:   cmd.RadiusM,
		Polygon:   cmd.Polygon,
		RouteName: cmd.RouteName,
		Priority:  cmd.Priority,
	}

	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		existing, err := uc.repo.Find(txCtx, string(cmd.TenantID), cmd.ID)
		if err != nil {
			return fmt.Errorf("geofence not found: %w", err)
		}
		zone.IsActive = existing.IsActive
		return uc.repo.Update(txCtx, zone)
	})
}

// SoftDelete deactivates a geofence so the worker stops evaluating it.
func (uc *ZoneCRUDUseCase) SoftDelete(ctx context.Context, tenantID shared.TenantID, id string) error {
	if id == "" {
		return errors.New("zone ID is required")
	}
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		return uc.repo.SoftDelete(txCtx, string(tenantID), id)
	})
}

func validateZoneShape(shape string, lat, lng, radius float64, polygon []domain.Point) error {
	switch shape {
	case domain.ShapeCircle:
		if lat == 0 || lng == 0 || radius <= 0 {
			return errors.New("circle zones require center_lat, center_lng and radius_m > 0")
		}
	case domain.ShapePolygon:
		if len(polygon) < 3 {
			return errors.New("polygon zones require at least 3 vertices")
		}
	default:
		return errors.New("invalid zone shape")
	}
	return nil
}

func validKind(kind string) bool {
	switch kind {
	case domain.KindPickup, domain.KindDrop, domain.KindDepot, domain.KindRestricted, domain.KindNoEntry:
		return true
	}
	return false
}
