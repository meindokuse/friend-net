package v1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (i *Implementation) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := i.services.GetUser.ByID(r.Context(), userID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (i *Implementation) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid user id")
		return
	}
	u, err := i.services.GetUser.ByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicUserResponse(u))
}

func (i *Implementation) GetUserByUsername(w http.ResponseWriter, r *http.Request) {
	u, err := i.services.GetUser.ByUsername(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicUserResponse(u))
}

func (i *Implementation) GetUsersByIDs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	users, err := i.services.GetUser.ByIDs(r.Context(), req.IDs)
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
