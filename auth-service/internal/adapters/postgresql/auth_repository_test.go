package postgresql

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/outbox"
)

// Эти тесты требуют запущенный PostgreSQL
// Запустить: docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres --name postgres-test postgres:latest

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skip("PostgreSQL not available:", err)
		return nil
	}

	// Проверяем подключение
	if err := pool.Ping(context.Background()); err != nil {
		t.Skip("PostgreSQL ping failed:", err)
		return nil
	}

	return pool
}

func TestAuthRepository_SaveWithOutbox(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := NewAuthRepository(pool)
	ctx := context.Background()

	// Создаём тестовый аккаунт
	account, err := domain.NewAccount("test@example.com", "hash123")
	if err != nil {
		t.Fatalf("NewAccount failed: %v", err)
	}

	// Создаём outbox event
	outboxEvent, err := outbox.NewAccountCreatedEvent(
		account.ID,
		account.Email,
		"Test User",
		account.CreatedAt,
	)
	if err != nil {
		t.Fatalf("NewAccountCreatedEvent failed: %v", err)
	}

	// Сохраняем
	accountID, err := repo.SaveWithOutbox(ctx, account, outboxEvent)
	if err != nil {
		t.Fatalf("SaveWithOutbox failed: %v", err)
	}

	if accountID == uuid.Nil {
		t.Fatal("expected non-nil accountID")
	}

	// Проверяем что можем найти
	found, err := repo.FindAccountByID(ctx, accountID)
	if err != nil {
		t.Fatalf("FindAccountByID failed: %v", err)
	}

	if found.Email != account.Email {
		t.Errorf("expected email %s, got %s", account.Email, found.Email)
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM outbox_events WHERE aggregate_id = $1", accountID)
	_, _ = pool.Exec(ctx, "DELETE FROM accounts WHERE id = $1", accountID)
}

func TestAuthRepository_FindAccount(t *testing.T) {
	pool := getTestPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	repo := NewAuthRepository(pool)
	ctx := context.Background()

	// Создаём тестовый аккаунт
	account, _ := domain.NewAccount("find@example.com", "hash123")
	outboxEvent, _ := outbox.NewAccountCreatedEvent(account.ID, account.Email, "Test", account.CreatedAt)
	accountID, err := repo.SaveWithOutbox(ctx, account, outboxEvent)
	if err != nil {
		t.Fatalf("SaveWithOutbox failed: %v", err)
	}

	// Ищем по email
	found, err := repo.FindAccount(ctx, domain.Login{Email: "find@example.com"})
	if err != nil {
		t.Fatalf("FindAccount failed: %v", err)
	}

	if found.ID != accountID {
		t.Errorf("expected ID %s, got %s", accountID, found.ID)
	}

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM outbox_events WHERE aggregate_id = $1", accountID)
	_, _ = pool.Exec(ctx, "DELETE FROM accounts WHERE id = $1", accountID)
}
