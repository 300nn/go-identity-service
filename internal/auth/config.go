package auth

import (
	"context"
	"time"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type RateLimitConfig struct {
	LoginLimit  int           `yaml:"login_limit" env:"LOGIN_LIMIT" env-default:"5" validate:"gte=0"`
	LoginWindow time.Duration `yaml:"login_window" env:"LOGIN_WINDOW" env-default:"1m" validate:"gt=0"`

	RegisterLimit  int           `yaml:"register_limit" env:"REGISTER_LIMIT" env-default:"3" validate:"gte=0"`
	RegisterWindow time.Duration `yaml:"register_window" env:"REGISTER_WINDOW" env-default:"1m" validate:"gt=0"`

	RefreshLimit  int           `yaml:"refresh_limit" env:"REFRESH_LIMIT" env-default:"20" validate:"gte=0"`
	RefreshWindow time.Duration `yaml:"refresh_window" env:"REFRESH_WINDOW" env-default:"1m" validate:"gt=0"`
}
