package postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
)

// OAuthRepository реализует работу с oauth_accounts через pgx pool.
type OAuthRepository struct {
	pool *pgxpool.Pool
}

// NewOAuthRepo создаёт новый репозиторий OAuth аккаунтов.
func NewOAuthRepo(pool *pgxpool.Pool) *OAuthRepository {
	return &OAuthRepository{pool: pool}
}

// GetByProviderID находит OAuth аккаунт по провайдеру и provider_id.
func (r *OAuthRepository) GetByProviderID(ctx context.Context, provider domain.OAuthProvider, providerID string) (*domain.OAuthAccount, error) {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"provider":    string(provider),
		"provider_id": providerID,
	})

	const query = `
		SELECT
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		FROM oauth_accounts
		WHERE provider = $1 AND provider_id = $2
		LIMIT 1
	`

	var account domain.OAuthAccount
	err := r.pool.QueryRow(ctx, query, provider, providerID).Scan(
		&account.ID,
		&account.AccountID,
		&account.Provider,
		&account.ProviderID,
		&account.Email,
		&account.AccessToken,
		&account.RefreshToken,
		&account.Expiry,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Не найден - это не ошибка
		}
		slog.ErrorContext(ctx, "postgres get oauth account by provider id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(ctx, "postgres oauth account loaded by provider id")
	return &account, nil
}

// GetByAccountID находит все OAuth аккаунты пользователя.
func (r *OAuthRepository) GetByAccountID(ctx context.Context, accountID string) ([]*domain.OAuthAccount, error) {
	ctx = sharedlogger.WithField(ctx, "account_id", accountID)

	const query = `
		SELECT
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		FROM oauth_accounts
		WHERE account_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, accountID)
	if err != nil {
		slog.ErrorContext(ctx, "postgres get oauth accounts by account id failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}
	defer rows.Close()

	var accounts []*domain.OAuthAccount
	for rows.Next() {
		var account domain.OAuthAccount
		err := rows.Scan(
			&account.ID,
			&account.AccountID,
			&account.Provider,
			&account.ProviderID,
			&account.Email,
			&account.AccessToken,
			&account.RefreshToken,
			&account.Expiry,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "postgres scan oauth account failed", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
		}
		accounts = append(accounts, &account)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "postgres rows error", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%w: %v", ErrDatabaseRead, err)
	}

	slog.InfoContext(ctx, "postgres oauth accounts loaded by account id", slog.Int("count", len(accounts)))
	return accounts, nil
}

// Create создаёт новый OAuth аккаунт.
func (r *OAuthRepository) Create(ctx context.Context, account *domain.OAuthAccount) error {
	ctx = sharedlogger.WithFields(ctx, map[string]interface{}{
		"account_id": account.AccountID,
		"provider":   string(account.Provider),
	})

	const query = `
		INSERT INTO oauth_accounts (
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
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
		slog.ErrorContext(ctx, "postgres create oauth account failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	slog.InfoContext(ctx, "postgres oauth account created")
	return nil
}

// UpdateTokens обновляет OAuth токены.
func (r *OAuthRepository) UpdateTokens(ctx context.Context, id string, accessToken, refreshToken string, expiry time.Time) error {
	ctx = sharedlogger.WithField(ctx, "oauth_account_id", id)

	const query = `
		UPDATE oauth_accounts
		SET access_token = $1, refresh_token = $2, expiry = $3, updated_at = $4
		WHERE id = $5
	`

	result, err := r.pool.Exec(ctx, query, accessToken, refreshToken, expiry, time.Now().UTC(), id)
	if err != nil {
		slog.ErrorContext(ctx, "postgres update oauth tokens failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	if result.RowsAffected() == 0 {
		slog.WarnContext(ctx, "postgres oauth account not found for token update")
		return fmt.Errorf("%w: oauth account not found", ErrNoRows)
	}

	slog.InfoContext(ctx, "postgres oauth tokens updated")
	return nil
}

// Delete удаляет OAuth аккаунт.
func (r *OAuthRepository) Delete(ctx context.Context, id string) error {
	ctx = sharedlogger.WithField(ctx, "oauth_account_id", id)

	const query = `DELETE FROM oauth_accounts WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		slog.ErrorContext(ctx, "postgres delete oauth account failed", slog.String("error", err.Error()))
		return fmt.Errorf("%w: %v", ErrDatabaseWrite, err)
	}

	if result.RowsAffected() == 0 {
		slog.WarnContext(ctx, "postgres oauth account not found for deletion")
		return fmt.Errorf("%w: oauth account not found", ErrNoRows)
	}

	slog.InfoContext(ctx, "postgres oauth account deleted")
	return nil
}
