package storage

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage/account"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage/session"
	"github.com/redis/go-redis/v9"
)

// Registry contains all storage implementations
type Registry struct {
	Account *account.Storage
	OAuth   *oauth.Storage
	Session *session.Storage
}

// NewRegistry creates a new storage registry
func NewRegistry(pool *pgxpool.Pool, rdb *redis.Client, sessionTTL int64) *Registry {
	return &Registry{
		Account: account.NewStorage(pool),
		OAuth:   oauth.NewStorage(pool),
		Session: session.NewStorage(rdb, sessionTTL),
	}
}
