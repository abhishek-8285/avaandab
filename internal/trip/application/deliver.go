package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// DeliverCommand contains parameters to transition a trip to delivered.
type DeliverCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

// DeliverUseCase orchestrates marking a trip as delivered.
type DeliverUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewDeliverUseCase creates a new DeliverUseCase.
func NewDeliverUseCase(uow ports.UnitOfWork, clock ports.Clock) *DeliverUseCase {
	return &DeliverUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *DeliverUseCase) Execute(ctx context.Context, cmd DeliverCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.Deliver(uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionDeliver, string(t.ID), nil, nil)
		return nil
	})
}
