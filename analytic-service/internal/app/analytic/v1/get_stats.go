package v1

import (
	"net/http"
	"time"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/application/service/analytic/get_stats"
)

// GetStats godoc
// GET /analytics/stats?from=<RFC3339>&to=<RFC3339>
func (i *Implementation) GetStats(w http.ResponseWriter, r *http.Request) {
	in := get_stats.Input{}

	if s := r.URL.Query().Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid 'from' timestamp: use RFC3339")
			return
		}
		in.From = &t
	}
	if s := r.URL.Query().Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid 'to' timestamp: use RFC3339")
			return
		}
		in.To = &t
	}

	out, err := i.services.GetStats.Execute(r.Context(), in)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}
