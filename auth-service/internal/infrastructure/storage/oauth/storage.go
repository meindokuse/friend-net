package oauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage/oauth/dao"
)

// Storage implements OAuth repository
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage creates a new OAuth storage
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// GetByProviderID finds OAuth account by provider and provider ID
func (s *Storage) GetByProviderID(ctx context.Context, provider entity.OAuthProvider, providerID string) (*entity.OAuthAccount, error) {
	const query = `
		SELECT
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		FROM oauth_accounts
		WHERE provider = $1 AND provider_id = $2
		LIMIT 1
	`

	var d dao.OAuthAccount
	err := s.pool.QueryRow(ctx, query, provider, providerID).Scan(
		&d.ID,
		&d.AccountID,
		&d.Provider,
		&d.ProviderID,
		&d.Email,
		&d.AccessToken,
		&d.RefreshToken,
		&d.Expiry,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("query: %w", err)
	}

	return d.ToEntity(), nil
}

// GetByAccountID finds all OAuth accounts for a user
func (s *Storage) GetByAccountID(ctx context.Context, accountID string) ([]*entity.OAuthAccount, error) {
	const query = `
		SELECT
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		FROM oauth_accounts
		WHERE account_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var accounts []*entity.OAuthAccount
	for rows.Next() {
		var d dao.OAuthAccount
		if err := rows.Scan(
			&d.ID,
			&d.AccountID,
			&d.Provider,
			&d.ProviderID,
			&d.Email,
			&d.AccessToken,
			&d.RefreshToken,
			&d.Expiry,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		accounts = append(accounts, d.ToEntity())
	}

	return accounts, nil
}

// Create creates a new OAuth account
func (s *Storage) Create(ctx context.Context, account *entity.OAuthAccount) error {
	const query = `
		INSERT INTO oauth_accounts (
			id, account_id, provider, provider_id, email,
			access_token, refresh_token, expiry, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.pool.Exec(ctx, query,
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
		return fmt.Errorf("insert: %w", err)
	}

	return nil
}

// UpdateTokens updates OAuth tokens
func (s *Storage) UpdateTokens(ctx context.Context, id string, accessToken, refreshToken string, expiry int64) error {
	const query = `
		UPDATE oauth_accounts
		SET access_token = $1, refresh_token = $2, expiry = $3, updated_at = $4
		WHERE id = $5
	`

	_, err := s.pool.Exec(ctx, query, accessToken, refreshToken, time.Unix(expiry, 0).UTC(), time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return nil
}

// Delete deletes an OAuth account
func (s *Storage) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM oauth_accounts WHERE id = $1`

	_, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}
