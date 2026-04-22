package usecase

import (
	"context"
	"time"

	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	domainuser "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

type DB interface {
	Save(ctx context.Context, userData domainuser.User) (string, error)
	FindUser(ctx context.Context, loginData domainuser.Login) (*domainuser.User, error)
	FindUserByID(ctx context.Context, userID string) (*domainuser.User, error)
}

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

type Auth struct {
	db         DB
	redis      Redis
	hasher     *pass.Hasher
	jwtManager *jwt.Manager
}

func NewAuth(db DB, redis Redis, hasher *pass.Hasher, jwtManager *jwt.Manager) *Auth {
	return &Auth{
		db:         db,
		redis:      redis,
		hasher:     hasher,
		jwtManager: jwtManager,
	}
}

func (a *Auth) RefreshTTL() time.Duration {
	if a == nil || a.jwtManager == nil {
		return 0
	}

	return a.jwtManager.RefreshTTL()
}
