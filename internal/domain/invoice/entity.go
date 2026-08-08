package invoice

import (
	"time"

	"transport-app/internal/domain/types"
)

// InvoiceStatus represents the lifecycle status of an invoice.
type InvoiceStatus string

const (
	InvoiceDraft       InvoiceStatus = "draft"
	InvoiceIssued      InvoiceStatus = "issued"
	InvoiceOutstanding InvoiceStatus = "outstanding"
	InvoicePaid        InvoiceStatus = "paid"
	InvoiceCancelled   InvoiceStatus = "cancelled"
)

// PaymentStatus represents the payment status of an invoice.
type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusPaid          PaymentStatus = "paid"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
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
	PaidAmount    float64
	Status        InvoiceStatus
	PaymentStatus PaymentStatus
	DueDate       *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OutstandingBalance calculates remaining amount due.
func (i Invoice) OutstandingBalance() float64 {
	balance := i.Total - i.PaidAmount
	if balance < 0 {
		return 0
	}
	return balance
}

// MarkIssued sets invoice status to issued/outstanding upon trip completion.
func (i *Invoice) MarkIssued(dueDate time.Time) {
	i.Status = InvoiceOutstanding
	i.PaymentStatus = PaymentStatusPending
	i.DueDate = &dueDate
	i.UpdatedAt = time.Now()
}

// ApplyPayment updates paid amount and recalculates payment status.
func (i *Invoice) ApplyPayment(amount float64) {
	i.PaidAmount += amount
	balance := i.OutstandingBalance()

	if balance <= 0 {
		i.Status = InvoicePaid
		i.PaymentStatus = PaymentStatusPaid
	} else {
		i.Status = InvoiceOutstanding
		i.PaymentStatus = PaymentStatusPartiallyPaid
	}
	i.UpdatedAt = time.Now()
}

// RecalculateTotal calculates Total = Subtotal + Tax - Discount.
func (i *Invoice) RecalculateTotal() {
	total := i.Subtotal + i.Tax - i.Discount
	if total < 0 {
		total = 0
	}
	i.Total = total
	i.UpdatedAt = time.Now()
}

// AdjustAmount allows dispatcher/admin to override or adjust subtotal, tax, and discount.
func (i *Invoice) AdjustAmount(subtotal, tax, discount float64) {
	i.Subtotal = subtotal
	i.Tax = tax
	i.Discount = discount
	i.RecalculateTotal()

	// Update payment status based on new Total
	balance := i.OutstandingBalance()
	if i.PaidAmount > 0 {
		if balance <= 0 {
			i.Status = InvoicePaid
			i.PaymentStatus = PaymentStatusPaid
		} else {
			i.Status = InvoiceOutstanding
			i.PaymentStatus = PaymentStatusPartiallyPaid
		}
	}
}
