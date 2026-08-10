package user

import "time"

type Profile struct {
	ID        int64
	UserID    int64
	Bio       string
	CreatedAt time.Time
	UpdatedAt time.Time
}
