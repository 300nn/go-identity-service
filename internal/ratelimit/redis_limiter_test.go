package ratelimit_test

import (
	"CrudTutorialProject/internal/ratelimit"
	"CrudTutorialProject/internal/testutils"
	"testing"
	"time"
)

func TestRedisLimiter_Allow(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	limiter := ratelimit.NewRedisLimiter(redisClient, "go-crud")

	key := "auth:login:email_ip:alex@example.com:127.0.0.1"

	ok, err := limiter.Allow(ctx, key, 2, time.Minute)

	if err != nil {
		t.Fatalf("first Allow returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected first request to be allowed")
	}

	ok, err = limiter.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("second Allow returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected second request to be allowed")
	}

	ok, err = limiter.Allow(ctx, key, 2, time.Minute)
	if err != nil {
		t.Fatalf("third Allow returned error: %v", err)
	}
	if ok {
		t.Fatal("expected third request to be rejected")
	}
}

func TestRedisLimiter_Allow_SetsTTL(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	limiter := ratelimit.NewRedisLimiter(redisClient, "test")

	key := "auth:register:ip:127.0.0.1"
	redisKey := "test:" + key

	ok, err := limiter.Allow(ctx, key, 3, time.Minute)
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected request to be allowed")
	}

	ttl, err := redisClient.TTL(ctx, redisKey).Result()
	if err != nil {
		t.Fatalf("TTL returned error: %v", err)
	}

	if ttl <= 0 {
		t.Fatalf("expected TTL to be positive, got %v", ttl)
	}

	if ttl > time.Minute {
		t.Fatalf("expected TTL <= %v, got %v", time.Minute, ttl)
	}
}

func TestRedisLimiter_Allow_ResetsAfterWindow(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	limiter := ratelimit.NewRedisLimiter(redisClient, "test")

	key := "auth:refresh:ip:127.0.0.1"
	window := 100 * time.Millisecond

	ok, err := limiter.Allow(ctx, key, 1, window)
	if err != nil {
		t.Fatalf("first Allow returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected first request to be allowed")
	}

	ok, err = limiter.Allow(ctx, key, 1, window)
	if err != nil {
		t.Fatalf("second Allow returned error: %v", err)
	}
	if ok {
		t.Fatal("expected second request to be rejected")
	}

	time.Sleep(window + 50*time.Millisecond)

	ok, err = limiter.Allow(ctx, key, 1, window)
	if err != nil {
		t.Fatalf("third Allow returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected request to be allowed after window reset")
	}
}
