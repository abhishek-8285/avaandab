package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// StartTripCommand contains parameters to transition a trip to started.
type StartTripCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

// StartTripUseCase orchestrates starting a trip.
type StartTripUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewStartTripUseCase creates a new StartTripUseCase.
func NewStartTripUseCase(uow ports.UnitOfWork, clock ports.Clock) *StartTripUseCase {
	return &StartTripUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *StartTripUseCase) Execute(ctx context.Context, cmd StartTripCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.Start(uc.clock.Now()); err != nil {
			return err
		}
		return repo.Save(txCtx, t)
	})
}
