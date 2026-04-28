package oauth

import (
	"time"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

// OAuthCallbackInput - входные данные для OAuth callback.
type OAuthCallbackInput struct {
	Provider    domain.OAuthProvider
	Code        string
	State       string
	RedirectURL string
	RequestData domain.RequestData
}

// OAuthLinkInput - входные данные для привязки OAuth аккаунта.
type OAuthLinkInput struct {
	Provider         domain.OAuthProvider
	Code             string
	State            string
	CurrentAccountID string // string потому что OAuthAccount.AccountID это string
}

// OAuthOutput - результат OAuth входа/привязки.
type OAuthOutput struct {
	AccountID        string // string для совместимости с domain.AuthResult
	Email            string
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	IsNewUser        bool
	IsLinked         bool
}
