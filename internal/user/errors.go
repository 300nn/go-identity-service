package user

import (
	"errors"
	"net/http"

	"CrudTutorialProject/internal/apperror"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidUserID      = errors.New("invalid user id")
	ErrInvalidUserInput   = errors.New("invalid user input")
)

func NewUserNotFoundError() error {
	return apperror.WrapPublicError(
		http.StatusNotFound,
		"user_not_found",
		"User not found",
		ErrUserNotFound,
	)
}

func NewEmailAlreadyExistsError() error {
	return apperror.WrapPublicError(
		http.StatusConflict,
		"email_already_exists",
		"Email already exists",
		ErrEmailAlreadyExists,
	)
}

func NewInvalidUserIDError() error {
	return apperror.WrapPublicError(
		http.StatusBadRequest,
		"invalid_user_id",
		"Invalid user id",
		ErrInvalidUserID,
	)
}

func NewInvalidUserInputError() error {
	return apperror.WrapPublicError(
		http.StatusBadRequest,
		"invalid_user_input",
		"Invalid user input",
		ErrInvalidUserInput,
	)
}
