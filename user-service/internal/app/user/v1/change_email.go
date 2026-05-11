package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_email"
)

func (i *Implementation) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Email   string `json:"email"`
		Version int    `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := i.services.ChangeEmail.Execute(r.Context(), change_email.Input{
		UserID:  userID,
		Email:   req.Email,
		Version: req.Version,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}
