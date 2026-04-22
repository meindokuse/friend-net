package usecase

import (
	"context"
	"fmt"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
)

func (a *Auth) Logout(ctx context.Context, input domain.LogoutInput) error {
	sessionID := input.SessionID
	userID := ""

	if sessionID == "" && input.RefreshToken != "" {
		parsedSessionID, _, err := a.jwtManager.ParseRefreshToken(input.RefreshToken)
		if err == nil {
			sessionID = parsedSessionID
		}
	}

	if input.AccessToken != "" {
		accessSessionID, accessUserID, _, _, err := a.jwtManager.ExtractAccessClaims(input.AccessToken)
		if err == nil {
			if sessionID == "" {
				sessionID = accessSessionID
			}
			userID = accessUserID
		}
	}

	if sessionID == "" {
		return ErrInvalidToken
	}

	if userID == "" {
		session, err := a.redis.GetSession(ctx, sessionID)
		if err != nil {
			return nil
		}
		userID = session.UserID
	}

	if err := a.redis.RevokeSession(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("auth: revoke: %w", err)
	}

	if input.AccessToken != "" {
		a.blacklistAccess(ctx, input.AccessToken)
	}

	return nil
}

func (a *Auth) LogoutAll(ctx context.Context, userID string) error {
	return a.redis.RevokeAllUserSessions(ctx, userID)
}
