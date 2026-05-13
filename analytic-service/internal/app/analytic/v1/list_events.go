package v1

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/list_events"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
)

type eventResponse struct {
	ID        uuid.UUID  `json:"id"`
	EventType string     `json:"event_type"`
	Service   string     `json:"service"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Payload   string     `json:"payload,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
	CreatedAt time.Time  `json:"created_at"`
}

type listEventsResponse struct {
	Events []*eventResponse `json:"events"`
	Total  int64            `json:"total"`
}

func toEventResponse(e *entity.Event) *eventResponse {
	return &eventResponse{
		ID:        e.ID(),
		EventType: e.EventType(),
		Service:   e.Service(),
		UserID:    e.UserID(),
		Payload:   e.Payload(),
		Timestamp: e.Timestamp(),
		CreatedAt: e.CreatedAt(),
	}
}

// ListEvents godoc
// GET /analytics/events?event_type=&service=&user_id=&from=&to=&limit=50&offset=0
func (i *Implementation) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := list_events.Filter{
		Limit:  parseIntQuery(q.Get("limit"), 50),
		Offset: parseIntQuery(q.Get("offset"), 0),
	}

	if s := q.Get("event_type"); s != "" {
		f.EventType = &s
	}
	if s := q.Get("service"); s != "" {
		f.Service = &s
	}
	if s := q.Get("user_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid user_id UUID")
			return
		}
		f.UserID = &id
	}
	if s := q.Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid 'from' timestamp: use RFC3339")
			return
		}
		f.From = &t
	}
	if s := q.Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid 'to' timestamp: use RFC3339")
			return
		}
		f.To = &t
	}

	out, err := i.services.ListEvents.Execute(r.Context(), f)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	resp := &listEventsResponse{Total: out.Total, Events: make([]*eventResponse, 0, len(out.Events))}
	for _, e := range out.Events {
		resp.Events = append(resp.Events, toEventResponse(e))
	}
	writeJSON(w, http.StatusOK, resp)
}
