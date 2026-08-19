package outbox

import (
	"context"
	"time"
)

type Store interface {
	Create(ctx context.Context, event Event) (Event, error)
	FetchBatch(ctx context.Context, limit int, lockTimeout time.Duration) ([]Event, error)
	MarkProcessed(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, reason string, maxAttempts int) error
}
