package usecase

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
)

func (uc *Auth) Register(ctx context.Context, registerData domain.Register) (string, error) {
	ctx = sharedlogger.WithField(ctx, "email", registerData.Email)
	slog.InfoContext(ctx, "register usecase started")

	if uc.db == nil {
		slog.ErrorContext(ctx, "register database dependency is missing")
		return "", ErrInternal
	}

	if uc.redis == nil {
		slog.ErrorContext(ctx, "register redis dependency is missing")
		return "", ErrInternal
	}

	if uc.jwtManager == nil {
		slog.ErrorContext(ctx, "register jwt manager dependency is missing")
		return "", ErrInternal
	}

	hasher := uc.hasher
	if hasher == nil {
		hasher = pass.New(pass.Config{})
	}

	passwordHash, err := hasher.Hash(registerData.Password)
	if err != nil {
		slog.ErrorContext(ctx, "register password hashing failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: hash password: %v", ErrInternal, err)
	}

	account := domain.NewAccount(registerData.Email, passwordHash)

	accountID, err := uc.db.Save(ctx, *account)
	if err != nil {
		slog.WarnContext(ctx, "register user save failed", slog.String("error", err.Error()))
		return "", mapPostgresError(err)
	}

	account.ID = accountID
	ctx = sharedlogger.WithField(ctx, "account_id", account.ID)

	slog.InfoContext(ctx, "register usecase completed")
	return accountID, nil
}
