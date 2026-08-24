package user

import (
	"CrudTutorialProject/internal/timex"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client  *redis.Client
	prefix  string
	timeout time.Duration
}

func NewRedisCache(client *redis.Client, prefix string, timeout time.Duration) *RedisCache {
	return &RedisCache{
		client:  client,
		prefix:  prefix,
		timeout: timeout,
	}
}

type cachedUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int32     `json:"age"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func cachedUserFromDomain(u User) cachedUser {
	return cachedUser{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Age:       u.Age,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (u cachedUser) toDomain() User {
	return User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Age:       u.Age,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func (c *RedisCache) GetUser(ctx context.Context, id int64) (User, bool, error) {
	ctx, cancel := timex.WithTimeout(ctx, c.timeout)
	defer cancel()

	raw, err := c.client.Get(ctx, c.userKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return User{}, false, nil
		}
		return User{}, false, fmt.Errorf("get user from redis failed: %w", err)
	}

	var cached cachedUser

	if err = json.Unmarshal([]byte(raw), &cached); err != nil {
		return User{}, false, fmt.Errorf("unmarshal cached user failed: %w", err)
	}

	return cached.toDomain(), true, nil
}

func (c *RedisCache) SetUser(ctx context.Context, u User, ttl time.Duration) error {
	ctx, cancel := timex.WithTimeout(ctx, c.timeout)
	defer cancel()

	data, err := json.Marshal(cachedUserFromDomain(u))

	if err != nil {
		return fmt.Errorf("marshal cached user failed: %w", err)
	}

	if err := c.client.Set(ctx, c.userKey(u.ID), data, ttl).Err(); err != nil {
		return fmt.Errorf("set user to redis cache failed: %w", err)
	}

	return nil
}

func (c *RedisCache) DeleteUser(ctx context.Context, id int64) error {
	ctx, cancel := timex.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.client.Del(ctx, c.userKey(id)).Err(); err != nil {
		return fmt.Errorf("delete user from redis cache failed: %w", err)
	}

	return nil
}

func (c *RedisCache) userKey(id int64) string {
	return c.prefix + ":user:id:" + strconv.FormatInt(id, 10)
}
