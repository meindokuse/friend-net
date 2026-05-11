package revoke_session

import (
	"context"
	"log/slog"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// SessionManager interface for session operations
type SessionManager interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	Revoke(ctx context.Context, sessionID, accountID string) error
}

// Service handles revoke session use case
type Service struct {
	sessions SessionManager
}

// NewService creates a new revoke session service
func NewService(
	sessions SessionManager,
) *Service {
	return &Service{
		sessions: sessions,
	}
}

// RevokeSession revokes a specific session
func (s *Service) RevokeSession(ctx context.Context, accountID, sessionID string) error {
	slog.DebugContext(ctx, "revoke-session: attempt",
		"account_id", accountID, "session_id", sessionID)

	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "revoke-session: session not found", "session_id", sessionID)
		return terror.NewNotFoundErr("session not found", err)
	}

	if session.AccountID != accountID {
		slog.WarnContext(ctx, "revoke-session: ownership mismatch",
			"account_id", accountID, "session_owner", session.AccountID, "session_id", sessionID)
		return terror.NewNotFoundErr("session not found", nil)
	}

	if err := s.sessions.Revoke(ctx, sessionID, accountID); err != nil {
		slog.ErrorContext(ctx, "revoke-session: storage error",
			"account_id", accountID, "session_id", sessionID, "error", err)
		return err
	}

	slog.InfoContext(ctx, "revoke-session: success",
		"account_id", accountID, "session_id", sessionID)
	return nil
}
