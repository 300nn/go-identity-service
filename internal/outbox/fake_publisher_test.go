package outbox_test

import (
	"CrudTutorialProject/internal/outbox"
	"context"
	"errors"
)

var errPublishFailed = errors.New("publish failed")

type fakePublisher struct {
	published []outbox.Event
	err       error
}

func (p *fakePublisher) Publish(ctx context.Context, event outbox.Event) error {
	if p.err != nil {
		return p.err
	}

	p.published = append(p.published, event)
	return nil
}
