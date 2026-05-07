package login

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

const maxSessionsPerUser = 3

// AccountProvider interface for account operations
type AccountProvider interface {
	FindByEmail(ctx context.Context, email string) (*entity.Account, error)
}

// SessionManager interface for session operations
type SessionManager interface {
	Create(ctx context.Context, session *entity.Session) error
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.Session, error)
	CountByAccountID(ctx context.Context, accountID string) (int64, error)
	Revoke(ctx context.Context, sessionID, accountID string) error
	SaveRefreshPair(ctx context.Context, sessionID string, pair *entity.RefreshPair) error
}

// Service handles login use case
type Service struct {
	accounts AccountProvider
	sessions SessionManager
	hasher   *pass.Hasher
	jwt      *jwt.Manager
}

// NewService creates a new login service
func NewService(
	accounts AccountProvider,
	sessions SessionManager,
	hasher *pass.Hasher,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		accounts: accounts,
		sessions: sessions,
		hasher:   hasher,
		jwt:      jwtManager,
	}
}

// DTO for login input
type LoginDTO struct {
	Email       string
	Password    string
	Fingerprint string
	IP          string
	UserAgent   string
}

// Result contains login result
type Result struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresIn        int64
	ExpiresAt        string
	RefreshExpiresAt string
	AccountID        string
}

// Login authenticates user and creates session
func (s *Service) Login(ctx context.Context, dto LoginDTO) (*Result, error) {
	// 1. Find account by email
	account, err := s.accounts.FindByEmail(ctx, dto.Email)
	if err != nil {
		return nil, terror.NewUnauthorizedErr("invalid credentials", err)
	}

	// 2. Verify password
	if err := s.hasher.Compare(account.PasswordHash, dto.Password); err != nil {
		return nil, terror.NewUnauthorizedErr("invalid credentials", err)
	}

	// 3. Check session limit
	count, err := s.sessions.CountByAccountID(ctx, account.ID.String())
	if err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}

	if count >= maxSessionsPerUser {
		// Evict oldest session
		if err := s.evictOldestSession(ctx, account.ID.String()); err != nil {
			return nil, fmt.Errorf("evict session: %w", err)
		}
	}

	// 4. Create session and tokens
	return s.createSessionAndTokens(ctx, account.ID.String(), dto.Fingerprint, dto.IP, dto.UserAgent)
}

func (s *Service) createSessionAndTokens(
	ctx context.Context,
	accountID, fingerprint, ip, ua string,
) (*Result, error) {
	sessionID := uuid.NewString()
	fingerprintHash := s.jwt.HashFingerprint(fingerprint)

	// Create session
	session := entity.NewSession(sessionID, accountID, fingerprintHash, ip, ua, s.jwt.RefreshTTL())
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Generate access token
	accessToken, err := s.jwt.GenerateAccessToken(sessionID, accountID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.jwt.GenerateRefreshToken(sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Save refresh pair
	_, randomPart, _ := s.jwt.ParseRefreshToken(refreshToken)
	pair := &entity.RefreshPair{
		Current: s.jwt.HashRefreshToken(randomPart),
	}
	if err := s.sessions.SaveRefreshPair(ctx, sessionID, pair); err != nil {
		return nil, fmt.Errorf("save refresh pair: %w", err)
	}

	return &Result{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(s.jwt.AccessTTL().Seconds()),
		ExpiresAt:        session.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		RefreshExpiresAt: session.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		AccountID:        accountID,
	}, nil
}

func (s *Service) evictOldestSession(ctx context.Context, accountID string) error {
	sessions, err := s.sessions.GetByAccountID(ctx, accountID)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return nil
	}

	oldest := sessions[0]
	for _, sess := range sessions[1:] {
		if sess.CreatedAt.Before(oldest.CreatedAt) {
			oldest = sess
		}
	}

	return s.sessions.Revoke(ctx, oldest.ID, accountID)
}
