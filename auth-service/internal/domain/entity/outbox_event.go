package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent represents an event in the outbox table
type OutboxEvent struct {
	ID            uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventType     string
	Payload       json.RawMessage
	CreatedAt     time.Time
	ProcessedAt   *time.Time
}

// NewOutboxEvent creates a new OutboxEvent
func NewOutboxEvent(aggregateType string, aggregateID uuid.UUID, eventType string, payload json.RawMessage) *OutboxEvent {
	return &OutboxEvent{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
	}
}
