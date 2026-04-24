package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// UpdateLastSeen — быстрый частичный апдейт (без чтения всего документа).
func (s *Service) UpdateLastSeen(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.UpdateLastSeen(ctx, userID); err != nil {
		return fmt.Errorf("update last seen: %w", err)
	}
	return nil
}