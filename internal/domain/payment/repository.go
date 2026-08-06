package payment

import (
	"context"

	"transport-app/internal/domain/types"
)

// PaymentRepository defines the interface for payment persistence.
type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment Payment) (Payment, error)
	GetPaymentByID(ctx context.Context, id types.PaymentID) (PaymentWithInvoice, error)
	DeletePayment(ctx context.Context, id types.PaymentID) error
	GetPaymentsByInvoice(ctx context.Context, invoiceID types.InvoiceID) ([]Payment, error)
	SumPaymentsByInvoice(ctx context.Context, invoiceID types.InvoiceID) (float64, error)
	SearchPayments(ctx context.Context, method string, limit, offset int) ([]PaymentWithInvoice, error)
	CountPayments(ctx context.Context, method string) (int64, error)
	GetTotalRevenue(ctx context.Context) (float64, error)
	GetMonthlyRevenue(ctx context.Context) ([]MonthlyRevenue, error)
	GetPaymentsByCustomer(ctx context.Context, customerID types.CustomerID, limit, offset int) ([]PaymentWithInvoice, error)
	CountPaymentsByCustomer(ctx context.Context, customerID types.CustomerID) (int64, error)
}

// PaymentWithInvoice includes the associated invoice details.
type PaymentWithInvoice struct {
	Payment
	InvoiceNumber        string
	InvoiceTotal         float64
	InvoicePaymentStatus string
	CustomerName         *string
}

// MonthlyRevenue is a month's revenue total.
type MonthlyRevenue struct {
	Month string
	Total float64
}
