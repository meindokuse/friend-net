package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/terror"
)

// Storage implements account repository
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage creates a new account storage
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// CreateWithOutbox creates account and outbox event in transaction
func (s *Storage) CreateWithOutbox(ctx context.Context, account *entity.Account, outbox *entity.OutboxEvent) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert account
	const insertAccountQuery = `
		INSERT INTO accounts (
			id, email, password_hash, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, insertAccountQuery,
		account.ID,
		account.Email,
		account.PasswordHash,
		account.IsActive,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return terror.NewConflictErr("account already exists", err)
		}
		return fmt.Errorf("insert account: %w", err)
	}

	// Insert outbox event
	const insertOutboxQuery = `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, insertOutboxQuery,
		outbox.ID,
		outbox.AggregateType,
		outbox.AggregateID,
		outbox.EventType,
		outbox.Payload,
		outbox.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// FindByEmail finds account by email
func (s *Storage) FindByEmail(ctx context.Context, email string) (*entity.Account, error) {
	const query = `
		SELECT
			id, email, password_hash, is_active,
			created_at, updated_at, last_login_at
		FROM accounts
		WHERE email = $1
		LIMIT 1
	`

	var account entity.Account
	var lastLoginAt *time.Time

	err := s.pool.QueryRow(ctx, query, email).Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, terror.NewNotFoundErr("account not found", err)
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	account.LastLoginAt = lastLoginAt
	return &account, nil
}

// FindByID finds account by ID
func (s *Storage) FindByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	const query = `
		SELECT
			id, email, password_hash, is_active,
			created_at, updated_at, last_login_at
		FROM accounts
		WHERE id = $1
		LIMIT 1
	`

	var account entity.Account
	var lastLoginAt *time.Time

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&lastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, terror.NewNotFoundErr("account not found", err)
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	account.LastLoginAt = lastLoginAt
	return &account, nil
}
