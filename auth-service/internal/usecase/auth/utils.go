package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
)

func (a *Auth) createSessionAndTokens(
	ctx context.Context,
	userID, fingerprintHash, ip, ua string,
) (*domain.AuthResult, error) {
	sessionID := uuid.NewString()

	// Сессия
	session := domainsession.NewSession(sessionID, userID, fingerprintHash, ip, ua, a.jwtManager.RefreshTTL())
	if err := a.redis.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}

	// Access JWT
	accessToken, err := a.jwtManager.GenerateAccessToken(sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: access token: %w", err)
	}

	// Refresh opaque
	refreshPlain, err := a.jwtManager.GenerateRefreshToken(sessionID)
	if err != nil {
		return nil, fmt.Errorf("auth: refresh token: %w", err)
	}

	// Хешируем random часть → Redis
	_, randomPart, _ := a.jwtManager.ParseRefreshToken(refreshPlain)
	pair := &domainsession.RefreshPair{
		Current: a.jwtManager.HashRefreshToken(randomPart),
	}
	if err := a.redis.SaveRefreshPair(ctx, sessionID, pair); err != nil {
		return nil, fmt.Errorf("auth: save refresh pair: %w", err)
	}

	return &domain.AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     refreshPlain,
		TokenType:        "Bearer",
		ExpiresIn:        int64(a.jwtManager.AccessTTL().Seconds()),
		ExpiresAt:        time.Now().UTC().Add(a.jwtManager.AccessTTL()),
		RefreshExpiresAt: time.Now().UTC().Add(a.jwtManager.RefreshTTL()),
		UserID:           userID,
	}, nil
}

func (a *Auth) rotateTokens(
	ctx context.Context,
	session *domainsession.Session,
	pair *domainsession.RefreshPair,
) (*domain.AuthResult, error) {
	// Новый refresh
	newRefreshPlain, err := a.jwtManager.GenerateRefreshToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh: %w", err)
	}

	_, newRandomPart, _ := a.jwtManager.ParseRefreshToken(newRefreshPlain)
	newHash := a.jwtManager.HashRefreshToken(newRandomPart)

	// Rotation: current → prev, new → current
	pair.Rotate(newHash, a.jwtManager.GracePeriod())

	if err := a.redis.SaveRefreshPair(ctx, session.ID, pair); err != nil {
		return nil, fmt.Errorf("auth: save pair: %w", err)
	}

	_ = a.redis.UpdateLastSeen(ctx, session.ID)

	// Новый access
	accessToken, err := a.jwtManager.GenerateAccessToken(session.ID, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: access token: %w", err)
	}

	return &domain.AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshPlain,
		TokenType:        "Bearer",
		ExpiresIn:        int64(a.jwtManager.AccessTTL().Seconds()),
		ExpiresAt:        time.Now().UTC().Add(a.jwtManager.AccessTTL()),
		RefreshExpiresAt: time.Now().UTC().Add(a.jwtManager.RefreshTTL()),
		UserID:           session.UserID,
	}, nil
}

func (a *Auth) rotateTokensGrace(
	ctx context.Context,
	session *domainsession.Session,
	pair *domainsession.RefreshPair,
) (*domain.AuthResult, error) {
	// Grace period: делаем rotation но prev НЕ перезаписываем
	// Потому что клиент может повторить ещё раз со старым токеном
	newRefreshPlain, err := a.jwtManager.GenerateRefreshToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: generate refresh: %w", err)
	}

	_, newRandomPart, _ := a.jwtManager.ParseRefreshToken(newRefreshPlain)
	newHash := a.jwtManager.HashRefreshToken(newRandomPart)

	// current = new, prev ОСТАЁТСЯ тем же (не перезаписываем)
	pair.Current = newHash
	// pair.Prev и pair.PrevExpiresAt не трогаем

	if err := a.redis.SaveRefreshPair(ctx, session.ID, pair); err != nil {
		return nil, fmt.Errorf("auth: save pair: %w", err)
	}

	_ = a.redis.UpdateLastSeen(ctx, session.ID)

	accessToken, err := a.jwtManager.GenerateAccessToken(session.ID, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: access token: %w", err)
	}

	return &domain.AuthResult{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshPlain,
		TokenType:        "Bearer",
		ExpiresIn:        int64(a.jwtManager.AccessTTL().Seconds()),
		ExpiresAt:        time.Now().UTC().Add(a.jwtManager.AccessTTL()),
		RefreshExpiresAt: time.Now().UTC().Add(a.jwtManager.RefreshTTL()),
		UserID:           session.UserID,
	}, nil
}

func (a *Auth) blacklistAccess(ctx context.Context, accessToken string) {
	// Парсим БЕЗ проверки expiration (токен может быть уже expired)
	_, _, jti, expiresAt, err := a.jwtManager.ExtractAccessClaims(accessToken)
	if err != nil {
		return // не критично
	}

	remaining := time.Until(expiresAt)
	if remaining > 0 {
		_ = a.redis.BlacklistAccessToken(ctx, jti, remaining)
	}
}

func (a *Auth) evictOldestSession(ctx context.Context, userID string) error {
	sessions, err := a.redis.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return nil
	}

	oldest := sessions[0]
	for _, s := range sessions[1:] {
		if s.CreatedAt.Before(oldest.CreatedAt) {
			oldest = s
		}
	}

	return a.redis.RevokeSession(ctx, oldest.ID, userID)
}
