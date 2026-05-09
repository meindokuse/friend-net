package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/application/service/user/create_user"
)

func (i *Implementation) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          *uuid.UUID `json:"id,omitempty"`
		Username    string     `json:"username"`
		Email       *string    `json:"email,omitempty"`
		Phone       *string    `json:"phone,omitempty"`
		DisplayName string     `json:"display_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := i.services.CreateUser.Execute(r.Context(), create_user.Input{
		ID:          req.ID,
		Username:    req.Username,
		Email:       req.Email,
		Phone:       req.Phone,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}
