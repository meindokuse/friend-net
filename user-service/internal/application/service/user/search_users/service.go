package search_users

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/pkg/apperr"
)

type Repository interface {
	Search(ctx context.Context, query string, limit, offset int) ([]*entity.User, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Input struct {
	Query  string
	Limit  int
	Offset int
}

func (s *Service) Execute(ctx context.Context, in Input) ([]*entity.User, error) {
	slog.DebugContext(ctx, "search_users.Execute", "query", in.Query, "limit", in.Limit, "offset", in.Offset)

	q := strings.TrimSpace(in.Query)
	if q == "" {
		return nil, fmt.Errorf("%w: empty query", apperr.ErrInvalidInput)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}
	users, err := s.repo.Search(ctx, q, limit, offset)
	if err != nil {
		slog.ErrorContext(ctx, "search_users.Execute: Search failed", "error", err, "query", q)
		return nil, err
	}
	return users, nil
}
