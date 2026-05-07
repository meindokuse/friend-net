package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// Registry contains OAuth provider clients
type Registry struct {
	Google *GoogleClient
}

// NewRegistry creates a new OAuth gateway registry
func NewRegistry(cfg config.OAuthConfig) *Registry {
	return &Registry{
		Google: NewGoogleClient(cfg.Google),
	}
}

// OAuthProviderInfo contains OAuth user info
type OAuthProviderInfo struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

// OAuthTokens contains OAuth tokens
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	Expiry       int64
}

// GoogleClient implements Google OAuth
type GoogleClient struct {
	config *oauth2.Config
	client *http.Client
}

// NewGoogleClient creates a new Google OAuth client
func NewGoogleClient(cfg config.GoogleOAuthConfig) *GoogleClient {
	return &GoogleClient{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     google.Endpoint,
		},
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthURL returns authorization URL
func (c *GoogleClient) AuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeToken exchanges code for tokens
func (c *GoogleClient) ExchangeToken(ctx context.Context, code string) (*OAuthTokens, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	return &OAuthTokens{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry.Unix(),
	}, nil
}

// GetUserInfo gets user info from Google
func (c *GoogleClient) GetUserInfo(ctx context.Context, accessToken string) (*OAuthProviderInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var user struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Picture  string `json:"picture"`
		Verified bool   `json:"verified_email"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &OAuthProviderInfo{
		ProviderID: user.ID,
		Email:      user.Email,
		Name:       user.Name,
		AvatarURL:  user.Picture,
	}, nil
}

// Provider returns the provider type
func (c *GoogleClient) Provider() entity.OAuthProvider {
	return entity.OAuthGoogle
}
