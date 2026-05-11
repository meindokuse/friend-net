package logout

import (
	"context"
	"log/slog"

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
	slog.DebugContext(ctx, "logout: attempt")

	sessionID := dto.SessionID
	accountID := ""

	if sessionID == "" && dto.RefreshToken != "" {
		parsedSessionID, _, err := s.jwt.ParseRefreshToken(dto.RefreshToken)
		if err == nil {
			sessionID = parsedSessionID
		}
	}

	if dto.AccessToken != "" {
		accessSessionID, accessUserID, jti, expiresAt, err := s.jwt.ExtractAccessClaims(dto.AccessToken)
		if err == nil {
			if sessionID == "" {
				sessionID = accessSessionID
			}
			accountID = accessUserID
			s.blacklistAccess(ctx, jti, expiresAt)
		}
	}

	if sessionID == "" {
		slog.WarnContext(ctx, "logout: no session identifier in request")
		return terror.NewBadRequestErr("session id required", nil)
	}

	if accountID == "" {
		session, err := s.sessions.Get(ctx, sessionID)
		if err != nil {
			slog.DebugContext(ctx, "logout: session not found, nothing to revoke", "session_id", sessionID)
			return nil
		}
		accountID = session.AccountID
	}

	if err := s.sessions.Revoke(ctx, sessionID, accountID); err != nil {
		slog.ErrorContext(ctx, "logout: revoke session failed",
			"session_id", sessionID, "account_id", accountID, "error", err)
		return err
	}

	slog.InfoContext(ctx, "logout: session revoked", "session_id", sessionID, "account_id", accountID)
	return nil
}

// LogoutAll revokes all user sessions
func (s *Service) LogoutAll(ctx context.Context, accountID string) error {
	slog.DebugContext(ctx, "logout-all: attempt", "account_id", accountID)

	if err := s.sessions.RevokeAllByAccountID(ctx, accountID); err != nil {
		slog.ErrorContext(ctx, "logout-all: failed", "account_id", accountID, "error", err)
		return err
	}

	slog.InfoContext(ctx, "logout-all: all sessions revoked", "account_id", accountID)
	return nil
}

func (s *Service) blacklistAccess(ctx context.Context, jti string, expiresAt int64) {
	// Calculate remaining TTL
	now := s.jwt.Now().Unix()
	remaining := expiresAt - now
	if remaining > 0 {
		_ = s.sessions.BlacklistAccessToken(ctx, jti, remaining)
	}
}
