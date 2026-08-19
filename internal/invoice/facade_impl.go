package invoice

import (
	"context"

	"transport-app/internal/invoice/application"
	"transport-app/internal/invoice/domain/aggregate"
)

type invoiceFacadeImpl struct {
	generateUC *application.GenerateInvoiceUseCase
}

// NewInvoiceFacade constructs a new InvoiceFacade implementation.
func NewInvoiceFacade(generateUC *application.GenerateInvoiceUseCase) InvoiceFacade {
	return &invoiceFacadeImpl{generateUC: generateUC}
}

func (f *invoiceFacadeImpl) GenerateInvoice(ctx context.Context, cmd GenerateInvoiceCommand) (aggregate.InvoiceID, error) {
	appCmd := application.GenerateInvoiceCommand{
		TenantID:   cmd.TenantID,
		BookingID:  cmd.BookingID,
		CustomerID: cmd.CustomerID,
		TripID:     cmd.TripID,
		Subtotal:   cmd.Subtotal,
		Tax:        cmd.Tax,
		Discount:   cmd.Discount,
		Total:      cmd.Total,
	}
	return f.generateUC.Execute(ctx, appCmd)
}
