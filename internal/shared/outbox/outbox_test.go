package outbox

import (
	"testing"

	bookingevents "transport-app/internal/domain/booking"
	tripevents "transport-app/internal/domain/trip"
	"transport-app/internal/events"
)

// TestGetEventTypeName_CanonicalCatalog proves the outbox writer persists the
// canonical event-type string (events.go catalog) for events subscribers
// listen on — never the Go type name (Spec 09 §5.1). This is the contract
// between outbox_events.event_type and bus subscribers.
func TestGetEventTypeName_CanonicalCatalog(t *testing.T) {
	cases := []struct {
		name string
		ev   any
		want string
	}{
		{"booking confirmed", bookingevents.BookingConfirmedEvent{}, events.BookingConfirmed},
		{"booking created", bookingevents.BookingCreatedEvent{}, events.BookingCreated},
		{"trip completed", tripevents.TripCompletedEvent{}, events.TripCompleted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := getEventTypeName(tc.ev)
			if got != tc.want {
				t.Fatalf("getEventTypeName(%T) = %q, want %q", tc.ev, got, tc.want)
			}
		})
	}
}

// TestGetEventTypeName_Fallback asserts unknown events still derive a stable
// type name via the Go type-name fallback.
func TestGetEventTypeName_Fallback(t *testing.T) {
	got := getEventTypeName(struct{}{})
	if got == "" {
		t.Fatal("expected non-empty fallback type name")
	}
}
