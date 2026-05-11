package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/delete_user"
)

func (i *Implementation) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Version int `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := i.services.DeleteUser.Execute(r.Context(), delete_user.Input{
		UserID:  userID,
		Version: req.Version,
	}); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
