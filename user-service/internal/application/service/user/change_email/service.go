package change_email

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email vo.Email) (*entity.User, error)
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
	Email   string
	Version int
}

func (s *Service) Execute(ctx context.Context, in Input) (*entity.User, error) {
	slog.DebugContext(ctx, "change_email.Execute", "user_id", in.UserID, "version", in.Version)

	e, err := vo.NewEmail(in.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "change_email.Execute: GetByID failed", "error", err, "user_id", in.UserID)
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, entity.ErrVersionConflict
	}
	existing, err := s.repo.GetByEmail(ctx, e)
	if err != nil && !errors.Is(err, entity.ErrUserNotFound) {
		slog.ErrorContext(ctx, "change_email.Execute: GetByEmail failed", "error", err, "user_id", in.UserID)
		return nil, err
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, entity.ErrEmailAlreadyTaken
	}
	u.ChangeEmail(e)
	if err := s.repo.Update(ctx, u); err != nil {
		slog.ErrorContext(ctx, "change_email.Execute: Update failed", "error", err, "user_id", in.UserID)
		return nil, err
	}
	return u, nil
}
