package ratelimit_test

import (
	"testing"
	"time"

	"CrudTutorialProject/internal/ratelimit"
)

func TestLimiter_Allow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	ctx := t.Context()

	limiter := ratelimit.NewLimiterWithClock(func() time.Time {
		return now
	})

	key := "login:alex@example.com:127.0.0.1"

	if ok, _ := limiter.Allow(ctx, key, 2, time.Minute); !ok {
		t.Fatal("expected first request to be allowed")
	}

	if ok, _ := limiter.Allow(ctx, key, 2, time.Minute); !ok {
		t.Fatal("expected second request to be allowed")
	}

	if ok, _ := limiter.Allow(ctx, key, 2, time.Minute); ok {
		t.Fatal("expected third request to be rejected")
	}

	now = now.Add(time.Minute + time.Second)

	if ok, _ := limiter.Allow(ctx, key, 2, time.Minute); !ok {
		t.Fatal("expected request to be allowed after window reset")
	}
}

func TestLimiter_Allow_DifferentKeys(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	ctx := t.Context()

	limiter := ratelimit.NewLimiterWithClock(func() time.Time {
		return now
	})

	if ok, _ := limiter.Allow(ctx, "key-1", 1, time.Minute); !ok {
		t.Fatal("expected key-1 first request to be allowed")
	}

	if ok, _ := limiter.Allow(ctx, "key-1", 1, time.Minute); ok {
		t.Fatal("expected key-1 second request to be rejected")
	}

	if ok, _ := limiter.Allow(ctx, "key-2", 1, time.Minute); !ok {
		t.Fatal("expected key-2 first request to be allowed")
	}
}
