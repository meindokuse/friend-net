package usecase

import (
	"context"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

func (a *Auth) ValidateAccessToken(ctx context.Context, accessToken string) (*domain.AccessTokenInfo, error) {
	claims, err := a.jwtManager.VerifyAccessToken(accessToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	blacklisted, err := a.redis.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, ErrInternal
	}
	if blacklisted {
		return nil, ErrInvalidToken
	}

	return &domain.AccessTokenInfo{
		AccountID: claims.Subject,
		SessionID: claims.SessionID,
		JTI:       claims.ID,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
