package payment_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/payment"
	"transport-app/internal/domain/types"
)

func TestPayment_Struct(t *testing.T) {
	now := time.Now()
	ref := "UPI-TXN-998877"

	p := payment.Payment{
		ID:          types.PaymentID("pmt-1"),
		InvoiceID:   types.InvoiceID("inv-1"),
		PaymentDate: now,
		Amount:      1500.0,
		Method:      payment.PaymentMethodUPI,
		Reference:   &ref,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if p.Amount != 1500.0 || p.Method != payment.PaymentMethodUPI || *p.Reference != ref {
		t.Fatalf("payment struct mismatch")
	}
}
