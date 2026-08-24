package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/300nn/go-identity-service/internal/timex"
)

type RefreshTokenDBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type RefreshTokenRepository struct {
	db           RefreshTokenDBTX
	queryTimeout time.Duration
}

func NewRefreshTokenRepository(db RefreshTokenDBTX, timeout time.Duration) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		db:           db,
		queryTimeout: timeout,
	}
}

func (r *RefreshTokenRepository) CreateRefreshToken(ctx context.Context, token RefreshToken) (RefreshToken, error) {
	ctx, cancel := timex.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	const sql = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at) 
		VALUES ($1, $2, $3)
		returning id, user_id, token_hash, expires_at, revoked_at, created_at;
		`

	var created RefreshToken

	err := r.db.QueryRow(ctx, sql, token.UserID, token.TokenHash, token.ExpiresAt).Scan(
		&created.ID,
		&created.UserID,
		&created.TokenHash,
		&created.ExpiresAt,
		&created.RevokedAt,
		&created.CreatedAt,
	)

	if err != nil {
		return RefreshToken{}, fmt.Errorf("insert refresh token: %w", err)
	}

	return created, nil
}

func (r *RefreshTokenRepository) FindRefreshTokenByHash(ctx context.Context, hash string) (RefreshToken, error) {
	ctx, cancel := timex.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	const sql = `
		select id, user_id, token_hash, expires_at, revoked_at, created_at
		from refresh_tokens 
		where token_hash = $1;
	`

	var token RefreshToken

	err := r.db.QueryRow(ctx, sql, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshToken{}, ErrRefreshTokenNotFound
		}
		return RefreshToken{}, fmt.Errorf("find refresh token: %w", err)
	}

	if token.ID == 0 {
		return RefreshToken{}, ErrRefreshTokenNotFound
	}

	return token, nil
}

func (r *RefreshTokenRepository) RevokeRefreshToken(ctx context.Context, id int64) error {
	ctx, cancel := timex.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	const sql = `
		update refresh_tokens 
		set revoked_at = now() 
		where id = $1 
		  and revoked_at is null;
	`

	tag, err := r.db.Exec(ctx, sql, id)

	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}
