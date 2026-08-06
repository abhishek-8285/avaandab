package aggregate

import (
	"time"

	"transport-app/internal/shared"
)

type PaymentID string
type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodUPI          PaymentMethod = "upi"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCheque       PaymentMethod = "cheque"
)

// PaymentAggregate is the aggregate root representing a payment transaction.
type PaymentAggregate struct {
	ID          PaymentID
	TenantID    shared.TenantID
	InvoiceID   string
	PaymentDate time.Time
	Amount      float64
	Method      PaymentMethod
	Reference   *string
	Remarks     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	events      []any
}

// NewPaymentAggregate constructs a new PaymentAggregate and records a received event.
func NewPaymentAggregate(
	id PaymentID,
	tenantID shared.TenantID,
	invoiceID string,
	paymentDate time.Time,
	amount float64,
	method PaymentMethod,
	reference *string,
	remarks *string,
	now time.Time,
) *PaymentAggregate {
	p := &PaymentAggregate{
		ID:          id,
		TenantID:    tenantID,
		InvoiceID:   invoiceID,
		PaymentDate: paymentDate,
		Amount:      amount,
		Method:      method,
		Reference:   reference,
		Remarks:     remarks,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	p.events = append(p.events, PaymentReceivedEvent{
		ID:        id,
		TenantID:  tenantID,
		InvoiceID: invoiceID,
		Amount:    amount,
		CreatedAt: now,
	})

	return p
}

// Events returns recorded domain events.
func (a *PaymentAggregate) Events() []any {
	return a.events
}

// ClearEvents clears the recorded events list.
func (a *PaymentAggregate) ClearEvents() {
	a.events = nil
}

// PaymentReceivedEvent emitted when a payment is processed.
type PaymentReceivedEvent struct {
	ID        PaymentID
	TenantID  shared.TenantID
	InvoiceID string
	Amount    float64
	CreatedAt time.Time
}
