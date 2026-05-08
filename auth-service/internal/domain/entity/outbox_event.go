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

// AccountCreatedPayload represents the payload for account.created event
type AccountCreatedPayload struct {
	AccountID   string `json:"account_id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
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

// NewAccountCreatedEvent creates an outbox event for account creation
func NewAccountCreatedEvent(accountID uuid.UUID, email, displayName string, createdAt time.Time) (*OutboxEvent, error) {
	username := createUsernameFromEmail(email)

	payload := AccountCreatedPayload{
		AccountID:   accountID.String(),
		Email:       email,
		Username:    username,
		DisplayName: displayName,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return NewOutboxEvent("account", accountID, "account.created", payloadBytes), nil
}

func createUsernameFromEmail(email string) string {
	for i, c := range email {
		if c == '@' {
			return email[:i]
		}
	}
	return email
}
