package oauth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

// createOAuthAccount создаёт новый аккаунт через OAuth.
// Создает запись в accounts (основной аккаунт) и oauth_accounts (связь с провайдером).
func (uc *OAuth) createOAuthAccount(
	ctx context.Context,
	userInfo *OAuthUserInfo,
	provider domain.OAuthProvider,
	oauthTokens *OAuthToken,
) (string, error) {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider":    string(provider),
		"provider_id": userInfo.ProviderID,
		"email":       userInfo.Email,
	})
	slog.InfoContext(ctx, "oauth account creation started")

	// Создаём Account (без пароля для OAuth - пустой password_hash)
	account, err := domain.NewAccount(userInfo.Email, "")
	if err != nil {
		slog.ErrorContext(ctx, "oauth account creation failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("create account: %w", err)
	}

	// Создаём outbox событие для синхронизации с user-service
	outboxEvent, err := outbox.NewAccountCreatedEvent(
		account.ID,
		account.Email,
		userInfo.Name,
		account.CreatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "oauth outbox event creation failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("create outbox event: %w", err)
	}

	// Сохраняем Account + Outbox Event в одной транзакции
	accountID, err := uc.accountRepo.SaveWithOutbox(ctx, account, outboxEvent)
	if err != nil {
		slog.ErrorContext(ctx, "oauth account save failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("save account: %w", err)
	}

	accountIDStr := accountID.String()
	ctx = sharedlogger.WithField(ctx, "account_id", accountIDStr)
	slog.InfoContext(ctx, "oauth account created in accounts table", slog.String("account_id", accountIDStr))

	// Создаём OAuth связь в таблице oauth_accounts
	if err := uc.createOAuthLink(ctx, accountIDStr, provider, userInfo, oauthTokens); err != nil {
		return "", err
	}

	slog.InfoContext(ctx, "oauth account creation completed (both accounts and oauth_accounts tables)")
	return accountIDStr, nil
}

// createOAuthLink создаёт связь между Account и OAuth провайдером в таблице oauth_accounts.
func (uc *OAuth) createOAuthLink(
	ctx context.Context,
	accountID string,
	provider domain.OAuthProvider,
	userInfo *OAuthUserInfo,
	oauthTokens *OAuthToken,
) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id":  accountID,
		"provider":    string(provider),
		"provider_id": userInfo.ProviderID,
	})

	// Создаём запись в таблице oauth_accounts
	oauthAccount := domain.NewOAuthAccount(accountID, provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = oauthTokens.AccessToken
	oauthAccount.RefreshToken = oauthTokens.RefreshToken
	oauthAccount.Expiry = oauthTokens.Expiry

	if err := uc.oauthRepo.Create(ctx, oauthAccount); err != nil {
		slog.ErrorContext(ctx, "oauth link creation failed", slog.String("error", err.Error()))
		return fmt.Errorf("create oauth link: %w", err)
	}

	slog.InfoContext(ctx, "oauth link created in oauth_accounts table",
		slog.String("oauth_account_id", oauthAccount.ID))
	return nil
}

// createSessionForAccount создаёт сессию для аккаунта.
func (uc *OAuth) createSessionForAccount(
	ctx context.Context,
	accountID string,
	email string,
	requestData domain.RequestData,
) (*OAuthOutput, error) {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id": accountID,
		"email":      email,
	})
	slog.InfoContext(ctx, "oauth session creation started")

	sessionID := uuid.NewString()

	// Генерируем JWT токены
	accessToken, err := uc.jwtManager.GenerateAccessToken(sessionID, accountID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth access token generation failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := uc.jwtManager.GenerateRefreshToken(sessionID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth refresh token generation failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Создаём сессию
	session := domainsession.NewSession(
		sessionID,
		accountID,
		uc.jwtManager.HashFingerprint(requestData.Fingerprint),
		requestData.IPAddress,
		requestData.UserAgent,
		uc.sessionTTL,
	)

	if err := uc.sessionRepo.CreateSession(ctx, session); err != nil {
		slog.ErrorContext(ctx, "oauth session save failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Сохраняем refresh pair
	_, randomPart, err := uc.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	pair := &domainsession.RefreshPair{
		Current: uc.jwtManager.HashRefreshToken(randomPart),
	}
	if err := uc.sessionRepo.SaveRefreshPair(ctx, sessionID, pair); err != nil {
		return nil, fmt.Errorf("save refresh pair: %w", err)
	}

	slog.InfoContext(ctx, "oauth session creation completed")

	return &OAuthOutput{
		AccountID:        accountID,
		Email:            email,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresAt:        time.Now().UTC().Add(uc.jwtManager.AccessTTL()),
		RefreshExpiresAt: time.Now().UTC().Add(uc.jwtManager.RefreshTTL()),
	}, nil
}
