package mongo

import (
	"time"

	"github.com/google/uuid"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
)

// userDocument — MongoDB документ для хранения User.
type userDocument struct {
	ID       uuid.UUID `bson:"_id"`
	Username string    `bson:"username"`
	Email    *string   `bson:"email,omitempty"`
	Phone    *string   `bson:"phone,omitempty"`

	Profile      profileDocument      `bson:"profile"`
	Settings     settingsDocument     `bson:"settings"`
	Verification verificationDocument `bson:"verification"`

	IsActive   bool       `bson:"is_active"`
	CreatedAt  time.Time  `bson:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at"`
	LastSeenAt *time.Time `bson:"last_seen_at,omitempty"`
	DeletedAt  *time.Time `bson:"deleted_at,omitempty"`
	Version    int        `bson:"version"`
}

type profileDocument struct {
	DisplayName string  `bson:"display_name"`
	Bio         *string `bson:"bio,omitempty"`
	AvatarURL   *string `bson:"avatar_url,omitempty"`
}

type settingsDocument struct {
	Privacy  privacySettingsDocument `bson:"privacy"`
	Language string                  `bson:"language"`
	Timezone string                  `bson:"timezone"`
}

type privacySettingsDocument struct {
	WhoCanMessage     string `bson:"who_can_message"`
	WhoCanSeeLastSeen string `bson:"who_can_see_last_seen"`
	WhoCanSeeProfile  string `bson:"who_can_see_profile"`
}

type verificationDocument struct {
	EmailVerified bool `bson:"email_verified"`
	PhoneVerified bool `bson:"phone_verified"`
}

// toDocument конвертирует domain User в MongoDB документ.
func toDocument(u *domainuser.User) *userDocument {
	doc := &userDocument{
		ID:       u.ID(),
		Username: u.Username().String(),
		Profile: profileDocument{
			DisplayName: u.Profile().DisplayName,
			Bio:         u.Profile().Bio,
			AvatarURL:   u.Profile().AvatarURL,
		},
		Settings: settingsDocument{
			Privacy: privacySettingsDocument{
				WhoCanMessage:     string(u.Settings().Privacy.WhoCanMessage),
				WhoCanSeeLastSeen: string(u.Settings().Privacy.WhoCanSeeLastSeen),
				WhoCanSeeProfile:  string(u.Settings().Privacy.WhoCanSeeProfile),
			},
			Language: u.Settings().Language,
			Timezone: u.Settings().Timezone,
		},
		Verification: verificationDocument{
			EmailVerified: u.Verification().EmailVerified,
			PhoneVerified: u.Verification().PhoneVerified,
		},
		IsActive:   u.IsActive(),
		CreatedAt:  u.CreatedAt(),
		UpdatedAt:  u.UpdatedAt(),
		LastSeenAt: u.LastSeenAt(),
		DeletedAt:  u.DeletedAt(),
		Version:    u.Version(),
	}

	if u.Email() != nil {
		email := u.Email().String()
		doc.Email = &email
	}

	if u.Phone() != nil {
		phone := u.Phone().String()
		doc.Phone = &phone
	}

	return doc
}

// fromDocument конвертирует MongoDB документ в domain User.
func fromDocument(doc *userDocument) (*domainuser.User, error) {
	username := vo.MustNewUsername(doc.Username)

	var email *vo.Email
	if doc.Email != nil {
		e := vo.MustNewEmail(*doc.Email)
		email = &e
	}

	var phone *vo.Phone
	if doc.Phone != nil {
		p := vo.MustNewPhone(*doc.Phone)
		phone = &p
	}

	profile := domainuser.Profile{
		DisplayName: doc.Profile.DisplayName,
		Bio:         doc.Profile.Bio,
		AvatarURL:   doc.Profile.AvatarURL,
	}

	settings := domainuser.Settings{
		Privacy: domainuser.PrivacySettings{
			WhoCanMessage:     domainuser.PrivacyLevel(doc.Settings.Privacy.WhoCanMessage),
			WhoCanSeeLastSeen: domainuser.PrivacyLevel(doc.Settings.Privacy.WhoCanSeeLastSeen),
			WhoCanSeeProfile:  domainuser.PrivacyLevel(doc.Settings.Privacy.WhoCanSeeProfile),
		},
		Language: doc.Settings.Language,
		Timezone: doc.Settings.Timezone,
	}

	verification := domainuser.Verification{
		EmailVerified: doc.Verification.EmailVerified,
		PhoneVerified: doc.Verification.PhoneVerified,
	}

	user := domainuser.Reconstruct(
		doc.ID,
		username,
		email,
		phone,
		profile,
		settings,
		verification,
		doc.IsActive,
		doc.CreatedAt,
		doc.UpdatedAt,
		doc.LastSeenAt,
		doc.DeletedAt,
		doc.Version,
	)

	return user, nil
}
