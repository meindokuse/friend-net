package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/search_users"
)

func (i *Implementation) SearchUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	users, err := i.services.SearchUsers.Execute(r.Context(), search_users.Input{
		Query:  q.Get("q"),
		Limit:  parseIntQuery(q.Get("limit"), 20),
		Offset: parseIntQuery(q.Get("offset"), 0),
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	items := make([]*publicUserResponse, 0, len(users))
	for _, u := range users {
		items = append(items, toPublicUserResponse(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
