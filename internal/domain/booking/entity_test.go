package booking_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/booking"
	"transport-app/internal/domain/types"
)

func TestBooking_CanConfirm(t *testing.T) {
	bk := booking.Booking{
		ID:     types.BookingID("bk-1"),
		Status: booking.BookingPending,
	}

	if err := bk.CanConfirm(); err != nil {
		t.Fatalf("expected pending booking to be confirmable, got %v", err)
	}

	bkCancelled := booking.Booking{
		ID:     types.BookingID("bk-2"),
		Status: booking.BookingCancelled,
	}
	if err := bkCancelled.CanConfirm(); err == nil {
		t.Fatalf("expected error confirming cancelled booking")
	}

	bkDraft := booking.Booking{
		ID:     types.BookingID("bk-3"),
		Status: booking.BookingDraft,
	}
	if err := bkDraft.CanConfirm(); err == nil {
		t.Fatalf("expected error confirming draft booking")
	}
}

func TestBooking_CanCancel(t *testing.T) {
	bk := booking.Booking{
		ID:     types.BookingID("bk-1"),
		Status: booking.BookingConfirmed,
	}

	if err := bk.CanCancel(); err != nil {
		t.Fatalf("expected confirmed booking to be cancellable, got %v", err)
	}

	bkCompleted := booking.Booking{
		ID:     types.BookingID("bk-2"),
		Status: booking.BookingCompleted,
	}
	if err := bkCompleted.CanCancel(); err == nil {
		t.Fatalf("expected error cancelling completed booking")
	}
}

func TestBooking_CanDelete(t *testing.T) {
	bkPending := booking.Booking{
		ID:     types.BookingID("bk-1"),
		Status: booking.BookingPending,
	}
	if err := bkPending.CanDelete(); err != nil {
		t.Fatalf("expected pending booking to be deletable, got %v", err)
	}

	bkConfirmed := booking.Booking{
		ID:     types.BookingID("bk-2"),
		Status: booking.BookingConfirmed,
	}
	if err := bkConfirmed.CanDelete(); err != nil {
		t.Fatalf("expected confirmed booking to be deletable, got %v", err)
	}

	bkCompleted := booking.Booking{
		ID:     types.BookingID("bk-3"),
		Status: booking.BookingCompleted,
	}
	if err := bkCompleted.CanDelete(); err == nil {
		t.Fatalf("expected error deleting completed booking")
	}
}

func TestBooking_StructFields(t *testing.T) {
	now := time.Now()
	cargo := 500.0
	notes := "Fragile cargo"

	bk := booking.Booking{
		ID:            types.BookingID("bk-struct"),
		BookingNumber: "BK-9999",
		CustomerID:    types.CustomerID("cust-1"),
		PickupDate:    now,
		CargoWeight:   &cargo,
		Notes:         &notes,
		Price:         1200.0,
		Status:        booking.BookingPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if bk.BookingNumber != "BK-9999" || *bk.CargoWeight != 500.0 || *bk.Notes != "Fragile cargo" {
		t.Fatalf("struct field mismatch")
	}
}
