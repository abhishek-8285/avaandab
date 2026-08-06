package application

import (
	"context"
	"errors"
	"time"

	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
	"transport-app/internal/trip/domain"
	"transport-app/internal/trip/domain/aggregate"
)

type CreateTripCommand struct {
	TenantID      shared.TenantID
	BookingID     *string
	RouteID       string
	DepartureTime time.Time
	Remarks       string
}

type CreateTripUseCase struct {
	uow   ports.UnitOfWork
	idGen ports.IDGenerator
	clock ports.Clock
}

func NewCreateTripUseCase(uow ports.UnitOfWork, idGen ports.IDGenerator, clock ports.Clock) *CreateTripUseCase {
	return &CreateTripUseCase{uow: uow, idGen: idGen, clock: clock}
}

func (uc *CreateTripUseCase) Execute(ctx context.Context, cmd CreateTripCommand) (aggregate.TripID, error) {
	if cmd.RouteID == "" {
		return "", errors.New("route ID is required")
	}
	if cmd.DepartureTime.IsZero() {
		return "", errors.New("departure time is required")
	}

	tripID := aggregate.TripID(uc.idGen.GenerateUUID())
	tripNumber := uc.idGen.GenerateDisplayID("TR")

	trip := aggregate.NewTripAggregate(
		tripID,
		cmd.TenantID,
		tripNumber,
		cmd.BookingID,
		cmd.RouteID,
		cmd.DepartureTime,
		cmd.Remarks,
		uc.clock.Now(),
	)

	err := uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Trips().(domain.TripRepository)
		if !ok {
			return errors.New("failed to retrieve trip repository")
		}
		if err := repo.Save(txCtx, trip); err != nil {
			return err
		}
		logAudit(txCtx, ActionCreate, string(trip.ID), nil, nil)
		return nil
	})

	if err != nil {
		return "", err
	}

	return tripID, nil
}
