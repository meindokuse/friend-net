package user

import (
	"context"
	"fmt"
	"strings"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 100
)

func (s *Service) SearchUsers(ctx context.Context, in SearchUsersInput) ([]*PublicUserOutput, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, fmt.Errorf("%w: empty query", ErrInvalidInput)
	}

	limit := in.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	users, err := s.repo.Search(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}

	return toPublicUserOutputs(users), nil
}