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

type OAuthRepository struct {
	db *sql.DB
}

func NewOAuthRepo(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

func (r *OAuthRepository) GetByProviderID(ctx context.Context, provider domain.OAuthProvider, providerID string) (*domain.OAuthAccount, error) {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider":    string(provider),
		"provider_id": providerID,
	})

	const query = `
		SELECT
			id,
			account_id,
			provider,
			provider_id,
			email,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at
		FROM oauth_accounts
		WHERE provider = $1 AND provider_id = $2
		LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, provider, providerID)
	account, err := scanOAuthAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		slog.WarnContext(ctx, "postgres oauth account not found by provider id")
		return nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "postgres get oauth account by provider id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "account_id", account.AccountID), "postgres oauth account loaded by provider id")
	return account, nil
}

func (r *OAuthRepository) GetByAccountID(ctx context.Context, userID string) ([]*domain.OAuthAccount, error) {
	ctx = sharedlogger.WithField(ctx, "account_id", userID)

	const query = `
		SELECT
			id,
			account_id,
			provider,
			provider_id,
			email,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at
		FROM oauth_accounts
		WHERE account_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		slog.ErrorContext(ctx, "postgres get oauth accounts by user id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}
	defer rows.Close()

	accounts := make([]*domain.OAuthAccount, 0)
	for rows.Next() {
		account, err := scanOAuthAccount(rows)
		if err != nil {
			slog.ErrorContext(ctx, "postgres scan oauth account failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "postgres iterate oauth accounts failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(ctx, "postgres oauth accounts loaded by user id", slog.Int("accounts_count", len(accounts)))
	return accounts, nil
}

func (r *OAuthRepository) Create(ctx context.Context, account *domain.OAuthAccount) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id":  account.AccountID,
		"provider":    string(account.Provider),
		"provider_id": account.ProviderID,
		"email":       account.Email,
	})

	const query = `
		INSERT INTO oauth_accounts (
			id,
			account_id,
			provider,
			provider_id,
			email,
			access_token,
			refresh_token,
			expiry,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now().UTC()
	if account.ID == "" {
		account.ID = uuid.NewString()
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		account.ID,
		account.AccountID,
		account.Provider,
		account.ProviderID,
		account.Email,
		account.AccessToken,
		account.RefreshToken,
		account.Expiry,
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			slog.WarnContext(ctx, "postgres create oauth account conflict", slog.String("error", err.Error()))
			return fmt.Errorf("%w: %v", ErrOAuthAccountExists, err)
		}

		slog.ErrorContext(ctx, "postgres create oauth account failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(sharedlogger.WithField(ctx, "oauth_account_id", account.ID), "postgres oauth account created")
	return nil
}

func (r *OAuthRepository) UpdateTokens(ctx context.Context, accountID, accessToken, refreshToken string, expiry time.Time) error {
	ctx = sharedlogger.WithField(ctx, "oauth_account_id", accountID)

	const query = `
		UPDATE oauth_accounts
		SET access_token = $2,
		    refresh_token = $3,
		    expiry = $4,
		    updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, accountID, accessToken, refreshToken, expiry, time.Now().UTC())
	if err != nil {
		slog.ErrorContext(ctx, "postgres update oauth tokens failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(ctx, "postgres oauth tokens updated")
	return nil
}

func (r *OAuthRepository) Delete(ctx context.Context, accountID string) error {
	ctx = sharedlogger.WithField(ctx, "oauth_account_id", accountID)

	const query = `DELETE FROM oauth_accounts WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, accountID)
	if err != nil {
		slog.ErrorContext(ctx, "postgres delete oauth account failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(ctx, "postgres oauth account deleted")
	return nil
}

type oauthAccountScanner interface {
	Scan(dest ...any) error
}

func scanOAuthAccount(scanner oauthAccountScanner) (*domain.OAuthAccount, error) {
	var account domain.OAuthAccount
	var provider string

	err := scanner.Scan(
		&account.ID,
		&account.AccountID,
		&provider,
		&account.ProviderID,
		&account.Email,
		&account.AccessToken,
		&account.RefreshToken,
		&account.Expiry,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	account.Provider = domain.OAuthProvider(provider)
	return &account, nil
}
