package auth_test

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/300nn/go-identity-service/internal/auth"
	"github.com/300nn/go-identity-service/internal/outbox"
	"github.com/300nn/go-identity-service/internal/user"

	"golang.org/x/crypto/bcrypt"
)

type testAuthApp struct {
	userRepo    *fakeUserRepository
	refreshRepo *fakeRefreshTokenRepository
	txFactory   *fakeTxFactory
	hasher      *auth.PasswordHasher
	tokens      *auth.TokenManager
	refresh     *auth.RefreshTokenManager
	service     *auth.Service
}

func newTestAuthApp(t *testing.T) testAuthApp {
	t.Helper()

	userRepo := newFakeUserRepository()
	refreshRepo := newFakeRefreshTokenRepository()

	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "identity-service")
	refresh := auth.NewRefreshTokenManager()

	outboxStore := newFakeOutboxStore()

	txFactory := newFakeTxFactory(userRepo, refreshRepo, outboxStore)

	service := auth.NewService(
		userRepo,
		refreshRepo,
		txFactory,
		hasher,
		tokens,
		refresh,
		30*24*time.Hour,
	)

	return testAuthApp{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		txFactory:   txFactory,
		hasher:      hasher,
		tokens:      tokens,
		refresh:     refresh,
		service:     service,
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

	if result.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
	}

	hash := app.refresh.Hash(result.RefreshToken)

	stored, err := app.refreshRepo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("FindRefreshTokenByHash returned error: %v", err)
	}

	if stored.TokenHash == result.RefreshToken {
		t.Fatal("stored refresh token must not equal plain refresh token")
	}

	if stored.UserID != result.User.ID {
		t.Fatalf("expected refresh token user id %d, got %d", result.User.ID, stored.UserID)
	}

	saved, err := app.userRepo.FindByEmail(ctx, "alex@example.com")
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

	if claims.UserID != result.User.ID {
		t.Fatalf("expected token user id %d, got %d", result.User.ID, claims.UserID)
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

	created, err := app.userRepo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("userRepo.Create returned error: %v", err)
	}

	result, err := app.service.Login(ctx, auth.LoginRequest{
		Email:    " Alex@Example.com ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
	}

	hash := app.refresh.Hash(result.RefreshToken)

	stored, err := app.refreshRepo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("FindRefreshTokenByHash returned error: %v", err)
	}

	if stored.TokenHash == result.RefreshToken {
		t.Fatal("stored refresh token must not equal plain refresh token")
	}

	if stored.UserID != result.User.ID {
		t.Fatalf("expected refresh token user id %d, got %d", result.User.ID, stored.UserID)
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

	_, err = app.userRepo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: passwordHash,
	})
	if err != nil {
		t.Fatalf("userRepo.Create returned error: %v", err)
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

	created, err := app.userRepo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("userRepo.Create returned error: %v", err)
	}

	result, err := app.service.Me(ctx, created.ID)
	if err != nil {
		t.Fatalf("Me returned error: %v", err)
	}

	if result.User.ID != created.ID {
		t.Fatalf("expected user id %d, got %d", created.ID, result.User.ID)
	}
}

func TestService_Refresh_Success(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	registerResult, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	refreshResult, err := app.service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: registerResult.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}

	if refreshResult.AccessToken == "" {
		t.Fatal("expected access token to be set")
	}

	if refreshResult.RefreshToken == "" {
		t.Fatal("expected refresh token to be set")
	}

	if refreshResult.RefreshToken == registerResult.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}

	if refreshResult.User.ID != registerResult.User.ID {
		t.Fatalf("expected user id %d, got %d", registerResult.User.ID, refreshResult.User.ID)
	}

	claims, err := app.tokens.Parse(refreshResult.AccessToken)
	if err != nil {
		t.Fatalf("Parse access token returned error: %v", err)
	}

	if claims.UserID != registerResult.User.ID {
		t.Fatalf("expected claims user id %d, got %d", registerResult.User.ID, claims.UserID)
	}
}

func TestService_Refresh_RotationRevokesOldToken(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	registerResult, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	firstRefreshToken := registerResult.RefreshToken

	_, err = app.service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: firstRefreshToken,
	})
	if err != nil {
		t.Fatalf("first Refresh returned error: %v", err)
	}

	_, err = app.service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: firstRefreshToken,
	})
	if err == nil {
		t.Fatal("expected old refresh token to be rejected")
	}

	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	_, err := app.service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: "invalid-refresh-token",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_Logout_RevokesRefreshToken(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	registerResult, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	err = app.service.Logout(ctx, auth.LogoutRequest{
		RefreshToken: registerResult.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}

	_, err = app.service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: registerResult.RefreshToken,
	})
	if err == nil {
		t.Fatal("expected refresh after logout to fail")
	}

	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_Logout_IsIdempotent(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	err := app.service.Logout(ctx, auth.LogoutRequest{
		RefreshToken: "unknown-refresh-token",
	})
	if err != nil {
		t.Fatalf("Logout with unknown token returned error: %v", err)
	}
}

func TestService_Refresh_CreateNewTokenFails_DoesNotRevokeOldToken(t *testing.T) {
	ctx := t.Context()

	userRepo := newFakeUserRepository()
	baseRefreshRepo := newFakeRefreshTokenRepository()
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "identity-service")
	refresh := auth.NewRefreshTokenManager()

	outboxStore := newFakeOutboxStore()

	normalTxFactory := newFakeTxFactory(userRepo, baseRefreshRepo, outboxStore)

	service := auth.NewService(
		userRepo,
		baseRefreshRepo,
		normalTxFactory,
		hasher,
		tokens,
		refresh,
		30*24*time.Hour,
	)

	registerResult, err := service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	failingRefreshRepo := &failingCreateRefreshTokenStore{
		RefreshTokenStore: baseRefreshRepo,
	}

	failingTxFactory := newFakeTxFactoryWithStores(userRepo, baseRefreshRepo, failingRefreshRepo, outboxStore)

	serviceWithFailingTx := auth.NewService(
		userRepo,
		baseRefreshRepo,
		failingTxFactory,
		hasher,
		tokens,
		refresh,
		30*24*time.Hour,
	)

	_, err = serviceWithFailingTx.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: registerResult.RefreshToken,
	})
	if err == nil {
		t.Fatal("expected refresh error, got nil")
	}

	_, err = service.Refresh(ctx, auth.RefreshRequest{
		RefreshToken: registerResult.RefreshToken,
	})
	if err != nil {
		t.Fatalf("expected old refresh token to remain active, got error: %v", err)
	}
}

func TestService_Register_CreateRefreshTokenFails_RollsBackUser(t *testing.T) {
	ctx := t.Context()

	userRepo := newFakeUserRepository()
	refreshRepo := newFakeRefreshTokenRepository()

	outboxStore := newFakeOutboxStore()

	failingRefreshStore := &failingCreateRefreshTokenStore{
		RefreshTokenStore: refreshRepo,
	}

	txFactory := newFakeTxFactoryWithStores(
		userRepo,
		refreshRepo,
		failingRefreshStore,
		outboxStore,
	)

	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "identity-service")
	refreshTokens := auth.NewRefreshTokenManager()

	service := auth.NewService(
		userRepo,
		refreshRepo,
		txFactory,
		hasher,
		tokens,
		refreshTokens,
		30*24*time.Hour,
	)

	_, err := service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected register error, got nil")
	}

	exists, err := userRepo.ExistsByEmail(ctx, "alex@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail returned error: %v", err)
	}

	if exists {
		t.Fatal("expected user to be rolled back")
	}

	if len(refreshRepo.tokens) != 0 {
		t.Fatalf("expected refresh tokens to be rolled back, got %d", len(refreshRepo.tokens))
	}
}

func TestService_Register_CreatesOutboxEvent(t *testing.T) {
	ctx := t.Context()
	app := newTestAuthApp(t)

	result, err := app.service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if len(app.txFactory.outboxStore.events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(app.txFactory.outboxStore.events))
	}

	var event outbox.Event
	for _, e := range app.txFactory.outboxStore.events {
		event = e
	}

	if event.EventType != outbox.EventTypeUserRegistered {
		t.Fatalf("expected event type %q, got %q", outbox.EventTypeUserRegistered, event.EventType)
	}

	if event.AggregateType != outbox.AggregateUser {
		t.Fatalf("expected aggregate type %q, got %q", outbox.AggregateUser, event.AggregateType)
	}

	if event.AggregateID != strconv.FormatInt(result.User.ID, 10) {
		t.Fatalf("expected aggregate id %d, got %s", result.User.ID, event.AggregateID)
	}

	if len(event.Payload) == 0 {
		t.Fatal("expected event payload to be set")
	}
}

func TestService_Register_CreateOutboxEventFails_RollsBackUserAndRefreshToken(t *testing.T) {
	ctx := t.Context()

	userRepo := newFakeUserRepository()
	refreshRepo := newFakeRefreshTokenRepository()
	outboxStore := newFakeOutboxStore()
	outboxStore.createErr = errCreateOutboxEventFailed

	txFactory := newFakeTxFactoryWithStores(
		userRepo,
		refreshRepo,
		refreshRepo,
		outboxStore,
	)

	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)
	tokens := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "identity-service")
	refreshTokens := auth.NewRefreshTokenManager()

	service := auth.NewService(
		userRepo,
		refreshRepo,
		txFactory,
		hasher,
		tokens,
		refreshTokens,
		30*24*time.Hour,
	)

	_, err := service.Register(ctx, auth.RegisterRequest{
		Name:     "Alex",
		Email:    "alex@example.com",
		Age:      25,
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected Register error, got nil")
	}

	exists, err := userRepo.ExistsByEmail(ctx, "alex@example.com")
	if err != nil {
		t.Fatalf("ExistsByEmail returned error: %v", err)
	}

	if exists {
		t.Fatal("expected user to be rolled back")
	}

	if len(refreshRepo.tokens) != 0 {
		t.Fatalf("expected refresh tokens to be rolled back, got %d", len(refreshRepo.tokens))
	}

	if len(outboxStore.events) != 0 {
		t.Fatalf("expected outbox events to be rolled back, got %d", len(outboxStore.events))
	}
}
