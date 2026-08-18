package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"CrudTutorialProject/internal/user"
	"testing"
	"time"
)

const testJWTSecret = "test-secret-with-at-least-32-characters"

func TestTokenManager_GenerateAndParse(t *testing.T) {
	manager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")

	token, err := manager.Generate(12, "alex@ee.com", "USER")
	if err != nil {
		t.Fatalf("Error generating token: %v", err)
	}

	if token == "" {
		t.Fatal("Expected token to be generated")
	}

	claims, err := manager.Parse(token)

	if err != nil {
		t.Fatalf("Error parsing token: %v", err)
	}

	if claims.Issuer != "go-crud-api" {
		t.Fatalf("Expected issuer to be %s, got %s", "go-crud-api", claims.Issuer)
	}

	if claims.UserID != 12 {
		t.Fatalf("Expected user id to be %d, got %d", 12, claims.UserID)
	}

	if claims.Email != "alex@ee.com" {
		t.Fatalf("Expected email %s, got %s", "alex@ee.com", claims.Email)
	}

	if claims.Role != string(user.RoleUser) {
		t.Fatalf("expected role %q, got %q", "USER", claims.Role)
	}
}

func TestTokenManager_Parse_WrongSecret(t *testing.T) {
	manager := auth.NewTokenManager(testJWTSecret, 15*time.Minute, "go-crud-api")

	token, err := manager.Generate(123, "alex@example.com", "USER")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	otherManager := auth.NewTokenManager(
		"another-secret-with-at-least-32-characters",
		15*time.Minute,
		"go-crud-api",
	)

	_, err = otherManager.Parse(token)
	if err == nil {
		t.Fatal("expected parse error with wrong secret, got nil")
	}
}

func TestTokenManager_Parse_ExpiredToken(t *testing.T) {
	manager := auth.NewTokenManager(testJWTSecret, -time.Minute, "go-crud-api")

	token, err := manager.Generate(123, "alex@example.com", "USER")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	_, err = manager.Parse(token)
	if err == nil {
		t.Fatal("expected expired token error, got nil")
	}
}
