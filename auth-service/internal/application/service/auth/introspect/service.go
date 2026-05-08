package introspect

import (
	"context"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// SessionChecker interface for session blacklist check
type SessionChecker interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// Service handles token introspection
type Service struct {
	sessions SessionChecker
	jwt      *jwt.Manager
}

// NewService creates a new introspect service
func NewService(
	sessions SessionChecker,
	jwtManager *jwt.Manager,
) *Service {
	return &Service{
		sessions: sessions,
		jwt:      jwtManager,
	}
}

// Result contains introspection result
type Result struct {
	Active    bool
	AccountID string
	SessionID string
	ExpiresAt int64
}

// Introspect validates an access token
func (s *Service) Introspect(ctx context.Context, accessToken string) (*Result, error) {
	claims, err := s.jwt.VerifyAccessToken(accessToken)
	if err != nil {
		return &Result{Active: false}, nil
	}

	// Check if blacklisted
	blacklisted, err := s.sessions.IsBlacklisted(ctx, claims.ID)
	if err != nil {
		return nil, terror.NewInternalErr("check blacklist", err)
	}

	if blacklisted {
		return &Result{Active: false}, nil
	}

	return &Result{
		Active:    true,
		AccountID: claims.Subject,
		SessionID: claims.SessionID,
		ExpiresAt: claims.ExpiresAt.Unix(),
	}, nil
}
