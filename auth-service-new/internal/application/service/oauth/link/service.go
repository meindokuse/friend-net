package link

import (
	"context"
	"errors"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// OAuthRepository interface for OAuth operations
type OAuthRepository interface {
	GetByProviderID(ctx context.Context, provider entity.OAuthProvider, providerID string) (*entity.OAuthAccount, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.OAuthAccount, error)
	Create(ctx context.Context, account *entity.OAuthAccount) error
}

// OAuthProviderGateway interface for OAuth provider operations
type OAuthProviderGateway interface {
	ExchangeToken(ctx context.Context, code string) (*OAuthToken, error)
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)
}

// OAuthToken represents OAuth tokens
type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       int64
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
}

// Service handles OAuth link use case
type Service struct {
	oauth     OAuthRepository
	providers map[entity.OAuthProvider]OAuthProviderGateway
}

// NewService creates a new link service
func NewService(
	oauth OAuthRepository,
	providers map[entity.OAuthProvider]OAuthProviderGateway,
) *Service {
	return &Service{
		oauth:     oauth,
		providers: providers,
	}
}

// LinkDTO contains link input
type LinkDTO struct {
	Provider  entity.OAuthProvider
	Code      string
	State     string
	AccountID string
}

// Link links an OAuth provider to existing account
func (s *Service) Link(ctx context.Context, dto LinkDTO) error {
	provider, ok := s.providers[dto.Provider]
	if !ok {
		return terror.NewBadRequestErr("unsupported provider", nil)
	}

	// Exchange code
	oauthToken, err := provider.ExchangeToken(ctx, dto.Code)
	if err != nil {
		return err
	}

	// Get user info
	userInfo, err := provider.GetUserInfo(ctx, oauthToken.AccessToken)
	if err != nil {
		return err
	}

	// Check if already linked to another account
	existingOAuth, err := s.oauth.GetByProviderID(ctx, dto.Provider, userInfo.ProviderID)
	if err != nil {
		return err
	}
	if existingOAuth != nil {
		return errors.New("this account is already linked to another user")
	}

	// Check if provider already linked to current user
	linkedAccounts, err := s.oauth.GetByAccountID(ctx, dto.AccountID)
	if err != nil {
		return err
	}
	for _, acc := range linkedAccounts {
		if acc.Provider == dto.Provider {
			return errors.New("you already have this provider linked")
		}
	}

	// Create OAuth link
	oauthAccount := entity.NewOAuthAccount(dto.AccountID, dto.Provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = oauthToken.AccessToken
	oauthAccount.RefreshToken = oauthToken.RefreshToken

	return s.oauth.Create(ctx, oauthAccount)
}
