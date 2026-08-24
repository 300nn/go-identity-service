package auth_test

import (
	"context"

	"github.com/300nn/go-identity-service/internal/outbox"

	"github.com/300nn/go-identity-service/internal/auth"
	"github.com/300nn/go-identity-service/internal/user"
)

type fakeTxFactory struct {
	userRepo     *fakeUserRepository
	refreshRepo  *fakeRefreshTokenRepository
	refreshStore auth.RefreshTokenStore
	outboxStore  *fakeOutboxStore
}

func newFakeTxFactory(
	userRepo *fakeUserRepository,
	refreshRepo *fakeRefreshTokenRepository,
	outboxStore *fakeOutboxStore,
) *fakeTxFactory {
	return &fakeTxFactory{
		userRepo:     userRepo,
		refreshRepo:  refreshRepo,
		refreshStore: refreshRepo,
		outboxStore:  outboxStore,
	}
}

func newFakeTxFactoryWithStores(
	userRepo *fakeUserRepository,
	refreshRepo *fakeRefreshTokenRepository,
	refreshStore auth.RefreshTokenStore,
	outboxStore *fakeOutboxStore,
) *fakeTxFactory {
	return &fakeTxFactory{
		userRepo:     userRepo,
		refreshRepo:  refreshRepo,
		refreshStore: refreshStore,
		outboxStore:  outboxStore,
	}
}

func (f *fakeTxFactory) WithinTx(
	_ context.Context,
	fn func(stores auth.TxStores) error,
) error {
	usersSnapshot := copyUsers(f.userRepo.users)
	userNextIDSnapshot := f.userRepo.nextID

	refreshSnapshot := copyRefreshTokens(f.refreshRepo.tokens)
	refreshNextIDSnapshot := f.refreshRepo.nextID

	outboxSnapshot := copyOutboxEvents(f.outboxStore.events)
	outboxNextIDSnapshot := f.outboxStore.nextID

	err := fn(auth.TxStores{
		UserStore:         f.userRepo,
		RefreshTokenStore: f.refreshStore,
		OutboxStore:       f.outboxStore,
	})
	if err != nil {
		f.userRepo.users = usersSnapshot
		f.userRepo.nextID = userNextIDSnapshot

		f.refreshRepo.tokens = refreshSnapshot
		f.refreshRepo.nextID = refreshNextIDSnapshot

		f.outboxStore.events = outboxSnapshot
		f.outboxStore.nextID = outboxNextIDSnapshot

		return err
	}

	return nil
}

func copyUsers(src map[int64]user.User) map[int64]user.User {
	dst := make(map[int64]user.User, len(src))

	for id, usr := range src {
		dst[id] = usr
	}

	return dst
}

func copyRefreshTokens(src map[int64]auth.RefreshToken) map[int64]auth.RefreshToken {
	dst := make(map[int64]auth.RefreshToken, len(src))

	for id, token := range src {
		dst[id] = token
	}

	return dst
}

func copyOutboxEvents(src map[int64]outbox.Event) map[int64]outbox.Event {
	dst := make(map[int64]outbox.Event, len(src))

	for id, event := range src {
		dst[id] = event
	}

	return dst
}
