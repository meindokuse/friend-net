package get_user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByUsername(ctx context.Context, username vo.Username) (*entity.User, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	slog.DebugContext(ctx, "get_user.ByID", "user_id", id)
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "get_user.ByID: failed", "error", err, "user_id", id)
		return nil, err
	}
	return u, nil
}

func (s *Service) ByUsername(ctx context.Context, raw string) (*entity.User, error) {
	slog.DebugContext(ctx, "get_user.ByUsername", "username", raw)
	uName, err := vo.NewUsername(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	u, err := s.repo.GetByUsername(ctx, uName)
	if err != nil {
		slog.ErrorContext(ctx, "get_user.ByUsername: failed", "error", err, "username", raw)
		return nil, err
	}
	return u, nil
}

func (s *Service) ByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	slog.DebugContext(ctx, "get_user.ByIDs", "count", len(ids))
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}
	if len(ids) > 500 {
		return nil, fmt.Errorf("%w: batch size exceeds 500", apperr.ErrInvalidInput)
	}
	users, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		slog.ErrorContext(ctx, "get_user.ByIDs: failed", "error", err, "count", len(ids))
		return nil, err
	}
	return users, nil
}
