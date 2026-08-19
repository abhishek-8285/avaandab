package realtime

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"transport-app/internal/events"
)

func TestHub_FanOutAndFrameFormat(t *testing.T) {
	hub := NewHub(15, nil)
	ctx := context.Background()

	ch1, unsub1 := hub.Subscribe(ctx, nil)
	defer unsub1()
	ch2, unsub2 := hub.Subscribe(ctx, nil)
	defer unsub2()

	payload := map[string]interface{}{
		"vehicle_id": "v-123",
		"lat":        12.34,
		"lng":        56.78,
		"speed":      45.0,
	}
	hub.Publish(ctx, events.Event{
		Type:    "telemetry.snapshot",
		Payload: payload,
	})

	select {
	case frame := <-ch1:
		frameStr := string(frame)
		if !strings.HasPrefix(frameStr, "event: telemetry\ndata: ") {
			t.Fatalf("unexpected frame format: %q", frameStr)
		}
		if !strings.Contains(frameStr, `"vehicle_id":"v-123"`) {
			t.Fatalf("missing vehicle_id in frame: %q", frameStr)
		}
		if !strings.HasSuffix(frameStr, "\n\n") {
			t.Fatalf("frame missing trailing double newline: %q", frameStr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("ch1 timed out waiting for event")
	}

	select {
	case frame := <-ch2:
		frameStr := string(frame)
		if !strings.Contains(frameStr, `"vehicle_id":"v-123"`) {
			t.Fatalf("ch2 missing vehicle_id in frame: %q", frameStr)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("ch2 timed out waiting for event")
	}
}

func TestHub_Filter(t *testing.T) {
	hub := NewHub(15, nil)
	ctx := context.Background()

	tripFilter := func(e events.Event) bool {
		if m, ok := e.Payload.(map[string]interface{}); ok {
			return m["trip_id"] == "trip-99"
		}
		return false
	}

	chFiltered, unsubFiltered := hub.Subscribe(ctx, tripFilter)
	defer unsubFiltered()

	// Publish non-matching event
	hub.Publish(ctx, events.Event{
		Type:    "telemetry.snapshot",
		Payload: map[string]interface{}{"trip_id": "trip-other", "vehicle_id": "v-1"},
	})

	select {
	case frame := <-chFiltered:
		t.Fatalf("expected filtered subscriber to NOT receive non-matching event, got: %s", string(frame))
	case <-time.After(50 * time.Millisecond):
		// Expected timeout
	}

	// Publish matching event
	hub.Publish(ctx, events.Event{
		Type:    "telemetry.snapshot",
		Payload: map[string]interface{}{"trip_id": "trip-99", "vehicle_id": "v-1"},
	})

	select {
	case frame := <-chFiltered:
		if !strings.Contains(string(frame), "trip-99") {
			t.Fatalf("expected frame with trip-99, got: %s", string(frame))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for matching event")
	}
}

func TestHub_SlowConsumerDrop(t *testing.T) {
	hub := NewHub(15, nil)
	ctx := context.Background()

	// Slow consumer that never reads
	chSlow, unsubSlow := hub.Subscribe(ctx, nil)
	defer unsubSlow()

	// Normal consumer that drains messages
	chFast, unsubFast := hub.Subscribe(ctx, nil)
	defer unsubFast()

	var fastReceived int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range chFast {
			fastReceived++
			if fastReceived == 100 {
				return
			}
		}
	}()

	// Publish 100 events (buffer capacity is 64; fast drainer keeps up, unread slow subscriber gets dropped)
	for i := 0; i < 100; i++ {
		hub.Publish(ctx, events.Event{
			Type:    "telemetry.snapshot",
			Payload: map[string]interface{}{"index": i, "vehicle_id": "v-1"},
		})
		time.Sleep(50 * time.Microsecond)
	}

	wg.Wait()
	if fastReceived != 100 {
		t.Fatalf("expected fast subscriber to receive 100 events, got %d", fastReceived)
	}

	// The slow subscriber channel should be closed due to drop
	drainCount := 0
	for range chSlow {
		drainCount++
	}
	// Channel closed: range terminates without hanging!
	if drainCount != defaultBufferCap {
		t.Logf("slow consumer drained %d buffered items before closure", drainCount)
	}

	// Verify that hub subs map only contains chFast, chSlow was removed
	hub.mu.RLock()
	subsCount := len(hub.subs)
	hub.mu.RUnlock()
	if subsCount != 1 {
		t.Fatalf("expected exactly 1 active subscriber in hub, got %d", subsCount)
	}

	// Ensure calling unsubSlow() afterwards is safe / does not panic
	unsubSlow()
}

func TestHub_HeartbeatKeepalive(t *testing.T) {
	hub := &Hub{
		subs:      make(map[chan []byte]filter),
		keepalive: 20 * time.Millisecond,
		logger:    nil,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, unsub := hub.Subscribe(ctx, nil)
	defer unsub()

	go hub.Run(ctx)

	select {
	case frame := <-ch:
		if string(frame) != ": keepalive\n\n" {
			t.Fatalf("expected keepalive comment frame, got: %q", string(frame))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for keepalive heartbeat")
	}
}
