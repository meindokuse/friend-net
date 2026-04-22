package oauth

import (
	"context"
	"time"
)

// OAuthUserInfo — данные пользователя из OAuth провайдера
type OAuthUserInfo struct {
	ProviderID string // ID в Google/GitHub
	Email      string
	Name       string
	AvatarURL  string
}

// OAuthToken — OAuth access/refresh токены
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// OAuthProviderService — интерфейс для конкретного провайдера (Google/GitHub)
// Каждый провайдер — отдельная реализация
type OAuthProviderService interface {
	// URL для перенаправления пользователя (авторизация)
	AuthURL(state string, redirectURI string) string

	// Обмен authorization code на токены
	ExchangeToken(ctx context.Context, code string) (*OAuthToken, error)

	// Получить данные пользователя по access token
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
}
