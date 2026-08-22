package response

import (
	"errors"
	"log/slog"
	"net/http"

	"CrudTutorialProject/internal/apperror"
)

type ErrorBody struct {
	Error ErrorInfo `json:"error"`
}

type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func HandleError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var validationErr *apperror.FieldValidationError
	if errors.As(err, &validationErr) {
		SendValidationError(w, validationErr.Fields)
		return
	}
	var publicErr *apperror.PublicError

	if errors.As(err, &publicErr) {
		SendError(w, publicErr.Status, publicErr.Code, publicErr.Message)
		return
	}

	logger.Error("internal server error", "error", err)
	SendError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}

func SendError(w http.ResponseWriter, code int, errorCode string, message string) {
	JSON(w, code, ErrorBody{
		Error: ErrorInfo{
			Code:    errorCode,
			Message: message,
		},
	})
}

func SendValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusBadRequest, ErrorBody{
		Error: ErrorInfo{
			Code:    "validation_error",
			Message: "Validation failed",
			Fields:  fields,
		},
	})
}
