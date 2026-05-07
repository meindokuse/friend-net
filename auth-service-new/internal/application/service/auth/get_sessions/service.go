package get_sessions

import (
	"context"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// SessionProvider interface for session operations
type SessionProvider interface {
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.Session, error)
}

// Service handles get sessions use case
type Service struct {
	sessions SessionProvider
}

// NewService creates a new get sessions service
func NewService(
	sessions SessionProvider,
) *Service {
	return &Service{
		sessions: sessions,
	}
}

// SessionInfo represents session info for response
type SessionInfo struct {
	ID         string `json:"id"`
	IPAddress  string `json:"ip_address"`
	UserAgent  string `json:"user_agent"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	Current    bool   `json:"current"`
}

// GetSessions returns all active sessions for a user
func (s *Service) GetSessions(ctx context.Context, accountID, currentSessionID string) ([]SessionInfo, error) {
	sessions, err := s.sessions.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	result := make([]SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		// Skip revoked sessions
		if sess.Status == entity.SessionStatusRevoked {
			continue
		}

		result = append(result, SessionInfo{
			ID:         sess.ID,
			IPAddress:  sess.IPAddress,
			UserAgent:  sess.UserAgent,
			CreatedAt:  sess.CreatedAt.Format("2006-01-02T15:04:05Z"),
			LastSeenAt: sess.LastSeenAt.Format("2006-01-02T15:04:05Z"),
			Current:    sess.ID == currentSessionID,
		})
	}

	return result, nil
}
