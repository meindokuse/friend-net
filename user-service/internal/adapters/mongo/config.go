package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Config содержит настройки подключения к MongoDB.
type Config struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// Connect устанавливает соединение с MongoDB и возвращает database instance.
func Connect(ctx context.Context, cfg Config) (*mongo.Database, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(cfg.Timeout).
		SetServerSelectionTimeout(cfg.Timeout)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return client.Database(cfg.Database), nil
}

// Disconnect закрывает соединение с MongoDB.
func Disconnect(ctx context.Context, db *mongo.Database) error {
	if db == nil {
		return nil
	}
	return db.Client().Disconnect(ctx)
}
