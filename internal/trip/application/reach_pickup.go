package application

import (
	"context"
	"errors"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

// ReachPickupCommand contains parameters to transition a trip to reached_pickup.
type ReachPickupCommand struct {
	TripID   aggregate.TripID
	TenantID shared.TenantID
}

// ReachPickupUseCase orchestrates marking a trip as having reached the pickup location.
type ReachPickupUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewReachPickupUseCase creates a new ReachPickupUseCase.
func NewReachPickupUseCase(uow ports.UnitOfWork, clock ports.Clock) *ReachPickupUseCase {
	return &ReachPickupUseCase{uow: uow, clock: clock}
}

// Execute performs the transition.
func (uc *ReachPickupUseCase) Execute(ctx context.Context, cmd ReachPickupCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		t, err := repo.Find(txCtx, cmd.TripID, cmd.TenantID)
		if err != nil {
			return err
		}
		if err := t.ReachPickup(uc.clock.Now()); err != nil {
			return err
		}
		if err := repo.Save(txCtx, t); err != nil {
			return err
		}
		logAudit(txCtx, ActionReachPickup, string(t.ID), nil, nil)
		return nil
	})
}
