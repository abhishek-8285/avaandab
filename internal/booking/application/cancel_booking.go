package application

import (
	"context"
	"errors"
	"transport-app/internal/booking/domain"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// CancelBookingCommand defines parameters for cancelling a booking.
type CancelBookingCommand struct {
	BookingID aggregate.BookingID
	TenantID  shared.TenantID
}

// CancelBookingUseCase orchestrates the cancellation of a booking.
type CancelBookingUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewCancelBookingUseCase creates a new CancelBookingUseCase.
func NewCancelBookingUseCase(uow ports.UnitOfWork, clock ports.Clock) *CancelBookingUseCase {
	return &CancelBookingUseCase{uow: uow, clock: clock}
}

// Execute marks the booking aggregate as cancelled within transactional boundaries.
func (uc *CancelBookingUseCase) Execute(ctx context.Context, cmd CancelBookingCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		booking, err := repo.Find(txCtx, cmd.BookingID, cmd.TenantID)
		if err != nil {
			return err
		}

		if err := booking.Cancel(uc.clock.Now()); err != nil {
			return err
		}

		if err := repo.Save(txCtx, booking); err != nil {
			return err
		}
		logAudit(txCtx, ActionCancel, string(booking.ID), nil, nil)
		return nil
	})
}
