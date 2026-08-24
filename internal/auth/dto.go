package auth

import "CrudTutorialProject/internal/user"

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Age      int32  `json:"age" validate:"gte=0,lte=150"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type AuthResponse struct {
	AccessToken  string            `json:"accessToken"`
	TokenType    string            `json:"tokenType"`
	RefreshToken string            `json:"refreshToken"`
	User         user.UserResponse `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type MeResponse struct {
	User user.UserResponse `json:"user"`
}
