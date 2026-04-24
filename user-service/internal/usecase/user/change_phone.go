package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (s *Service) ChangePhone(ctx context.Context, in ChangePhoneInput) (*UserOutput, error) {
	phoneVO, err := vo.NewPhone(in.Phone)
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

	existing, err := s.repo.GetByPhone(ctx, phoneVO)
	if err != nil && !errors.Is(err, domainuser.ErrUserNotFound) {
		return nil, fmt.Errorf("change phone: check existing: %w", err)
	}
	if existing != nil && existing.ID() != u.ID() {
		return nil, domainuser.ErrPhoneAlreadyTaken
	}

	u.ChangePhone(phoneVO)

	if err := s.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrVersionConflict) ||
			errors.Is(err, domainuser.ErrPhoneAlreadyTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("change phone: %w", err)
	}
	return toUserOutput(u), nil
}