package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxBodySize = 1 << 20

func DecodeJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var dst T

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&dst); err != nil {
		writeDecodeError(w, err)
		return dst, false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		SendError(w, http.StatusBadRequest, "invalid_json", "Request body must contain only one JSON object")
		return dst, false
	}

	return dst, true
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var maxBytesErr *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxErr):
		SendError(w, http.StatusBadRequest, "invalid_json", "Malformed JSON body")

	case errors.Is(err, io.ErrUnexpectedEOF):
		SendError(w, http.StatusBadRequest, "invalid_json", "Malformed JSON body")

	case errors.As(err, &typeErr):
		message := fmt.Sprintf("Invalid value for field %q", typeErr.Field)
		SendError(w, http.StatusBadRequest, "invalid_json_type", message)

	case errors.Is(err, io.EOF):
		SendError(w, http.StatusBadRequest, "empty_body", "Request body must not be empty")

	case errors.As(err, &maxBytesErr):
		SendError(w, http.StatusRequestEntityTooLarge, "body_too_large", "Request body is too large")

	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.TrimPrefix(err.Error(), "json: unknown field ")
		message := fmt.Sprintf("Unknown field %s", field)
		SendError(w, http.StatusBadRequest, "unknown_json_field", message)

	default:
		SendError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
	}
}
