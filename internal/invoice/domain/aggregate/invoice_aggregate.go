package aggregate

import (
	"errors"
	"time"

	"transport-app/internal/shared"
)

type InvoiceID string
type PaymentStatus string
type InvoiceStatus string

const (
	PaymentStatusPending       PaymentStatus = "pending"
	PaymentStatusPaid          PaymentStatus = "paid"
	PaymentStatusPartiallyPaid PaymentStatus = "partially_paid"
)

const (
	InvoiceStatusDraft       InvoiceStatus = "draft"
	InvoiceStatusIssued      InvoiceStatus = "issued"
	InvoiceStatusOutstanding InvoiceStatus = "outstanding"
	InvoiceStatusPaid        InvoiceStatus = "paid"
	InvoiceStatusCancelled   InvoiceStatus = "cancelled"
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
	PaidAmount    float64
	Status        InvoiceStatus
	DueDate       *time.Time
	FinancialYear string
	CreditBalance float64
	Remarks       string
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
		Status:        InvoiceStatusOutstanding,
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

// RehydrateInvoiceAggregate reconstructs an invoice from persistence without emitting events.
func RehydrateInvoiceAggregate(
	id InvoiceID, tenantID shared.TenantID, invoiceNumber string,
	bookingID, customerID string, tripID *string,
	subtotal, tax, discount, total float64,
	paymentStatus PaymentStatus, invoiceStatus InvoiceStatus,
	paidAmount, creditBalance float64,
	dueDate *time.Time, financialYear, remarks string,
	createdAt, updatedAt time.Time, version int64,
) *InvoiceAggregate {
	return &InvoiceAggregate{
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
		PaymentStatus: paymentStatus,
		PaidAmount:    paidAmount,
		Status:        invoiceStatus,
		DueDate:       dueDate,
		FinancialYear: financialYear,
		CreditBalance: creditBalance,
		Remarks:       remarks,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Version:       version,
	}
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

// OutstandingBalance returns Total - PaidAmount.
func (a *InvoiceAggregate) OutstandingBalance() float64 {
	return a.Total - a.PaidAmount
}

// ApplyPayment records a payment against this invoice, updates PaidAmount and PaymentStatus.
// If payment exceeds outstanding, the excess is recorded in CreditBalance.
// Returns error if invoice is cancelled.
func (a *InvoiceAggregate) ApplyPayment(amount float64, now time.Time) error {
	if a.Status == InvoiceStatusCancelled {
		return errors.New("cannot apply payment to cancelled invoice")
	}

	a.PaidAmount += amount
	outstanding := a.Total - a.PaidAmount

	if outstanding < 0 {
		a.CreditBalance = -outstanding
		a.PaidAmount = a.Total
		a.PaymentStatus = PaymentStatusPaid
		a.Status = InvoiceStatusPaid
	} else if outstanding == 0 {
		a.PaymentStatus = PaymentStatusPaid
		a.Status = InvoiceStatusPaid
	} else {
		if a.PaidAmount > 0 {
			a.PaymentStatus = PaymentStatusPartiallyPaid
		} else {
			a.PaymentStatus = PaymentStatusPending
		}
	}

	a.UpdatedAt = now

	a.events = append(a.events, InvoicePaymentAppliedEvent{
		ID:            a.ID,
		TenantID:      a.TenantID,
		Amount:        amount,
		PaidAmount:    a.PaidAmount,
		PaymentStatus: a.PaymentStatus,
		OccurredAt:    now,
	})

	return nil
}

// MarkIssued transitions invoice to issued status with a due date.
// Returns error if invoice is not in draft status.
func (a *InvoiceAggregate) MarkIssued(dueDate time.Time, now time.Time) error {
	if a.Status != InvoiceStatusDraft {
		return errors.New("only draft invoices can be issued")
	}

	a.Status = InvoiceStatusIssued
	a.DueDate = &dueDate
	a.UpdatedAt = now

	a.events = append(a.events, InvoiceIssuedEvent{
		ID:         a.ID,
		TenantID:   a.TenantID,
		DueDate:    dueDate,
		OccurredAt: now,
	})

	return nil
}

// Void transitions invoice to cancelled status.
// Returns error if invoice is paid or already cancelled.
func (a *InvoiceAggregate) Void(now time.Time) error {
	if a.Status == InvoiceStatusPaid {
		return errors.New("paid invoices cannot be voided")
	}
	if a.Status == InvoiceStatusCancelled {
		return errors.New("invoice already cancelled")
	}

	a.Status = InvoiceStatusCancelled
	a.UpdatedAt = now

	a.events = append(a.events, InvoiceVoidedEvent{
		ID:         a.ID,
		TenantID:   a.TenantID,
		OccurredAt: now,
	})

	return nil
}

// ValidateInvoiceNumber returns error if invoice number is empty.
func ValidateInvoiceNumber(number string) error {
	if number == "" {
		return errors.New("invoice number is required")
	}
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

// InvoicePaymentAppliedEvent emitted when a payment is applied to an invoice.
type InvoicePaymentAppliedEvent struct {
	ID            InvoiceID
	TenantID      shared.TenantID
	Amount        float64
	PaidAmount    float64
	PaymentStatus PaymentStatus
	OccurredAt    time.Time
}

// InvoiceIssuedEvent emitted when an invoice transitions to issued status.
type InvoiceIssuedEvent struct {
	ID         InvoiceID
	TenantID   shared.TenantID
	DueDate    time.Time
	OccurredAt time.Time
}

// InvoiceVoidedEvent emitted when an invoice is voided.
type InvoiceVoidedEvent struct {
	ID         InvoiceID
	TenantID   shared.TenantID
	OccurredAt time.Time
}
