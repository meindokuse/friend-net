package get_user

import (
	"context"
	"fmt"

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
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ByUsername(ctx context.Context, raw string) (*entity.User, error) {
	uName, err := vo.NewUsername(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	return s.repo.GetByUsername(ctx, uName)
}

func (s *Service) ByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}
	if len(ids) > 500 {
		return nil, fmt.Errorf("%w: batch size exceeds 500", apperr.ErrInvalidInput)
	}
	return s.repo.GetByIDs(ctx, ids)
}
