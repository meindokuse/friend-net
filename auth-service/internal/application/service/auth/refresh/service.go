package refresh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// SessionManager interface for session operations
type SessionManager interface {
	Get(ctx context.Context, sessionID string) (*entity.Session, error)
	SaveRefreshPair(ctx context.Context, sessionID string, pair *entity.RefreshPair) error
	GetRefreshPair(ctx context.Context, sessionID string) (*entity.RefreshPair, error)
	UpdateLastSeen(ctx context.Context, sessionID string) error
	Revoke(ctx context.Context, sessionID, accountID string) error
}

// Service handles refresh use case
type Service struct {
	sessions SessionManager
	jwt      *jwt.Manager
}

// NewService creates a new refresh service
func NewService(
	sessions SessionManager,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		sessions: sessions,
		jwt:      jwtManager,
	}
}

// DTO for refresh input
type RefreshDTO struct {
	RefreshToken string
	Fingerprint  string
}

// Result contains refresh result
type Result struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresIn        int64
	ExpiresAt        string
	RefreshExpiresAt string
	AccountID        string
}

// Refresh validates refresh token and creates new tokens
func (s *Service) Refresh(ctx context.Context, dto RefreshDTO) (*Result, error) {
	slog.DebugContext(ctx, "refresh: attempt")

	sessionID, randomPart, err := s.jwt.ParseRefreshToken(dto.RefreshToken)
	if err != nil {
		slog.WarnContext(ctx, "refresh: invalid token format", "error", err)
		return nil, terror.NewUnauthorizedErr("invalid refresh token", err)
	}

	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		slog.WarnContext(ctx, "refresh: session not found", "session_id", sessionID)
		return nil, terror.NewNotFoundErr("session not found", err)
	}

	if !session.IsActive() {
		slog.WarnContext(ctx, "refresh: session is revoked", "session_id", sessionID)
		return nil, terror.NewUnauthorizedErr("session revoked", nil)
	}

	fingerprintHash := s.jwt.HashFingerprint(dto.Fingerprint)
	if session.FingerprintHash != fingerprintHash {
		slog.WarnContext(ctx, "refresh: fingerprint mismatch",
			"session_id", sessionID, "account_id", session.AccountID)
		return nil, terror.NewUnauthorizedErr("fingerprint mismatch", nil)
	}

	pair, err := s.sessions.GetRefreshPair(ctx, sessionID)
	if err != nil {
		slog.ErrorContext(ctx, "refresh: get refresh pair failed", "session_id", sessionID, "error", err)
		return nil, terror.NewNotFoundErr("refresh pair not found", err)
	}

	incomingHash := s.jwt.HashRefreshToken(randomPart)

	switch pair.Match(incomingHash) {
	case entity.RefreshMatchCurrent:
		slog.DebugContext(ctx, "refresh: rotating tokens", "session_id", sessionID)
		result, err := s.rotateTokens(ctx, session, pair)
		if err != nil {
			slog.ErrorContext(ctx, "refresh: rotate tokens failed", "session_id", sessionID, "error", err)
			return nil, err
		}
		slog.InfoContext(ctx, "refresh: tokens rotated",
			"session_id", sessionID, "account_id", session.AccountID)
		return result, nil

	case entity.RefreshMatchPrev:
		slog.DebugContext(ctx, "refresh: grace period rotation", "session_id", sessionID)
		result, err := s.rotateTokensGrace(ctx, session, pair)
		if err != nil {
			slog.ErrorContext(ctx, "refresh: grace rotation failed", "session_id", sessionID, "error", err)
			return nil, err
		}
		slog.InfoContext(ctx, "refresh: grace rotation success",
			"session_id", sessionID, "account_id", session.AccountID)
		return result, nil

	case entity.RefreshMatchNone:
		slog.WarnContext(ctx, "refresh: token reuse detected — revoking session",
			"session_id", sessionID, "account_id", session.AccountID)
		_ = s.sessions.Revoke(ctx, sessionID, session.AccountID)
		return nil, terror.NewUnauthorizedErr("token reuse detected", nil)
	}

	return nil, terror.NewUnauthorizedErr("invalid token", nil)
}

func (s *Service) rotateTokens(ctx context.Context, session *entity.Session, pair *entity.RefreshPair) (*Result, error) {
	// Generate new refresh token
	newRefreshToken, err := s.jwt.GenerateRefreshToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh: %w", err)
	}

	_, newRandomPart, _ := s.jwt.ParseRefreshToken(newRefreshToken)
	newHash := s.jwt.HashRefreshToken(newRandomPart)

	// Rotation: current -> prev, new -> current
	pair.Rotate(newHash, s.jwt.GracePeriod())

	if err := s.sessions.SaveRefreshPair(ctx, session.ID, pair); err != nil {
		return nil, fmt.Errorf("save pair: %w", err)
	}

	_ = s.sessions.UpdateLastSeen(ctx, session.ID)

	// Generate new access token
	accessToken, err := s.jwt.GenerateAccessToken(session.ID, session.AccountID)
	if err != nil {
		return nil, fmt.Errorf("generate access: %w", err)
	}

	return s.buildResult(accessToken, newRefreshToken, session.AccountID), nil
}

func (s *Service) rotateTokensGrace(ctx context.Context, session *entity.Session, pair *entity.RefreshPair) (*Result, error) {
	// Grace period: rotate but don't overwrite prev
	newRefreshToken, err := s.jwt.GenerateRefreshToken(session.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh: %w", err)
	}

	_, newRandomPart, _ := s.jwt.ParseRefreshToken(newRefreshToken)
	newHash := s.jwt.HashRefreshToken(newRandomPart)

	// Only update current, keep prev for potential retry
	pair.SetCurrent(newHash)

	if err := s.sessions.SaveRefreshPair(ctx, session.ID, pair); err != nil {
		return nil, fmt.Errorf("save pair: %w", err)
	}

	_ = s.sessions.UpdateLastSeen(ctx, session.ID)

	accessToken, err := s.jwt.GenerateAccessToken(session.ID, session.AccountID)
	if err != nil {
		return nil, fmt.Errorf("generate access: %w", err)
	}

	return s.buildResult(accessToken, newRefreshToken, session.AccountID), nil
}

func (s *Service) buildResult(accessToken, refreshToken, accountID string) *Result {
	return &Result{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(s.jwt.AccessTTL().Seconds()),
		ExpiresAt:        s.jwt.ExpiresAt(),
		RefreshExpiresAt: s.jwt.RefreshExpiresAt(),
		AccountID:        accountID,
	}
}

// Error definitions
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionRevoked  = errors.New("session revoked")
	ErrTokenReuse      = errors.New("token reuse detected")
)
