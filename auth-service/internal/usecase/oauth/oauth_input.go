package oauth

import (
	"time"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

type OAuthCallbackInput struct {
	Provider    domain.OAuthProvider // "google" или "github"
	Code        string               // Authorization code от провайдера
	State       string               // CSRF token (проверяем)
	RedirectURL string               // Куда вернуться после успеха
	RequestData domain.RequestData
}

// OAuthLinkInput — привязка аккаунта (для уже залогиненного пользователя)
type OAuthLinkInput struct {
	Provider         domain.OAuthProvider
	Code             string
	State            string
	CurrentAccountID string // ID текущего залогиненного аккаунта (из сессии)
}

// OAuthOutput — результат OAuth входа/привязки
type OAuthOutput struct {
	AccountID        string
	Email            string
	AccessToken      string // Наш JWT access token
	RefreshToken     string // Ключ сессии, обычно уходит в cookie
	TokenType        string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	IsNewUser        bool //true = пользователь создан (первый вход)
	IsLinked         bool // true = аккаунт привязан (не новое создание)
}
