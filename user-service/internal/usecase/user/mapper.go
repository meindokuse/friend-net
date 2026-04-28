package user

import (
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// toUserOutput — полный профиль.
func toUserOutput(u *domainuser.User) *UserOutput {
	var email, phone *string
	if u.Email() != nil {
		s := u.Email().String()
		email = &s
	}
	if u.Phone() != nil {
		s := u.Phone().String()
		phone = &s
	}

	return &UserOutput{
		ID:          u.ID(),
		Username:    u.Username().String(),
		Email:       email,
		Phone:       phone,
		DisplayName: u.Profile().DisplayName,
		Bio:         u.Profile().Bio,
		AvatarURL:   u.Profile().AvatarURL,

		EmailVerified: u.Verification().EmailVerified,
		PhoneVerified: u.Verification().PhoneVerified,

		Privacy: PrivacyOutput{
			WhoCanMessage:     string(u.Settings().Privacy.WhoCanMessage),
			WhoCanSeeLastSeen: string(u.Settings().Privacy.WhoCanSeeLastSeen),
			WhoCanSeeProfile:  string(u.Settings().Privacy.WhoCanSeeProfile),
		},
		Language: u.Settings().Language,
		Timezone: u.Settings().Timezone,

		IsActive:   u.IsActive(),
		CreatedAt:  u.CreatedAt(),
		UpdatedAt:  u.UpdatedAt(),
		LastSeenAt: u.LastSeenAt(),
		Version:    u.Version(),
	}
}

// toPublicUserOutput — публичный профиль (для чужого пользователя).
func toPublicUserOutput(u *domainuser.User) *PublicUserOutput {
	return &PublicUserOutput{
		ID:          u.ID(),
		Username:    u.Username().String(),
		DisplayName: u.Profile().DisplayName,
		Bio:         u.Profile().Bio,
		AvatarURL:   u.Profile().AvatarURL,
		LastSeenAt:  u.LastSeenAt(),
	}
}

func toPublicUserOutputs(users []*domainuser.User) []*PublicUserOutput {
	out := make([]*PublicUserOutput, 0, len(users))
	for _, u := range users {
		out = append(out, toPublicUserOutput(u))
	}
	return out
}
