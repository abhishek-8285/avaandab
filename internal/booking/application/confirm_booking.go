package application

import (
	"context"
	"errors"
	"transport-app/internal/booking/domain"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

// ConfirmBookingCommand defines parameters for confirming a booking.
type ConfirmBookingCommand struct {
	BookingID aggregate.BookingID
	TenantID  shared.TenantID
}

// ConfirmBookingUseCase orchestrates the validation and update of a booking confirmation.
type ConfirmBookingUseCase struct {
	uow   ports.UnitOfWork
	clock ports.Clock
}

// NewConfirmBookingUseCase creates a new ConfirmBookingUseCase.
func NewConfirmBookingUseCase(uow ports.UnitOfWork, clock ports.Clock) *ConfirmBookingUseCase {
	return &ConfirmBookingUseCase{uow: uow, clock: clock}
}

// Execute marks the booking aggregate as confirmed within transactional boundaries.
func (uc *ConfirmBookingUseCase) Execute(ctx context.Context, cmd ConfirmBookingCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		booking, err := repo.Find(txCtx, cmd.BookingID, cmd.TenantID)
		if err != nil {
			return err
		}

		if err := booking.Confirm(uc.clock.Now()); err != nil {
			return err
		}

		if err := repo.Save(txCtx, booking); err != nil {
			return err
		}
		logAudit(txCtx, ActionConfirm, string(booking.ID), nil, nil)
		return nil
	})
}
