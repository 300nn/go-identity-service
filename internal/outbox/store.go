package outbox

import "context"

type Store interface {
	Create(ctx context.Context, event Event) (Event, error)
	FetchBatch(ctx context.Context, limit int) ([]Event, error)
	MarkProcessed(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, reason string, maxAttempts int) error
}
