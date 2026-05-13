package analytic

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AnalyticEvent struct {
	EventID   uuid.UUID       `json:"event_id"`
	EventType string          `json:"event_type"`
	Service   string          `json:"service"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}
