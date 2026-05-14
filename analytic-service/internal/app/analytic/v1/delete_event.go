package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// DeleteEvent godoc
// DELETE /analytics/events/{id}
// Fires an async ClickHouse mutation. Returns 202 Accepted immediately.
func (i *Implementation) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid event id UUID")
		return
	}

	if err := i.services.DeleteEvent.Execute(r.Context(), id); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
