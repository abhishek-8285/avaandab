package workflow

import (
	"context"

	"transport-app/internal/booking/application"
	"transport-app/internal/booking/domain/aggregate"
	"transport-app/internal/shared"
)

// BookingWorkflow orchestrates the booking lifecycle across use-case boundaries.
// It coordinates state transitions and cross-module side-effects.
type BookingWorkflow struct {
	confirmUC   *application.ConfirmBookingUseCase
	completeUC  *application.CompleteBookingUseCase
	cancelUC    *application.CancelBookingUseCase
}

// NewBookingWorkflow creates a new BookingWorkflow.
func NewBookingWorkflow(
	confirmUC *application.ConfirmBookingUseCase,
	completeUC *application.CompleteBookingUseCase,
	cancelUC *application.CancelBookingUseCase,
) *BookingWorkflow {
	return &BookingWorkflow{
		confirmUC:  confirmUC,
		completeUC: completeUC,
		cancelUC:   cancelUC,
	}
}

// ConfirmBooking marks a booking as confirmed.
// In a future iteration this would trigger trip scheduling via the TripWorkflow.
func (w *BookingWorkflow) ConfirmBooking(ctx context.Context, bookingID aggregate.BookingID, tenantID shared.TenantID) error {
	return w.confirmUC.Execute(ctx, application.ConfirmBookingCommand{
		BookingID: bookingID,
		TenantID:  tenantID,
	})
}

// CompleteBooking marks a confirmed booking as completed.
func (w *BookingWorkflow) CompleteBooking(ctx context.Context, bookingID aggregate.BookingID, tenantID shared.TenantID) error {
	return w.completeUC.Execute(ctx, application.CompleteBookingCommand{
		BookingID: bookingID,
		TenantID:  tenantID,
	})
}

// CancelBooking cancels a booking.
// In a future iteration this would cancel the associated trip.
func (w *BookingWorkflow) CancelBooking(ctx context.Context, bookingID aggregate.BookingID, tenantID shared.TenantID) error {
	return w.cancelUC.Execute(ctx, application.CancelBookingCommand{
		BookingID: bookingID,
		TenantID:  tenantID,
	})
}

// CanComplete returns whether the booking is eligible for completion
// (i.e. it is in the confirmed state).
func (w *BookingWorkflow) CanComplete(status aggregate.BookingStatus) bool {
	return status == aggregate.BookingConfirmed
}

// CanCancel returns whether the booking is eligible for cancellation.
func (w *BookingWorkflow) CanCancel(status aggregate.BookingStatus) bool {
	return status != aggregate.BookingCompleted
}
