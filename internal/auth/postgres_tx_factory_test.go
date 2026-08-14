package auth_test

import (
	"errors"
	"testing"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/testutils"
	"CrudTutorialProject/internal/user"
)

func TestPostgresTxFactory_WithinTx_RollsBack(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := auth.NewPostgresTxFactory(pool)

	expectedErr := errors.New("force rollback")

	err := txFactory.WithinTx(ctx, func(stores auth.TxStores) error {
		_, err := stores.UserStore.Create(ctx, user.User{
			Name:         "Rollback",
			Email:        "rollback@example.com",
			Age:          25,
			PasswordHash: "hash",
			Role:         user.RoleUser,
		})
		if err != nil {
			return err
		}

		return expectedErr
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	userRepo := user.NewPostgresRepository(pool)

	_, err = userRepo.FindByEmail(ctx, "rollback@example.com")
	if err == nil {
		t.Fatal("expected user not to be persisted after rollback")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestPostgresTxFactory_WithinTx_Commits(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	txFactory := auth.NewPostgresTxFactory(pool)

	var createdID int64

	err := txFactory.WithinTx(ctx, func(stores auth.TxStores) error {
		created, err := stores.UserStore.Create(ctx, user.User{
			Name:         "Commit",
			Email:        "commit@example.com",
			Age:          25,
			PasswordHash: "hash",
			Role:         user.RoleUser,
		})
		if err != nil {
			return err
		}

		createdID = created.ID
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTx returned error: %v", err)
	}

	userRepo := user.NewPostgresRepository(pool)

	found, err := userRepo.FindByID(ctx, createdID)
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}

	if found.Email != "commit@example.com" {
		t.Fatalf("expected email %q, got %q", "commit@example.com", found.Email)
	}
}
