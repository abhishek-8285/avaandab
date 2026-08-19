package realtime

import (
	"context"
	"strings"
	"testing"
	"time"

	"transport-app/internal/events"
)

func TestAttachToBus(t *testing.T) {
	bus := events.NewInMemoryBus()
	hub := NewHub(15, nil)
	AttachToBus(bus, hub)

	ctx := context.Background()
	ch, unsub := hub.Subscribe(ctx, nil)
	defer unsub()

	// 1. Test telemetry.snapshot
	bus.Publish(ctx, events.Event{
		Type: "telemetry.snapshot",
		Payload: map[string]interface{}{
			"vehicle_id": "v-bus-1",
			"lat":        19.0760,
			"lng":        72.8777,
		},
	})

	select {
	case frame := <-ch:
		if !strings.Contains(string(frame), "v-bus-1") {
			t.Fatalf("expected frame to contain v-bus-1, got %s", string(frame))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for telemetry.snapshot event")
	}

	// 2. Test maintenance.due
	bus.Publish(ctx, events.Event{
		Type: "maintenance.due",
		Payload: map[string]interface{}{
			"vehicle_id": "v-bus-2",
			"reason":     "oil_change",
		},
	})

	select {
	case frame := <-ch:
		if !strings.Contains(string(frame), "v-bus-2") {
			t.Fatalf("expected frame to contain v-bus-2, got %s", string(frame))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for maintenance.due event")
	}
}
