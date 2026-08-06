package booking

import (
	"context"
	"transport-app/internal/booking/application"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
)

type bookingFacadeImpl struct {
	createUC  *application.CreateBookingUseCase
	confirmUC *application.ConfirmBookingUseCase
	cancelUC  *application.CancelBookingUseCase
}

// NewBookingFacade creates a new concrete implementation of BookingFacade.
func NewBookingFacade(
	createUC *application.CreateBookingUseCase,
	confirmUC *application.ConfirmBookingUseCase,
	cancelUC *application.CancelBookingUseCase,
) BookingFacade {
	return &bookingFacadeImpl{
		createUC:  createUC,
		confirmUC: confirmUC,
		cancelUC:  cancelUC,
	}
}

func (f *bookingFacadeImpl) CreateBooking(ctx context.Context, cmd CreateBookingCommand) (aggregate.BookingID, error) {
	return f.createUC.Execute(ctx, application.CreateBookingCommand(cmd))
}

func (f *bookingFacadeImpl) ConfirmBooking(ctx context.Context, id aggregate.BookingID, tenantID shared.TenantID) error {
	return f.confirmUC.Execute(ctx, application.ConfirmBookingCommand{
		BookingID: id,
		TenantID:  tenantID,
	})
}

func (f *bookingFacadeImpl) CancelBooking(ctx context.Context, id aggregate.BookingID, tenantID shared.TenantID) error {
	return f.cancelUC.Execute(ctx, application.CancelBookingCommand{
		BookingID: id,
		TenantID:  tenantID,
	})
}
