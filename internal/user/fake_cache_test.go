package user_test

import (
	"context"
	"time"

	"CrudTutorialProject/internal/user"
)

type fakeUserCache struct {
	users map[int64]user.User

	getCalls    int
	setCalls    int
	deleteCalls int

	getErr    error
	setErr    error
	deleteErr error
}

func newFakeUserCache() *fakeUserCache {
	return &fakeUserCache{
		users: make(map[int64]user.User),
	}
}

func (c *fakeUserCache) GetUser(ctx context.Context, id int64) (user.User, bool, error) {
	c.getCalls++

	if c.getErr != nil {
		return user.User{}, false, c.getErr
	}

	u, ok := c.users[id]
	return u, ok, nil
}

func (c *fakeUserCache) SetUser(ctx context.Context, u user.User, ttl time.Duration) error {
	c.setCalls++

	if c.setErr != nil {
		return c.setErr
	}

	c.users[u.ID] = u
	return nil
}

func (c *fakeUserCache) DeleteUser(ctx context.Context, id int64) error {
	c.deleteCalls++

	if c.deleteErr != nil {
		return c.deleteErr
	}

	delete(c.users, id)
	return nil
}
