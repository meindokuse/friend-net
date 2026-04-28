package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

// UnlinkAccount отвязывает OAuth провайдера от аккаунта.
func (uc *OAuth) UnlinkAccount(ctx context.Context, userID string, provider domain.OAuthProvider) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id": userID,
		"provider":   string(provider),
	})
	slog.InfoContext(ctx, "oauth unlink started")

	accounts, err := uc.oauthRepo.GetByAccountID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth unlink linked accounts lookup failed", slog.String("error", err.Error()))
		return fmt.Errorf("get linked accounts: %w", err)
	}

	var targetAccount *domain.OAuthAccount
	for _, acc := range accounts {
		if acc.Provider == provider {
			targetAccount = acc
			break
		}
	}

	if targetAccount == nil {
		slog.WarnContext(ctx, "oauth unlink rejected: provider not linked")
		return errors.New("provider not linked")
	}

	if err := uc.oauthRepo.Delete(ctx, targetAccount.ID); err != nil {
		slog.ErrorContext(ctx, "oauth unlink delete failed", slog.String("error", err.Error()))
		return err
	}

	slog.InfoContext(ctx, "oauth unlink completed")
	return nil
}

// GetLinkedAccounts возвращает список привязанных OAuth провайдеров.
func (uc *OAuth) GetLinkedAccounts(ctx context.Context, userID string) ([]*domain.OAuthAccount, error) {
	ctx = sharedlogger.WithField(ctx, "account_id", userID)
	slog.InfoContext(ctx, "oauth get linked accounts started")

	accounts, err := uc.oauthRepo.GetByAccountID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth get linked accounts failed", slog.String("error", err.Error()))
		return nil, err
	}

	slog.InfoContext(ctx, "oauth get linked accounts completed", slog.Int("accounts_count", len(accounts)))
	return accounts, nil
}
