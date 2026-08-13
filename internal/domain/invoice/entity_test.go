package invoice_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/invoice"
	"transport-app/internal/domain/types"
)

func TestInvoice_LifecycleAndOutstandingBalance(t *testing.T) {
	inv := &invoice.Invoice{
		ID:            types.InvoiceID("inv-1"),
		InvoiceNumber: "INV-2026-001",
		BookingID:     types.BookingID("bk-1"),
		CustomerID:    types.CustomerID("cust-1"),
		Subtotal:      1000.0,
		Tax:           180.0,
		Discount:      50.0,
		Total:         1130.0,
		PaidAmount:    0.0,
		Status:        invoice.InvoiceDraft,
	}

	dueDate := time.Now().Add(14 * 24 * time.Hour)
	inv.MarkIssued(dueDate)

	if inv.Status != invoice.InvoiceOutstanding {
		t.Errorf("expected status %s, got %s", invoice.InvoiceOutstanding, inv.Status)
	}

	if inv.OutstandingBalance() != 1130.0 {
		t.Errorf("expected outstanding balance 1130.0, got %f", inv.OutstandingBalance())
	}

	// Partial payment
	inv.ApplyPayment(500.0)
	if inv.Status != invoice.InvoiceOutstanding {
		t.Errorf("expected status %s after partial payment, got %s", invoice.InvoiceOutstanding, inv.Status)
	}
	if inv.PaymentStatus != invoice.PaymentStatusPartiallyPaid {
		t.Errorf("expected payment status %s, got %s", invoice.PaymentStatusPartiallyPaid, inv.PaymentStatus)
	}
	if inv.OutstandingBalance() != 630.0 {
		t.Errorf("expected outstanding balance 630.0, got %f", inv.OutstandingBalance())
	}

	// Full payment completion
	inv.ApplyPayment(630.0)
	if inv.Status != invoice.InvoicePaid {
		t.Errorf("expected status %s after full payment, got %s", invoice.InvoicePaid, inv.Status)
	}
	if inv.PaymentStatus != invoice.PaymentStatusPaid {
		t.Errorf("expected payment status %s, got %s", invoice.PaymentStatusPaid, inv.PaymentStatus)
	}
	if inv.OutstandingBalance() != 0.0 {
		t.Errorf("expected outstanding balance 0.0, got %f", inv.OutstandingBalance())
	}
}

func TestInvoice_AdjustAmount(t *testing.T) {
	inv := &invoice.Invoice{
		ID:       types.InvoiceID("inv-2"),
		Subtotal: 1000.0,
		Tax:      100.0,
		Discount: 50.0,
		Total:    1050.0,
	}

	// Dispatcher adjusts price/discount manually
	inv.AdjustAmount(1200.0, 120.0, 100.0)

	if inv.Total != 1220.0 {
		t.Errorf("expected adjusted total 1220.0, got %f", inv.Total)
	}
	if inv.OutstandingBalance() != 1220.0 {
		t.Errorf("expected outstanding balance 1220.0, got %f", inv.OutstandingBalance())
	}
}

func TestInvoice_Cancel(t *testing.T) {
	inv := &invoice.Invoice{
		ID:     types.InvoiceID("inv-cancel"),
		Status: invoice.InvoiceDraft,
	}

	inv.Cancel()
	if inv.Status != invoice.InvoiceCancelled {
		t.Fatalf("expected status cancelled, got %s", inv.Status)
	}

	invPaid := &invoice.Invoice{
		ID:     types.InvoiceID("inv-paid"),
		Status: invoice.InvoicePaid,
	}
	invPaid.Cancel() // Cannot cancel paid invoice
	if invPaid.Status != invoice.InvoicePaid {
		t.Fatalf("expected paid invoice status to remain paid")
	}
}
