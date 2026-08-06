package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// StartTransitCommand contains parameters to transition a trip to in_transit.
type StartTransitCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

// StartTransitUseCase orchestrates marking a trip as in_transit.
type StartTransitUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewStartTransitUseCase creates a new StartTransitUseCase.
func NewStartTransitUseCase(uow ports.UnitOfWork, clock ports.Clock) *StartTransitUseCase {
	return &StartTransitUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *StartTransitUseCase) Execute(ctx context.Context, cmd StartTransitCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.StartTransit(uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionStartTransit, string(t.ID), nil, nil)
		return nil
	})
}
