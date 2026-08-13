package ewaybill_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/ewaybill"
	"transport-app/internal/domain/types"
)

func TestEWayBill_ValidateAndActive(t *testing.T) {
	now := time.Now()
	validUntil := now.Add(48 * time.Hour)
	transporter := "27AAAAA0000A1Z5"
	vehicleNo := "MH-12-AB-1234"

	ewb := ewaybill.EWayBill{
		ID:             "ewb-1",
		TripID:         types.TripID("trp-1"),
		EWBNumber:      "123456789012",
		GenerationDate: now,
		ValidUntil:     validUntil,
		TransporterID:  &transporter,
		VehicleNumber:  &vehicleNo,
		Status:         "active",
		CreatedAt:      now,
	}

	if !ewb.IsActive(now) {
		t.Fatalf("expected EWB to be active")
	}

	if err := ewb.Validate(now); err != nil {
		t.Fatalf("expected valid EWB, got %v", err)
	}

	// Invalid EWB number length
	ewbShort := ewb
	ewbShort.EWBNumber = "123"
	if err := ewbShort.Validate(now); err != ewaybill.ErrInvalidEWBNumber {
		t.Fatalf("expected ErrInvalidEWBNumber, got %v", err)
	}

	// Expired EWB
	ewbExpired := ewb
	ewbExpired.ValidUntil = now.Add(-1 * time.Hour)
	if err := ewbExpired.Validate(now); err != ewaybill.ErrEWBExpired {
		t.Fatalf("expected ErrEWBExpired, got %v", err)
	}
	if ewbExpired.IsActive(now) {
		t.Fatalf("expected expired EWB to not be active")
	}

	// Cancelled EWB
	ewbCancelled := ewb
	ewbCancelled.Status = "cancelled"
	if ewbCancelled.IsActive(now) {
		t.Fatalf("expected cancelled EWB to not be active")
	}
}
