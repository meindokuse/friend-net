package update_profile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
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
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	Version     int
}

func (s *Service) Execute(ctx context.Context, in Input) (*entity.User, error) {
	slog.DebugContext(ctx, "update_profile.Execute", "user_id", in.UserID, "version", in.Version)

	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "update_profile.Execute: GetByID failed", "error", err, "user_id", in.UserID)
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	if err := u.UpdateProfile(in.DisplayName, in.Bio, in.AvatarURL); err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	if err := s.repo.Update(ctx, u); err != nil {
		slog.ErrorContext(ctx, "update_profile.Execute: Update failed", "error", err, "user_id", in.UserID)
		return nil, err
	}
	return u, nil
}
