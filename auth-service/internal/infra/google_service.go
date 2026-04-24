package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/meindokuse/cloud-drive/auth-service/internal/usecase/oauth"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

type GoogleService struct {
	config *oauth2.Config
	client *http.Client
}

type GoogleServiceConfig struct {
	ClientID     string   `yaml:"clientID"     env:"GOOGLE_CLIENT_ID"     env-required:"true"`
	ClientSecret string   `yaml:"clientSecret" env:"GOOGLE_CLIENT_SECRET" env-required:"true"`
	RedirectURL  string   `yaml:"redirectURL"  env:"GOOGLE_REDIRECT_URL"`
	Scopes       []string `yaml:"scopes"`
}

func NewGoogleService(cfg GoogleServiceConfig) *GoogleService {
	return &GoogleService{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
			Endpoint:     googleoauth.Endpoint,
		},
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// AuthURL возвращает URL для перенаправления пользователя
func (s *GoogleService) AuthURL(state string, redirectURI string) string {
	cfg := *s.config
	if redirectURI != "" {
		cfg.RedirectURL = redirectURI
	}

	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeToken обменивает code на токены
func (s *GoogleService) ExchangeToken(ctx context.Context, code string) (*oauth.OAuthToken, error) {
	token, err := s.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &oauth.OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}, nil
}

// GetUserInfo получает данные пользователя из Google
func (s *GoogleService) GetUserInfo(ctx context.Context, accessToken string) (*oauth.OAuthUserInfo, error) {
	// Google API: получения инфо о пользователе
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.client.Do(req)
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

	return &oauth.OAuthUserInfo{
		ProviderID: user.ID,
		Email:      user.Email,
		Name:       user.Name,
		AvatarURL:  user.Picture,
	}, nil
}
