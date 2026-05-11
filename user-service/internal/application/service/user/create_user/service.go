package create_user

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
	Create(ctx context.Context, u *entity.User) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Input struct {
	ID          *uuid.UUID
	Username    string
	Email       *string
	Phone       *string
	DisplayName string
}

func (s *Service) Execute(ctx context.Context, in Input) (*entity.User, error) {
	slog.DebugContext(ctx, "create_user.Execute",
		"username", in.Username,
		"has_email", in.Email != nil,
		"has_phone", in.Phone != nil,
		"explicit_id", in.ID != nil,
	)

	username, err := vo.NewUsername(in.Username)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
	}
	var emailVO *vo.Email
	if in.Email != nil {
		e, err := vo.NewEmail(*in.Email)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
		}
		emailVO = &e
	}
	var phoneVO *vo.Phone
	if in.Phone != nil {
		p, err := vo.NewPhone(*in.Phone)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", apperr.ErrInvalidInput, err)
		}
		phoneVO = &p
	}
	id := uuid.New()
	if in.ID != nil {
		id = *in.ID
	}
	u, err := entity.NewUser(id, username, emailVO, phoneVO, in.DisplayName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, u); err != nil {
		slog.ErrorContext(ctx, "create_user.Execute: repo.Create failed",
			"error", err, "username", in.Username)
		return nil, err
	}
	return u, nil
}
