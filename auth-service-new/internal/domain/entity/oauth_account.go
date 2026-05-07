package entity

import (
	"time"

	"github.com/google/uuid"
)

// OAuthProvider represents OAuth provider type
type OAuthProvider string

const (
	OAuthGoogle OAuthProvider = "google"
	OAuthGitHub OAuthProvider = "github"
	OAuthVK     OAuthProvider = "vk"
)

// OAuthAccount represents a link between user account and OAuth provider
type OAuthAccount struct {
	ID           string        // UUID
	AccountID    string        // Reference to accounts.id
	Provider     OAuthProvider // google/github/vk
	ProviderID   string        // User ID in Google/GitHub
	Email        string        // Email from OAuth (may differ from main email)
	AccessToken  string        // OAuth access token
	RefreshToken string        // OAuth refresh token
	Expiry       time.Time     // When access token expires
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewOAuthAccount creates a new OAuthAccount entity
func NewOAuthAccount(accountID string, provider OAuthProvider, providerID, email string) *OAuthAccount {
	return &OAuthAccount{
		ID:         uuid.NewString(),
		AccountID:  accountID,
		Provider:   provider,
		ProviderID: providerID,
		Email:      email,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
}

// UpdateTokens updates OAuth tokens
func (o *OAuthAccount) UpdateTokens(accessToken, refreshToken string, expiry time.Time) {
	o.AccessToken = accessToken
	o.RefreshToken = refreshToken
	o.Expiry = expiry
	o.UpdatedAt = time.Now().UTC()
}
