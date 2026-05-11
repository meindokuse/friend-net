package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/change_phone"
)

func (i *Implementation) ChangePhone(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Phone   string `json:"phone"`
		Version int    `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := i.services.ChangePhone.Execute(r.Context(), change_phone.Input{
		UserID:  userID,
		Phone:   req.Phone,
		Version: req.Version,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}
