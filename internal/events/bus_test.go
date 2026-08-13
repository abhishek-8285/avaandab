package events_test

import (
	"context"
	"testing"

	"transport-app/internal/events"
)

func TestInMemoryBus_PublishSubscribeUnsubscribe(t *testing.T) {
	bus := events.NewInMemoryBus()
	ctx := context.Background()

	calledCount := 0
	unsub := bus.Subscribe("TestEvent", func(ctx context.Context, e events.Event) error {
		calledCount++
		return nil
	})

	bus.Publish(ctx, events.Event{Type: "TestEvent", Payload: "hello"})

	if calledCount != 1 {
		t.Fatalf("expected handler to be called 1 time, got %d", calledCount)
	}

	// Unsubscribe
	unsub()

	bus.Publish(ctx, events.Event{Type: "TestEvent", Payload: "hello again"})

	if calledCount != 1 {
		t.Fatalf("expected handler to not be called after unsubscribe, got %d", calledCount)
	}
}
