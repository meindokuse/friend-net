package get_stats

import (
	"context"
	"time"
)

type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     uint64 `json:"count"`
}

type ServiceCount struct {
	Service string `json:"service"`
	Count   uint64 `json:"count"`
}

type Output struct {
	Total       uint64           `json:"total"`
	ByEventType []EventTypeCount `json:"by_event_type"`
	ByService   []ServiceCount   `json:"by_service"`
	From        *time.Time       `json:"from,omitempty"`
	To          *time.Time       `json:"to,omitempty"`
}

type Repository interface {
	GetStats(ctx context.Context, from, to *time.Time) (*Output, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type Input struct {
	From *time.Time
	To   *time.Time
}

func (s *Service) Execute(ctx context.Context, in Input) (*Output, error) {
	out, err := s.repo.GetStats(ctx, in.From, in.To)
	if err != nil {
		return nil, err
	}
	out.From = in.From
	out.To = in.To
	return out, nil
}
