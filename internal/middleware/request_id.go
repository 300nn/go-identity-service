package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey string

const requestIDKey contextKey = "request_id"
const RequestIDHeader = "X-Request-ID"

func RequestId(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get(RequestIDHeader)

		if requestId == "" {
			requestId = generateRequestId()
		}

		w.Header().Set(RequestIDHeader, requestId)

		ctx := context.WithValue(r.Context(), requestIDKey, requestId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	requestId, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}

	return requestId
}

func generateRequestId() string {
	var b [16]byte

	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}

	return hex.EncodeToString(b[:])
}
