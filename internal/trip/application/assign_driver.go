package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	driverDomain "transport-app/internal/driver/domain"
	driverAgg "transport-app/internal/driver/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

type AssignDriverCommand struct {
	TripID   aggregate.TripID
	DriverID string
	TenantID shared.TenantID
}

type AssignDriverUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

func NewAssignDriverUseCase(uow ports.UnitOfWork, clock ports.Clock) *AssignDriverUseCase {
	return &AssignDriverUseCase{uow: uow, clock: clock}
}

func (uc *AssignDriverUseCase) Execute(ctx context.Context, cmd AssignDriverCommand) error {
	if cmd.DriverID == "" {
		return errors.New("driver ID is required")
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
		if err := uc.checkDriverCompliance(txCtx, cmd.DriverID, cmd.TenantID); err != nil {
			return err
		}
		conflicts, err := repo.CheckDriverConflict(txCtx, cmd.DriverID, cmd.TenantID, string(cmd.TripID))
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("driver %s has conflicting trips: %s", cmd.DriverID, conflicts[0].TripNumber)
		}
		if err := t.AssignDriver(cmd.DriverID, uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionAssign, string(t.ID), nil, nil)
		return nil
	})
}

func (uc *AssignDriverUseCase) checkDriverCompliance(ctx ports.TxContext, driverID string, tenantID shared.TenantID) error {
	driverRepo, ok := ctx.Repositories().Drivers().(driverDomain.DriverRepository)
	if !ok {
		return errors.New("failed to retrieve driver repository")
	}
	d, err := driverRepo.Find(ctx, driverAgg.DriverID(driverID), tenantID)
	if err != nil {
		return fmt.Errorf("driver %s not found: %w", driverID, err)
	}
	if d.Status == driverAgg.DriverInactive || d.Status == driverAgg.DriverLeave {
		return fmt.Errorf("driver %s is not assignable (status: %s)", driverID, d.Status)
	}
	if !d.LicenseExpiry.IsZero() && d.LicenseExpiry.Before(uc.clock.Now().Truncate(24*time.Hour)) {
		return fmt.Errorf("driver %s license expired on %s", driverID, d.LicenseExpiry.Format("2006-01-02"))
	}
	return nil
}
