package user_test

import (
	"context"
	"errors"
	"time"

	"github.com/300nn/go-identity-service/internal/outbox"
)

var errCreateOutboxEvent = errors.New("create outbox event failed")

type fakeOutboxStore struct {
	nextID int64
	events map[int64]outbox.Event

	createErr   error
	createCalls int
}

func newFakeOutboxStore() *fakeOutboxStore {
	return &fakeOutboxStore{
		nextID: 1,
		events: make(map[int64]outbox.Event),
	}
}

func (s *fakeOutboxStore) Create(_ context.Context, event outbox.Event) (outbox.Event, error) {
	s.createCalls++

	if s.createErr != nil {
		return outbox.Event{}, s.createErr
	}

	event.ID = s.nextID
	event.Status = outbox.StatusNew
	event.CreatedAt = time.Now().UTC()
	event.Payload = append([]byte(nil), event.Payload...)

	s.events[event.ID] = event
	s.nextID++

	return event, nil
}
