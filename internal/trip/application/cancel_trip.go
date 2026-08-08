package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// CancelTripCommand contains parameters to transition a trip to cancelled.
type CancelTripCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

// CancelTripUseCase orchestrates cancelling a trip.
type CancelTripUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewCancelTripUseCase creates a new CancelTripUseCase.
func NewCancelTripUseCase(uow ports.UnitOfWork, clock ports.Clock) *CancelTripUseCase {
	return &CancelTripUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *CancelTripUseCase) Execute(ctx context.Context, cmd CancelTripCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.Cancel(uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionCancel, string(t.ID), nil, nil)
		return nil
	})
}
