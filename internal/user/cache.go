package user

import (
	"context"
	"time"
)

type Cache interface {
	GetUser(ctx context.Context, id int64) (User, bool, error)
	SetUser(ctx context.Context, user User, ttl time.Duration) error
	DeleteUser(ctx context.Context, id int64) error
}

type CacheConfig struct {
	UserTTL time.Duration `yaml:"user_ttl" env:"USER_TTL" env-default:"5m" validate:"gt=0"`
}
