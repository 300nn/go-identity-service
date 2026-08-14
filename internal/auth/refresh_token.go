package auth

import "time"

type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	return !t.ExpiresAt.After(now)
}

func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t *RefreshToken) IsActive(now time.Time) bool {
	return !t.IsRevoked() && !t.IsExpired(now)
}
