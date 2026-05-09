package delete_user

import (
	"context"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Update(ctx context.Context, u *entity.User) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Input struct {
	UserID  uuid.UUID
	Version int
}

func (s *Service) Execute(ctx context.Context, in Input) error {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if u.Version() != in.Version {
		return entity.ErrVersionConflict
	}
	if err := u.SoftDelete(); err != nil {
		return err
	}
	return s.repo.Update(ctx, u)
}
