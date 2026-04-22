package domain

import (
	"time"

	"github.com/google/uuid"
)

// OAuthProvider — тип провайдера
type OAuthProvider string

const (
	OAuthGoogle OAuthProvider = "google"
	OAuthGitHub OAuthProvider = "github"
	OAuthVK     OAuthProvider = "vk"
)

// OAuthAccount — связь пользователя с OAuth-провайдером
type OAuthAccount struct {
	ID           string        // UUID
	UserID       string        // Ссылка на users.id
	Provider     OAuthProvider // google/github/vk
	ProviderID   string        // ID пользователя в Google/GitHub (например, "1082348923489")
	Email        string        // Email из OAuth (может не совпадать с основным email)
	AccessToken  string        // OAuth access token (если нужно для API провайдера)
	RefreshToken string        // OAuth refresh token (для обновления access token)
	Expiry       time.Time     // Когда истекает access token
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewOAuthAccount(userID string, provider OAuthProvider, providerID, email string) *OAuthAccount {
	return &OAuthAccount{
		ID:         uuid.NewString(),
		UserID:     userID,
		Provider:   provider,
		ProviderID: providerID,
		Email:      email,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}
