package postgresql

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	domain "github.com/meindokuse/cloud-drive/auth-service/internal/domain/account"
)

func TestAccountRepositorySaveUsesAccountsTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewAccountRepo(db)
	account := domain.Account{
		ID:           "a-1",
		Email:        "acc@example.com",
		PasswordHash: "hash",
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO accounts`)).
		WithArgs(
			account.ID,
			account.Email,
			account.PasswordHash,
			account.IsActive,
			account.CreatedAt,
			account.UpdatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := repo.Save(context.Background(), account); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAccountRepositoryFindAccountByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewAccountRepo(db)
	now := time.Now().UTC()
	email := "acc@example.com"

	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "is_active", "created_at", "updated_at", "last_login_at",
	}).AddRow("a-1", email, "hash", true, now, now, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM accounts`)).
		WithArgs(email).
		WillReturnRows(rows)

	account, err := repo.FindAccount(context.Background(), domain.Login{Email: email})
	if err != nil {
		t.Fatalf("FindAccount returned error: %v", err)
	}

	if account.Email != email {
		t.Fatalf("expected email %s, got %s", email, account.Email)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
