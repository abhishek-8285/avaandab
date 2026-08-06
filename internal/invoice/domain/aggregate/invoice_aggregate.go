package aggregate

import (
	"time"

	"transport-app/internal/shared"
)

type InvoiceID string
type PaymentStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusPaid          PaymentStatus = "paid"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
)

// InvoiceAggregate is the aggregate root representing a billing invoice.
type InvoiceAggregate struct {
	ID            InvoiceID
	TenantID      shared.TenantID
	InvoiceNumber string
	BookingID     string
	CustomerID    string
	TripID        *string
	Subtotal      float64
	Tax           float64
	Discount      float64
	Total         float64
	PaymentStatus PaymentStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Version       int64
	events        []any
}

// NewInvoiceAggregate constructs a new InvoiceAggregate and registers a created event.
func NewInvoiceAggregate(
	id InvoiceID,
	tenantID shared.TenantID,
	invoiceNumber string,
	bookingID string,
	customerID string,
	tripID *string,
	subtotal float64,
	tax float64,
	discount float64,
	total float64,
	status PaymentStatus,
	now time.Time,
) *InvoiceAggregate {
	inv := &InvoiceAggregate{
		ID:            id,
		TenantID:      tenantID,
		InvoiceNumber: invoiceNumber,
		BookingID:     bookingID,
		CustomerID:    customerID,
		TripID:        tripID,
		Subtotal:      subtotal,
		Tax:           tax,
		Discount:      discount,
		Total:         total,
		PaymentStatus: status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	inv.events = append(inv.events, InvoiceGeneratedEvent{
		ID:            id,
		TenantID:      tenantID,
		InvoiceNumber: invoiceNumber,
		Total:         total,
		CreatedAt:     now,
	})

	return inv
}

// Events returns recorded domain events.
func (a *InvoiceAggregate) Events() []any {
	return a.events
}

// ClearEvents clears the recorded events list.
func (a *InvoiceAggregate) ClearEvents() {
	a.events = nil
}

// UpdatePaymentStatus updates the status of the invoice and records a payment received event.
func (a *InvoiceAggregate) UpdatePaymentStatus(status PaymentStatus, now time.Time) error {
	a.PaymentStatus = status
	a.UpdatedAt = now

	a.events = append(a.events, InvoicePaymentUpdatedEvent{
		ID:            a.ID,
		TenantID:      a.TenantID,
		PaymentStatus: status,
		UpdatedAt:     now,
	})

	return nil
}

// InvoiceGeneratedEvent emitted when an invoice is generated.
type InvoiceGeneratedEvent struct {
	ID            InvoiceID
	TenantID      shared.TenantID
	InvoiceNumber string
	Total         float64
	CreatedAt     time.Time
}

// InvoicePaymentUpdatedEvent emitted when an invoice payment status changes.
type InvoicePaymentUpdatedEvent struct {
	ID            InvoiceID
	TenantID      shared.TenantID
	PaymentStatus PaymentStatus
	UpdatedAt     time.Time
}
