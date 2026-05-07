package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/meindokuse/cloud-drive/user-service-new/config"
)

func NewDatabase(ctx context.Context, cfg config.MongoConfig) (*mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetServerSelectionTimeout(cfg.Timeout))
	if err != nil {
		return nil, fmt.Errorf("connect mongo: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return client.Database(cfg.Database), nil
}
