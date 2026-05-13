package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrEventNotFound    = errors.New("event: not found")
	ErrInvalidEventType = errors.New("event: event_type is required")
	ErrInvalidService   = errors.New("event: service is required")
)

type Event struct {
	id        uuid.UUID
	eventType string
	service   string
	userID    *uuid.UUID
	payload   string
	timestamp time.Time
	createdAt time.Time
}

func NewEvent(id uuid.UUID, eventType, service string, userID *uuid.UUID, payload string, ts time.Time) (*Event, error) {
	if eventType == "" {
		return nil, ErrInvalidEventType
	}
	if service == "" {
		return nil, ErrInvalidService
	}
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &Event{
		id:        id,
		eventType: eventType,
		service:   service,
		userID:    userID,
		payload:   payload,
		timestamp: ts.UTC(),
		createdAt: time.Now().UTC(),
	}, nil
}

func (e *Event) ID() uuid.UUID        { return e.id }
func (e *Event) EventType() string    { return e.eventType }
func (e *Event) Service() string      { return e.service }
func (e *Event) UserID() *uuid.UUID   { return e.userID }
func (e *Event) Payload() string      { return e.payload }
func (e *Event) Timestamp() time.Time { return e.timestamp }
func (e *Event) CreatedAt() time.Time { return e.createdAt }
