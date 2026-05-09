package v1

import (
	"net/http"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/update_profile"
)

func (i *Implementation) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		DisplayName string  `json:"display_name"`
		Bio         *string `json:"bio,omitempty"`
		AvatarURL   *string `json:"avatar_url,omitempty"`
		Version     int     `json:"version"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := i.services.UpdateProfile.Execute(r.Context(), update_profile.Input{
		UserID:      userID,
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
		Version:     req.Version,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}
