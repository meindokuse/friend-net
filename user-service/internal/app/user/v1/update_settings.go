package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_settings"
)

func (i *Implementation) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		WhoCanMessage     string `json:"who_can_message"`
		WhoCanSeeLastSeen string `json:"who_can_see_last_seen"`
		WhoCanSeeProfile  string `json:"who_can_see_profile"`
		Language          string `json:"language"`
		Timezone          string `json:"timezone"`
		Version           int    `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := i.services.UpdateSettings.Execute(r.Context(), update_settings.Input{
		UserID:            userID,
		WhoCanMessage:     req.WhoCanMessage,
		WhoCanSeeLastSeen: req.WhoCanSeeLastSeen,
		WhoCanSeeProfile:  req.WhoCanSeeProfile,
		Language:          req.Language,
		Timezone:          req.Timezone,
		Version:           req.Version,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}
