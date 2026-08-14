package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

type RefreshTokenManager struct{}

func NewRefreshTokenManager() *RefreshTokenManager {
	return &RefreshTokenManager{}
}

func (m *RefreshTokenManager) Generate() (string, string, error) {
	raw := make([]byte, 32)

	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("error generating refresh token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := m.Hash(token)

	return token, hash, nil
}

func (m *RefreshTokenManager) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
