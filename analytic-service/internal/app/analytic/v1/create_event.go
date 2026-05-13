package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/ingest_event"
)

type createEventRequest struct {
	EventType string          `json:"event_type"`
	Service   string          `json:"service"`
	UserID    *uuid.UUID      `json:"user_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp *time.Time      `json:"timestamp,omitempty"`
}

// CreateEvent godoc
// POST /analytics/events
// Manually inserts an analytic event (bypasses Kafka, enqueues directly to batcher).
func (i *Implementation) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	ts := time.Now().UTC()
	if req.Timestamp != nil {
		ts = *req.Timestamp
	}

	in := ingest_event.Input{
		EventID:   uuid.New(),
		EventType: req.EventType,
		Service:   req.Service,
		UserID:    req.UserID,
		Payload:   req.Payload,
		Timestamp: ts,
	}

	if err := i.services.IngestEvent.Execute(r.Context(), in); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": in.EventID.String()})
}
