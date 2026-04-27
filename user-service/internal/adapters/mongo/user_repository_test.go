package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
	domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
)

// Тесты требуют запущенный MongoDB.
// docker run -d --name mongo-test -p 27017:27017 mongo:7

var testDB *mongo.Database

// TestMain — одно соединение на весь прогон.
func TestMain(m *testing.M) {
	cfg := Config{
		URI:      "mongodb://localhost:27017",
		Database: "user_service_test",
		Timeout:  5 * time.Second,
	}

	ctx := context.Background()
	db, err := Connect(ctx, cfg)
	if err != nil {
		os.Exit(m.Run()) // MongoDB недоступна — тесты пропустятся внутри setupTestRepo
	}
	testDB = db

	code := m.Run()
	_ = Disconnect(context.Background(), testDB)
	os.Exit(code)
}

func setupTestRepo(t *testing.T) *UserRepository {
	t.Helper()
	if testDB == nil {
		t.Skip("MongoDB not available")
	}
	ctx := context.Background()
	if err := testDB.Collection(collectionName).Drop(ctx); err != nil {
		t.Fatalf("drop collection: %v", err)
	}
	repo, err := NewUserRepository(testDB)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	return repo
}

func newTestUser(t *testing.T, username, email, displayName string) *domainuser.User {
	t.Helper()
	u := vo.MustNewUsername(username)
	e := vo.MustNewEmail(email)
	user, err := domainuser.NewUser(uuid.New(), u, &e, nil, displayName)
	if err != nil {
		t.Fatalf("NewUser(%s): %v", username, err)
	}
	return user
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreate_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "testuser", "test@example.com", "Test User")

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.ID() != user.ID() {
		t.Errorf("ID mismatch: want %v, got %v", user.ID(), found.ID())
	}
	if found.Username().String() != user.Username().String() {
		t.Errorf("Username mismatch: want %v, got %v", user.Username(), found.Username())
	}
}

func TestCreate_DuplicateUsername(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user1 := newTestUser(t, "testuser", "first@example.com", "First")
	if err := repo.Create(ctx, user1); err != nil {
		t.Fatalf("Create user1: %v", err)
	}

	user2 := newTestUser(t, "testuser", "second@example.com", "Second")
	err := repo.Create(ctx, user2)
	if err != domainuser.ErrUsernameAlreadyTaken {
		t.Errorf("want ErrUsernameAlreadyTaken, got %v", err)
	}
}

func TestCreate_DuplicateEmail(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user1 := newTestUser(t, "userone", "same@example.com", "One")
	if err := repo.Create(ctx, user1); err != nil {
		t.Fatalf("Create user1: %v", err)
	}

	user2 := newTestUser(t, "usertwo", "same@example.com", "Two")
	err := repo.Create(ctx, user2)
	if err != domainuser.ErrEmailAlreadyTaken {
		t.Errorf("want ErrEmailAlreadyTaken, got %v", err)
	}
}

// ─── Update ──────────────────────────────────────────────────────────────────

func TestUpdate_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "testuser", "test@example.com", "Test User")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	bio := "new bio"
	if err := user.UpdateProfile("Updated Name", &bio, nil); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.Profile().DisplayName != "Updated Name" {
		t.Errorf("DisplayName: want 'Updated Name', got %q", found.Profile().DisplayName)
	}
	if found.Profile().Bio == nil || *found.Profile().Bio != "new bio" {
		t.Errorf("Bio: want 'new bio', got %v", found.Profile().Bio)
	}
}

func TestUpdate_OptimisticLock(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "testuser", "test@example.com", "Test User")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Две независимые копии одного пользователя
	copy1, _ := repo.GetByID(ctx, user.ID())
	copy2, _ := repo.GetByID(ctx, user.ID())

	copy1.UpdateProfile("Name from copy1", nil, nil)
	if err := repo.Update(ctx, copy1); err != nil {
		t.Fatalf("Update copy1: %v", err)
	}

	copy2.UpdateProfile("Name from copy2", nil, nil)
	err := repo.Update(ctx, copy2)
	if err != domainuser.ErrVersionConflict {
		t.Errorf("want ErrVersionConflict, got %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	// Пользователь не сохранён в БД
	user := newTestUser(t, "ghost", "ghost@example.com", "Ghost")
	user.UpdateProfile("Ghost Updated", nil, nil) // version становится 2

	err := repo.Update(ctx, user)
	if err != domainuser.ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

// ─── GetBy* ──────────────────────────────────────────────────────────────────

func TestGetByID_NotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	if err != domainuser.ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}

func TestGetByUsername_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "alice", "alice@example.com", "Alice")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.GetByUsername(ctx, user.Username())
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if found.ID() != user.ID() {
		t.Errorf("ID mismatch: want %v, got %v", user.ID(), found.ID())
	}
}

func TestGetByEmail_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "alice", "alice@example.com", "Alice")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.GetByEmail(ctx, *user.Email())
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if found.ID() != user.ID() {
		t.Errorf("ID mismatch: want %v, got %v", user.ID(), found.ID())
	}
}

// ─── Soft delete visibility ───────────────────────────────────────────────────

func TestGetByID_SoftDeleted_NotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "todelete", "del@example.com", "To Delete")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := user.SoftDelete(); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update after SoftDelete: %v", err)
	}

	_, err := repo.GetByID(ctx, user.ID())
	if err != domainuser.ErrUserNotFound {
		t.Errorf("want ErrUserNotFound for deleted user, got %v", err)
	}
}

// ─── GetByIDs ────────────────────────────────────────────────────────────────

func TestGetByIDs_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	users := []*domainuser.User{
		newTestUser(t, "usera", "usera@example.com", "User A"),
		newTestUser(t, "userb", "userb@example.com", "User B"),
		newTestUser(t, "userc", "userc@example.com", "User C"),
	}
	ids := make([]uuid.UUID, len(users))
	for i, u := range users {
		ids[i] = u.ID()
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("Create user%d: %v", i, err)
		}
	}

	found, err := repo.GetByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(found) != len(users) {
		t.Errorf("want %d users, got %d", len(users), len(found))
	}
}

func TestGetByIDs_Empty(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	found, err := repo.GetByIDs(ctx, []uuid.UUID{})
	if err != nil {
		t.Fatalf("GetByIDs empty: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("want 0, got %d", len(found))
	}
}

// ─── List (keyset pagination) ─────────────────────────────────────────────────

func TestList_FirstPage(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	for _, u := range []struct{ username, email, display string }{
		{"alice", "alice@example.com", "Alice"},
		{"bob", "bob@example.com", "Bob"},
		{"charlie", "charlie@example.com", "Charlie"},
		{"david", "david@example.com", "David"},
	} {
		if err := repo.Create(ctx, newTestUser(t, u.username, u.email, u.display)); err != nil {
			t.Fatalf("Create %s: %v", u.username, err)
		}
	}

	params := domainuser.ListParams{Limit: 2}
	result, nextCursor, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("want 2 users, got %d", len(result))
	}
	if !nextCursor.HasMore {
		t.Error("want HasMore=true")
	}
	if result[0].Username().String() != "alice" {
		t.Errorf("want first=alice, got %s", result[0].Username())
	}
	if result[1].Username().String() != "bob" {
		t.Errorf("want second=bob, got %s", result[1].Username())
	}
}

func TestList_SecondPage(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	for _, u := range []struct{ username, email, display string }{
		{"alice", "alice@example.com", "Alice"},
		{"bob", "bob@example.com", "Bob"},
		{"charlie", "charlie@example.com", "Charlie"},
		{"david", "david@example.com", "David"},
	} {
		if err := repo.Create(ctx, newTestUser(t, u.username, u.email, u.display)); err != nil {
			t.Fatalf("Create %s: %v", u.username, err)
		}
	}

	// Первая страница
	page1, cursor1, err := repo.List(ctx, domainuser.ListParams{Limit: 2})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if !cursor1.HasMore {
		t.Fatal("want HasMore after page1")
	}

	// Вторая страница
	page2, cursor2, err := repo.List(ctx, domainuser.ListParams{
		Limit:  2,
		Cursor: &cursor1.NextCursor,
	})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("want 2 on page2, got %d", len(page2))
	}
	if cursor2.HasMore {
		t.Error("want HasMore=false after last page")
	}
	if page2[0].Username().String() != "charlie" {
		t.Errorf("want charlie, got %s", page2[0].Username())
	}
	if page2[1].Username().String() != "david" {
		t.Errorf("want david, got %s", page2[1].Username())
	}

	// Нет пересечений между страницами
	seen := map[uuid.UUID]bool{}
	for _, u := range append(page1, page2...) {
		if seen[u.ID()] {
			t.Errorf("duplicate user %s across pages", u.Username())
		}
		seen[u.ID()] = true
	}
}

func TestList_NoCursor_AllResults(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	for _, u := range []struct{ username, email, display string }{
		{"alice", "alice@example.com", "Alice"},
		{"bob", "bob@example.com", "Bob"},
	} {
		if err := repo.Create(ctx, newTestUser(t, u.username, u.email, u.display)); err != nil {
			t.Fatalf("Create %s: %v", u.username, err)
		}
	}

	result, cursor, err := repo.List(ctx, domainuser.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("want 2, got %d", len(result))
	}
	if cursor.HasMore {
		t.Error("want HasMore=false when all fit in one page")
	}
}

func TestList_SoftDeleted_Excluded(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	active := newTestUser(t, "active", "active@example.com", "Active")
	deleted := newTestUser(t, "deleted", "deleted@example.com", "Deleted")

	if err := repo.Create(ctx, active); err != nil {
		t.Fatalf("Create active: %v", err)
	}
	if err := repo.Create(ctx, deleted); err != nil {
		t.Fatalf("Create deleted: %v", err)
	}

	deleted.SoftDelete()
	if err := repo.Update(ctx, deleted); err != nil {
		t.Fatalf("Update deleted: %v", err)
	}

	result, _, err := repo.List(ctx, domainuser.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("want 1 active user, got %d", len(result))
	}
	if result[0].Username().String() != "active" {
		t.Errorf("want 'active', got %s", result[0].Username())
	}
}

// ─── UpdateLastSeen ───────────────────────────────────────────────────────────

func TestUpdateLastSeen_OK(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := newTestUser(t, "testuser", "test@example.com", "Test User")
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.UpdateLastSeen(ctx, user.ID()); err != nil {
		t.Fatalf("UpdateLastSeen: %v", err)
	}

	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.LastSeenAt() == nil {
		t.Error("want LastSeenAt to be set")
	}
	// version не должен измениться
	if found.Version() != user.Version() {
		t.Errorf("version must not change: want %d, got %d", user.Version(), found.Version())
	}
}

func TestUpdateLastSeen_NotFound(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	err := repo.UpdateLastSeen(ctx, uuid.New())
	if err != domainuser.ErrUserNotFound {
		t.Errorf("want ErrUserNotFound, got %v", err)
	}
}
