package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
	vehicleDomain "transport-app/internal/vehicle/domain"
	vehicleAgg "transport-app/internal/vehicle/domain/aggregate"
)

type AssignVehicleCommand struct {
	TripID    aggregate.TripID
	VehicleID string
	TenantID  shared.TenantID
}

type AssignVehicleUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

func NewAssignVehicleUseCase(uow ports.UnitOfWork, clock ports.Clock) *AssignVehicleUseCase {
	return &AssignVehicleUseCase{uow: uow, clock: clock}
}

func (uc *AssignVehicleUseCase) Execute(ctx context.Context, cmd AssignVehicleCommand) error {
	if cmd.VehicleID == "" {
		return errors.New("vehicle ID is required")
	}
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := uc.checkVehicleCompliance(txCtx, cmd.VehicleID, cmd.TenantID); err != nil {
			return err
		}
		conflicts, err := repo.CheckVehicleConflict(txCtx, cmd.VehicleID, cmd.TenantID, string(cmd.TripID))
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("vehicle %s has conflicting trips: %s", cmd.VehicleID, conflicts[0].TripNumber)
		}
		if err := t.AssignVehicle(cmd.VehicleID, uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionAssign, string(t.ID), nil, nil)
		return nil
	})
}

func (uc *AssignVehicleUseCase) checkVehicleCompliance(ctx ports.TxContext, vehicleID string, tenantID shared.TenantID) error {
	vehicleRepo, ok := ctx.Repositories().Vehicles().(vehicleDomain.VehicleRepository)
	if !ok {
		return errors.New("failed to retrieve vehicle repository")
	}
	v, err := vehicleRepo.Find(ctx, vehicleAgg.VehicleID(vehicleID), tenantID)
	if err != nil {
		return fmt.Errorf("vehicle %s not found: %w", vehicleID, err)
	}
	if v.Status == vehicleAgg.VehicleInactive || v.Status == vehicleAgg.VehicleMaintenance {
		return fmt.Errorf("vehicle %s is not assignable (status: %s)", vehicleID, v.Status)
	}
	now := uc.clock.Now().Truncate(24 * time.Hour)
	for _, expiry := range []struct {
		name string
		when time.Time
	}{{"insurance", v.InsuranceExpiry}, {"fitness", v.FitnessExpiry}, {"permit", v.PermitExpiry}} {
		if !expiry.when.IsZero() && expiry.when.Before(now) {
			return fmt.Errorf("vehicle %s %s expired on %s", vehicleID, expiry.name, expiry.when.Format("2006-01-02"))
		}
	}
	return nil
}
