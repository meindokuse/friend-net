package v1

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
)

func (i *Implementation) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := parseIntQuery(q.Get("limit"), 20)
	params := entity.ListParams{Limit: limit}

	if raw := q.Get("cursor"); raw != "" {
		b, err := base64.URLEncoding.DecodeString(raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid cursor")
			return
		}
		var payload struct {
			Username string    `json:"username"`
			ID       uuid.UUID `json:"id"`
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid cursor")
			return
		}
		params.Cursor = &entity.UsernameCursor{Username: payload.Username, ID: payload.ID}
	}

	_, paged, err := i.services.ListUsers.Execute(r.Context(), params)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	items := make([]*publicUserResponse, 0, len(paged.Items))
	for _, u := range paged.Items {
		items = append(items, toPublicUserResponse(u))
	}

	resp := map[string]any{"items": items, "has_more": paged.HasMore}
	if paged.HasMore {
		b, _ := json.Marshal(map[string]any{
			"username": paged.NextCursor.Username,
			"id":       paged.NextCursor.ID,
		})
		resp["next_cursor"] = base64.URLEncoding.EncodeToString(b)
	}
	writeJSON(w, http.StatusOK, resp)
}
