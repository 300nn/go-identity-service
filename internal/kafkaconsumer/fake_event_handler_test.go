package kafkaconsumer_test

import (
	"CrudTutorialProject/internal/kafkaconsumer"
	"context"
)

type fakeEventHandler struct {
	calls int
	err   error
}

func (h *fakeEventHandler) Handle(ctx context.Context, event kafkaconsumer.Event) error {
	h.calls++
	return h.err
}
