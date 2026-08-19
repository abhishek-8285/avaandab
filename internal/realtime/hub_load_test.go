package realtime

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"transport-app/internal/events"
)

func TestHub_100ConcurrentSubscribers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	hub := NewHub(15, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	const subscriberCount = 100
	const eventCount = 10

	var wg sync.WaitGroup
	received := make([]int, subscriberCount)

	// Launch 100 concurrent subscribers
	for i := 0; i < subscriberCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ch, unsub := hub.Subscribe(ctx, nil)
			defer unsub()

			for j := 0; j < eventCount; j++ {
				select {
				case <-ch:
					received[idx]++
				case <-time.After(5 * time.Second):
					return
				}
			}
		}(i)
	}

	// Give all goroutines time to subscribe
	time.Sleep(50 * time.Millisecond)

	// Publish events
	for i := 0; i < eventCount; i++ {
		hub.Publish(ctx, events.Event{
			Type:    "telemetry.snapshot",
			Payload: map[string]interface{}{"index": i, "vehicle_id": "v-scale"},
		})
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()

	// Assert all 100 subscribers received all 10 events
	for i := 0; i < subscriberCount; i++ {
		assert.Equal(t, eventCount, received[i], "subscriber %d should receive all events", i)
	}
}
