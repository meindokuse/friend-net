package delete_event

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Execute(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
