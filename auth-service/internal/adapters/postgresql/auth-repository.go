package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

// AccountRepository реализует работу с accounts через pgx pool.
type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAccountRepo создаёт новый репозиторий аккаунтов.
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// SaveWithOutbox сохраняет Account и OutboxEvent в одной транзакции.
func (r *AuthRepository) SaveWithOutbox(ctx context.Context, account *domain.Account, outboxEvent *outbox.OutboxEvent) (uuid.UUID, error) {
	ctx = sharedlogger.WithField(ctx, "email", account.Email)

	// Начинаем транзакцию
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to begin transaction", slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%w: begin tx: %v", ErrDatabaseWrite, err)
	}
	defer tx.Rollback(ctx)

	// 1. INSERT account
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
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			slog.WarnContext(ctx, "postgres save account conflict", slog.String("error", err.Error()))
			return uuid.Nil, fmt.Errorf("%w: %v", ErrUserAlreadyExists, err)
		}

		slog.ErrorContext(ctx, "postgres save account failed", slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	// 2. INSERT outbox event
	const insertOutboxQuery = `
		INSERT INTO outbox_events (
			id, aggregate_type, aggregate_id, event_type, payload, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, insertOutboxQuery,
		outboxEvent.ID,
		outboxEvent.AggregateType,
		outboxEvent.AggregateID,
		outboxEvent.EventType,
		outboxEvent.Payload,
		outboxEvent.CreatedAt,
	)
	if err != nil {
		slog.ErrorContext(ctx, "postgres save outbox event failed", slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%w: failed to save outbox event: %v", ErrDatabaseWrite, err)
	}

	// 3. Коммит транзакции
	if err = tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit transaction failed", slog.String("error", err.Error()))
		return uuid.Nil, fmt.Errorf("%w: commit tx: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(
		sharedlogger.WithField(ctx, "account_id", account.ID),
		"account and outbox event saved in transaction",
		slog.String("outbox_event_id", outboxEvent.ID.String()),
	)

	return account.ID, nil
}

// FindAccount находит аккаунт по email.
func (r *AuthRepository) FindAccount(ctx context.Context, loginData domain.Login) (*domain.Account, error) {
	ctx = sharedlogger.WithField(ctx, "email", loginData.Email)

	const query = `
		SELECT
			id, email, password_hash, is_active,
			created_at, updated_at, last_login_at
		FROM accounts
		WHERE email = $1
		LIMIT 1
	`

	var account domain.Account
	err := r.pool.QueryRow(ctx, query, loginData.Email).Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.WarnContext(ctx, "postgres account not found by email")
			return nil, fmt.Errorf("%w: account with email %s not found", ErrNoRows, loginData.Email)
		}

		slog.ErrorContext(ctx, "postgres find account by email failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "account_id", account.ID), "postgres account loaded by email")
	return &account, nil
}

// FindAccountByID находит аккаунт по ID.
func (r *AuthRepository) FindAccountByID(ctx context.Context, accountID uuid.UUID) (*domain.Account, error) {
	ctx = sharedlogger.WithField(ctx, "account_id", accountID)

	const query = `
		SELECT
			id, email, password_hash, is_active,
			created_at, updated_at, last_login_at
		FROM accounts
		WHERE id = $1
		LIMIT 1
	`

	var account domain.Account
	err := r.pool.QueryRow(ctx, query, accountID).Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.LastLoginAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.WarnContext(ctx, "postgres account not found by id")
			return nil, fmt.Errorf("%w: account with id %s not found", ErrNoRows, accountID)
		}

		slog.ErrorContext(ctx, "postgres find account by id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "email", account.Email), "postgres account loaded by id")
	return &account, nil
}
