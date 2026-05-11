package login

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/providers"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/events/account_created"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// AuthRepository interface for account operations
type AuthRepository interface {
	FindByEmail(ctx context.Context, email string) (*entity.Account, error)
	CreateWithOutbox(ctx context.Context, account *entity.Account, outbox *entity.OutboxEvent) error
}

// OAuthRepository interface for OAuth operations
type OAuthRepository interface {
	GetByProviderID(ctx context.Context, provider entity.OAuthProvider, providerID string) (*entity.OAuthAccount, error)
	Create(ctx context.Context, account *entity.OAuthAccount) error
	UpdateTokens(ctx context.Context, id string, accessToken, refreshToken string, expiry int64) error
}

// SessionManager interface for session operations
type SessionManager interface {
	Create(ctx context.Context, session *entity.Session) error
	SaveRefreshPair(ctx context.Context, sessionID string, pair *entity.RefreshPair) error
}

// Service handles OAuth login
type Service struct {
	accounts  AuthRepository
	oauth     OAuthRepository
	sessions  SessionManager
	providers map[entity.OAuthProvider]providers.OAuthProviderGateway
	jwt       *jwt.Manager
	ttl       int64
}

// NewService creates a new OAuth login service
func NewService(
	accounts AuthRepository,
	oauth OAuthRepository,
	sessions SessionManager,
	providers map[entity.OAuthProvider]providers.OAuthProviderGateway,
	jwtManager *jwt.Manager,
	ttl int64,
) *Service {
	return &Service{
		accounts:  accounts,
		oauth:     oauth,
		sessions:  sessions,
		providers: providers,
		jwt:       jwtManager,
		ttl:       ttl,
	}
}

// DTO for OAuth login input
type LoginDTO struct {
	Provider    entity.OAuthProvider
	Code        string
	State       string
	Fingerprint string
	IP          string
	UserAgent   string
}

// Result contains OAuth login result
type Result struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresIn        int64
	ExpiresAt        string
	RefreshExpiresAt string
	AccountID        string
	IsNewUser        bool
}

// Login handles OAuth login flow
func (s *Service) Login(ctx context.Context, dto LoginDTO) (*Result, error) {
	slog.DebugContext(ctx, "oauth-login: attempt", "provider", dto.Provider, "ip", dto.IP)

	provider, ok := s.providers[dto.Provider]
	if !ok {
		slog.WarnContext(ctx, "oauth-login: unsupported provider", "provider", dto.Provider)
		return nil, terror.NewBadRequestErr("unsupported provider", nil)
	}

	oauthToken, err := provider.ExchangeToken(ctx, dto.Code)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-login: token exchange failed",
			"provider", dto.Provider, "error", err)
		return nil, fmt.Errorf("exchange token: %w", err)
	}

	userInfo, err := provider.GetUserInfo(ctx, oauthToken.AccessToken)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-login: get user info failed",
			"provider", dto.Provider, "error", err)
		return nil, fmt.Errorf("get user info: %w", err)
	}

	existingOAuth, err := s.oauth.GetByProviderID(ctx, dto.Provider, userInfo.ProviderID)
	if err != nil {
		return nil, err
	}

	var accountID string
	isNewUser := false

	if existingOAuth != nil {
		accountID = existingOAuth.AccountID
		slog.DebugContext(ctx, "oauth-login: existing oauth account, updating tokens",
			"account_id", accountID, "provider", dto.Provider)
		if err := s.oauth.UpdateTokens(ctx, existingOAuth.ID, oauthToken.AccessToken, oauthToken.RefreshToken, oauthToken.Expiry); err != nil {
			slog.ErrorContext(ctx, "oauth-login: update tokens failed",
				"account_id", accountID, "error", err)
			return nil, err
		}
	} else {
		existingAccount, err := s.accounts.FindByEmail(ctx, userInfo.Email)
		if err == nil && existingAccount != nil {
			accountID = existingAccount.ID.String()
			slog.DebugContext(ctx, "oauth-login: linking to existing account by email",
				"account_id", accountID, "provider", dto.Provider)
			if err := s.createOAuthLink(ctx, accountID, dto.Provider, userInfo, oauthToken); err != nil {
				slog.ErrorContext(ctx, "oauth-login: create oauth link failed",
					"account_id", accountID, "error", err)
				return nil, err
			}
		} else {
			slog.DebugContext(ctx, "oauth-login: creating new account", "provider", dto.Provider)
			accountID, err = s.createOAuthAccount(ctx, userInfo, dto.Provider, oauthToken)
			if err != nil {
				slog.ErrorContext(ctx, "oauth-login: create account failed",
					"provider", dto.Provider, "error", err)
				return nil, err
			}
			isNewUser = true
		}
	}

	result, err := s.createSession(ctx, accountID, dto.Fingerprint, dto.IP, dto.UserAgent, isNewUser)
	if err != nil {
		slog.ErrorContext(ctx, "oauth-login: create session failed",
			"account_id", accountID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "oauth-login: success",
		"account_id", accountID, "provider", dto.Provider, "is_new_user", isNewUser)
	return result, nil
}

func (s *Service) createOAuthLink(
	ctx context.Context,
	accountID string,
	provider entity.OAuthProvider,
	userInfo *providers.OAuthUserInfo,
	token *providers.OAuthToken,
) error {
	oauthAccount := entity.NewOAuthAccount(accountID, provider, userInfo.ProviderID, userInfo.Email)
	oauthAccount.AccessToken = token.AccessToken
	oauthAccount.RefreshToken = token.RefreshToken
	oauthAccount.Expiry = s.jwt.Now()

	return s.oauth.Create(ctx, oauthAccount)
}

func (s *Service) createOAuthAccount(
	ctx context.Context,
	userInfo *providers.OAuthUserInfo,
	provider entity.OAuthProvider,
	token *providers.OAuthToken,
) (string, error) {
	// Create account (no password for OAuth)
	account, err := entity.NewAccount(userInfo.Email, "")
	if err != nil {
		return "", err
	}

	// Create outbox event
	outboxEvent, err := account_created.New(
		account.ID,
		account.Email,
		userInfo.Name,
		account.CreatedAt,
	)
	if err != nil {
		return "", err
	}

	// Save account with outbox
	if err := s.accounts.CreateWithOutbox(ctx, account, outboxEvent); err != nil {
		return "", err
	}

	// Create OAuth link
	accountID := account.ID.String()
	if err := s.createOAuthLink(ctx, accountID, provider, userInfo, token); err != nil {
		return "", err
	}

	return accountID, nil
}

func (s *Service) createSession(
	ctx context.Context,
	accountID, fingerprint, ip, ua string,
	isNewUser bool,
) (*Result, error) {
	sessionID := uuid.NewString()
	fingerprintHash := s.jwt.HashFingerprint(fingerprint)

	// Create session
	session := entity.NewSession(sessionID, accountID, fingerprintHash, ip, ua, s.jwt.RefreshTTL())
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Generate tokens
	accessToken, err := s.jwt.GenerateAccessToken(sessionID, accountID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := s.jwt.GenerateRefreshToken(sessionID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Save refresh pair
	_, randomPart, _ := s.jwt.ParseRefreshToken(refreshToken)
	pair := &entity.RefreshPair{
		Current: s.jwt.HashRefreshToken(randomPart),
	}
	if err := s.sessions.SaveRefreshPair(ctx, sessionID, pair); err != nil {
		return nil, fmt.Errorf("save refresh pair: %w", err)
	}

	return &Result{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(s.jwt.AccessTTL().Seconds()),
		ExpiresAt:        session.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		RefreshExpiresAt: session.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		AccountID:        accountID,
		IsNewUser:        isNewUser,
	}, nil
}
