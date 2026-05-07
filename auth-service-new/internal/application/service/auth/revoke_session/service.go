package revoke_session

import (
	"context"

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
	// Verify session belongs to user
	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return terror.NewNotFoundErr("session not found", err)
	}

	if session.AccountID != accountID {
		// Don't reveal that session belongs to another user
		return terror.NewNotFoundErr("session not found", nil)
	}

	return s.sessions.Revoke(ctx, sessionID, accountID)
}
