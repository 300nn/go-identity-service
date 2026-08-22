package auth

import (
	"errors"
	"net/http"

	"CrudTutorialProject/internal/apperror"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

func NewRefreshTokenNotFoundError() error {
	return apperror.WrapPublicError(
		http.StatusNotFound,
		"refresh_token_not_found",
		"Refresh token not found",
		ErrRefreshTokenNotFound,
	)
}

func NewInvalidCredentialsError() error {
	return apperror.WrapPublicError(
		http.StatusUnauthorized,
		"invalid_credentials",
		"invalid email or password",
		ErrInvalidCredentials,
	)
}

func NewUnauthorizedError() error {
	return apperror.WrapPublicError(
		http.StatusUnauthorized,
		"unauthorized",
		"unauthorized",
		ErrUnauthorized,
	)
}
