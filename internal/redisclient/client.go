package redisclient

import (
	"context"
	"fmt"

	"CrudTutorialProject/internal/config"

	"github.com/redis/go-redis/v9"
)

func New(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis client ping failed: %w", err)
	}

	return client, nil
}
