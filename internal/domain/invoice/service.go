package invoice

import (
	"context"

	"transport-app/internal/domain/payment"
	"transport-app/internal/domain/types"
)

// InvoiceService defines the interface for invoice business operations.
type InvoiceService interface {
	GenerateInvoiceFromTrip(ctx context.Context, tripID types.TripID) (Invoice, error)
	GetInvoice(ctx context.Context, id types.InvoiceID) (InvoiceWithJoins, error)
	GetInvoiceByNumber(ctx context.Context, number string) (InvoiceWithJoins, error)
	ListInvoices(ctx context.Context, query, status string, limit, offset int) ([]InvoiceWithJoins, int64, error)
	GetPendingInvoices(ctx context.Context) ([]InvoiceWithJoins, error)
	UpdateInvoice(ctx context.Context, id types.InvoiceID, bookingID types.BookingID, customerID types.CustomerID, tripID *types.TripID, subtotal, tax, discount, total float64, paymentStatus PaymentStatus) (Invoice, error)
	DeleteInvoice(ctx context.Context, id types.InvoiceID) error
	GetPaymentsForInvoice(ctx context.Context, invoiceID types.InvoiceID) ([]payment.Payment, error)
	GetBalance(ctx context.Context, invoiceID types.InvoiceID) (float64, error)
}
