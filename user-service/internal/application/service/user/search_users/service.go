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
	Search(ctx context.Context, params entity.SearchParams) ([]*entity.User, entity.PagedSearchUsers, error)
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
	Cursor *entity.SearchCursor
}

func (s *Service) Execute(ctx context.Context, in Input) (entity.PagedSearchUsers, error) {
	slog.DebugContext(ctx, "search_users.Execute", "query", in.Query, "limit", in.Limit)

	q := strings.TrimSpace(in.Query)
	if q == "" {
		return entity.PagedSearchUsers{}, fmt.Errorf("%w: empty query", apperr.ErrInvalidInput)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	_, paged, err := s.repo.Search(ctx, entity.SearchParams{
		Query:  q,
		Limit:  limit,
		Cursor: in.Cursor,
	})
	if err != nil {
		slog.ErrorContext(ctx, "search_users.Execute: Search failed", "error", err, "query", q)
		return entity.PagedSearchUsers{}, err
	}
	return paged, nil
}
