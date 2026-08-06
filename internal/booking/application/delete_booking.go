package application

import (
	"context"
	"errors"

	"transport-app/internal/booking/domain"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

type DeleteBookingCommand struct {
	BookingID aggregate.BookingID
	TenantID  shared.TenantID
}

type DeleteBookingUseCase struct {
	uow ports.UnitOfWork
}

func NewDeleteBookingUseCase(uow ports.UnitOfWork) *DeleteBookingUseCase {
	return &DeleteBookingUseCase{uow: uow}
}

func (uc *DeleteBookingUseCase) Execute(ctx context.Context, cmd DeleteBookingCommand) error {
	return uc.uow.Execute(ctx, func(txCtx ports.TxContext) error {
		repo, ok := txCtx.Repositories().Bookings().(domain.BookingRepository)
		if !ok {
			return errors.New("failed to retrieve booking repository")
		}

		exists, err := repo.Exists(txCtx, cmd.BookingID, cmd.TenantID)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("booking not found")
		}

		if err := repo.Delete(txCtx, cmd.BookingID, cmd.TenantID); err != nil {
			return err
		}
		logAudit(txCtx, ActionDelete, string(cmd.BookingID), nil, nil)
		return nil
	})
}
