package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	postgresqlerrors "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/postgresql"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

func (uc *OAuth) Login(ctx context.Context, input OAuthCallbackInput) (*OAuthOutput, error) {
	ctx = sharedlogger.WithField(ctx, "provider", string(input.Provider))
	slog.InfoContext(ctx, "oauth login started")

	if input.State == "" {
		slog.WarnContext(ctx, "oauth login rejected: csrf token missing")
		return nil, errors.New("csrf token missing")
	}

	provider, ok := uc.oauthProviders[input.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth login rejected: unsupported provider")
		return nil, fmt.Errorf("unsupported provider: %s", input.Provider)
	}

	oauthTokens, err := provider.ExchangeToken(ctx, input.Code)
	if err != nil {
		slog.ErrorContext(ctx, "oauth token exchange failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("exchange oauth token: %w", err)
	}

	userInfo, err := provider.GetUserInfo(ctx, oauthTokens.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "oauth user info loading failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("load oauth user info: %w", err)
	}

	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider_id": userInfo.ProviderID,
		"email":       userInfo.Email,
	})

	existingOAuth, err := uc.oauthRepo.GetByProviderID(ctx, input.Provider, userInfo.ProviderID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth account lookup failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("get oauth account: %w", err)
	}

	var accountID string
	isNewUser := false

	if existingOAuth != nil {
		accountID = existingOAuth.AccountID
		ctx = sharedlogger.WithField(ctx, "account_id", accountID)
		slog.InfoContext(ctx, "oauth account found")

		if err := uc.oauthRepo.UpdateTokens(ctx, existingOAuth.ID, oauthTokens.AccessToken, oauthTokens.RefreshToken, oauthTokens.Expiry); err != nil {
			slog.ErrorContext(ctx, "oauth token update failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("update oauth tokens: %w", err)
		}
	} else {
		account, err := uc.accountRepo.FindAccount(ctx, domain.Login{Email: userInfo.Email})
		if err == nil && account != nil {
			accountID = account.ID.String()
			ctx = sharedlogger.WithField(ctx, "account_id", accountID)
			slog.InfoContext(ctx, "oauth matched existing account")

			if err := uc.createOAuthLink(ctx, accountID, input.Provider, userInfo, oauthTokens); err != nil {
				return nil, err
			}
		} else if isUserNotFound(err) {
			accountID, err = uc.createOAuthAccount(ctx, userInfo, input.Provider, oauthTokens)
			if err != nil {
				return nil, err
			}
			isNewUser = true
			ctx = sharedlogger.WithField(ctx, "account_id", accountID)
			slog.InfoContext(ctx, "oauth created new account")
		} else {
			slog.ErrorContext(ctx, "oauth account lookup failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("find account by email: %w", err)
		}
	}

	output, err := uc.createSessionForAccount(ctx, accountID, userInfo.Email, input.RequestData)
	if err != nil {
		return nil, err
	}

	output.IsNewUser = isNewUser
	output.IsLinked = false

	slog.InfoContext(ctx, "oauth login completed", slog.Bool("is_new_user", isNewUser))
	return output, nil
}

func isUserNotFound(err error) bool {
	return errors.Is(err, postgresqlerrors.ErrNoRows)
}
