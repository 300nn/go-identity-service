package user_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"CrudTutorialProject/internal/apperror"
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/eventcodec"
	"CrudTutorialProject/internal/outbox"
	"CrudTutorialProject/internal/user"

	"golang.org/x/crypto/bcrypt"
)

type testApp struct {
	repo        *FakeRepository
	outboxStore *fakeOutboxStore
	txFactory   *fakeTxFactory
	service     *user.Service
	hasher      user.Hasher
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	repo := newFakeRepository()
	outboxStore := newFakeOutboxStore()
	txFactory := newFakeTxFactory(repo, outboxStore)
	service := user.NewService(repo, txFactory, hasher)

	return &testApp{
		repo:        repo,
		outboxStore: outboxStore,
		txFactory:   txFactory,
		service:     service,
		hasher:      hasher,
	}
}

func TestService_CreateUser_Success(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	got, err := app.service.CreateUser(
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
		t.Fatalf("expected name %q, got %q", "John", got.Name)
	}

	if got.Email != "a@asdf.ru" {
		t.Fatalf("expected email %q, got %q", "a@asdf.ru", got.Email)
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

	assertUserRegisteredOutboxEvent(t, app.outboxStore, got)
}

func TestService_CreateUser_DuplicateEmail(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	_, err := app.service.CreateUser(ctx, user.CreateUserInput{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("first CreateUser returned error: %v", err)
	}

	_, err = app.service.CreateUser(ctx, user.CreateUserInput{
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

	if len(app.outboxStore.events) != 1 {
		t.Fatalf("expected exactly 1 outbox event after duplicate attempt, got %d", len(app.outboxStore.events))
	}
}

func TestService_CreateUser_OutboxFailure_RollsBackUser(t *testing.T) {
	ctx := t.Context()
	app := newTestApp(t)
	app.outboxStore.createErr = errCreateOutboxEvent

	got, err := app.service.CreateUser(ctx, user.CreateUserInput{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected CreateUser error, got nil")
	}

	if !errors.Is(err, errCreateOutboxEvent) {
		t.Fatalf("expected outbox error, got %v", err)
	}
	if got.ID != 0 {
		t.Fatalf("expected zero user on rollback, got ID %d", got.ID)
	}

	exists, err := app.repo.ExistsByEmail(ctx, "alex@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail returned error: %v", err)
	}
	if exists {
		t.Fatal("expected created user to be rolled back")
	}

	if len(app.outboxStore.events) != 0 {
		t.Fatalf("expected no outbox events after rollback, got %d", len(app.outboxStore.events))
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

			app := newTestApp(t)

			_, err := app.service.CreateUser(ctx, tt.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestService_CreateUser_ValidationFields(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	_, err := app.service.CreateUser(ctx, user.CreateUserInput{
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

func TestService_CreateUserWithProfile_CreatesOutboxEvent(t *testing.T) {
	ctx := t.Context()
	app := newTestApp(t)

	result, err := app.service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:     "Alex",
		Email:    "alex.profile@example.com",
		Age:      25,
		Password: "password123",
		Bio:      "Go developer",
	})
	if err != nil {
		t.Fatalf("CreateUserWithProfile returned error: %v", err)
	}

	if result.Profile.UserID != result.User.ID {
		t.Fatalf("expected profile user id %d, got %d", result.User.ID, result.Profile.UserID)
	}

	assertUserRegisteredOutboxEvent(t, app.outboxStore, result.User)
}

func TestService_CreateUserWithProfile_OutboxFailure_RollsBackUserAndProfile(t *testing.T) {
	ctx := t.Context()
	app := newTestApp(t)
	app.outboxStore.createErr = errCreateOutboxEvent

	_, err := app.service.CreateUserWithProfile(ctx, user.CreateUserWithProfileInput{
		Name:     "Alex",
		Email:    "alex.profile@example.com",
		Age:      25,
		Password: "password123",
		Bio:      "Go developer",
	})
	if err == nil {
		t.Fatal("expected CreateUserWithProfile error, got nil")
	}

	if !errors.Is(err, errCreateOutboxEvent) {
		t.Fatalf("expected outbox error, got %v", err)
	}

	exists, err := app.repo.ExistsByEmail(ctx, "alex.profile@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail returned error: %v", err)
	}
	if exists {
		t.Fatal("expected user to be rolled back")
	}

	if len(app.repo.profiles) != 0 {
		t.Fatalf("expected profiles to be rolled back, got %d", len(app.repo.profiles))
	}

	if len(app.outboxStore.events) != 0 {
		t.Fatalf("expected outbox events to be rolled back, got %d", len(app.outboxStore.events))
	}
}

func TestService_GetUser_Success(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	created, err := app.repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	got, err := app.service.GetUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("expected ID %d, got %d", created.ID, got.ID)
	}
}

func TestService_GetUser_NotFound(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	_, err := app.service.GetUser(ctx, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestService_GetUser_InvalidID(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	_, err := app.service.GetUser(ctx, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, user.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestService_ListUsers(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)

	_, _ = app.repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
	})

	_, _ = app.repo.Create(ctx, user.User{
		Name:  "Bob",
		Email: "bob@example.com",
		Age:   30,
	})

	_, _ = app.repo.Create(ctx, user.User{
		Name:  "Alice",
		Email: "alice@user_test.com",
		Age:   22,
	})

	result, err := app.service.ListUsers(ctx, user.ListUsersInput{
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

	app := newTestApp(t)
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
		app.repo,
		nil,
		app.hasher,
		user.WithCache(cache, time.Minute),
	)

	found, err := service.GetUser(ctx, cachedUser.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if found.Email != cachedUser.Email {
		t.Fatalf("expected email %q, got %q", cachedUser.Email, found.Email)
	}

	if app.repo.findByIDCalls != 0 {
		t.Fatalf("expected repo not to be called, got %d calls", app.repo.findByIDCalls)
	}
}

func TestService_GetUser_CacheMissLoadsFromRepoAndStoresCache(t *testing.T) {
	ctx := t.Context()

	app := newTestApp(t)
	cache := newFakeUserCache()

	created, err := app.repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		app.repo,
		nil,
		app.hasher,
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

	app := newTestApp(t)
	cache := newFakeUserCache()
	cache.getErr = errors.New("redis unavailable")

	created, err := app.repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		app.repo,
		nil,
		app.hasher,
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

	app := newTestApp(t)
	cache := newFakeUserCache()

	created, err := app.repo.Create(ctx, user.User{
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	service := user.NewService(
		app.repo,
		nil,
		app.hasher,
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

	_, err = service.GetUser(ctx, created.ID)

	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
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

	app := newTestApp(t)
	cache := newFakeUserCache()

	created, err := app.repo.Create(ctx, user.User{
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
		app.repo,
		nil,
		app.hasher,
		user.WithCache(cache, time.Minute),
	)

	if err := service.DeleteUser(ctx, created.ID); err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	if _, ok := cache.users[created.ID]; ok {
		t.Fatal("expected user to be deleted from cache")
	}
}

func assertUserRegisteredOutboxEvent(t *testing.T, store *fakeOutboxStore, usr user.User) {
	t.Helper()

	if len(store.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(store.events))
	}

	var event outbox.Event
	for _, stored := range store.events {
		event = stored
	}

	if event.EventType != outbox.EventTypeUserRegistered {
		t.Fatalf("expected event type %q, got %q", outbox.EventTypeUserRegistered, event.EventType)
	}
	if event.AggregateType != outbox.AggregateUser {
		t.Fatalf("expected aggregate type %q, got %q", outbox.AggregateUser, event.AggregateType)
	}
	if event.AggregateID != strconv.FormatInt(usr.ID, 10) {
		t.Fatalf("expected aggregate id %q, got %q", strconv.FormatInt(usr.ID, 10), event.AggregateID)
	}
	if event.ContentType != eventcodec.ContentTypeProtobuf {
		t.Fatalf("expected content type %q, got %q", eventcodec.ContentTypeProtobuf, event.ContentType)
	}
	if event.ProtoMessage != eventcodec.ProtoMessageUserRegistered {
		t.Fatalf("expected proto message %q, got %q", eventcodec.ProtoMessageUserRegistered, event.ProtoMessage)
	}
	if event.EventVersion != eventcodec.EventVersionV1 {
		t.Fatalf("expected event version %q, got %q", eventcodec.EventVersionV1, event.EventVersion)
	}

	payload, err := eventcodec.UnmarshalUserRegistered(event.Payload)
	if err != nil {
		t.Fatalf("UnmarshalUserRegistered returned error: %v", err)
	}
	if payload.GetUserId() != usr.ID {
		t.Fatalf("expected payload user id %d, got %d", usr.ID, payload.GetUserId())
	}
	if payload.GetEmail() != usr.Email {
		t.Fatalf("expected payload email %q, got %q", usr.Email, payload.GetEmail())
	}
	if payload.GetRole() != string(usr.Role) {
		t.Fatalf("expected payload role %q, got %q", usr.Role, payload.GetRole())
	}
}
