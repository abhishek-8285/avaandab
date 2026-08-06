package application

import (
	"context"
	"errors"

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
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.AssignDriver(cmd.DriverID, uc.clock.Now()); err != nil {
			return err
		}
		return repo.Save(txCtx, t)
	})
}
