package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

func (uc *Service) CreateUser(ctx context.Context, in CreateUserInput) (*UserOutput, error) {
	username, err := vo.NewUsername(in.Username)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	var emailVO *vo.Email
	if in.Email != nil {
		e, err := vo.NewEmail(*in.Email)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		emailVO = &e
	}

	var phoneVO *vo.Phone
	if in.Phone != nil {
		p, err := vo.NewPhone(*in.Phone)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
		}
		phoneVO = &p
	}

	// 2. ID: используем переданный или генерим новый
	var id uuid.UUID
	if in.ID != nil {
		id = *in.ID
	} else {
		id = uuid.New()
	}

	// 3. Создаём доменную сущность (инварианты проверятся внутри)
	u, err := domainuser.NewUser(id, username, emailVO, phoneVO, in.DisplayName)
	if err != nil {
		return nil, err
	}

	// 4. Сохраняем
	if err := uc.repo.Create(ctx, u); err != nil {
		// Пробрасываем доменные ошибки как есть — выше (handler) маппит в HTTP-коды
		if errors.Is(err, domainuser.ErrUsernameAlreadyTaken) ||
			errors.Is(err, domainuser.ErrEmailAlreadyTaken) ||
			errors.Is(err, domainuser.ErrPhoneAlreadyTaken) {
			return nil, err
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return toUserOutput(u), nil

}