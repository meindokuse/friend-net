package user

import (
	"context"
	"errors"
	"fmt"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	// Fast-fail до обращения к БД
	if u.Version() != in.Version {
		return nil, domainuser.ErrVersionConflict
	}

	if err := u.UpdateProfile(in.DisplayName, in.Bio, in.AvatarURL); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrVersionConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return toUserOutput(u), nil
}
