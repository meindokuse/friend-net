package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// Create создаёт нового пользователя.
func (r *UserRepository) Create(ctx context.Context, u *domainuser.User) error {
	_, err := r.collection.InsertOne(ctx, toDocument(u))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return r.parseDuplicateKeyError(err)
		}
		return err
	}
	return nil
}

// Update обновляет существующего пользователя с optimistic locking.
// В entity version уже увеличен через touch(), поэтому ищем version-1.
func (r *UserRepository) Update(ctx context.Context, u *domainuser.User) error {
	filter := bson.M{
		"_id":     u.ID(),
		"version": u.Version() - 1,
	}

	result, err := r.collection.ReplaceOne(ctx, filter, toDocument(u))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return r.parseDuplicateKeyError(err)
		}
		return err
	}

	if result.MatchedCount == 0 {
		count, _ := r.collection.CountDocuments(ctx, bson.M{"_id": u.ID()})
		if count == 0 {
			return domainuser.ErrUserNotFound
		}
		return domainuser.ErrVersionConflict
	}

	return nil
}
