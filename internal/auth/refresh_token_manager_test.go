package auth_test

import (
	"testing"

	"github.com/300nn/go-identity-service/internal/auth"
)

func TestRefreshTokenManager_Generate(t *testing.T) {
	manager := auth.NewRefreshTokenManager()

	token, hash, err := manager.Generate()
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if token == "" {
		t.Fatal("expected token to be set")
	}

	if hash == "" {
		t.Fatal("expected hash to be set")
	}

	if token == hash {
		t.Fatal("hash must not equal plain token")
	}

	if manager.Hash(token) != hash {
		t.Fatal("expected Hash(token) to equal generated hash")
	}
}

func TestRefreshTokenManager_Hash_IsStable(t *testing.T) {
	manager := auth.NewRefreshTokenManager()

	first := manager.Hash("refresh-token")
	second := manager.Hash("refresh-token")

	if first != second {
		t.Fatalf("expected stable hash, got %q and %q", first, second)
	}
}

func TestRefreshTokenManager_Generate_ReturnsDifferentTokens(t *testing.T) {
	manager := auth.NewRefreshTokenManager()

	firstToken, firstHash, err := manager.Generate()
	if err != nil {
		t.Fatalf("first Generate returned error: %v", err)
	}

	secondToken, secondHash, err := manager.Generate()
	if err != nil {
		t.Fatalf("second Generate returned error: %v", err)
	}

	if firstToken == secondToken {
		t.Fatal("expected different refresh tokens")
	}

	if firstHash == secondHash {
		t.Fatal("expected different refresh token hashes")
	}
}
