// Package events provides a lightweight event bus for publishing and
// subscribing to domain events. A synchronous in-memory implementation
// is provided; the interface allows swapping in an async broker later.
package events

import (
	"context"
	"sync"
)

// Event is a domain event with a type and payload.
type Event struct {
	// Type uniquely identifies the event kind, e.g. "booking.confirmed".
	Type string
	// Payload carries the event-specific data.
	Payload any
}

// Handler processes an event. Return an error to signal failure.
type Handler func(ctx context.Context, e Event) error

// EventBus allows services to publish events and register handlers.
type EventBus interface {
	Publish(ctx context.Context, e Event)
	Subscribe(eventType string, h Handler) (unsubscribe func())
}

// InMemoryBus is a synchronous event bus safe for concurrent use.
type InMemoryBus struct {
	mu   sync.RWMutex
	subs map[string][]Handler
}

// NewInMemoryBus creates a new empty InMemoryBus.
func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{
		subs: make(map[string][]Handler),
	}
}

// Publish synchronously dispatches an event to all registered handlers
// for the event type. Handlers run in the caller's goroutine.
func (b *InMemoryBus) Publish(ctx context.Context, e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.subs[e.Type]))
	copy(handlers, b.subs[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		_ = h(ctx, e)
	}
}

// Subscribe registers a handler for a given event type and returns an
// unsubscribe function.
func (b *InMemoryBus) Subscribe(eventType string, h Handler) (unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs[eventType] = append(b.subs[eventType], h)

	// Return a closure that removes this handler
	idx := len(b.subs[eventType]) - 1
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if subs, ok := b.subs[eventType]; ok {
			if idx < len(subs) {
				b.subs[eventType] = append(subs[:idx], subs[idx+1:]...)
			}
		}
	}
}
