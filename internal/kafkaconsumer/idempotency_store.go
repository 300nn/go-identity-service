package kafkaconsumer

import "context"

type IdempotencyStore interface {
	WasProcessed(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, event Event) error
}
