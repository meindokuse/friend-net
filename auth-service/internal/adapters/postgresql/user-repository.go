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

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/user"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepository {
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, userData domain.User) (string, error) {
	ctx = sharedlogger.WithField(ctx, "email", userData.Email)

	const query = `
		INSERT INTO users (
			id,
			email,
			password_hash,
			mfa_enabled,
			mfa_secret,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`

	if userData.ID == "" {
		userData.ID = uuid.NewString()
	}

	if userData.CreatedAt.IsZero() {
		userData.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, query,
		userData.ID,
		userData.Email,
		userData.PasswordHash,
		userData.MFAEnabled,
		userData.MFASecret,
		userData.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.WarnContext(ctx, "postgres save user conflict", slog.String("error", err.Error()))
			return "", fmt.Errorf("%w: %v", ErrUserAlreadyExists, err)
		}

		slog.ErrorContext(ctx, "postgres save user failed", slog.String("error", err.Error()))
		return "", fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "user_id", userData.ID), "postgres user saved")
	return userData.ID, nil
}

func (r *UserRepository) FindUser(ctx context.Context, loginData domain.Login) (*domain.User, error) {
	ctx = sharedlogger.WithField(ctx, "email", loginData.Email)

	const query = `
		SELECT
			id,
			email,
			password_hash,
			mfa_enabled,
			mfa_secret,
			created_at
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, loginData.Email)

	var user domain.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.MFAEnabled,
		&user.MFASecret,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "postgres user not found by email")
			return nil, fmt.Errorf("%w: user with email %s not found", ErrNoRows, loginData.Email)
		}

		slog.ErrorContext(ctx, "postgres find user by email failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "user_id", user.ID), "postgres user loaded by email")
	return &user, nil
}

func (r *UserRepository) FindUserByID(ctx context.Context, userID string) (*domain.User, error) {
	ctx = sharedlogger.WithField(ctx, "user_id", userID)

	const query = `
		SELECT
			id,
			email,
			password_hash,
			mfa_enabled,
			mfa_secret,
			created_at
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, userID)

	var user domain.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.MFAEnabled,
		&user.MFASecret,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.WarnContext(ctx, "postgres user not found by id")
			return nil, fmt.Errorf("%w: user with id %s not found", ErrNoRows, userID)
		}

		slog.ErrorContext(ctx, "postgres find user by id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "email", user.Email), "postgres user loaded by id")
	return &user, nil
}
