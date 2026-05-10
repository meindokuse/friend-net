package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collectionName = "processed_events"
	ttlDays        = 7
)

type record struct {
	ID          string    `bson:"_id"`
	ProcessedAt time.Time `bson:"processed_at"`
	Topic       string    `bson:"topic"`
	Partition   int       `bson:"partition"`
	Offset      int64     `bson:"offset"`
}

type Storage struct {
	col *mongo.Collection
}

func NewStorage(db *mongo.Database) (*Storage, error) {
	col := db.Collection(collectionName)
	ttlSeconds := int32(ttlDays * 24 * 3600)
	_, err := col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "processed_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(ttlSeconds),
	})
	if err != nil {
		return nil, fmt.Errorf("idempotency: create ttl index: %w", err)
	}
	return &Storage{col: col}, nil
}

// IsProcessed returns true when the key was previously marked as processed.
func (s *Storage) IsProcessed(ctx context.Context, key string) (bool, error) {
	err := s.col.FindOne(ctx, bson.M{"_id": key}).Err()
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, fmt.Errorf("idempotency: lookup %q: %w", key, err)
}

// MarkProcessed records the key as processed. Duplicate inserts are silently ignored.
func (s *Storage) MarkProcessed(ctx context.Context, key, topic string, partition int, offset int64) error {
	_, err := s.col.InsertOne(ctx, &record{
		ID:          key,
		ProcessedAt: time.Now().UTC(),
		Topic:       topic,
		Partition:   partition,
		Offset:      offset,
	})
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("idempotency: mark %q: %w", key, err)
	}
	return nil
}
