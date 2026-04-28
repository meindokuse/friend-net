package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainaccount "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

// DB - контракт репозитория для работы с аккаунтами.
type DB interface {
	SaveWithOutbox(ctx context.Context, accountData *domainaccount.Account, outbox *outbox.OutboxEvent) (uuid.UUID, error)
	FindAccount(ctx context.Context, loginData domainaccount.Login) (*domainaccount.Account, error)
	FindAccountByID(ctx context.Context, accountID uuid.UUID) (*domainaccount.Account, error)
}

// Redis - контракт репозитория для работы с сессиями.
type Redis interface {
	CreateSession(ctx context.Context, s *domainsession.Session) error
	GetSession(ctx context.Context, sessionID string) (*domainsession.Session, error)
	UpdateLastSeen(ctx context.Context, sessionID string) error
	RevokeSession(ctx context.Context, sessionID, userID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	GetUserSessions(ctx context.Context, userID string) ([]*domainsession.Session, error)
	CountUserSessions(ctx context.Context, userID string) (int64, error)
	SaveRefreshPair(ctx context.Context, sessionID string, pair *domainsession.RefreshPair) error
	GetRefreshPair(ctx context.Context, sessionID string) (*domainsession.RefreshPair, error)
	DeleteRefreshPair(ctx context.Context, sessionID string) error
	BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// Auth - usecase для аутентификации и регистрации.
type Auth struct {
	db         DB
	redis      Redis
	hasher     *pass.Hasher
	jwtManager *jwt.Manager
}

// NewAuth создаёт новый usecase.
func NewAuth(db DB, redis Redis, hasher *pass.Hasher, jwtManager *jwt.Manager) *Auth {
	return &Auth{
		db:         db,
		redis:      redis,
		hasher:     hasher,
		jwtManager: jwtManager,
	}
}

// RefreshTTL возвращает TTL refresh токена.
func (a *Auth) RefreshTTL() time.Duration {
	if a == nil || a.jwtManager == nil {
		return 0
	}
	return a.jwtManager.RefreshTTL()
}
