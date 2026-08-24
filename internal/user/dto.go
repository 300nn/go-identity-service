package user

import (
	"time"
)

type ListUsersRequest struct {
	Limit  int    `query:"limit" validate:"gte=1,lte=100"`
	Offset int    `query:"offset" validate:"gte=0"`
	Email  string `query:"email" validate:"omitempty,max=250"`
	Sort   string `query:"sort" validate:"oneof=id_asc id_desc email_asc email_desc created_at_asc created_at_desc"`
}

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Age      int32  `json:"age" validate:"gte=0,lte=150"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type UpdateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email,max=255"`
	Age   int32  `json:"age" validate:"gte=0,lte=150"`
}

type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int32     `json:"age"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateUserWithProfileRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Age      int32  `json:"age" validate:"gte=0,lte=150"`
	Bio      string `json:"bio" validate:"max=1000"`
	Password string `json:"password" validate:"required,min=8,max=72"`
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
		Role:      u.Role,
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
