package outbox_test

import (
	"context"
	"errors"

	"github.com/300nn/go-identity-service/internal/outbox"
)

var errPublishFailed = errors.New("publish failed")

type fakePublisher struct {
	published []outbox.Event
	err       error
}

func (p *fakePublisher) Publish(_ context.Context, event outbox.Event) error {
	if p.err != nil {
		return p.err
	}

	p.published = append(p.published, event)
	return nil
}
