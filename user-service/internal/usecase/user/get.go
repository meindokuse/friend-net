package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
)

// GetUserByID — полный профиль (для /users/me).
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUserOutput(u), nil
}

// GetPublicUserByID — публичный профиль (для просмотра чужого).
func (s *Service) GetPublicUserByID(ctx context.Context, id uuid.UUID) (*PublicUserOutput, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutput(u), nil
}

// GetPublicUserByUsername — публичный профиль по username.
func (s *Service) GetPublicUserByUsername(ctx context.Context, rawUsername string) (*PublicUserOutput, error) {
	username, err := vo.NewUsername(rawUsername)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return toPublicUserOutput(u), nil
}