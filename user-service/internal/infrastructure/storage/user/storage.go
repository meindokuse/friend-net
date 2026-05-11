package user

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/value_object"
	"github.com/meindokuse/cloud-drive/user-service-new/internal/infrastructure/storage/user/dao"
)

// paginationIndex is the compound index that backs both List and Search.
// Sort order must stay in sync with SetSort calls below.
var paginationIndex = bson.D{
	{Key: "deleted_at", Value: 1},
	{Key: "username", Value: 1},
	{Key: "_id", Value: 1},
}

type Storage struct {
	collection *mongo.Collection
}

func NewStorage(db *mongo.Database) (*Storage, error) {
	s := &Storage{collection: db.Collection("users")}
	if err := s.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) ensureIndexes(ctx context.Context) error {
	_, err := s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "phone", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "deleted_at", Value: 1}}},
		{
            Keys: bson.D{
                {Key: "deleted_at", Value: 1}, 
                {Key: "username", Value: 1}, 
                {Key: "_id", Value: 1},
            },
        },
	})
	return err
}

func activeFilter(extra bson.M) bson.M {
	extra["deleted_at"] = bson.M{"$eq": nil}
	return extra
}

func mapDupError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "username"):
		return entity.ErrUsernameAlreadyTaken
	case strings.Contains(msg, "email"):
		return entity.ErrEmailAlreadyTaken
	case strings.Contains(msg, "phone"):
		return entity.ErrPhoneAlreadyTaken
	default:
		return err
	}
}

// isDomainErr returns true for expected business errors that are not infrastructure
// failures and should not be logged at ERROR level by the storage layer.
func isDomainErr(err error) bool {
	return errors.Is(err, entity.ErrUserNotFound) ||
		errors.Is(err, entity.ErrVersionConflict) ||
		errors.Is(err, entity.ErrUsernameAlreadyTaken) ||
		errors.Is(err, entity.ErrEmailAlreadyTaken) ||
		errors.Is(err, entity.ErrPhoneAlreadyTaken) ||
		errors.Is(err, entity.ErrAlreadyDeleted)
}

// logOp emits one structured log record per storage operation:
//   - ERROR for unexpected infrastructure failures (not domain errors)
//   - WARN  for queries that exceed 500 ms
//   - DEBUG for all other completions (including domain errors)
func (s *Storage) logOp(ctx context.Context, op string, start time.Time, err error) {
	ms := time.Since(start).Milliseconds()
	col := s.collection.Name()
	switch {
	case err != nil && !isDomainErr(err):
		slog.ErrorContext(ctx, "storage op failed",
			"op", op, "collection", col, "duration_ms", ms, "error", err)
	case ms > 500:
		slog.WarnContext(ctx, "slow storage op",
			"op", op, "collection", col, "duration_ms", ms)
	default:
		slog.DebugContext(ctx, "storage op",
			"op", op, "collection", col, "duration_ms", ms)
	}
}

func (s *Storage) Create(ctx context.Context, u *entity.User) error {
	start := time.Now()
	_, err := s.collection.InsertOne(ctx, dao.FromEntity(u))
	if err != nil && mongo.IsDuplicateKeyError(err) {
		err = mapDupError(err)
	}
	s.logOp(ctx, "insert", start, err)
	return err
}

func (s *Storage) Update(ctx context.Context, u *entity.User) error {
	start := time.Now()
	filter := bson.M{"_id": u.ID(), "version": u.Version() - 1}
	res, err := s.collection.ReplaceOne(ctx, filter, dao.FromEntity(u))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			err = mapDupError(err)
		}
		s.logOp(ctx, "replace", start, err)
		return err
	}
	if res.MatchedCount == 0 {
		count, _ := s.collection.CountDocuments(ctx, bson.M{"_id": u.ID()})
		if count == 0 {
			s.logOp(ctx, "replace", start, entity.ErrUserNotFound)
			return entity.ErrUserNotFound
		}
		s.logOp(ctx, "replace", start, entity.ErrVersionConflict)
		return entity.ErrVersionConflict
	}
	s.logOp(ctx, "replace", start, nil)
	return nil
}

func (s *Storage) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	start := time.Now()
	var d dao.User
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"_id": id})).Decode(&d)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		err = entity.ErrUserNotFound
	}
	s.logOp(ctx, "find_one", start, err)
	if err != nil {
		return nil, err
	}
	return d.ConvertTo(), nil
}

func (s *Storage) GetByUsername(ctx context.Context, username vo.Username) (*entity.User, error) {
	start := time.Now()
	var d dao.User
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"username": username.String()})).Decode(&d)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		err = entity.ErrUserNotFound
	}
	s.logOp(ctx, "find_one", start, err)
	if err != nil {
		return nil, err
	}
	return d.ConvertTo(), nil
}

func (s *Storage) GetByEmail(ctx context.Context, email vo.Email) (*entity.User, error) {
	start := time.Now()
	var d dao.User
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"email": email.String()})).Decode(&d)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		err = entity.ErrUserNotFound
	}
	s.logOp(ctx, "find_one", start, err)
	if err != nil {
		return nil, err
	}
	return d.ConvertTo(), nil
}

func (s *Storage) GetByPhone(ctx context.Context, phone vo.Phone) (*entity.User, error) {
	start := time.Now()
	var d dao.User
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"phone": phone.String()})).Decode(&d)
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		err = entity.ErrUserNotFound
	}
	s.logOp(ctx, "find_one", start, err)
	if err != nil {
		return nil, err
	}
	return d.ConvertTo(), nil
}

func (s *Storage) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}
	start := time.Now()
	cur, err := s.collection.Find(ctx, activeFilter(bson.M{"_id": bson.M{"$in": ids}}))
	if err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []dao.User
	if err := cur.All(ctx, &docs); err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, err
	}
	s.logOp(ctx, "find", start, nil)
	out := make([]*entity.User, 0, len(docs))
	for i := range docs {
		out = append(out, docs[i].ConvertTo())
	}
	return out, nil
}

func (s *Storage) Search(ctx context.Context, params entity.SearchParams) ([]*entity.User, entity.PagedSearchUsers, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	start := time.Now()

	safeQuery := regexp.QuoteMeta(params.Query)
	matchFilter := bson.M{"$or": []bson.M{
		{"username": bson.M{"$regex": "^" + safeQuery, "$options": "i"}},
		{"display_name": bson.M{"$regex": safeQuery, "$options": "i"}},
	}}

	var filter bson.M
	if params.Cursor == nil {
		filter = activeFilter(matchFilter)
	} else {
		filter = bson.M{
			"deleted_at": bson.M{"$eq": nil},
			"$and": []bson.M{
				matchFilter,
				{"$or": []bson.M{
					{"username": bson.M{"$gt": params.Cursor.Username}},
					{"$and": []bson.M{
						{"username": params.Cursor.Username},
						{"_id": bson.M{"$gt": params.Cursor.ID}},
					}},
				}},
			},
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "username", Value: 1}, {Key: "_id", Value: 1}}).
		SetLimit(int64(limit + 1)).
		SetHint(paginationIndex)
	cur, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, entity.PagedSearchUsers{}, err
	}
	defer cur.Close(ctx)
	var docs []dao.User
	if err := cur.All(ctx, &docs); err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, entity.PagedSearchUsers{}, err
	}
	s.logOp(ctx, "find", start, nil)

	hasMore := len(docs) > limit
	if hasMore {
		docs = docs[:limit]
	}
	out := make([]*entity.User, 0, len(docs))
	for i := range docs {
		out = append(out, docs[i].ConvertTo())
	}
	paged := entity.PagedSearchUsers{Items: out, HasMore: hasMore}
	if hasMore && len(docs) > 0 {
		last := docs[len(docs)-1]
		paged.NextCursor = entity.SearchCursor{Username: last.Username, ID: last.ID}
	}
	return out, paged, nil
}

func (s *Storage) List(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	start := time.Now()
	var filter bson.M
	if params.Cursor == nil {
		filter = activeFilter(bson.M{})
	} else {
		filter = bson.M{
			"deleted_at": bson.M{"$eq": nil},
			"$or": []bson.M{
				{"username": bson.M{"$gt": params.Cursor.Username}},
				{"$and": []bson.M{
					{"username": params.Cursor.Username},
					{"_id": bson.M{"$gt": params.Cursor.ID}},
				}},
			},
		}
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "username", Value: 1}, {Key: "_id", Value: 1}}).
		SetLimit(int64(limit + 1)).
		SetHint(paginationIndex)
	cur, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, entity.PagedUsers{}, err
	}
	defer cur.Close(ctx)
	var docs []dao.User
	if err := cur.All(ctx, &docs); err != nil {
		s.logOp(ctx, "find", start, err)
		return nil, entity.PagedUsers{}, err
	}
	s.logOp(ctx, "find", start, nil)
	hasMore := len(docs) > limit
	if hasMore {
		docs = docs[:limit]
	}
	out := make([]*entity.User, 0, len(docs))
	for i := range docs {
		out = append(out, docs[i].ConvertTo())
	}
	paged := entity.PagedUsers{Items: out, HasMore: hasMore}
	if hasMore && len(docs) > 0 {
		last := docs[len(docs)-1]
		paged.NextCursor = entity.UsernameCursor{Username: last.Username, ID: last.ID}
	}
	return out, paged, nil
}

func (s *Storage) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	res, err := s.collection.UpdateOne(
		ctx,
		activeFilter(bson.M{"_id": id}),
		bson.M{"$set": bson.M{"last_seen_at": time.Now().UTC()}},
	)
	if err != nil {
		s.logOp(ctx, "update_one", start, err)
		return err
	}
	if res.MatchedCount == 0 {
		s.logOp(ctx, "update_one", start, entity.ErrUserNotFound)
		return entity.ErrUserNotFound
	}
	s.logOp(ctx, "update_one", start, nil)
	return nil
}
