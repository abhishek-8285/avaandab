package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// CompleteTripCommand contains parameters to transition a trip to completed.
type CompleteTripCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
	// OnCompleted runs inside the same UnitOfWork transaction after the trip
	// is saved, letting callers attach detentions/invoices atomically with
	// the completion (Spec 02 §6 — no torn states).
	OnCompleted func(txCtx ports.TxContext, trip *aggregate.TripAggregate) error
}

// CompleteTripUseCase orchestrates completing a trip.
type CompleteTripUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewCompleteTripUseCase creates a new CompleteTripUseCase.
func NewCompleteTripUseCase(uow ports.UnitOfWork, clock ports.Clock) *CompleteTripUseCase {
	return &CompleteTripUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *CompleteTripUseCase) Execute(ctx context.Context, cmd CompleteTripCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.Complete(uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionComplete, string(t.ID), nil, nil)
		if cmd.OnCompleted != nil {
			return cmd.OnCompleted(txCtx, t)
		}
		return nil
	})
}
