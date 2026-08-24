package auth_test

import (
	"context"
	"strings"
	"time"

	"github.com/300nn/go-identity-service/internal/user"
)

type fakeUserRepository struct {
	nextID int64
	users  map[int64]user.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		nextID: 1,
		users:  make(map[int64]user.User),
	}
}

func (r *fakeUserRepository) Create(ctx context.Context, usr user.User) (user.User, error) {
	if exists, _ := r.ExistsByEmail(ctx, usr.Email); exists {
		return user.User{}, user.ErrEmailAlreadyExists
	}

	now := time.Now().UTC()

	usr.ID = r.nextID
	usr.CreatedAt = now
	usr.UpdatedAt = now

	r.users[usr.ID] = usr
	r.nextID++

	return usr, nil
}

func (r *fakeUserRepository) FindByID(_ context.Context, id int64) (user.User, error) {
	found, ok := r.users[id]
	if !ok {
		return user.User{}, user.ErrUserNotFound
	}

	return found, nil
}

func (r *fakeUserRepository) FindByEmail(_ context.Context, email string) (user.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	for _, usr := range r.users {
		if strings.ToLower(usr.Email) == email {
			return usr, nil
		}
	}

	return user.User{}, user.ErrUserNotFound
}

func (r *fakeUserRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	for _, usr := range r.users {
		if strings.ToLower(usr.Email) == email {
			return true, nil
		}
	}

	return false, nil
}
