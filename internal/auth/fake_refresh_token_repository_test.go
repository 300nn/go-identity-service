package auth_test

import (
	"CrudTutorialProject/internal/auth"
	"context"
	"errors"
	"time"
)

type fakeRefreshTokenRepository struct {
	nextID int64
	tokens map[int64]auth.RefreshToken
}

func newFakeRefreshTokenRepository() *fakeRefreshTokenRepository {
	return &fakeRefreshTokenRepository{
		nextID: 1,
		tokens: make(map[int64]auth.RefreshToken),
	}
}

func (r *fakeRefreshTokenRepository) CreateRefreshToken(ctx context.Context, token auth.RefreshToken) (auth.RefreshToken, error) {
	token.ID = r.nextID
	token.CreatedAt = time.Now()

	r.tokens[token.ID] = token
	r.nextID++

	return token, nil
}

func (r *fakeRefreshTokenRepository) FindRefreshTokenByHash(ctx context.Context, hash string) (auth.RefreshToken, error) {
	for _, token := range r.tokens {
		if token.TokenHash == hash {
			return token, nil
		}
	}

	return auth.RefreshToken{}, auth.ErrRefreshTokenNotFound
}

func (r *fakeRefreshTokenRepository) RevokeRefreshToken(ctx context.Context, id int64) error {
	token, ok := r.tokens[id]
	if !ok {
		return auth.ErrRefreshTokenNotFound
	}
	if token.IsRevoked() {
		return auth.ErrRefreshTokenNotFound
	}

	now := time.Now().UTC()
	token.RevokedAt = &now

	r.tokens[token.ID] = token

	return nil
}

func (r *fakeRefreshTokenRepository) countActiveTokens(userID int64) int {
	now := time.Now().UTC()

	count := 0
	for _, token := range r.tokens {
		if token.UserID == userID && token.IsActive(now) {
			count++
		}
	}

	return count
}

type failingCreateRefreshTokenStore struct {
	auth.RefreshTokenStore
}

var errCreateRefreshTokenFailed = errors.New("create refresh token failed")

func (s *failingCreateRefreshTokenStore) CreateRefreshToken(
	ctx context.Context,
	token auth.RefreshToken,
) (auth.RefreshToken, error) {
	return auth.RefreshToken{}, errCreateRefreshTokenFailed
}
