package usecase

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

const MaxSessionsPerUser = 3

func (uc *Auth) LoginUser(ctx context.Context, input domain.LoginInput) (*domain.AuthResult, error) {
	ctx = sharedlogger.WithField(ctx, "email", input.Email)
	slog.InfoContext(ctx, "login usecase started")

	user, err := uc.db.FindUser(ctx, domain.Login{Email: input.Email})
	if err != nil {
		slog.WarnContext(ctx, "login user lookup failed", slog.String("error", err.Error()))
		return nil, mapPostgresError(err)
	}

	ctx = sharedlogger.WithField(ctx, "user_id", user.ID)

	hasher := uc.hasher
	if hasher == nil {
		hasher = pass.New(pass.Config{})
	}

	if err := hasher.Compare(user.PasswordHash, input.Password); err != nil {
		slog.WarnContext(ctx, "login password validation failed")
		return nil, ErrInvalidCredentials
	}

	// 3. Проверяем лимит сессий
	count, err := uc.redis.CountUserSessions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: count sessions: %w", err)
	}
	if count >= MaxSessionsPerUser {
		// Стратегия: убиваем самую старую
		if err := uc.evictOldestSession(ctx, user.ID); err != nil {
			return nil, fmt.Errorf("auth: evict session: %w", err)
		}
	}

	slog.InfoContext(ctx, "login usecase completed")
	fingerprintHash := uc.jwtManager.HashFingerprint(input.Fingerprint)
	return uc.createSessionAndTokens(ctx, user.ID, fingerprintHash, input.IP, input.UserAgent)
}
