package user_test

import (
	"context"

	"github.com/300nn/go-identity-service/internal/outbox"
	"github.com/300nn/go-identity-service/internal/user"
)

type fakeTxFactory struct {
	repo        *FakeRepository
	outboxStore *fakeOutboxStore

	withinTxCalls int
}

func newFakeTxFactory(repo *FakeRepository, outboxStore *fakeOutboxStore) *fakeTxFactory {
	return &fakeTxFactory{
		repo:        repo,
		outboxStore: outboxStore,
	}
}

func (f *fakeTxFactory) WithinTx(_ context.Context, fn func(stores user.TxStores) error) error {
	f.withinTxCalls++

	usersSnapshot := copyUsers(f.repo.users)
	profilesSnapshot := copyProfiles(f.repo.profiles)
	userEventsSnapshot := copyUserEvents(f.repo.events)
	outboxSnapshot := copyOutboxEvents(f.outboxStore.events)

	nextUserIDSnapshot := f.repo.nextUserID
	nextProfileIDSnapshot := f.repo.nextProfileID
	nextEventIDSnapshot := f.repo.nextEventID
	nextOutboxIDSnapshot := f.outboxStore.nextID

	err := fn(user.TxStores{
		UserRepo:    f.repo,
		OutBoxStore: f.outboxStore,
	})
	if err == nil {
		return nil
	}

	f.repo.users = usersSnapshot
	f.repo.profiles = profilesSnapshot
	f.repo.events = userEventsSnapshot
	f.repo.nextUserID = nextUserIDSnapshot
	f.repo.nextProfileID = nextProfileIDSnapshot
	f.repo.nextEventID = nextEventIDSnapshot

	f.outboxStore.events = outboxSnapshot
	f.outboxStore.nextID = nextOutboxIDSnapshot

	return err
}

func copyUsers(src map[int64]user.User) map[int64]user.User {
	dst := make(map[int64]user.User, len(src))
	for id, usr := range src {
		dst[id] = usr
	}
	return dst
}

func copyProfiles(src map[int64]user.Profile) map[int64]user.Profile {
	dst := make(map[int64]user.Profile, len(src))
	for id, profile := range src {
		dst[id] = profile
	}
	return dst
}

func copyUserEvents(src map[int64]user.Event) map[int64]user.Event {
	dst := make(map[int64]user.Event, len(src))
	for id, event := range src {
		dst[id] = event
	}
	return dst
}

func copyOutboxEvents(src map[int64]outbox.Event) map[int64]outbox.Event {
	dst := make(map[int64]outbox.Event, len(src))
	for id, event := range src {
		event.Payload = append([]byte(nil), event.Payload...)
		dst[id] = event
	}
	return dst
}
