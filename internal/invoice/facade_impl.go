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
	return f.generateUC.Execute(ctx, application.GenerateInvoiceCommand(cmd))
}
