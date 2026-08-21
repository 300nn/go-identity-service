package grpcapi_test

import (
	userapiv1 "CrudTutorialProject/internal/gen/api/user/v1"
	"CrudTutorialProject/internal/grpcapi"
	"context"
	"testing"
	"time"

	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func contextWithPrincipal(userID int64, role user.Role) context.Context {
	return auth.ContextWithUser(context.Background(), auth.Principal{
		UserID: userID,
		Email:  "user@example.com",
		Role:   role,
	})
}

func assertGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()

	if status.Code(err) != code {
		t.Fatalf("expected gRPC code %s, got %s, err: %v", code, status.Code(err), err)
	}
}

func testUser(id int64, role user.Role) user.User {
	now := time.Now().UTC()

	return user.User{
		ID:        id,
		Name:      "Alex",
		Email:     "alex@example.com",
		Age:       25,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestUserService_GetUser_Unauthenticated(t *testing.T) {
	reader := newFakeUserReader()
	reader.users[1] = testUser(1, user.RoleUser)

	service := grpcapi.NewUserService(reader)

	_, err := service.GetUser(context.Background(), &userapiv1.GetUserRequest{
		Id: 1,
	})
	assertGRPCCode(t, err, codes.Unauthenticated)
}

func TestUserService_GetUser_Self(t *testing.T) {
	reader := newFakeUserReader()
	reader.users[1] = testUser(1, user.RoleUser)

	service := grpcapi.NewUserService(reader)

	resp, err := service.GetUser(
		contextWithPrincipal(1, user.RoleUser),
		&userapiv1.GetUserRequest{
			Id: 1,
		},
	)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if resp.GetUser().GetId() != 1 {
		t.Fatalf("expected user id 1, got %d", resp.GetUser().GetId())
	}

	if reader.getCalls != 1 {
		t.Fatalf("expected GetUser calls 1, got %d", reader.getCalls)
	}
}

func TestUserService_GetUser_OtherUserForbidden(t *testing.T) {
	reader := newFakeUserReader()
	reader.users[2] = testUser(2, user.RoleUser)

	service := grpcapi.NewUserService(reader)

	_, err := service.GetUser(
		contextWithPrincipal(1, user.RoleUser),
		&userapiv1.GetUserRequest{
			Id: 2,
		},
	)

	assertGRPCCode(t, err, codes.PermissionDenied)

	if reader.getCalls != 0 {
		t.Fatalf("expected GetUser not to be called, got %d", reader.getCalls)
	}
}

func TestUserService_GetUser_AdminCanGetAnyUser(t *testing.T) {
	reader := newFakeUserReader()
	reader.users[2] = testUser(2, user.RoleUser)

	service := grpcapi.NewUserService(reader)

	resp, err := service.GetUser(
		contextWithPrincipal(1, user.RoleAdmin),
		&userapiv1.GetUserRequest{
			Id: 2,
		},
	)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if resp.GetUser().GetId() != 2 {
		t.Fatalf("expected user id 2, got %d", resp.GetUser().GetId())
	}
}

func TestUserService_ListUsers_Unauthenticated(t *testing.T) {
	reader := newFakeUserReader()
	service := grpcapi.NewUserService(reader)

	_, err := service.ListUsers(context.Background(), &userapiv1.ListUsersRequest{
		Limit: 20,
	})

	assertGRPCCode(t, err, codes.Unauthenticated)
}

func TestUserService_ListUsers_UserForbidden(t *testing.T) {
	reader := newFakeUserReader()
	service := grpcapi.NewUserService(reader)

	_, err := service.ListUsers(
		contextWithPrincipal(1, user.RoleUser),
		&userapiv1.ListUsersRequest{
			Limit: 20,
		},
	)

	assertGRPCCode(t, err, codes.PermissionDenied)

	if reader.listCalls != 0 {
		t.Fatalf("expected ListUsers not to be called, got %d", reader.listCalls)
	}
}

func TestUserService_ListUsers_Admin(t *testing.T) {
	reader := newFakeUserReader()
	reader.list = user.ListUsersOutput{
		Users: []user.User{
			testUser(1, user.RoleUser),
			testUser(2, user.RoleAdmin),
		},
		Total: 2,
	}

	service := grpcapi.NewUserService(reader)

	resp, err := service.ListUsers(
		contextWithPrincipal(1, user.RoleAdmin),
		&userapiv1.ListUsersRequest{
			Limit:  20,
			Offset: 0,
			Sort:   "id_asc",
		},
	)
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if len(resp.GetUsers()) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.GetUsers()))
	}

	if resp.GetTotal() != 2 {
		t.Fatalf("expected total 2, got %d", resp.GetTotal())
	}

	if reader.listCalls != 1 {
		t.Fatalf("expected ListUsers calls 1, got %d", reader.listCalls)
	}
}

func TestUserService_ListUsers_InvalidLimit(t *testing.T) {
	reader := newFakeUserReader()
	service := grpcapi.NewUserService(reader)

	_, err := service.ListUsers(
		contextWithPrincipal(1, user.RoleAdmin),
		&userapiv1.ListUsersRequest{
			Limit: 101,
		},
	)

	assertGRPCCode(t, err, codes.InvalidArgument)

	if reader.listCalls != 0 {
		t.Fatalf("expected ListUsers not to be called, got %d", reader.listCalls)
	}
}
