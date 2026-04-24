package usecase

import (
	"context"
	"fmt"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

func (a *Auth) Logout(ctx context.Context, input domain.LogoutInput) error {
	sessionID := input.SessionID
	accountID := ""

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
			accountID = accessUserID
		}
	}

	if sessionID == "" {
		return ErrInvalidToken
	}

	if accountID == "" {
		session, err := a.redis.GetSession(ctx, sessionID)
		if err != nil {
			return nil
		}
		accountID = session.AccountID
	}

	if err := a.redis.RevokeSession(ctx, sessionID, accountID); err != nil {
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
