package oauth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	postgresqlerrors "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/postgresql"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	domainsession "github.com/meindokuse/cloud-drive/auth-service/internal/domain/session"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

type AccountRepository interface {
	Save(ctx context.Context, accountData domain.Account) (string, error)
	FindAccount(ctx context.Context, loginData domain.Login) (*domain.Account, error)
}

type OAuthRepository interface {
	GetByProviderID(ctx context.Context, provider domain.OAuthProvider, providerID string) (*domain.OAuthAccount, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*domain.OAuthAccount, error)
	Create(ctx context.Context, account *domain.OAuthAccount) error
	UpdateTokens(ctx context.Context, accountID, accessToken, refreshToken string, expiry time.Time) error
	Delete(ctx context.Context, accountID string) error
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session *domainsession.Session) error
	SaveRefreshPair(ctx context.Context, sessionID string, pair *domainsession.RefreshPair) error
}

type OAuthUseCase interface {
	Login(ctx context.Context, input OAuthCallbackInput) (*OAuthOutput, error)
	LinkAccount(ctx context.Context, input OAuthLinkInput) error
	UnlinkAccount(ctx context.Context, accountID string, provider domain.OAuthProvider) error
	GetLinkedAccounts(ctx context.Context, accountID string) ([]*domain.OAuthAccount, error)
	RegisterProvider(provider domain.OAuthProvider, svc OAuthProviderService)
}

type oauthUseCase struct {
	accountRepo    AccountRepository
	oauthRepo      OAuthRepository
	sessionRepo    SessionRepository
	jwtSvc         *jwt.Manager
	oauthProviders map[domain.OAuthProvider]OAuthProviderService
	sessionTTL     time.Duration
}

func NewOAuthUseCase(
	accountRepo AccountRepository,
	oauthRepo OAuthRepository,
	sessionRepo SessionRepository,
	jwtSvc *jwt.Manager,
	sessionTTL time.Duration,
) OAuthUseCase {
	return &oauthUseCase{
		accountRepo:    accountRepo,
		oauthRepo:      oauthRepo,
		sessionRepo:    sessionRepo,
		jwtSvc:         jwtSvc,
		sessionTTL:     sessionTTL,
		oauthProviders: make(map[domain.OAuthProvider]OAuthProviderService),
	}
}

func (uc *oauthUseCase) RegisterProvider(provider domain.OAuthProvider, svc OAuthProviderService) {
	uc.oauthProviders[provider] = svc
	slog.InfoContext(context.Background(), "oauth provider registered", slog.String("provider", string(provider)))
}

func (uc *oauthUseCase) Login(ctx context.Context, input OAuthCallbackInput) (*OAuthOutput, error) {
	ctx = sharedlogger.WithField(ctx, "provider", string(input.Provider))
	slog.InfoContext(ctx, "oauth login started")

	if input.State == "" {
		slog.WarnContext(ctx, "oauth login rejected because csrf token is missing")
		return nil, errors.New("csrf token missing")
	}

	provider, ok := uc.oauthProviders[input.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth login rejected because provider is unsupported")
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

	switch {
	case existingOAuth != nil:
		accountID = existingOAuth.AccountID
		ctx = sharedlogger.WithField(ctx, "account_id", accountID)
		slog.InfoContext(ctx, "oauth account found for provider user")
		if err := uc.oauthRepo.UpdateTokens(ctx, existingOAuth.ID, oauthTokens.AccessToken, oauthTokens.RefreshToken, oauthTokens.Expiry); err != nil {
			slog.ErrorContext(ctx, "oauth token update failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("update oauth tokens: %w", err)
		}
	default:
		account, err := uc.accountRepo.FindAccount(ctx, domain.Login{Email: userInfo.Email})
		switch {
		case err == nil && account != nil:
			accountID = account.ID
			ctx = sharedlogger.WithField(ctx, "account_id", accountID)
			slog.InfoContext(ctx, "oauth matched existing local user by email")

			oauthAccount := domain.NewOAuthAccount(accountID, input.Provider, userInfo.ProviderID, userInfo.Email)
			oauthAccount.AccessToken = oauthTokens.AccessToken
			oauthAccount.RefreshToken = oauthTokens.RefreshToken
			oauthAccount.Expiry = oauthTokens.Expiry

			if err := uc.oauthRepo.Create(ctx, oauthAccount); err != nil {
				slog.ErrorContext(ctx, "oauth account creation for existing user failed", slog.String("error", err.Error()))
				return nil, fmt.Errorf("create oauth account: %w", err)
			}
		case isUserNotFound(err):
			accountID, err = uc.createOAuthAccount(ctx, userInfo, input.Provider, oauthTokens)
			if err != nil {
				return nil, err
			}
			isNewUser = true
			ctx = sharedlogger.WithField(ctx, "account_id", accountID)
			slog.InfoContext(ctx, "oauth created new local user")
		default:
			slog.ErrorContext(ctx, "oauth local user lookup failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("find user by email: %w", err)
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

func (uc *oauthUseCase) LinkAccount(ctx context.Context, input OAuthLinkInput) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider":   string(input.Provider),
		"account_id": input.CurrentAccountID,
	})
	slog.InfoContext(ctx, "oauth link started")

	provider, ok := uc.oauthProviders[input.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth link rejected because provider is unsupported")
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
		slog.WarnContext(ctx, "oauth link rejected because provider account is already linked")
		return errors.New("this account is already linked")
	}

	existingAccounts, err := uc.oauthRepo.GetByAccountID(ctx, input.CurrentAccountID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth link current user linked accounts lookup failed", slog.String("error", err.Error()))
		return fmt.Errorf("get linked accounts: %w", err)
	}
	for _, acc := range existingAccounts {
		if acc.Provider == input.Provider {
			slog.WarnContext(ctx, "oauth link rejected because provider is already attached to current user")
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

func (uc *oauthUseCase) UnlinkAccount(ctx context.Context, userID string, provider domain.OAuthProvider) error {
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
		slog.WarnContext(ctx, "oauth unlink rejected because provider is not linked")
		return errors.New("provider not linked")
	}

	if err := uc.oauthRepo.Delete(ctx, targetAccount.ID); err != nil {
		slog.ErrorContext(ctx, "oauth unlink delete failed", slog.String("error", err.Error()))
		return err
	}

	slog.InfoContext(ctx, "oauth unlink completed")
	return nil
}

func (uc *oauthUseCase) GetLinkedAccounts(ctx context.Context, userID string) ([]*domain.OAuthAccount, error) {
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

func (uc *oauthUseCase) createOAuthAccount(
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
	slog.InfoContext(ctx, "oauth user creation started")

	account := domain.NewAccount(userInfo.Email, "")
	account.ID = uuid.NewString()

	accountID, err := uc.accountRepo.Save(ctx, *account)
	if err != nil {
		slog.ErrorContext(ctx, "oauth local user creation failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("create oauth user: %w", err)
	}

	ctx = sharedlogger.WithField(ctx, "account_id", accountID)

	oauthAccount := domain.NewOAuthAccount(accountID, provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = oauthTokens.AccessToken
	oauthAccount.RefreshToken = oauthTokens.RefreshToken
	oauthAccount.Expiry = oauthTokens.Expiry

	if err := uc.oauthRepo.Create(ctx, oauthAccount); err != nil {
		slog.ErrorContext(ctx, "oauth account creation for new user failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("create oauth account: %w", err)
	}

	slog.InfoContext(ctx, "oauth user creation completed")
	return accountID, nil
}

func (uc *oauthUseCase) createSessionForAccount(
	ctx context.Context,
	accountID, email string,
	requestData domain.RequestData,
) (*OAuthOutput, error) {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id": accountID,
		"email":      email,
	})
	slog.InfoContext(ctx, "oauth session creation started")

	sessionID := uuid.NewString()

	accessToken, err := uc.jwtSvc.GenerateAccessToken(sessionID, accountID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth jwt token generation failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := uc.jwtSvc.GenerateRefreshToken(sessionID)
	if err != nil {
		slog.ErrorContext(ctx, "oauth refresh token generation failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	session := domainsession.NewSession(
		sessionID,
		accountID,
		uc.jwtSvc.HashFingerprint(requestData.Fingerprint),
		requestData.IPAddress,
		requestData.UserAgent,
		uc.sessionTTL,
	)

	if err := uc.sessionRepo.CreateSession(ctx, session); err != nil {
		slog.ErrorContext(ctx, "oauth session persistence failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("create oauth session: %w", err)
	}

	_, randomPart, err := uc.jwtSvc.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	pair := &domainsession.RefreshPair{
		Current: uc.jwtSvc.HashRefreshToken(randomPart),
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
		ExpiresAt:        time.Now().UTC().Add(uc.jwtSvc.AccessTTL()),
		RefreshExpiresAt: time.Now().UTC().Add(uc.jwtSvc.RefreshTTL()),
	}, nil
}

func isUserNotFound(err error) bool {
	return errors.Is(err, postgresqlerrors.ErrNoRows)
}

var _ OAuthUseCase = (*oauthUseCase)(nil)
