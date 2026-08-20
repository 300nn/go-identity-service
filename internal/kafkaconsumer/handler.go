package kafkaconsumer

import "context"

type EventHandler interface {
	Handle(ctx context.Context, event Event, stores TxStores) error
}
