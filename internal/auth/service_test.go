package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type TestAuthApp struct {
	repo    auth.UserStore
	hasher  *auth.PasswordHasher
	tokens  *auth.TokenManager
	service *auth.Service
}

func newTestAuthApp(t *testing.T) *TestAuthApp {
	t.Helper()

	repo := newFakeUserRepository()
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)

	service := auth.NewService(repo, hasher, tokens)

	return &TestAuthApp{
		repo:    repo,
		hasher:  hasher,
		tokens:  tokens,
		service: service,
	}
}

func TestService_Register_Success(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	result, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     " Alex ",
		Email:    " Alex@Example.com ",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if result.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if result.TokenType != "Bearer" {
		t.Fatalf("expected token type %q, got %q", "Bearer", result.TokenType)
	}

	if result.User.ID == 0 {
		t.Fatal("expected user ID to be set")
	}

	if result.User.Name != "Alex" {
		t.Fatalf("expected name %q, got %q", "Alex", result.User.Name)
	}

	if result.User.Email != "alex@example.com" {
		t.Fatalf("expected email %q, got %q", "alex@example.com", result.User.Email)
	}

	saved, err := app.repo.FindByEmail(ctx, "alex@example.com")
	if err != nil {
		t.Fatalf("FindByEmail returned error: %v", err)
	}

	if saved.PasswordHash == "" {
		t.Fatal("expected password hash to be saved")
	}

	if saved.PasswordHash == "password123" {
		t.Fatal("password hash must not equal plain password")
	}

	if !app.hasher.Compare(saved.PasswordHash, "password123") {
		t.Fatal("expected saved hash to match password")
	}

	claims, err := app.tokens.Parse(result.AccessToken)
	if err != nil {
		t.Fatalf("Parse access token returned error: %v", err)
	}

	if claims.UserId != result.User.ID {
		t.Fatalf("expected token user id %d, got %d", result.User.ID, claims.UserId)
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	_, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}

	_, err = app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Another Alex",
		Email:    " Alex@Example.com ",
		Age:      30,
		Password: "password456",
	})
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestService_Login_Success(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	passwordHash, err := app.hasher.Hash("password123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	created, err := app.repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	result, err := app.service.Login(ctx, auth.LoginRequest{
		Email:    " Alex@Example.com ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if result.User.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, result.User.ID)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	passwordHash, err := app.hasher.Hash("password123")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	_, err = app.repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	_, err = app.service.Login(ctx, auth.LoginRequest{
		Email:    "alex@example.com",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected invalid credentials error, got nil")
	}

	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Login_UnknownEmail(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	_, err := app.service.Login(ctx, auth.LoginRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected invalid credentials error, got nil")
	}

	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Me_Success(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	created, err := app.repo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("repo.Create returned error: %v", err)
	}

	result, err := app.service.Me(ctx, created.ID)
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}

	if result.User.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, result.User.ID)
	}
}
