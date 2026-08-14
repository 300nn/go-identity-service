package auth_test

import (
	"context"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"
)

type fakeTxFactory struct {
	userRepo     *fakeUserRepository
	refreshRepo  *fakeRefreshTokenRepository
	refreshStore auth.RefreshTokenStore
}

func newFakeTxFactory(
	userRepo *fakeUserRepository,
	refreshRepo *fakeRefreshTokenRepository,
) *fakeTxFactory {
	return &fakeTxFactory{
		userRepo:     userRepo,
		refreshRepo:  refreshRepo,
		refreshStore: refreshRepo,
	}
}

func newFakeTxFactoryWithRefreshStore(
	userRepo *fakeUserRepository,
	refreshRepo *fakeRefreshTokenRepository,
	refreshStore auth.RefreshTokenStore,
) *fakeTxFactory {
	return &fakeTxFactory{
		userRepo:     userRepo,
		refreshRepo:  refreshRepo,
		refreshStore: refreshStore,
	}
}

func (f *fakeTxFactory) WithinTx(
	ctx context.Context,
	fn func(stores auth.TxStores) error,
) error {
	usersSnapshot := copyUsers(f.userRepo.users)
	userNextIDSnapshot := f.userRepo.nextID

	refreshSnapshot := copyRefreshTokens(f.refreshRepo.tokens)
	refreshNextIDSnapshot := f.refreshRepo.nextID

	err := fn(auth.TxStores{
		UserStore:         f.userRepo,
		RefreshTokenStore: f.refreshStore,
	})
	if err != nil {
		f.userRepo.users = usersSnapshot
		f.userRepo.nextID = userNextIDSnapshot

		f.refreshRepo.tokens = refreshSnapshot
		f.refreshRepo.nextID = refreshNextIDSnapshot

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
