package list_users

import (
	"context"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
)

type Repository interface {
	List(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Execute(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error) {
	return s.repo.List(ctx, params)
}
