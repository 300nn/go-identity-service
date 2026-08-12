package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHasher_HashAndCompare(t *testing.T) {
	hasher := auth.NewPasswordHasherWithCost(bcrypt.MinCost)

	hash, err := hasher.Hash("password123")

	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash returned empty string")
	}

	if !hasher.Compare(hash, "password123") {
		t.Fatal("Expected password to match hash")
	}

	if hasher.Compare(hash, "wrong-password") {
		t.Fatal("Expected wrong password not to match hash")
	}
}
