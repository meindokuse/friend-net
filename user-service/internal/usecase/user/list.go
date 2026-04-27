package user

import (
	"context"
	"fmt"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// List возвращает страницу пользователей с keyset-пагинацией.
func (s *Service) List(ctx context.Context, params domainuser.ListParams) ([]*domainuser.User, domainuser.PagedUsers, error) {
	users, paged, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, domainuser.PagedUsers{}, fmt.Errorf("list users: %w", err)
	}
	return users, paged, nil
}
