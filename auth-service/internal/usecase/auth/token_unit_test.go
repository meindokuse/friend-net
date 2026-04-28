package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	rderr "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/redis"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
)

type tokenTestDB struct{}

func (tokenTestDB) SaveWithOutbox(context.Context, *domain.Account, *outbox.OutboxEvent) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (tokenTestDB) FindAccount(context.Context, domain.Login) (*domain.Account, error) {
	return nil, nil
}
func (tokenTestDB) FindAccountByID(context.Context, uuid.UUID) (*domain.Account, error) {
	return nil, nil
}

type tokenTestRedis struct {
	blacklisted bool
}

func (r tokenTestRedis) CreateSession(context.Context, *domainsession.Session) error { return nil }
func (r tokenTestRedis) GetSession(context.Context, string) (*domainsession.Session, error) {
	return nil, rderr.ErrSessionNotFound
}
func (r tokenTestRedis) UpdateLastSeen(context.Context, string) error        { return nil }
func (r tokenTestRedis) RevokeSession(context.Context, string, string) error { return nil }
func (r tokenTestRedis) RevokeAllUserSessions(context.Context, string) error { return nil }
func (r tokenTestRedis) GetUserSessions(context.Context, string) ([]*domainsession.Session, error) {
	return nil, nil
}
func (r tokenTestRedis) CountUserSessions(context.Context, string) (int64, error) { return 0, nil }
func (r tokenTestRedis) SaveRefreshPair(context.Context, string, *domainsession.RefreshPair) error {
	return nil
}
func (r tokenTestRedis) GetRefreshPair(context.Context, string) (*domainsession.RefreshPair, error) {
	return nil, rderr.ErrRefreshNotFound
}
func (r tokenTestRedis) DeleteRefreshPair(context.Context, string) error { return nil }
func (r tokenTestRedis) BlacklistAccessToken(context.Context, string, time.Duration) error {
	return nil
}
func (r tokenTestRedis) IsBlacklisted(context.Context, string) (bool, error) {
	return r.blacklisted, nil
}

func TestValidateAccessToken_RejectsBlacklisted(t *testing.T) {
	jwtManager, err := jwt.NewManager(jwt.Config{
		SecretKey:     "01234567890123456789012345678901",
		RefreshSecret: "abcdefghijklmnopqrstuvwxyz123456",
		Issuer:        "auth-test",
		AccessTTL:     10 * time.Minute,
		RefreshTTL:    24 * time.Hour,
		GracePeriod:   30 * time.Second,
	})
	if err != nil {
		t.Fatalf("jwt.NewManager: %v", err)
	}

	access, err := jwtManager.GenerateAccessToken("s-1", "a-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	uc := NewAuth(tokenTestDB{}, tokenTestRedis{blacklisted: true}, nil, jwtManager)
	_, err = uc.ValidateAccessToken(context.Background(), access)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
