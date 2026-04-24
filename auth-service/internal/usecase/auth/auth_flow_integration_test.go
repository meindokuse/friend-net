package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	pgerr "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/postgresql"
	rderr "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/redis"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

type fakeAccountDB struct {
	byEmail map[string]domain.Account
	byID    map[string]domain.Account
}

func newFakeAccountDB() *fakeAccountDB {
	return &fakeAccountDB{
		byEmail: make(map[string]domain.Account),
		byID:    make(map[string]domain.Account),
	}
}

func (f *fakeAccountDB) Save(_ context.Context, accountData domain.Account) (string, error) {
	if _, exists := f.byEmail[accountData.Email]; exists {
		return "", pgerr.ErrUserAlreadyExists
	}
	if accountData.ID == "" {
		accountData.ID = uuid.NewString()
	}
	f.byEmail[accountData.Email] = accountData
	f.byID[accountData.ID] = accountData
	return accountData.ID, nil
}

func (f *fakeAccountDB) FindAccount(_ context.Context, loginData domain.Login) (*domain.Account, error) {
	acc, ok := f.byEmail[loginData.Email]
	if !ok {
		return nil, pgerr.ErrNoRows
	}
	out := acc
	return &out, nil
}

func (f *fakeAccountDB) FindAccountByID(_ context.Context, accountID string) (*domain.Account, error) {
	acc, ok := f.byID[accountID]
	if !ok {
		return nil, pgerr.ErrNoRows
	}
	out := acc
	return &out, nil
}

type fakeRedisStore struct {
	sessions     map[string]*domainsession.Session
	accountIndex map[string]map[string]struct{}
	refreshPairs map[string]*domainsession.RefreshPair
	blacklist    map[string]time.Time
}

func newFakeRedisStore() *fakeRedisStore {
	return &fakeRedisStore{
		sessions:     make(map[string]*domainsession.Session),
		accountIndex: make(map[string]map[string]struct{}),
		refreshPairs: make(map[string]*domainsession.RefreshPair),
		blacklist:    make(map[string]time.Time),
	}
}

func (f *fakeRedisStore) CreateSession(_ context.Context, s *domainsession.Session) error {
	c := *s
	f.sessions[s.ID] = &c
	if _, ok := f.accountIndex[s.AccountID]; !ok {
		f.accountIndex[s.AccountID] = make(map[string]struct{})
	}
	f.accountIndex[s.AccountID][s.ID] = struct{}{}
	return nil
}

func (f *fakeRedisStore) GetSession(_ context.Context, sessionID string) (*domainsession.Session, error) {
	s, ok := f.sessions[sessionID]
	if !ok {
		return nil, rderr.ErrSessionNotFound
	}
	c := *s
	return &c, nil
}

func (f *fakeRedisStore) UpdateLastSeen(_ context.Context, sessionID string) error {
	s, ok := f.sessions[sessionID]
	if !ok {
		return rderr.ErrSessionNotFound
	}
	s.LastSeenAt = time.Now().UTC()
	return nil
}

func (f *fakeRedisStore) RevokeSession(_ context.Context, sessionID, accountID string) error {
	s, ok := f.sessions[sessionID]
	if !ok {
		return rderr.ErrSessionNotFound
	}
	s.Status = domainsession.SessionStatusRevoked
	delete(f.refreshPairs, sessionID)
	if idx, ok := f.accountIndex[accountID]; ok {
		delete(idx, sessionID)
	}
	return nil
}

func (f *fakeRedisStore) RevokeAllUserSessions(_ context.Context, accountID string) error {
	ids := f.accountIndex[accountID]
	for sid := range ids {
		if s, ok := f.sessions[sid]; ok {
			s.Status = domainsession.SessionStatusRevoked
		}
		delete(f.refreshPairs, sid)
	}
	delete(f.accountIndex, accountID)
	return nil
}

func (f *fakeRedisStore) GetUserSessions(_ context.Context, accountID string) ([]*domainsession.Session, error) {
	ids := f.accountIndex[accountID]
	out := make([]*domainsession.Session, 0, len(ids))
	for sid := range ids {
		if s, ok := f.sessions[sid]; ok && s.Status == domainsession.SessionStatusActive {
			c := *s
			out = append(out, &c)
		}
	}
	return out, nil
}

func (f *fakeRedisStore) CountUserSessions(_ context.Context, accountID string) (int64, error) {
	return int64(len(f.accountIndex[accountID])), nil
}

func (f *fakeRedisStore) SaveRefreshPair(_ context.Context, sessionID string, pair *domainsession.RefreshPair) error {
	c := *pair
	f.refreshPairs[sessionID] = &c
	return nil
}

func (f *fakeRedisStore) GetRefreshPair(_ context.Context, sessionID string) (*domainsession.RefreshPair, error) {
	p, ok := f.refreshPairs[sessionID]
	if !ok {
		return nil, rderr.ErrRefreshNotFound
	}
	c := *p
	return &c, nil
}

func (f *fakeRedisStore) DeleteRefreshPair(_ context.Context, sessionID string) error {
	delete(f.refreshPairs, sessionID)
	return nil
}

func (f *fakeRedisStore) BlacklistAccessToken(_ context.Context, jti string, ttl time.Duration) error {
	f.blacklist[jti] = time.Now().UTC().Add(ttl)
	return nil
}

func (f *fakeRedisStore) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	exp, ok := f.blacklist[jti]
	if !ok {
		return false, nil
	}
	if time.Now().UTC().After(exp) {
		delete(f.blacklist, jti)
		return false, nil
	}
	return true, nil
}

func buildTestAuth(t *testing.T) *Auth {
	t.Helper()
	db := newFakeAccountDB()
	redis := newFakeRedisStore()
	hasher := pass.New(pass.Config{Cost: 4})
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
	return NewAuth(db, redis, hasher, jwtManager)
}

func TestAuthFlowIntegration_RegisterLoginRefreshLogout(t *testing.T) {
	uc := buildTestAuth(t)
	ctx := context.Background()

	accountID, err := uc.Register(ctx, domain.Register{
		Email:    "account@example.com",
		Password: "StrongPass123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if accountID == "" {
		t.Fatal("expected non-empty accountID")
	}

	loginRes, err := uc.LoginUser(ctx, domain.LoginInput{
		Email:       "account@example.com",
		Password:    "StrongPass123",
		Fingerprint: "fp-1",
		IP:          "127.0.0.1",
		UserAgent:   "go-test",
	})
	if err != nil {
		t.Fatalf("LoginUser failed: %v", err)
	}
	if loginRes.AccountID != accountID {
		t.Fatalf("expected accountID %s, got %s", accountID, loginRes.AccountID)
	}

	if _, err := uc.ValidateAccessToken(ctx, loginRes.AccessToken); err != nil {
		t.Fatalf("ValidateAccessToken after login failed: %v", err)
	}

	refreshRes, err := uc.Refresh(ctx, domain.RefreshInput{
		RefreshToken: loginRes.RefreshToken,
		Fingerprint:  "fp-1",
	})
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if refreshRes.AccountID != accountID {
		t.Fatalf("expected refreshed accountID %s, got %s", accountID, refreshRes.AccountID)
	}

	sessions, err := uc.GetUserSessions(ctx, accountID, "")
	if err != nil {
		t.Fatalf("GetUserSessions failed: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least one active session")
	}

	if err := uc.Logout(ctx, domain.LogoutInput{
		AccessToken:  refreshRes.AccessToken,
		RefreshToken: refreshRes.RefreshToken,
	}); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	if _, err := uc.ValidateAccessToken(ctx, refreshRes.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after logout, got %v", err)
	}
}

func TestAuthFlowIntegration_RefreshFingerprintMismatch(t *testing.T) {
	uc := buildTestAuth(t)
	ctx := context.Background()

	if _, err := uc.Register(ctx, domain.Register{
		Email:    "fp@example.com",
		Password: "StrongPass123",
	}); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	loginRes, err := uc.LoginUser(ctx, domain.LoginInput{
		Email:       "fp@example.com",
		Password:    "StrongPass123",
		Fingerprint: "fp-good",
		IP:          "127.0.0.1",
		UserAgent:   "go-test",
	})
	if err != nil {
		t.Fatalf("LoginUser failed: %v", err)
	}

	_, err = uc.Refresh(ctx, domain.RefreshInput{
		RefreshToken: loginRes.RefreshToken,
		Fingerprint:  "fp-bad",
	})
	if !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("expected ErrFingerprintMismatch, got %v", err)
	}
}
