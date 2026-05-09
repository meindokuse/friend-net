package logout

import (
	"context"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// SessionManager interface for session operations
type SessionManager interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	Revoke(ctx context.Context, sessionID, accountID string) error
	RevokeAllByAccountID(ctx context.Context, accountID string) error
	BlacklistAccessToken(ctx context.Context, jti string, ttl int64) error
}

// Service handles logout use case
type Service struct {
	sessions SessionManager
	jwt      *jwt.Manager
}

// NewService creates a new logout service
func NewService(
	sessions SessionManager,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		sessions: sessions,
		jwt:      jwtManager,
	}
}

// DTO for logout input
type LogoutDTO struct {
	AccessToken  string
	RefreshToken string
	SessionID    string
}

// Logout revokes current session
func (s *Service) Logout(ctx context.Context, dto LogoutDTO) error {
	sessionID := dto.SessionID
	accountID := ""

	// Extract sessionID from refresh token if not provided
	if sessionID == "" && dto.RefreshToken != "" {
		parsedSessionID, _, err := s.jwt.ParseRefreshToken(dto.RefreshToken)
		if err == nil {
			sessionID = parsedSessionID
		}
	}

	// Extract from access token
	if dto.AccessToken != "" {
		accessSessionID, accessUserID, jti, expiresAt, err := s.jwt.ExtractAccessClaims(dto.AccessToken)
		if err == nil {
			if sessionID == "" {
				sessionID = accessSessionID
			}
			accountID = accessUserID

			// Blacklist access token
			s.blacklistAccess(ctx, jti, expiresAt)
		}
	}

	if sessionID == "" {
		return terror.NewBadRequestErr("session id required", nil)
	}

	// Get accountID if not extracted from token
	if accountID == "" {
		session, err := s.sessions.Get(ctx, sessionID)
		if err != nil {
			return nil // Session not found, nothing to logout
		}
		accountID = session.AccountID
	}

	return s.sessions.Revoke(ctx, sessionID, accountID)
}

// LogoutAll revokes all user sessions
func (s *Service) LogoutAll(ctx context.Context, accountID string) error {
	return s.sessions.RevokeAllByAccountID(ctx, accountID)
}

func (s *Service) blacklistAccess(ctx context.Context, jti string, expiresAt int64) {
	// Calculate remaining TTL
	now := s.jwt.Now().Unix()
	remaining := expiresAt - now
	if remaining > 0 {
		_ = s.sessions.BlacklistAccessToken(ctx, jti, remaining)
	}
}
