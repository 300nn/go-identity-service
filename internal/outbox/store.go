package outbox

import "context"

type Store interface {
	Create(ctx context.Context, event Event) (Event, error)
}
