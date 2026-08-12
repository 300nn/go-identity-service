package user_test

import (
	"CrudTutorialProject/internal/user"
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func countRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()

	var count int64

	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	return count
}

func TestPostgresTxRepositoryFactory_WithinTx_Commits(t *testing.T) {
	ctx := t.Context()

	pool := newTestPostgresPool(t)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)

	var createdID int64

	err := txFactory.WithinTx(ctx, func(repo user.Repository) error {
		created, err := repo.Create(ctx, user.User{
			Name:         "Alex",
			Email:        "alex.tx.commit@example.com",
			Age:          25,
			PasswordHash: "password-hash",
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

	repo := user.NewPostgresRepository(pool)

	found, err := repo.FindByID(ctx, createdID)
	if err != nil {
		t.Fatalf("FindByID after commit returned error: %v", err)
	}

	if found.Email != "alex.tx.commit@example.com" {
		t.Fatalf("expected email %q, got %q", "alex.tx.commit@example.com", found.Email)
	}
}

func TestPostgresTxRepositoryFactory_WithinTx_RollsBack(t *testing.T) {
	ctx := t.Context()

	pool := newTestPostgresPool(t)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)

	expectedErr := errors.New("force rollback")

	err := txFactory.WithinTx(ctx, func(repo user.Repository) error {
		_, err := repo.Create(ctx, user.User{
			Name:         "Rollback",
			Email:        "rollback@example.com",
			Age:          25,
			PasswordHash: "password-hash",
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
		t.Fatalf("expected rollback error %v, got %v", expectedErr, err)
	}

	repo := user.NewPostgresRepository(pool)

	_, err = repo.FindByEmail(ctx, "rollback@example.com")
	if err == nil {
		t.Fatal("expected user not to be persisted after rollback")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound after rollback, got %v", err)
	}
}

func TestService_CreateUserWithProfile_Commits(t *testing.T) {
	ctx := t.Context()

	pool := newTestPostgresPool(t)

	repo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	service := user.NewService(repo, txFactory)

	created, err := service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:  "Alex",
		Email: "alex.with.profile@example.com",
		Age:   25,
		Bio:   "Go backend developer",
	})
	if err != nil {
		t.Fatalf("CreateUserWithProfile returned error: %v", err)
	}

	if created.User.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	if created.Profile.ID == 0 {
		t.Fatal("expected profile ID to be set")
	}

	if created.Profile.UserID != created.User.ID {
		t.Fatalf("expected profile user_id %d, got %d", created.User.ID, created.Profile.UserID)
	}

	usersCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM users WHERE id = $1`,
		created.User.ID,
	)

	if usersCount != 1 {
		t.Fatalf("expected users count %d, got %d", 1, usersCount)
	}

	profilesCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM user_profiles WHERE user_id = $1`,
		created.User.ID,
	)

	if profilesCount != 1 {
		t.Fatalf("expected profiles count %d, got %d", 1, profilesCount)
	}

	eventsCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM user_events WHERE user_id = $1`,
		created.User.ID,
	)

	if eventsCount != 1 {
		t.Fatalf("expected events count %d, got %d", 1, eventsCount)
	}
}

func TestService_CreateUserWithProfile_DuplicateEmail(t *testing.T) {
	ctx := t.Context()

	pool := newTestPostgresPool(t)

	repo := user.NewPostgresRepository(pool)
	txFactory := user.NewPostgresTxRepositoryFactory(pool)
	service := user.NewService(repo, txFactory)

	_, err := service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:  "Alex",
		Email: "duplicate.profile@example.com",
		Age:   25,
		Bio:   "First profile",
	})
	if err != nil {
		t.Fatalf("first CreateUserWithProfile returned error: %v", err)
	}

	_, err = service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:  "Another Alex",
		Email: "duplicate.profile@example.com",
		Age:   30,
		Bio:   "Second profile",
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	usersCount := countRows(
		t,
		ctx,
		pool,
		`SELECT count(*) FROM users WHERE email = $1`,
		"duplicate.profile@example.com",
	)

	if usersCount != 1 {
		t.Fatalf("expected users count %d, got %d", 1, usersCount)
	}

	profilesCount := countRows(
		t,
		ctx,
		pool,
		`
		SELECT count(*)
		FROM user_profiles p
		JOIN users u ON u.id = p.user_id
		WHERE u.email = $1
		`,
		"duplicate.profile@example.com",
	)

	if profilesCount != 1 {
		t.Fatalf("expected profiles count %d, got %d", 1, profilesCount)
	}

	eventsCount := countRows(
		t,
		ctx,
		pool,
		`
		SELECT count(*)
		FROM user_events e
		JOIN users u ON u.id = e.user_id
		WHERE u.email = $1
		`,
		"duplicate.profile@example.com",
	)

	if eventsCount != 1 {
		t.Fatalf("expected events count %d, got %d", 1, eventsCount)
	}
}
