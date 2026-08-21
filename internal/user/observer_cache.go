package user

import (
	"context"
	"time"
)

type CacheObserver interface {
	Hit(cacheName string)
	Miss(cacheName string)
}

type ObservedCache struct {
	cache    Cache
	observer CacheObserver
	name     string
}

func NewObservedCache(cache Cache, observer CacheObserver, name string) *ObservedCache {
	return &ObservedCache{
		cache:    cache,
		observer: observer,
		name:     name,
	}
}

func (c *ObservedCache) GetUser(ctx context.Context, id int64) (User, bool, error) {
	usr, ok, err := c.cache.GetUser(ctx, id)

	if err == nil {
		if ok {
			c.observer.Hit(c.name)
		} else {
			c.observer.Miss(c.name)
		}
	}

	return usr, ok, err
}

func (c *ObservedCache) SetUser(ctx context.Context, user User, ttl time.Duration) error {
	return c.cache.SetUser(ctx, user, ttl)
}

func (c *ObservedCache) DeleteUser(ctx context.Context, id int64) error {
	return c.cache.DeleteUser(ctx, id)
}
