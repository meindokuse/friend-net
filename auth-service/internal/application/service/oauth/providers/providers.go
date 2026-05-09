package providers

import "context"

type OAuthProviderGateway interface {
	ExchangeToken(ctx context.Context, code string) (*OAuthToken, error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
}

// OAuthToken represents OAuth tokens
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       int64
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}
