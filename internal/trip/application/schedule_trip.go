package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

type ScheduleTripCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

type ScheduleTripUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

func NewScheduleTripUseCase(uow ports.UnitOfWork, clock ports.Clock) *ScheduleTripUseCase {
	return &ScheduleTripUseCase{uow: uow, clock: clock}
}

func (uc *ScheduleTripUseCase) Execute(ctx context.Context, cmd ScheduleTripCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.Schedule(uc.clock.Now()); err != nil {
			return err
		}
		return repo.Save(txCtx, t)
	})
}
