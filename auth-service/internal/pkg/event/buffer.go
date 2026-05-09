package event

import (
	"context"
	"sync"
)

// Flusher publishes a batch of events.
type Flusher interface {
	Flush(ctx context.Context, events Events) error
}

// Buffer accumulates events and flushes them via a Flusher.
type Buffer struct {
	mu      sync.Mutex
	events  Events
	flusher Flusher
}

// NewBuffer creates a new Buffer backed by the given Flusher.
func NewBuffer(flusher Flusher) *Buffer {
	return &Buffer{flusher: flusher}
}

// Add appends an event to the buffer.
func (b *Buffer) Add(e *Event) {
	b.mu.Lock()
	b.events = append(b.events, e)
	b.mu.Unlock()
}

// Flush drains the buffer and forwards all events to the Flusher.
func (b *Buffer) Flush(ctx context.Context) error {
	b.mu.Lock()
	events := b.events
	b.events = nil
	b.mu.Unlock()

	if len(events) == 0 {
		return nil
	}
	return b.flusher.Flush(ctx, events)
}
