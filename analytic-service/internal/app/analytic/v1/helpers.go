package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/analytic-service/internal/pkg/apperr"
)

type reqLogCtx struct{ errMsg string }
type reqLogCtxKey struct{}

func storeErrMsg(ctx context.Context, msg string) {
	if rlc, ok := ctx.Value(reqLogCtxKey{}).(*reqLogCtx); ok {
		rlc.errMsg = msg
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	storeErrMsg(r.Context(), msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseIntQuery(s string, def int) int {
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n < 0 {
		return def
	}
	return n
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperr.ErrInvalidInput),
		errors.Is(err, entity.ErrInvalidEventType),
		errors.Is(err, entity.ErrInvalidService):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrEventNotFound):
		writeError(w, r, http.StatusNotFound, "event not found")
	default:
		slog.ErrorContext(r.Context(), "unhandled usecase error", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal server error")
	}
}
