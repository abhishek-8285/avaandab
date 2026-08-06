package application

import (
	"context"
	"errors"

	"transport-app/internal/booking/domain"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// CompleteBookingCommand defines parameters for completing a booking.
type CompleteBookingCommand struct {
	BookingID aggregate.BookingID
	TenantID  shared.TenantID
}

// CompleteBookingUseCase orchestrates the completion of a booking.
type CompleteBookingUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewCompleteBookingUseCase creates a new CompleteBookingUseCase.
func NewCompleteBookingUseCase(uow ports.UnitOfWork, clock ports.Clock) *CompleteBookingUseCase {
	return &CompleteBookingUseCase{uow: uow, clock: clock}
}

// Execute marks the booking aggregate as completed within transactional boundaries.
func (uc *CompleteBookingUseCase) Execute(ctx context.Context, cmd CompleteBookingCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		booking, err := repo.Find(txCtx, cmd.BookingID, cmd.TenantID)
		if err != nil {
			return err
		}

		if err := booking.Complete(uc.clock.Now()); err != nil {
			return err
		}

		if err := repo.Save(txCtx, booking); err != nil {
			return err
		}
		logAudit(txCtx, ActionComplete, string(booking.ID), nil, nil)
		return nil
	})
}
