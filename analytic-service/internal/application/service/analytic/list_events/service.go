package list_events

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/analytic-service/internal/domain/entity"
)

type Filter struct {
	EventType *string
	Service   *string
	UserID    *uuid.UUID
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type Output struct {
	Events []*entity.Event
	Total  int64
}

type Repository interface {
	List(ctx context.Context, f Filter) ([]*entity.Event, int64, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Execute(ctx context.Context, in Filter) (*Output, error) {
	if in.Limit <= 0 || in.Limit > 1000 {
		in.Limit = 50
	}
	if in.Offset < 0 {
		in.Offset = 0
	}

	events, total, err := s.repo.List(ctx, in)
	if err != nil {
		return nil, err
	}
	return &Output{Events: events, Total: total}, nil
}
