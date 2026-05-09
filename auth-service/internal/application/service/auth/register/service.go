package register

import (
	"context"
	"fmt"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/events/account_created"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// AccountCreator interface for account creation
type AccountCreator interface {
	CreateWithOutbox(ctx context.Context, account *entity.Account, outbox *entity.OutboxEvent) error
	FindByEmail(ctx context.Context, email string) (*entity.Account, error)
}

// Service handles register use case
type Service struct {
	accounts AccountCreator
	hasher   *pass.Hasher
}

// NewService creates a new register service
func NewService(
	accounts AccountCreator,
	hasher *pass.Hasher,
) *Service {
	return &Service{
		accounts: accounts,
		hasher:   hasher,
	}
}

// DTO for register input
type RegisterDTO struct {
	Email       string
	Password    string
	DisplayName string
}

// Register creates a new account
func (s *Service) Register(ctx context.Context, dto RegisterDTO) (string, error) {
	existing, err := s.accounts.FindByEmail(ctx, dto.Email)
	if err == nil && existing != nil {
		return "", terror.NewConflictErr("account already exists", nil)
	}

	passwordHash, err := s.hasher.Hash(dto.Password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	account, err := entity.NewAccount(dto.Email, passwordHash)
	if err != nil {
		return "", fmt.Errorf("create account: %w", err)
	}

	outboxEvent, err := account_created.New(
		account.ID,
		account.Email,
		dto.DisplayName,
		account.CreatedAt,
	)
	if err != nil {
		return "", fmt.Errorf("create outbox event: %w", err)
	}

	if err := s.accounts.CreateWithOutbox(ctx, account, outboxEvent); err != nil {
		return "", err
	}

	return account.ID.String(), nil
}
