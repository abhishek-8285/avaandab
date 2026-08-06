package workflow

import (
	"context"

	"transport-app/internal/payment/application"
	"transport-app/internal/payment/domain/aggregate"
)

// PaymentWorkflow orchestrates payment lifecycle operations,
// coordinating with the invoice module when payments are recorded.
type PaymentWorkflow struct {
	recordUC  *application.RecordPaymentUseCase
	getUC     *application.GetPaymentUseCase
	listUC    *application.ListPaymentsUseCase
}

// NewPaymentWorkflow creates a new PaymentWorkflow.
func NewPaymentWorkflow(
	recordUC *application.RecordPaymentUseCase,
	getUC *application.GetPaymentUseCase,
	listUC *application.ListPaymentsUseCase,
) *PaymentWorkflow {
	return &PaymentWorkflow{
		recordUC: recordUC,
		getUC:    getUC,
		listUC:   listUC,
	}
}

// RecordPayment records a payment for an invoice.
func (w *PaymentWorkflow) RecordPayment(ctx context.Context, cmd application.RecordPaymentCommand) (aggregate.PaymentID, error) {
	return w.recordUC.Execute(ctx, cmd)
}

// CanProcessPayment returns whether the invoice status allows payment processing.
func (w *PaymentWorkflow) CanProcessPayment(invoiceStatus string) bool {
	return invoiceStatus == "pending" || invoiceStatus == "partially_paid"
}
