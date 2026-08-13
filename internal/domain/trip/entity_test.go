package trip_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/trip"
	"transport-app/internal/domain/types"
)

func TestTrip_CanSchedule(t *testing.T) {
	trp := trip.Trip{
		ID:            types.TripID("trp-1"),
		DepartureTime: time.Now(),
		Status:        trip.TripDraft,
	}

	if err := trp.CanSchedule(); err != nil {
		t.Fatalf("expected draft trip to be schedulable, got: %v", err)
	}

	trp.Status = trip.TripScheduled
	if err := trp.CanSchedule(); err == nil {
		t.Fatalf("expected error scheduling non-draft trip")
	}
}

func TestTrip_CanStart(t *testing.T) {
	trp := trip.Trip{
		ID:     types.TripID("trp-2"),
		Status: trip.TripAssigned,
	}

	if err := trp.CanStart(); err != nil {
		t.Fatalf("expected assigned trip to be startable, got: %v", err)
	}

	trp.Status = trip.TripDraft
	if err := trp.CanStart(); err == nil {
		t.Fatalf("expected error starting draft trip")
	}
}

func TestTrip_CanDeliver(t *testing.T) {
	trp := trip.Trip{
		ID:     types.TripID("trp-3"),
		Status: trip.TripStarted,
	}

	if err := trp.CanDeliver(); err != nil {
		t.Fatalf("expected started trip to be deliverable, got: %v", err)
	}

	trp.Status = trip.TripInTransit
	if err := trp.CanDeliver(); err != nil {
		t.Fatalf("expected in_transit trip to be deliverable, got: %v", err)
	}

	trp.Status = trip.TripDraft
	if err := trp.CanDeliver(); err == nil {
		t.Fatalf("expected error delivering draft trip")
	}
}

func TestTrip_CanComplete(t *testing.T) {
	trp := trip.Trip{
		ID:     types.TripID("trp-4"),
		Status: trip.TripDelivered,
	}

	if err := trp.CanComplete(); err != nil {
		t.Fatalf("expected delivered trip to be completeable, got: %v", err)
	}

	trp.Status = trip.TripDraft
	if err := trp.CanComplete(); err == nil {
		t.Fatalf("expected error completing draft trip")
	}
}

func TestTrip_CanCancel(t *testing.T) {
	trp := trip.Trip{
		ID:     types.TripID("trp-5"),
		Status: trip.TripAssigned,
	}

	if err := trp.CanCancel(); err != nil {
		t.Fatalf("expected assigned trip to be cancellable, got: %v", err)
	}

	trp.Status = trip.TripCompleted
	if err := trp.CanCancel(); err == nil {
		t.Fatalf("expected error cancelling completed trip")
	}
}
