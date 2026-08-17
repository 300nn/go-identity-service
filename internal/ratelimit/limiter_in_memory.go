package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	entries map[string]entry
	now     func() time.Time
}

type entry struct {
	count   int
	resetAt time.Time
}

func NewLimiter() *Limiter {
	return &Limiter{
		entries: make(map[string]entry),
		now:     time.Now,
	}
}

func NewLimiterWithClock(now func() time.Time) *Limiter {
	return &Limiter{
		entries: make(map[string]entry),
		now:     now,
	}
}

func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	if limit <= 0 {
		return true
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	current, ok := l.entries[key]
	if !ok || !current.resetAt.After(now) {
		l.entries[key] = entry{
			count:   1,
			resetAt: now.Add(window),
		}
		return true
	}

	if current.count >= limit {
		return false
	}

	current.count++
	l.entries[key] = current

	return true
}
