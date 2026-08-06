package payment

import (
	"context"
	"time"

	"transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
)

// RecordPaymentCommand contains parameters to record a payment.
type RecordPaymentCommand struct {
	TenantID    shared.TenantID
	InvoiceID   string
	PaymentDate time.Time
	Amount      float64
	Method      aggregate.PaymentMethod
	Reference   *string
	Remarks     *string
}

// PaymentFacade is the gateway into the payment module.
type PaymentFacade interface {
	RecordPayment(ctx context.Context, cmd RecordPaymentCommand) (aggregate.PaymentID, error)
}
