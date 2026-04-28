package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

// LinkAccount привязывает OAuth провайдера к существующему аккаунту.
func (uc *OAuth) LinkAccount(ctx context.Context, input OAuthLinkInput) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider":   string(input.Provider),
		"account_id": input.CurrentAccountID,
	})
	slog.InfoContext(ctx, "oauth link started")

	provider, ok := uc.oauthProviders[input.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth link rejected: unsupported provider")
		return fmt.Errorf("unsupported provider: %s", input.Provider)
	}

	oauthTokens, err := provider.ExchangeToken(ctx, input.Code)
	if err != nil {
		slog.ErrorContext(ctx, "oauth link token exchange failed", slog.String("error", err.Error()))
		return fmt.Errorf("exchange oauth token: %w", err)
	}

	userInfo, err := provider.GetUserInfo(ctx, oauthTokens.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "oauth link user info loading failed", slog.String("error", err.Error()))
		return fmt.Errorf("load oauth user info: %w", err)
	}

	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider_id": userInfo.ProviderID,
		"email":       userInfo.Email,
	})

	existingOAuth, err := uc.oauthRepo.GetByProviderID(ctx, input.Provider, userInfo.ProviderID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth link provider account lookup failed", slog.String("error", err.Error()))
		return fmt.Errorf("get oauth account: %w", err)
	}
	if existingOAuth != nil {
		slog.WarnContext(ctx, "oauth link rejected: provider account already linked")
		return errors.New("this account is already linked")
	}

	existingAccounts, err := uc.oauthRepo.GetByAccountID(ctx, input.CurrentAccountID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth link current user linked accounts lookup failed", slog.String("error", err.Error()))
		return fmt.Errorf("get linked accounts: %w", err)
	}
	for _, acc := range existingAccounts {
		if acc.Provider == input.Provider {
			slog.WarnContext(ctx, "oauth link rejected: provider already attached to current user")
			return errors.New("you already have this provider linked")
		}
	}

	oauthAccount := domain.NewOAuthAccount(input.CurrentAccountID, input.Provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = oauthTokens.AccessToken
	oauthAccount.RefreshToken = oauthTokens.RefreshToken
	oauthAccount.Expiry = oauthTokens.Expiry

	if err := uc.oauthRepo.Create(ctx, oauthAccount); err != nil {
		slog.ErrorContext(ctx, "oauth link account creation failed", slog.String("error", err.Error()))
		return err
	}

	slog.InfoContext(ctx, "oauth link completed")
	return nil
}
