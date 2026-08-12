package user

import "time"

type User struct {
	ID           int64
	Name         string
	Email        string
	Age          int
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
