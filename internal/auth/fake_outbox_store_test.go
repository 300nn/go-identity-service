package auth_test

import (
	"context"
	"errors"

	"CrudTutorialProject/internal/outbox"
)

var errCreateOutboxEventFailed = errors.New("create outbox event failed")

type fakeOutboxStore struct {
	nextID int64
	events map[int64]outbox.Event

	createErr error
}

func newFakeOutboxStore() *fakeOutboxStore {
	return &fakeOutboxStore{
		nextID: 1,
		events: make(map[int64]outbox.Event),
	}
}

func (s *fakeOutboxStore) Create(ctx context.Context, event outbox.Event) (outbox.Event, error) {
	if s.createErr != nil {
		return outbox.Event{}, s.createErr
	}

	event.ID = s.nextID
	event.Status = outbox.StatusNew

	s.events[event.ID] = event
	s.nextID++

	return event, nil
}
