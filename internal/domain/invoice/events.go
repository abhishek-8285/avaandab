package invoice

import (
	"time"

	"transport-app/internal/domain/types"
)

// InvoiceGenerated is emitted when an invoice is generated from a completed trip.
type InvoiceGenerated struct {
	InvoiceID     types.InvoiceID
	InvoiceNumber string
	TripID        types.TripID
	Total         float64
	OccurredAt    time.Time
}

// InvoicePaid is emitted when an invoice payment status changes to paid.
type InvoicePaid struct {
	InvoiceID   types.InvoiceID
	Amount      float64
	OccurredAt  time.Time
}

// InvoicePartiallyPaid is emitted when a partial payment is made.
type InvoicePartiallyPaid struct {
	InvoiceID   types.InvoiceID
	Amount      float64
	OccurredAt  time.Time
}
