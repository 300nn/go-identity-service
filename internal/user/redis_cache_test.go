package user_test

import (
	"testing"
	"time"

	"CrudTutorialProject/internal/testutils"
	"CrudTutorialProject/internal/user"
)

func TestRedisCache_SetAndGetUser(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	cache := user.NewRedisCache(redisClient, "test")

	usr := user.User{
		ID:    1,
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	}

	if err := cache.SetUser(ctx, usr, time.Minute); err != nil {
		t.Fatalf("SetUser returned error: %v", err)
	}

	found, ok, err := cache.GetUser(ctx, usr.ID)

	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if !ok {
		t.Fatalf("expected user to be found")
	}

	if found.ID != usr.ID {
		t.Fatalf("expected user id %v, got %v", usr.ID, found.ID)
	}

	if found.Name != usr.Name {
		t.Fatalf("expected user name to be %s, got %s", usr.Name, found.Name)
	}

	if found.Email != usr.Email {
		t.Fatalf("expected user email to be %s, got %s", usr.Email, found.Email)
	}
}

func TestRedisCache_DeleteUser(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	cache := user.NewRedisCache(redisClient, "test")

	usr := user.User{
		ID:    1,
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	}

	if err := cache.SetUser(ctx, usr, time.Minute); err != nil {
		t.Fatalf("SetUser returned error: %v", err)
	}

	if err := cache.DeleteUser(ctx, usr.ID); err != nil {
		t.Fatalf("DeleteUser returned error: %v", err)
	}

	_, ok, err := cache.GetUser(ctx, usr.ID)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestRedisCache_SetUser_WithTTL(t *testing.T) {
	ctx := t.Context()

	redisClient := testutils.NewTestRedisClient(t)
	cache := user.NewRedisCache(redisClient, "test")

	usr := user.User{
		ID:    1,
		Name:  "Alex",
		Email: "alex@example.com",
		Age:   25,
		Role:  user.RoleUser,
	}

	if err := cache.SetUser(ctx, usr, 100*time.Millisecond); err != nil {
		t.Fatalf("SetUser returned error: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	_, ok, err := cache.GetUser(ctx, usr.ID)

	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if ok {
		t.Fatal("expected cache miss after TTL expired")
	}
}
