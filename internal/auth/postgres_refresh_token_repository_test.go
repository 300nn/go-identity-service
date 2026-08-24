package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/300nn/go-identity-service/internal/testutils"

	"github.com/300nn/go-identity-service/internal/auth"
	"github.com/300nn/go-identity-service/internal/user"
)

func TestPostgresRefreshTokenRepository_CreateAndFindByHash(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)

	userRepo := user.NewPostgresRepository(pool, time.Second)
	refreshRepo := auth.NewRefreshTokenRepository(pool, time.Second)

	createdUser, err := userRepo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex@example.com",
		Age:          25,
		PasswordHash: "hash",
		Role:         user.RoleUser,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	manager := auth.NewRefreshTokenManager()

	_, hash, err := manager.Generate()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	createdToken, err := refreshRepo.CreateRefreshToken(ctx, auth.RefreshToken{
		UserID:    createdUser.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}

	foundToken, err := refreshRepo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("FindRefreshTokenByHash returned error: %v", err)
	}

	if foundToken.ID != createdToken.ID {
		t.Fatalf("expected token id %d, got %d", createdToken.ID, foundToken.ID)
	}

	if foundToken.UserID != createdUser.ID {
		t.Fatalf("expected user id %d, got %d", createdUser.ID, foundToken.UserID)
	}

	if foundToken.TokenHash != hash {
		t.Fatalf("expected token hash %q, got %q", hash, foundToken.TokenHash)
	}
}

func TestPostgresRefreshTokenRepository_Revoke(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)

	userRepo := user.NewPostgresRepository(pool, time.Second)
	refreshRepo := auth.NewRefreshTokenRepository(pool, time.Second)

	createdUser, err := userRepo.Create(ctx, user.User{
		Name:         "Alex",
		Email:        "alex.revoke@example.com",
		Age:          25,
		PasswordHash: "hash",
		Role:         user.RoleUser,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	manager := auth.NewRefreshTokenManager()

	_, hash, err := manager.Generate()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	createdToken, err := refreshRepo.CreateRefreshToken(ctx, auth.RefreshToken{
		UserID:    createdUser.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}

	if err := refreshRepo.RevokeRefreshToken(ctx, createdToken.ID); err != nil {
		t.Fatalf("RevokeRefreshToken returned error: %v", err)
	}

	foundToken, err := refreshRepo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		t.Fatalf("FindRefreshTokenByHash returned error: %v", err)
	}

	if foundToken.RevokedAt == nil {
		t.Fatal("expected RevokedAt to be set")
	}
}

func TestPostgresRefreshTokenRepository_FindByHash_NotFound(t *testing.T) {
	ctx := t.Context()

	pool := testutils.NewTestPostgresPool(t)
	refreshRepo := auth.NewRefreshTokenRepository(pool, time.Second)

	_, err := refreshRepo.FindRefreshTokenByHash(ctx, "missing-hash")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, auth.ErrRefreshTokenNotFound) {
		t.Fatalf("expected ErrRefreshTokenNotFound, got %v", err)
	}
}
