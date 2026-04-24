package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "users"

// UserRepository — MongoDB реализация user.UserRepository.
type UserRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

// NewUserRepository создаёт новый репозиторий с индексами.
func NewUserRepository(db *mongo.Database) (*UserRepository, error) {
	repo := &UserRepository{
		db:         db,
		collection: db.Collection(collectionName),
	}

	if err := repo.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

// ensureIndexes создаёт необходимые индексы.
func (r *UserRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{Key: "phone", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{Key: "deleted_at", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "profile.display_name", Value: "text"},
				{Key: "username", Value: "text"},
			},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// parseDuplicateKeyError определяет, какое поле вызвало дубликат.
func (r *UserRepository) parseDuplicateKeyError(err error) error {
	if err == nil {
		return nil
	}
	return parseDuplicateKey(err)
}
