package payment

import (
	"time"

	"transport-app/internal/domain/types"
)

// Payment represents a payment made against an invoice.
type Payment struct {
	ID          types.PaymentID
	InvoiceID   types.InvoiceID
	PaymentDate time.Time
	Amount      float64
	Method      PaymentMethod
	Reference   *string
	Remarks     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PaymentMethod represents the method of payment.
type PaymentMethod string

const (
	PaymentMethodCash         PaymentMethod = "cash"
	PaymentMethodUPI          PaymentMethod = "upi"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodCheque       PaymentMethod = "cheque"
	PaymentMethodRazorpay     PaymentMethod = "razorpay"
)
