package user

import "time"

type User struct {
	ID        int64
	Name      string
	Email     string
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
}
