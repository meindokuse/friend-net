package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/meindokuse/cloud-drive/auth-service/internal/adapters/redis"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
)

func (a *Auth) Refresh(ctx context.Context, input domain.RefreshInput) (*domain.AuthResult, error) {
	// 1. Парсим refresh → session_id + random
	sessionID, randomPart, err := a.jwtManager.ParseRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("auth: parse refresh: %w", err)
	}

	// 2. Получаем сессию
	session, err := a.redis.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, redis.ErrSessionNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, ErrInternal
	}

	if !session.IsActive() {
		return nil, ErrSessionRevoked
	}

	// 3. Проверяем fingerprint
	fpHash := a.jwtManager.HashFingerprint(input.Fingerprint)
	if session.FingerprintHash != fpHash {
		return nil, ErrFingerprintMismatch
	}

	// 4. Получаем refresh pair
	pair, err := a.redis.GetRefreshPair(ctx, sessionID)
	if err != nil {
		return nil, mapRedisError(err)
	}

	// 5. Хешируем random часть, сверяем
	incomingHash := a.jwtManager.HashRefreshToken(randomPart)

	switch pair.Match(incomingHash) {
	case domainsession.RefreshMatchCurrent:
		return a.rotateTokens(ctx, session, pair)

	case domainsession.RefreshMatchPrev:
		// Grace period — тоже rotation, но prev не перезаписываем
		return a.rotateTokensGrace(ctx, session, pair)

	case domainsession.RefreshMatchNone:
		_ = a.redis.RevokeSession(ctx, sessionID, session.AccountID)
		return nil, ErrTokenReuse
	}

	return nil, ErrTokenReuse
}
