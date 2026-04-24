package user

import (
	"context"
	"errors"
	"fmt"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (s *Service) UpdateSettings(ctx context.Context, in UpdateSettingsInput) (*UserOutput, error) {
	u, err := s.repo.GetByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if u.Version() != in.Version {
		return nil, domainuser.ErrVersionConflict
	}

	settings := domainuser.Settings{
		Privacy: domainuser.PrivacySettings{
			WhoCanMessage:     domainuser.PrivacyLevel(in.WhoCanMessage),
			WhoCanSeeLastSeen: domainuser.PrivacyLevel(in.WhoCanSeeLastSeen),
			WhoCanSeeProfile:  domainuser.PrivacyLevel(in.WhoCanSeeProfile),
		},
		Language: in.Language,
		Timezone: in.Timezone,
	}

	if err := u.UpdateSettings(settings); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domainuser.ErrVersionConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("update settings: %w", err)
	}

	return toUserOutput(u), nil
}