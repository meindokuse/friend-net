package usecase

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

const MaxSessionsPerUser = 3

func (uc *Auth) LoginUser(ctx context.Context, input domain.LoginInput) (*domain.AuthResult, error) {
	ctx = sharedlogger.WithField(ctx, "email", input.Email)
	slog.InfoContext(ctx, "login usecase started")

	account, err := uc.db.FindAccount(ctx, domain.Login{Email: input.Email})
	if err != nil {
		slog.WarnContext(ctx, "login user lookup failed", slog.String("error", err.Error()))
		return nil, mapPostgresError(err)
	}

	ctx = sharedlogger.WithField(ctx, "account_id", account.ID)

	hasher := uc.hasher
	if hasher == nil {
		hasher = pass.New(pass.Config{})
	}

	if err := hasher.Compare(account.PasswordHash, input.Password); err != nil {
		slog.WarnContext(ctx, "login password validation failed")
		return nil, ErrInvalidCredentials
	}

	// 3. Проверяем лимит сессий
	count, err := uc.redis.CountUserSessions(ctx, account.ID.String())
	if err != nil {
		return nil, fmt.Errorf("auth: count sessions: %w", err)
	}
	if count >= MaxSessionsPerUser {
		// Стратегия: убиваем самую старую
		if err := uc.evictOldestSession(ctx, account.ID.String()); err != nil {
			return nil, fmt.Errorf("auth: evict session: %w", err)
		}
	}

	slog.InfoContext(ctx, "login usecase completed")
	fingerprintHash := uc.jwtManager.HashFingerprint(input.Fingerprint)
	return uc.createSessionAndTokens(ctx, account.ID.String(), fingerprintHash, input.IP, input.UserAgent)
}
