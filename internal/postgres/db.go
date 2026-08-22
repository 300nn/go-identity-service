package postgres

import (
	"context"
	"fmt"
	"time"

	"CrudTutorialProject/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, cfg *config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseUrlWithSSL())

	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolConfig.MaxConns = cfg.Pool.MaxCons
	poolConfig.MinConns = cfg.Pool.MinCons
	poolConfig.MaxConnLifetime = cfg.Pool.MaxConLifeTime
	poolConfig.MaxConnIdleTime = cfg.Pool.MaxConIdleTime
	poolConfig.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
