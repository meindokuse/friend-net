package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepo(db *sql.DB) *AccountRepository {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return &AccountRepository{db: db}
}

func (r *AccountRepository) Save(ctx context.Context, accountData domain.Account) (string, error) {
	ctx = sharedlogger.WithField(ctx, "email", accountData.Email)

	const query = `
		INSERT INTO accounts (
			id,
			email,
			password_hash,
			is_active,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	if accountData.ID == "" {
		accountData.ID = uuid.NewString()
	}

	now := time.Now().UTC()
	if accountData.CreatedAt.IsZero() {
		accountData.CreatedAt = now
	}
	if accountData.UpdatedAt.IsZero() {
		accountData.UpdatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		accountData.ID,
		accountData.Email,
		accountData.PasswordHash,
		accountData.IsActive,
		accountData.CreatedAt,
		accountData.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.WarnContext(ctx, "postgres save account conflict", slog.String("error", err.Error()))
			return "", fmt.Errorf("%w: %v", ErrUserAlreadyExists, err)
		}

		slog.ErrorContext(ctx, "postgres save account failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "account_id", accountData.ID), "postgres account saved")
	return accountData.ID, nil
}

func (r *AccountRepository) FindAccount(ctx context.Context, loginData domain.Login) (*domain.Account, error) {
	ctx = sharedlogger.WithField(ctx, "email", loginData.Email)

	const query = `
		SELECT
			id,
			email,
			password_hash,
			is_active,
			created_at,
			updated_at,
			last_login_at
		FROM accounts
		WHERE email = $1
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, loginData.Email)

	var account domain.Account
	err := row.Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "postgres account not found by email")
			return nil, fmt.Errorf("%w: account with email %s not found", ErrNoRows, loginData.Email)
		}

		slog.ErrorContext(ctx, "postgres find account by email failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "account_id", account.ID), "postgres account loaded by email")
	return &account, nil
}

func (r *AccountRepository) FindAccountByID(ctx context.Context, accountID string) (*domain.Account, error) {
	ctx = sharedlogger.WithField(ctx, "account_id", accountID)

	const query = `
		SELECT
			id,
			email,
			password_hash,
			is_active,
			created_at,
			updated_at,
			last_login_at
		FROM accounts
		WHERE id = $1
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, accountID)

	var account domain.Account
	err := row.Scan(
		&account.ID,
		&account.Email,
		&account.PasswordHash,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
		&account.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "postgres account not found by id")
			return nil, fmt.Errorf("%w: account with id %s not found", ErrNoRows, accountID)
		}

		slog.ErrorContext(ctx, "postgres find account by id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "email", account.Email), "postgres account loaded by id")
	return &account, nil
}
