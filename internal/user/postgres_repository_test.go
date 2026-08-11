package user_test

import (
	"CrudTutorialProject/internal/user"
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:17",
		postgres.WithDatabase("go_crud_test"),
		postgres.WithUsername("go_crud"),
		postgres.WithPassword("go_crud"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")

	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}

	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db: %v", err)
		}
	})

	migratePostgres(t, ctx, sqlDB)

	poolConfig, err := pgxpool.ParseConfig(dsn)

	if err != nil {
		t.Fatalf("parse pgxpool config: %v", err)
	}

	poolConfig.MaxConns = 4
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		t.Fatalf("create pgxpool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	return pool
}

func migratePostgres(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	if err := goose.SetDialect("pgx"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}

	if err := goose.UpContext(ctx, db, migrationsDir(t)); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	_, fileName, _, ok := runtime.Caller(0)

	if !ok {
		t.Fatal("get current file path")
	}

	return filepath.Join(filepath.Dir(fileName), "..", "..", "migrations")
}

func TestPostgresRepository_FindByID(t *testing.T) {
	ctx := t.Context()

	pool := newTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
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

	pool := newTestPostgresPool(t)
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

	pool := newTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:  "Another Alex",
		Email: "alex@example.com",
		Age:   30,
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

	pool := newTestPostgresPool(t)
	repo := user.NewPostgresRepository(pool)

	_, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("create Alex: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   30,
	})
	if err != nil {
		t.Fatalf("create Bob: %v", err)
	}

	_, err = repo.Create(ctx, user.User{
		Name:  "Alice",
		Email: "alice@test.com",
		Age:   22,
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
