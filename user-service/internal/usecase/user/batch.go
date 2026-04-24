package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const maxBatchSize = 500

// GetUsersByIDs — для inter-service вызовов (chat-service просит обогатить участников).
// Возвращает публичные профили.
func (s *Service) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*PublicUserOutput, error) {
	if len(ids) == 0 {
		return []*PublicUserOutput{}, nil
	}
	if len(ids) > maxBatchSize {
		return nil, fmt.Errorf("%w: batch size exceeds %d", ErrInvalidInput, maxBatchSize)
	}

	users, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}

	return toPublicUserOutputs(users), nil
}