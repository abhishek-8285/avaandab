package payment

import (
	"context"

	"transport-app/internal/payment/application"
	"transport-app/internal/payment/domain/aggregate"
)

type paymentFacadeImpl struct {
	recordUC *application.RecordPaymentUseCase
}

// NewPaymentFacade constructs a new PaymentFacade implementation.
func NewPaymentFacade(recordUC *application.RecordPaymentUseCase) PaymentFacade {
	return &paymentFacadeImpl{recordUC: recordUC}
}

func (f *paymentFacadeImpl) RecordPayment(ctx context.Context, cmd RecordPaymentCommand) (aggregate.PaymentID, error) {
	return f.recordUC.Execute(ctx, application.RecordPaymentCommand(cmd))
}
