package testutils

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	postgresOnce sync.Once
	postgresMu   sync.Mutex

	postgresPool      *pgxpool.Pool
	postgresContainer *postgres.PostgresContainer
	postgresErr       error
)

func NewTestPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	postgresMu.Lock()

	t.Cleanup(func() {
		if postgresPool != nil {
			cleanPostgres(t, postgresPool)
		}
		postgresMu.Unlock()
	})

	pool := sharedPostgresPool(t)

	cleanPostgres(t, pool)

	return pool
}

func sharedPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	postgresOnce.Do(func() {
		ctx := context.Background()

		postgresContainer, postgresErr = postgres.Run(
			ctx,
			"postgres:17",
			postgres.WithDatabase("identity_service_test"),
			postgres.WithUsername("identity_service"),
			postgres.WithPassword("identity_service"),
			postgres.BasicWaitStrategies(),
		)
		if postgresErr != nil {
			return
		}

		dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			postgresErr = err
			return
		}

		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			postgresErr = err
			return
		}

		if err := goose.SetDialect("postgres"); err != nil {
			_ = sqlDB.Close()
			postgresErr = err
			return
		}

		if err := goose.UpContext(ctx, sqlDB, migrationsDir(t)); err != nil {
			_ = sqlDB.Close()
			postgresErr = err
			return
		}

		_ = sqlDB.Close()

		poolConfig, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			postgresErr = err
			return
		}

		poolConfig.MaxConns = 8
		poolConfig.MinConns = 1
		poolConfig.MaxConnLifetime = time.Hour
		poolConfig.MaxConnIdleTime = 30 * time.Minute

		postgresPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			postgresErr = err
			return
		}

		if err := postgresPool.Ping(ctx); err != nil {
			postgresErr = err
			return
		}
	})

	if postgresErr != nil {
		t.Fatalf("start shared postgres: %v", postgresErr)
	}

	return postgresPool
}

func cleanPostgres(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const query = `
		TRUNCATE TABLE
		    user_audit_events,
		    processed_kafka_events,
			outbox_events,
			refresh_tokens,
			user_profiles,
			users
		RESTART IDENTITY CASCADE
	`

	if _, err := pool.Exec(ctx, query); err != nil {
		t.Fatalf("clean postgres: %v", err)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("get current file path")
	}

	return filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
}
