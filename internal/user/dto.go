package user

import (
	"time"
)

type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type UpdateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserWithProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
	Bio   string `json:"bio"`
}

type UserWithProfileResponse struct {
	User    UserResponse    `json:"user"`
	Profile ProfileResponse `json:"profile"`
}

type ProfileResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ProfileToResponse(profile Profile) ProfileResponse {
	return ProfileResponse{
		ID:        profile.ID,
		UserID:    profile.UserID,
		Bio:       profile.Bio,
		CreatedAt: profile.CreatedAt,
		UpdatedAt: profile.UpdatedAt,
	}
}

func ToResponse(u User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Age:       u.Age,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func ToResponseList(users []User) []UserResponse {
	res := make([]UserResponse, 0, len(users))
	for _, val := range users {
		res = append(res, ToResponse(val))
	}
	return res
}
