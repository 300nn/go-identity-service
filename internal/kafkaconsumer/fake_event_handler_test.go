package kafkaconsumer_test

import (
	"CrudTutorialProject/internal/kafkaconsumer"
	"context"
)

type fakeEventHandler struct {
	calls int
	err   error

	receivedEvent  kafkaconsumer.Event
	receivedStores kafkaconsumer.TxStores
}

func (h *fakeEventHandler) Handle(
	ctx context.Context,
	event kafkaconsumer.Event,
	stores kafkaconsumer.TxStores,
) error {
	h.calls++
	h.receivedEvent = event
	h.receivedStores = stores

	return h.err
}
