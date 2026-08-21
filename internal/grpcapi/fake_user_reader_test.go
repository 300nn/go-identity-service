package grpcapi_test

import (
	"context"
	"errors"

	"CrudTutorialProject/internal/user"
)

var errFakeUserReaderFailed = errors.New("fake user reader failed")

type fakeUserReader struct {
	users map[int64]user.User
	list  user.ListUsersOutput

	getCalls  int
	listCalls int

	getErr  error
	listErr error
}

func newFakeUserReader() *fakeUserReader {
	return &fakeUserReader{
		users: make(map[int64]user.User),
	}
}

func (r *fakeUserReader) GetUser(ctx context.Context, id int64) (user.User, error) {
	r.getCalls++

	if r.getErr != nil {
		return user.User{}, r.getErr
	}

	usr, ok := r.users[id]
	if !ok {
		return user.User{}, user.NewUserNotFoundError()
	}

	return usr, nil
}

func (r *fakeUserReader) ListUsers(ctx context.Context, input user.ListUsersInput) (user.ListUsersOutput, error) {
	r.listCalls++

	if r.listErr != nil {
		return user.ListUsersOutput{}, r.listErr
	}

	return r.list, nil
}
