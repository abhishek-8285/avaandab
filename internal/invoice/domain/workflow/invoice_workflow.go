package workflow

import (
	"context"
	"errors"

	"transport-app/internal/invoice/application"
	"transport-app/internal/invoice/domain/aggregate"
)

// InvoiceWorkflow orchestrates invoice lifecycle operations.
// It coordinates invoice generation, payment status updates, and
// integration with the payment module when invoices are paid.
type InvoiceWorkflow struct {
	generateUC *application.GenerateInvoiceUseCase
	getUC      *application.GetInvoiceUseCase
	listUC     *application.ListInvoicesUseCase
	voidUC     *application.VoidInvoiceUseCase
}

// NewInvoiceWorkflow creates a new InvoiceWorkflow.
func NewInvoiceWorkflow(
	generateUC *application.GenerateInvoiceUseCase,
	getUC *application.GetInvoiceUseCase,
	listUC *application.ListInvoicesUseCase,
	voidUC *application.VoidInvoiceUseCase,
) *InvoiceWorkflow {
	return &InvoiceWorkflow{
		generateUC: generateUC,
		getUC:      getUC,
		listUC:     listUC,
		voidUC:     voidUC,
	}
}

// GenerateInvoice creates a new invoice for a booking.
func (w *InvoiceWorkflow) GenerateInvoice(ctx context.Context, cmd application.GenerateInvoiceCommand) (aggregate.InvoiceID, error) {
	if cmd.BookingID == "" {
		return "", errors.New("booking ID is required")
	}
	if cmd.CustomerID == "" {
		return "", errors.New("customer ID is required")
	}
	return w.generateUC.Execute(ctx, cmd)
}

// CanPay returns whether the invoice status allows payment processing.
func (w *InvoiceWorkflow) CanPay(status aggregate.PaymentStatus) bool {
	return status == aggregate.PaymentStatusPending || status == aggregate.PaymentStatusPartiallyPaid
}

func (w *InvoiceWorkflow) VoidInvoice(ctx context.Context, cmd application.VoidInvoiceCommand) error {
	return w.voidUC.Execute(ctx, cmd)
}
