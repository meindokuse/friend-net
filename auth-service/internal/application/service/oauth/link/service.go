package link

import (
	"context"
	"errors"
	"log/slog"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/providers"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// OAuthRepository interface for OAuth operations
type OAuthRepository interface {
	GetByProviderID(ctx context.Context, provider entity.OAuthProvider, providerID string) (*entity.OAuthAccount, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*entity.OAuthAccount, error)
	Create(ctx context.Context, account *entity.OAuthAccount) error
}


// Service handles OAuth link use case
type Service struct {
	oauth     OAuthRepository
	providers map[entity.OAuthProvider]providers.OAuthProviderGateway
}

// NewService creates a new link service
func NewService(
	oauth OAuthRepository,
	providers map[entity.OAuthProvider]providers.OAuthProviderGateway,
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
	slog.DebugContext(ctx, "oauth-link: attempt",
		"account_id", dto.AccountID, "provider", dto.Provider)

	provider, ok := s.providers[dto.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth-link: unsupported provider", "provider", dto.Provider)
		return terror.NewBadRequestErr("unsupported provider", nil)
	}

	oauthToken, err := provider.ExchangeToken(ctx, dto.Code)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-link: token exchange failed",
			"account_id", dto.AccountID, "provider", dto.Provider, "error", err)
		return err
	}

	userInfo, err := provider.GetUserInfo(ctx, oauthToken.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-link: get user info failed",
			"account_id", dto.AccountID, "provider", dto.Provider, "error", err)
		return err
	}

	existingOAuth, err := s.oauth.GetByProviderID(ctx, dto.Provider, userInfo.ProviderID)
	if err != nil {
		return err
	}
	if existingOAuth != nil {
		slog.WarnContext(ctx, "oauth-link: provider already linked to another account",
			"account_id", dto.AccountID, "provider", dto.Provider)
		return errors.New("this account is already linked to another user")
	}

	linkedAccounts, err := s.oauth.GetByAccountID(ctx, dto.AccountID)
	if err != nil {
		return err
	}
	for _, acc := range linkedAccounts {
		if acc.Provider == dto.Provider {
			slog.WarnContext(ctx, "oauth-link: provider already linked to this account",
				"account_id", dto.AccountID, "provider", dto.Provider)
			return errors.New("you already have this provider linked")
		}
	}

	oauthAccount := entity.NewOAuthAccount(dto.AccountID, dto.Provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = oauthToken.AccessToken
	oauthAccount.RefreshToken = oauthToken.RefreshToken

	if err := s.oauth.Create(ctx, oauthAccount); err != nil {
		slog.ErrorContext(ctx, "oauth-link: create link failed",
			"account_id", dto.AccountID, "provider", dto.Provider, "error", err)
		return err
	}

	slog.InfoContext(ctx, "oauth-link: provider linked",
		"account_id", dto.AccountID, "provider", dto.Provider)
	return nil
}
