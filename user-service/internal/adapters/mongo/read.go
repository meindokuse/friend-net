package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// activeFilter возвращает фильтр, исключающий soft-deleted документы.
func activeFilter(extra bson.M) bson.M {
	extra["deleted_at"] = bson.M{"$eq": nil}
	return extra
}

// GetByID получает пользователя по ID. Не возвращает soft-deleted.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domainuser.User, error) {
	var doc userDocument
	err := r.collection.FindOne(ctx, activeFilter(bson.M{"_id": id})).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return fromDocument(&doc)
}

// GetByUsername получает пользователя по username. Не возвращает soft-deleted.
func (r *UserRepository) GetByUsername(ctx context.Context, username vo.Username) (*domainuser.User, error) {
	var doc userDocument
	err := r.collection.FindOne(ctx, activeFilter(bson.M{"username": username.String()})).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return fromDocument(&doc)
}

// GetByEmail получает пользователя по email. Не возвращает soft-deleted.
func (r *UserRepository) GetByEmail(ctx context.Context, email vo.Email) (*domainuser.User, error) {
	var doc userDocument
	err := r.collection.FindOne(ctx, activeFilter(bson.M{"email": email.String()})).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return fromDocument(&doc)
}

// GetByPhone получает пользователя по телефону. Не возвращает soft-deleted.
func (r *UserRepository) GetByPhone(ctx context.Context, phone vo.Phone) (*domainuser.User, error) {
	var doc userDocument
	err := r.collection.FindOne(ctx, activeFilter(bson.M{"phone": phone.String()})).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domainuser.ErrUserNotFound
		}
		return nil, err
	}
	return fromDocument(&doc)
}

// GetByIDs получает пользователей по списку ID (batch). Не возвращает soft-deleted.
func (r *UserRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*domainuser.User, error) {
	if len(ids) == 0 {
		return []*domainuser.User{}, nil
	}

	filter := activeFilter(bson.M{"_id": bson.M{"$in": ids}})
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	users := make([]*domainuser.User, 0, len(docs))
	for i := range docs {
		u, err := fromDocument(&docs[i])
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// Search ищет пользователей по query (username или display_name). Не возвращает soft-deleted.
func (r *UserRepository) Search(ctx context.Context, query string, limit, offset int) ([]*domainuser.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := activeFilter(bson.M{
		"$or": []bson.M{
			{"username": bson.M{"$regex": query, "$options": "i"}},
			{"profile.display_name": bson.M{"$regex": query, "$options": "i"}},
		},
	})

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "username", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	users := make([]*domainuser.User, 0, len(docs))
	for i := range docs {
		u, err := fromDocument(&docs[i])
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateLastSeen обновляет last_seen_at без изменения version. Не работает с soft-deleted.
func (r *UserRepository) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	result, err := r.collection.UpdateOne(
		ctx,
		activeFilter(bson.M{"_id": id}),
		bson.M{"$set": bson.M{"last_seen_at": time.Now().UTC()}},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return domainuser.ErrUserNotFound
	}
	return nil
}
