package ratelimit_test

import (
	"CrudTutorialProject/internal/ratelimit"
	"testing"
	"time"
)

func TestLimiter_Allow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	limiter := ratelimit.NewLimiterWithClock(func() time.Time {
		return now
	})

	key := "login:alex@example.com:127.0.0.1"

	if !limiter.Allow(key, 2, time.Minute) {
		t.Fatal("expected first request to be allowed")
	}

	if !limiter.Allow(key, 2, time.Minute) {
		t.Fatal("expected second request to be allowed")
	}

	if limiter.Allow(key, 2, time.Minute) {
		t.Fatal("expected third request to be rejected")
	}

	now = now.Add(time.Minute + time.Second)

	if !limiter.Allow(key, 2, time.Minute) {
		t.Fatal("expected request to be allowed after window reset")
	}
}

func TestLimiter_Allow_DifferentKeys(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	limiter := ratelimit.NewLimiterWithClock(func() time.Time {
		return now
	})

	if !limiter.Allow("key-1", 1, time.Minute) {
		t.Fatal("expected key-1 first request to be allowed")
	}

	if limiter.Allow("key-1", 1, time.Minute) {
		t.Fatal("expected key-1 second request to be rejected")
	}

	if !limiter.Allow("key-2", 1, time.Minute) {
		t.Fatal("expected key-2 first request to be allowed")
	}
}
