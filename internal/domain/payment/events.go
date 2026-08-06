package payment

import (
	"time"

	"transport-app/internal/domain/types"
)

// PaymentRecorded is emitted when a payment is recorded.
type PaymentRecorded struct {
	PaymentID  types.PaymentID
	InvoiceID  types.InvoiceID
	Amount     float64
	Method     PaymentMethod
	OccurredAt time.Time
}

// PaymentDeleted is emitted when a payment is deleted.
type PaymentDeleted struct {
	PaymentID  types.PaymentID
	InvoiceID  types.InvoiceID
	Amount     float64
	OccurredAt time.Time
}
