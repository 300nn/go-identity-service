package user_test

import (
	"CrudTutorialProject/internal/apperror"
	"CrudTutorialProject/internal/user"
	"errors"
	"testing"
	"time"
)

func TestService_CreateUser_Success(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	got, err := service.CreateUser(
		ctx,
		user.CreateUserInput{
			Email: "a@asdf.ru",
			Name:  "John",
			Age:   25,
		},
	)

	if err != nil {
		t.Fatalf("Create user returned error: %v", err)
	}

	if got.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	if got.Name != "John" {
		t.Fatalf("expected name %q, got %q", "Alex", got.Name)
	}

	if got.Email != "a@asdf.ru" {
		t.Fatalf("expected email %q, got %q", "alex@example.com", got.Email)
	}

	if got.Age != 25 {
		t.Fatalf("expected age %d, got %d", 25, got.Age)
	}

	if got.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if got.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestService_CreateUser_DuplicateEmail(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	_, err := service.CreateUser(ctx, user.CreateUserInput{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("first CreateUser returned error: %v", err)
	}

	_, err = service.CreateUser(ctx, user.CreateUserInput{
		Name:  "Another Alex",
		Email: " Alex@Example.com ",
		Age:   30,
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestService_CreateUser_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input user.CreateUserInput
	}{
		{
			name: "empty name",
			input: user.CreateUserInput{
				Name:  "",
				Email: "alex@example.com",
				Age:   25,
			},
		},
		{
			name: "short name",
			input: user.CreateUserInput{
				Name:  "A",
				Email: "alex@example.com",
				Age:   25,
			},
		},
		{
			name: "empty email",
			input: user.CreateUserInput{
				Name:  "Alex",
				Email: "",
				Age:   25,
			},
		},
		{
			name: "invalid email",
			input: user.CreateUserInput{
				Name:  "Alex",
				Email: "wrong-email",
				Age:   25,
			},
		},
		{
			name: "negative age",
			input: user.CreateUserInput{
				Name:  "Alex",
				Email: "alex@example.com",
				Age:   -1,
			},
		},
		{
			name: "too large age",
			input: user.CreateUserInput{
				Name:  "Alex",
				Email: "alex@example.com",
				Age:   151,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			repo := newFakeRepository()
			service := user.NewService(repo, nil)

			_, err := service.CreateUser(ctx, tt.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestService_CreateUser_ValidationFields(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	_, err := service.CreateUser(ctx, user.CreateUserInput{
		Name:  "",
		Email: "wrong-email",
		Age:   -1,
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var validationErr *apperror.FieldValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected FieldValidationError, got %T: %v", err, err)
	}

	if validationErr.Fields["name"] == "" {
		t.Fatal("expected validation error for field name")
	}

	if validationErr.Fields["email"] == "" {
		t.Fatal("expected validation error for field email")
	}

	if validationErr.Fields["age"] == "" {
		t.Fatal("expected validation error for field age")
	}
}

func TestService_GetUser_Success(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	got, err := service.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestService_GetUser_NotFound(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	_, err := service.GetUser(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestService_GetUser_InvalidID(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	_, err := service.GetUser(ctx, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, user.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestService_ListUsers(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	service := user.NewService(repo, nil)

	_, _ = repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})

	_, _ = repo.Create(ctx, user.User{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   30,
	})

	_, _ = repo.Create(ctx, user.User{
		Name:  "Alice",
		Email: "alice@user_test.com",
		Age:   22,
	})

	result, err := service.ListUsers(ctx, user.ListUsersInput{
		Limit:  10,
		Offset: 0,
		Email:  "example",
		Sort:   "id_asc",
	})
	if err != nil {
		t.Fatalf("ListUsers returned error: %v", err)
	}

	if result.Total != 2 {
		t.Fatalf("expected total %d, got %d", 2, result.Total)
	}

	if len(result.Users) != 2 {
		t.Fatalf("expected users len %d, got %d", 2, len(result.Users))
	}

	if result.Users[0].Email != "alex@example.com" {
		t.Fatalf("expected first user email %q, got %q", "alex@example.com", result.Users[0].Email)
	}
}

func TestService_GetUser_ReturnsFromCache(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	cache := newFakeUserCache()

	cachedUser := user.User{
		ID:    1,
		Name:  "Cached Alex",
		Email: "cached@example.com",
		Age:   25,
		Role:  user.RoleUser,
	}

	cache.users[cachedUser.ID] = cachedUser

	service := user.NewService(
		repo,
		nil,
		user.WithCache(cache, time.Minute),
	)

	found, err := service.GetUser(ctx, cachedUser.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if found.Email != cachedUser.Email {
		t.Fatalf("expected email %q, got %q", cachedUser.Email, found.Email)
	}

	if repo.findByIDCalls != 0 {
		t.Fatalf("expected repo not to be called, got %d calls", repo.findByIDCalls)
	}
}

func TestService_GetUser_CacheMissLoadsFromRepoAndStoresCache(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	cache := newFakeUserCache()

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		repo,
		nil,
		user.WithCache(cache, time.Minute),
	)

	found, err := service.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, found.ID)
	}

	if cache.setCalls != 1 {
		t.Fatalf("expected cache set calls 1, got %d", cache.setCalls)
	}

	cached, ok := cache.users[created.ID]
	if !ok {
		t.Fatal("expected user to be stored in cache")
	}

	if cached.Email != created.Email {
		t.Fatalf("expected cached email %q, got %q", created.Email, cached.Email)
	}
}

func TestService_GetUser_CacheErrorFallsBackToRepo(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	cache := newFakeUserCache()
	cache.getErr = errors.New("redis unavailable")

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		repo,
		nil,
		user.WithCache(cache, time.Minute),
	)

	found, err := service.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, found.ID)
	}
}

func TestService_UpdateUser_UpdatesCache(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	cache := newFakeUserCache()

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		repo,
		nil,
		user.WithCache(cache, time.Minute),
	)

	updated, err := service.UpdateUser(ctx, created.ID, user.UpdateUserInput{
		Name:  "Updated Alex",
		Email: "updated@example.com",
		Age:   26,
	})
	if err != nil {
		t.Fatalf("UpdateUser returned error: %v", err)
	}

	cached, ok := cache.users[created.ID]
	if !ok {
		t.Fatal("expected updated user to be cached")
	}

	if cached.Email != updated.Email {
		t.Fatalf("expected cached email %q, got %q", updated.Email, cached.Email)
	}
}

func TestService_DeleteUser_DeletesCache(t *testing.T) {
	ctx := t.Context()

	repo := newFakeRepository()
	cache := newFakeUserCache()

	created, err := repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	cache.users[created.ID] = created

	service := user.NewService(
		repo,
		nil,
		user.WithCache(cache, time.Minute),
	)

	if err := service.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	if _, ok := cache.users[created.ID]; ok {
		t.Fatal("expected user to be deleted from cache")
	}
}
