package user_test

import (
	"errors"
	"testing"

	"github.com/300nn/go-identity-service/internal/testutils"
	"github.com/300nn/go-identity-service/internal/user"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresRepository_FindByID(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	created, err := repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "password-hash",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, found.ID)
	}

	if found.Email != created.Email {
		t.Fatalf("expected email %q, got %q", created.Email, found.Email)
	}
}

func TestPostgresRepository_FindByID_NotFound(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	_, err := repo.FindByID(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestPostgresRepository_Create_DuplicateEmail(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "password-hash",
	})
	if err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:         "Another Alex",
		Email:        "alex@example.com",
		Age:          30,
		PasswordHash: "password-hash",
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestPostgresRepository_List(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "password-hash",
	})
	if err != nil {
		t.Fatalf("create Alex: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:         "Bob",
		Email:        "bob@example.com",
		Age:          30,
		PasswordHash: "password-hash",
	})
	if err != nil {
		t.Fatalf("create Bob: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:         "Alice",
		Email:        "alice@test.com",
		Age:          22,
		PasswordHash: "password-hash",
	})
	if err != nil {
		t.Fatalf("create Alice: %v", err)
	}

	result, err := repo.List(ctx, user.ListUsersFilter{
		Limit:  1,
		Offset: 0,
		Email:  "example",
		Sort:   "id_asc",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if result.Total != 2 {
		t.Fatalf("expected total %d, got %d", 2, result.Total)
	}

	if len(result.Users) != 1 {
		t.Fatalf("expected users len %d, got %d", 1, len(result.Users))
	}

	if result.Users[0].Email != "alex@example.com" {
		t.Fatalf("expected first email %q, got %q", "alex@example.com", result.Users[0].Email)
	}
}
