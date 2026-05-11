package unlink

import (
	"context"
	"errors"
	"log/slog"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// OAuthRepository interface for OAuth operations
type OAuthRepository interface {
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.OAuthAccount, error)
	Delete(ctx context.Context, id string) error
}

// Service handles OAuth unlink use case
type Service struct {
	oauth OAuthRepository
}

// NewService creates a new unlink service
func NewService(
	oauth OAuthRepository,
) *Service {
	return &Service{
		oauth: oauth,
	}
}

// Unlink removes OAuth provider from account
func (s *Service) Unlink(ctx context.Context, accountID string, provider entity.OAuthProvider) error {
	slog.DebugContext(ctx, "oauth-unlink: attempt",
		"account_id", accountID, "provider", provider)

	accounts, err := s.oauth.GetByAccountID(ctx, accountID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-unlink: get linked accounts failed",
			"account_id", accountID, "error", err)
		return err
	}

	var targetAccount *entity.OAuthAccount
	for _, acc := range accounts {
		if acc.Provider == provider {
			targetAccount = acc
			break
		}
	}

	if targetAccount == nil {
		slog.WarnContext(ctx, "oauth-unlink: provider not linked",
			"account_id", accountID, "provider", provider)
		return errors.New("provider not linked")
	}

	if err := s.oauth.Delete(ctx, targetAccount.ID); err != nil {
		slog.ErrorContext(ctx, "oauth-unlink: delete failed",
			"account_id", accountID, "provider", provider, "error", err)
		return err
	}

	slog.InfoContext(ctx, "oauth-unlink: provider unlinked",
		"account_id", accountID, "provider", provider)
	return nil
}
