package change_phone

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByPhone(ctx context.Context, phone vo.Phone) (*entity.User, error)
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
	Phone   string
	Version int
}

func (s *Service) Execute(ctx context.Context, in Input) (*entity.User, error) {
	p, err := vo.NewPhone(in.Phone)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	existing, err := s.repo.GetByPhone(ctx, p)
	if err != nil && !errors.Is(err, entity.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, entity.ErrPhoneAlreadyTaken
	}
	u.ChangePhone(p)
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
