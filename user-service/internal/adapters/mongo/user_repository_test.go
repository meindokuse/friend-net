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

// Эти тесты требуют запущенный MongoDB (integration tests).
// Для запуска: docker run -d -p 27017:27017 mongo:7

var testDB *mongo.Database

// TestMain устанавливает одно соединение на весь прогон тестов.
func TestMain(m *testing.M) {
	cfg := Config{
		URI:      "mongodb://localhost:27017",
		Database: "user_service_test",
		Timeout:  5 * time.Second,
	}

	ctx := context.Background()
	db, err := Connect(ctx, cfg)
	if err != nil {
		// MongoDB недоступна — все тесты будут пропущены внутри setupTestRepo
		os.Exit(m.Run())
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

	// Очищаем коллекцию перед каждым тестом
	if err := testDB.Collection(collectionName).Drop(ctx); err != nil {
		t.Fatalf("failed to drop collection: %v", err)
	}

	repo, err := NewUserRepository(testDB)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	return repo
}

func createTestUser(t *testing.T) *domainuser.User {
	t.Helper()

	username := vo.MustNewUsername("testuser")
	email := vo.MustNewEmail("test@example.com")

	user, err := domainuser.NewUser(
		uuid.New(),
		username,
		&email,
		nil,
		"Test User",
	)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestUserRepository_Create(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Проверяем, что пользователь создан
	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if found.ID() != user.ID() {
		t.Errorf("expected ID %v, got %v", user.ID(), found.ID())
	}
	if found.Username().String() != user.Username().String() {
		t.Errorf("expected username %v, got %v", user.Username(), found.Username())
	}
}

func TestUserRepository_Create_DuplicateUsername(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user1 := createTestUser(t)
	if err := repo.Create(ctx, user1); err != nil {
		t.Fatalf("Create user1 failed: %v", err)
	}

	// Пытаемся создать пользователя с тем же username
	email2 := vo.MustNewEmail("another@example.com")
	user2, _ := domainuser.NewUser(
		uuid.New(),
		user1.Username(),
		&email2,
		nil,
		"Another User",
	)

	err := repo.Create(ctx, user2)
	if err != domainuser.ErrUsernameAlreadyTaken {
		t.Errorf("expected ErrUsernameAlreadyTaken, got %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Обновляем профиль
	bio := "New bio"
	if err := user.UpdateProfile("Updated Name", &bio, nil); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Проверяем обновление
	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if found.Profile().DisplayName != "Updated Name" {
		t.Errorf("expected display name 'Updated Name', got %v", found.Profile().DisplayName)
	}
	if found.Profile().Bio == nil || *found.Profile().Bio != "New bio" {
		t.Errorf("expected bio 'New bio', got %v", found.Profile().Bio)
	}
}

func TestUserRepository_Update_OptimisticLock(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Получаем две копии пользователя
	user1, _ := repo.GetByID(ctx, user.ID())
	user2, _ := repo.GetByID(ctx, user.ID())

	// Обновляем первую копию
	user1.UpdateProfile("Name 1", nil, nil)
	if err := repo.Update(ctx, user1); err != nil {
		t.Fatalf("Update user1 failed: %v", err)
	}

	// Пытаемся обновить вторую копию (должен быть конфликт версий)
	user2.UpdateProfile("Name 2", nil, nil)
	err := repo.Update(ctx, user2)
	if err != domainuser.ErrVersionConflict {
		t.Errorf("expected ErrVersionConflict, got %v", err)
	}
}

func TestUserRepository_GetByUsername(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.GetByUsername(ctx, user.Username())
	if err != nil {
		t.Fatalf("GetByUsername failed: %v", err)
	}

	if found.ID() != user.ID() {
		t.Errorf("expected ID %v, got %v", user.ID(), found.ID())
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	found, err := repo.GetByEmail(ctx, *user.Email())
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if found.ID() != user.ID() {
		t.Errorf("expected ID %v, got %v", user.ID(), found.ID())
	}
}

func TestUserRepository_GetByIDs(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	// Создаём несколько пользователей
	users := make([]*domainuser.User, 3)
	ids := make([]uuid.UUID, 3)

	for i := 0; i < 3; i++ {
		username := vo.MustNewUsername("user" + string(rune('a'+i)))
		email := vo.MustNewEmail("user" + string(rune('a'+i)) + "@example.com")
		user, _ := domainuser.NewUser(uuid.New(), username, &email, nil, "User "+string(rune('A'+i)))
		users[i] = user
		ids[i] = user.ID()

		if err := repo.Create(ctx, user); err != nil {
			t.Fatalf("Create user %d failed: %v", i, err)
		}
	}

	// Получаем batch
	found, err := repo.GetByIDs(ctx, ids)
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}

	if len(found) != 3 {
		t.Errorf("expected 3 users, got %d", len(found))
	}
}

func TestUserRepository_Search(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	// Создаём пользователей для поиска
	users := []struct {
		username string
		email    string
		display  string
	}{
		{"alice", "alice@example.com", "Alice Smith"},
		{"bob", "bob@example.com", "Bob Johnson"},
		{"charlie", "charlie@example.com", "Charlie Brown"},
	}

	for _, u := range users {
		username := vo.MustNewUsername(u.username)
		email := vo.MustNewEmail(u.email)
		user, _ := domainuser.NewUser(uuid.New(), username, &email, nil, u.display)
		if err := repo.Create(ctx, user); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Поиск по username
	results, err := repo.Search(ctx, "ali", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Username().String() != "alice" {
		t.Errorf("expected username 'alice', got %v", results[0].Username())
	}
}

func TestUserRepository_UpdateLastSeen(t *testing.T) {
	repo := setupTestRepo(t)
	ctx := context.Background()

	user := createTestUser(t)
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Обновляем last_seen
	if err := repo.UpdateLastSeen(ctx, user.ID()); err != nil {
		t.Fatalf("UpdateLastSeen failed: %v", err)
	}

	// Проверяем, что last_seen обновился
	found, err := repo.GetByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if found.LastSeenAt() == nil {
		t.Error("expected last_seen_at to be set")
	}

	// Проверяем, что version НЕ изменился
	if found.Version() != user.Version() {
		t.Errorf("expected version %d, got %d (last_seen should not bump version)", user.Version(), found.Version())
	}
}
