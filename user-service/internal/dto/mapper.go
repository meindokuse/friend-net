package dto

import (
	usecase "github.com/meindokuse/cloud-drive/user-service/internal/usecase/user"
)

// FromUserOutput конвертирует полный usecase-output в HTTP response.
func FromUserOutput(u *usecase.UserOutput) *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Phone:         u.Phone,
		DisplayName:   u.DisplayName,
		Bio:           u.Bio,
		AvatarURL:     u.AvatarURL,
		EmailVerified: u.EmailVerified,
		PhoneVerified: u.PhoneVerified,
		Privacy: PrivacyResponse{
			WhoCanMessage:     u.Privacy.WhoCanMessage,
			WhoCanSeeLastSeen: u.Privacy.WhoCanSeeLastSeen,
			WhoCanSeeProfile:  u.Privacy.WhoCanSeeProfile,
		},
		Language:   u.Language,
		Timezone:   u.Timezone,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
		LastSeenAt: u.LastSeenAt,
		Version:    u.Version,
	}
}

// FromPublicUserOutput конвертирует публичный usecase-output в HTTP response.
func FromPublicUserOutput(u *usecase.PublicUserOutput) *PublicUserResponse {
	return &PublicUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Bio:         u.Bio,
		AvatarURL:   u.AvatarURL,
		LastSeenAt:  u.LastSeenAt,
	}
}

// FromPublicUserOutputs конвертирует слайс публичных профилей.
func FromPublicUserOutputs(users []*usecase.PublicUserOutput) []*PublicUserResponse {
	out := make([]*PublicUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, FromPublicUserOutput(u))
	}
	return out
}
