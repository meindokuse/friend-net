package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/meindokuse/cloud-drive/user-service-new/internal/domain/entity"
	vo "github.com/meindokuse/cloud-drive/user-service-new/internal/domain/valueobject"
)

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
	})
	return err
}

func activeFilter(extra bson.M) bson.M { extra["deleted_at"] = bson.M{"$eq": nil}; return extra }

type userDoc struct {
	ID           uuid.UUID  `bson:"_id"`
	Username     string     `bson:"username"`
	Email        *string    `bson:"email,omitempty"`
	Phone        *string    `bson:"phone,omitempty"`
	DisplayName  string     `bson:"display_name"`
	Bio          *string    `bson:"bio,omitempty"`
	AvatarURL    *string    `bson:"avatar_url,omitempty"`
	WhoCanMsg    string     `bson:"who_can_message"`
	WhoCanSeen   string     `bson:"who_can_see_last_seen"`
	WhoCanProf   string     `bson:"who_can_see_profile"`
	Language     string     `bson:"language"`
	Timezone     string     `bson:"timezone"`
	EmailVer     bool       `bson:"email_verified"`
	PhoneVer     bool       `bson:"phone_verified"`
	IsActive     bool       `bson:"is_active"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
	LastSeenAt   *time.Time `bson:"last_seen_at,omitempty"`
	DeletedAt    *time.Time `bson:"deleted_at,omitempty"`
	Version      int        `bson:"version"`
}

func toDoc(u *entity.User) *userDoc {
	var email, phone *string
	if u.Email() != nil { s := u.Email().String(); email = &s }
	if u.Phone() != nil { s := u.Phone().String(); phone = &s }
	return &userDoc{
		ID: u.ID(), Username: u.Username().String(), Email: email, Phone: phone,
		DisplayName: u.Profile().DisplayName, Bio: u.Profile().Bio, AvatarURL: u.Profile().AvatarURL,
		WhoCanMsg: string(u.Settings().Privacy.WhoCanMessage), WhoCanSeen: string(u.Settings().Privacy.WhoCanSeeLastSeen), WhoCanProf: string(u.Settings().Privacy.WhoCanSeeProfile),
		Language: u.Settings().Language, Timezone: u.Settings().Timezone,
		EmailVer: u.Verification().EmailVerified, PhoneVer: u.Verification().PhoneVerified,
		IsActive: u.IsActive(), CreatedAt: u.CreatedAt(), UpdatedAt: u.UpdatedAt(), LastSeenAt: u.LastSeenAt(), DeletedAt: u.DeletedAt(), Version: u.Version(),
	}
}

func fromDoc(d *userDoc) (*entity.User, error) {
	username := vo.MustNewUsername(d.Username)
	var email *vo.Email
	if d.Email != nil { e := vo.MustNewEmail(*d.Email); email = &e }
	var phone *vo.Phone
	if d.Phone != nil { p := vo.MustNewPhone(*d.Phone); phone = &p }
	return entity.Reconstruct(
		d.ID, username, email, phone,
		entity.Profile{DisplayName: d.DisplayName, Bio: d.Bio, AvatarURL: d.AvatarURL},
		entity.Settings{Privacy: entity.PrivacySettings{WhoCanMessage: entity.PrivacyLevel(d.WhoCanMsg), WhoCanSeeLastSeen: entity.PrivacyLevel(d.WhoCanSeen), WhoCanSeeProfile: entity.PrivacyLevel(d.WhoCanProf)}, Language: d.Language, Timezone: d.Timezone},
		entity.Verification{EmailVerified: d.EmailVer, PhoneVerified: d.PhoneVer},
		d.IsActive, d.CreatedAt, d.UpdatedAt, d.LastSeenAt, d.DeletedAt, d.Version,
	), nil
}

func mapDup(err error) error {
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

func (s *Storage) Create(ctx context.Context, u *entity.User) error {
	_, err := s.collection.InsertOne(ctx, toDoc(u))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) { return mapDup(err) }
		return err
	}
	return nil
}
func (s *Storage) Update(ctx context.Context, u *entity.User) error {
	filter := bson.M{"_id": u.ID(), "version": u.Version() - 1}
	res, err := s.collection.ReplaceOne(ctx, filter, toDoc(u))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) { return mapDup(err) }
		return err
	}
	if res.MatchedCount == 0 {
		count, _ := s.collection.CountDocuments(ctx, bson.M{"_id": u.ID()})
		if count == 0 { return entity.ErrUserNotFound }
		return entity.ErrVersionConflict
	}
	return nil
}
func (s *Storage) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var d userDoc
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"_id": id})).Decode(&d)
	if err != nil { if errors.Is(err, mongo.ErrNoDocuments) { return nil, entity.ErrUserNotFound }; return nil, err }
	return fromDoc(&d)
}
func (s *Storage) GetByUsername(ctx context.Context, username vo.Username) (*entity.User, error) {
	var d userDoc
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"username": username.String()})).Decode(&d)
	if err != nil { if errors.Is(err, mongo.ErrNoDocuments) { return nil, entity.ErrUserNotFound }; return nil, err }
	return fromDoc(&d)
}
func (s *Storage) GetByEmail(ctx context.Context, email vo.Email) (*entity.User, error) {
	var d userDoc
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"email": email.String()})).Decode(&d)
	if err != nil { if errors.Is(err, mongo.ErrNoDocuments) { return nil, entity.ErrUserNotFound }; return nil, err }
	return fromDoc(&d)
}
func (s *Storage) GetByPhone(ctx context.Context, phone vo.Phone) (*entity.User, error) {
	var d userDoc
	err := s.collection.FindOne(ctx, activeFilter(bson.M{"phone": phone.String()})).Decode(&d)
	if err != nil { if errors.Is(err, mongo.ErrNoDocuments) { return nil, entity.ErrUserNotFound }; return nil, err }
	return fromDoc(&d)
}
func (s *Storage) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	if len(ids) == 0 { return []*entity.User{}, nil }
	cur, err := s.collection.Find(ctx, activeFilter(bson.M{"_id": bson.M{"$in": ids}}))
	if err != nil { return nil, err }
	defer cur.Close(ctx)
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil { return nil, err }
	out := make([]*entity.User, 0, len(docs))
	for i := range docs { u, err := fromDoc(&docs[i]); if err != nil { return nil, err }; out = append(out, u) }
	return out, nil
}
func (s *Storage) Search(ctx context.Context, query string, limit, offset int) ([]*entity.User, error) {
	if limit <= 0 { limit = 20 }; if limit > 100 { limit = 100 }; if offset < 0 { offset = 0 }
	filter := activeFilter(bson.M{"$or": []bson.M{{"username": bson.M{"$regex": query, "$options": "i"}}, {"display_name": bson.M{"$regex": query, "$options": "i"}}}})
	opts := options.Find().SetSort(bson.D{{Key: "username", Value: 1}}).SetLimit(int64(limit)).SetSkip(int64(offset))
	cur, err := s.collection.Find(ctx, filter, opts); if err != nil { return nil, err }
	defer cur.Close(ctx)
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil { return nil, err }
	out := make([]*entity.User, 0, len(docs))
	for i := range docs { u, err := fromDoc(&docs[i]); if err != nil { return nil, err }; out = append(out, u) }
	return out, nil
}
func (s *Storage) List(ctx context.Context, params entity.ListParams) ([]*entity.User, entity.PagedUsers, error) {
	limit := params.Limit; if limit <= 0 { limit = 20 }; if limit > 100 { limit = 100 }
	filter := activeFilter(bson.M{})
	if params.Cursor != nil {
		filter = activeFilter(bson.M{"$or": []bson.M{{"username": bson.M{"$gt": params.Cursor.Username}}, {"$and": []bson.M{{"username": params.Cursor.Username}, {"_id": bson.M{"$gt": params.Cursor.ID}}}}}})
	}
	opts := options.Find().SetSort(bson.D{{Key: "username", Value: 1}, {Key: "_id", Value: 1}}).SetLimit(int64(limit + 1))
	cur, err := s.collection.Find(ctx, filter, opts); if err != nil { return nil, entity.PagedUsers{}, err }
	defer cur.Close(ctx)
	var docs []userDoc
	if err := cur.All(ctx, &docs); err != nil { return nil, entity.PagedUsers{}, err }
	hasMore := len(docs) > limit; if hasMore { docs = docs[:limit] }
	out := make([]*entity.User, 0, len(docs))
	for i := range docs { u, err := fromDoc(&docs[i]); if err != nil { return nil, entity.PagedUsers{}, err }; out = append(out, u) }
	paged := entity.PagedUsers{Items: out, HasMore: hasMore}
	if hasMore && len(docs) > 0 { last := docs[len(docs)-1]; paged.NextCursor = entity.UsernameCursor{Username: last.Username, ID: last.ID} }
	return out, paged, nil
}
func (s *Storage) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	res, err := s.collection.UpdateOne(ctx, activeFilter(bson.M{"_id": id}), bson.M{"$set": bson.M{"last_seen_at": time.Now().UTC()}})
	if err != nil { return err }
	if res.MatchedCount == 0 { return entity.ErrUserNotFound }
	return nil
}
