package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (s *Service) ChangeEmail(ctx context.Context, in ChangeEmailInput) (*UserOutput, error) {
	emailVO, err := vo.NewEmail(in.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, domainuser.ErrVersionConflict
	}

	// Проверка уникальности: email не должен принадлежать другому юзеру
	existing, err := s.repo.GetByEmail(ctx, emailVO)
	if err != nil && !errors.Is(err, domainuser.ErrUserNotFound) {
		return nil, fmt.Errorf("change email: check existing: %w", err)
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, domainuser.ErrEmailAlreadyTaken
	}

	u.ChangeEmail(emailVO)

	if err := s.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrVersionConflict) ||
			errors.Is(err, domainuser.ErrEmailAlreadyTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("change email: %w", err)
	}
	return toUserOutput(u), nil
}