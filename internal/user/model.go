package user

import "time"

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID           int64
	Name         string
	Email        string
	Age          int32
	Role         Role
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
