package invoice

import (
	"context"

	"transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/shared"
)

// GenerateInvoiceCommand contains parameters to create an invoice.
type GenerateInvoiceCommand struct {
	TenantID   shared.TenantID
	BookingID  string
	CustomerID string
	TripID     *string
	Subtotal   float64
	Tax        float64
	Discount   float64
	Total      float64
}

// InvoiceFacade defines entry points into the invoice module.
type InvoiceFacade interface {
	GenerateInvoice(ctx context.Context, cmd GenerateInvoiceCommand) (aggregate.InvoiceID, error)
}
