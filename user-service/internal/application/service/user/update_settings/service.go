package update_settings

import (
	"context"
	"fmt"

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
	UserID            uuid.UUID
	WhoCanMessage     string
	WhoCanSeeLastSeen string
	WhoCanSeeProfile  string
	Language          string
	Timezone          string
	Version           int
}

func (s *Service) Execute(ctx context.Context, in Input) (*entity.User, error) {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	settings := entity.Settings{
		Privacy: entity.PrivacySettings{
			WhoCanMessage:     entity.PrivacyLevel(in.WhoCanMessage),
			WhoCanSeeLastSeen: entity.PrivacyLevel(in.WhoCanSeeLastSeen),
			WhoCanSeeProfile:  entity.PrivacyLevel(in.WhoCanSeeProfile),
		},
		Language: in.Language,
		Timezone: in.Timezone,
	}
	if err := u.UpdateSettings(settings); err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
