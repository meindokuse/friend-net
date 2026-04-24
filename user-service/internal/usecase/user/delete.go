package user

import (
	"context"
	"errors"
	"fmt"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (s *Service) DeleteUser(ctx context.Context, in DeleteUserInput) error {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return err
	}
	if u.Version() != in.Version {
		return domainuser.ErrVersionConflict
	}

	if err := u.SoftDelete(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrVersionConflict) {
			return err
		}
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}