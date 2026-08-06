package invoice

import (
	"time"

	"transport-app/internal/domain/types"
)

// Invoice represents a billing invoice generated from a completed trip.
type Invoice struct {
	ID            types.InvoiceID
	InvoiceNumber string
	BookingID     types.BookingID
	CustomerID    types.CustomerID
	TripID        *types.TripID
	Subtotal      float64
	Tax           float64
	Discount      float64
	Total         float64
	PaymentStatus PaymentStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PaymentStatus represents the payment status of an invoice.
type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusPaid          PaymentStatus = "paid"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
)
