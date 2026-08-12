package auth

import (
	"CrudTutorialProject/internal/apperror"
	"errors"
	"net/http"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

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
