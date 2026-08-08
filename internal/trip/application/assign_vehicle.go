package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
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
