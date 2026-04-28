package usecase

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
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

	account, err := domain.NewAccount(registerData.Email, passwordHash)
	if err != nil {
		slog.ErrorContext(ctx, "register create account failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: create account: %v", ErrInternal, err)
	}

	outboxEvent, err := outbox.NewAccountCreatedEvent(
		account.ID,
		account.Email,
		registerData.DisplayName,
		account.CreatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "register create outbox event failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: create outbox: %v", ErrInternal, err)
	}

	accountID, err := uc.db.SaveWithOutbox(ctx, account, outboxEvent)
	if err != nil {
		slog.WarnContext(ctx, "register user save failed", slog.String("error", err.Error()))
		return "", mapPostgresError(err)
	}

	ctx = sharedlogger.WithField(ctx, "account_id", accountID)

	slog.InfoContext(ctx, "register usecase completed")
	return accountID.String(), nil
}
