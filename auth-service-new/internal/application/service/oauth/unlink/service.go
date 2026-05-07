package unlink

import (
	"context"
	"errors"

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
	accounts, err := s.oauth.GetByAccountID(ctx, accountID)
	if err != nil {
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
		return errors.New("provider not linked")
	}

	return s.oauth.Delete(ctx, targetAccount.ID)
}
