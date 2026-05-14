package dao

import (
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
)

// EventDAO maps to the analytic_events ClickHouse table.
// UserID uses the zero UUID as a sentinel for "no user" since ClickHouse UUID is non-nullable.
type EventDAO struct {
	EventID   uuid.UUID `ch:"event_id"`
	EventType string    `ch:"event_type"`
	Service   string    `ch:"service"`
	UserID    uuid.UUID `ch:"user_id"`
	Payload   string    `ch:"payload"`
	Timestamp time.Time `ch:"timestamp"`
	CreatedAt time.Time `ch:"created_at"`
}

func FromEntity(e *entity.Event) *EventDAO {
	userID := uuid.Nil
	if e.UserID() != nil {
		userID = *e.UserID()
	}
	return &EventDAO{
		EventID:   e.ID(),
		EventType: e.EventType(),
		Service:   e.Service(),
		UserID:    userID,
		Payload:   e.Payload(),
		Timestamp: e.Timestamp(),
		CreatedAt: e.CreatedAt(),
	}
}

func ToEntity(d *EventDAO) *entity.Event {
	var userID *uuid.UUID
	if d.UserID != uuid.Nil {
		id := d.UserID
		userID = &id
	}
	e, _ := entity.NewEvent(d.EventID, d.EventType, d.Service, userID, d.Payload, d.Timestamp)
	return e
}
