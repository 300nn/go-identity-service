package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type PasswordHasher struct {
	cost int
}

func NewPasswordHasherWithCost(cost int) *PasswordHasher {
	return &PasswordHasher{
		cost: cost,
	}
}

func NewPasswordHasher() *PasswordHasher {
	return NewPasswordHasherWithCost(bcryptCost)
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)

	if err != nil {
		return "", fmt.Errorf("generate pasword hash: %w", err)
	}

	return string(hash), nil
}

func (h *PasswordHasher) Compare(hash string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
