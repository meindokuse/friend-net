package oauth

import (
	"context"
	"time"

	"github.com/google/uuid"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
)

// OAuthUseCase - интерфейс для OAuth usecase (для handlers).
type OAuthUseCase interface {
	Login(ctx context.Context, input OAuthCallbackInput) (*OAuthOutput, error)
	LinkAccount(ctx context.Context, input OAuthLinkInput) error
	UnlinkAccount(ctx context.Context, accountID string, provider domain.OAuthProvider) error
	GetLinkedAccounts(ctx context.Context, accountID string) ([]*domain.OAuthAccount, error)
	RegisterProvider(provider domain.OAuthProvider, svc OAuthProviderService)
}

// AccountRepository - контракт для работы с аккаунтами.
type AccountRepository interface {
	SaveWithOutbox(ctx context.Context, account *domain.Account, outbox *outbox.OutboxEvent) (uuid.UUID, error)
	FindAccount(ctx context.Context, loginData domain.Login) (*domain.Account, error)
	FindAccountByID(ctx context.Context, accountID uuid.UUID) (*domain.Account, error)
}

// OAuthRepository - контракт для работы с OAuth аккаунтами.
type OAuthRepository interface {
	GetByProviderID(ctx context.Context, provider domain.OAuthProvider, providerID string) (*domain.OAuthAccount, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*domain.OAuthAccount, error)
	Create(ctx context.Context, account *domain.OAuthAccount) error
	UpdateTokens(ctx context.Context, id string, accessToken, refreshToken string, expiry time.Time) error
	Delete(ctx context.Context, id string) error
}

// SessionRepository - контракт для работы с сессиями.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *domainsession.Session) error
	SaveRefreshPair(ctx context.Context, sessionID string, pair *domainsession.RefreshPair) error
}

// OAuth - usecase для OAuth аутентификации.
type OAuth struct {
	accountRepo    AccountRepository
	oauthRepo      OAuthRepository
	sessionRepo    SessionRepository
	jwtManager     *jwt.Manager
	oauthProviders map[domain.OAuthProvider]OAuthProviderService
	sessionTTL     time.Duration
}

// NewOAuth создаёт новый OAuth usecase.
func NewOAuth(
	accountRepo AccountRepository,
	oauthRepo OAuthRepository,
	sessionRepo SessionRepository,
	jwtManager *jwt.Manager,
	sessionTTL time.Duration,
) *OAuth {
	return &OAuth{
		accountRepo:    accountRepo,
		oauthRepo:      oauthRepo,
		sessionRepo:    sessionRepo,
		jwtManager:     jwtManager,
		sessionTTL:     sessionTTL,
		oauthProviders: make(map[domain.OAuthProvider]OAuthProviderService),
	}
}

// RegisterProvider регистрирует OAuth провайдера.
func (uc *OAuth) RegisterProvider(provider domain.OAuthProvider, svc OAuthProviderService) {
	uc.oauthProviders[provider] = svc
}

// Проверка что OAuth реализует OAuthUseCase
var _ OAuthUseCase = (*OAuth)(nil)
