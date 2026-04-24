package usecase

import (
	"context"
	"fmt"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

func (a *Auth) GetUserSessions(ctx context.Context, userID, currentSessionID string) ([]domain.SessionInfo, error) {
	sessions, err := a.redis.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: get sessions: %w", err)
	}

	result := make([]domain.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, domain.SessionInfo{
			ID:         s.ID,
			IPAddress:  s.IPAddress,
			UserAgent:  s.UserAgent,
			CreatedAt:  s.CreatedAt,
			LastSeenAt: s.LastSeenAt,
			Current:    s.ID == currentSessionID,
		})
	}

	return result, nil
}

// ─── Revoke конкретной сессии (со страницы "мои устройства") ───

func (a *Auth) RevokeSession(ctx context.Context, userID, sessionID string) error {
	// Проверяем что сессия принадлежит этому пользователю
	session, err := a.redis.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("auth: get session: %w", err)
	}

	if session.AccountID != userID {
		return ErrSessionNotFound // не раскрываем что сессия чужая
	}

	return a.redis.RevokeSession(ctx, sessionID, userID)
}
