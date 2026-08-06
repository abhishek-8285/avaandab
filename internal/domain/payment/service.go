package payment

import (
	"context"

	"transport-app/internal/domain/types"
)

// PaymentService defines the interface for payment business operations.
type PaymentService interface {
	RecordPayment(ctx context.Context, invoiceID types.InvoiceID, amount float64, method PaymentMethod, reference, remarks string, paymentDate string) (Payment, error)
	GetPayment(ctx context.Context, id types.PaymentID) (PaymentWithInvoice, error)
	ListPayments(ctx context.Context, method string, limit, offset int) ([]PaymentWithInvoice, int64, error)
	DeletePayment(ctx context.Context, id types.PaymentID) error
	GetTotalRevenue(ctx context.Context) (float64, error)
	GetMonthlyRevenue(ctx context.Context) ([]MonthlyRevenue, error)
	GetCustomerPayments(ctx context.Context, customerID types.CustomerID, limit, offset int) ([]PaymentWithInvoice, int64, error)
}
