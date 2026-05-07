package get_linked

import (
	"context"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
)

// OAuthRepository interface for OAuth operations
type OAuthRepository interface {
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.OAuthAccount, error)
}

// Service handles get linked accounts use case
type Service struct {
	oauth OAuthRepository
}

// NewService creates a new get linked service
func NewService(
	oauth OAuthRepository,
) *Service {
	return &Service{
		oauth: oauth,
	}
}

// LinkedAccount represents a linked OAuth account
type LinkedAccount struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

// GetLinked returns all linked OAuth accounts
func (s *Service) GetLinked(ctx context.Context, accountID string) ([]LinkedAccount, error) {
	accounts, err := s.oauth.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	result := make([]LinkedAccount, 0, len(accounts))
	for _, acc := range accounts {
		result = append(result, LinkedAccount{
			ID:        acc.ID,
			Provider:  string(acc.Provider),
			Email:     acc.Email,
			CreatedAt: acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return result, nil
}
