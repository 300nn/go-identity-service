package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	skipSuccessPaths := map[string]struct{}{
		"/health":  {},
		"/ready":   {},
		"/ping":    {},
		"/info":    {},
		"/version": {},
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			requestID := GetRequestID(r.Context())

			status := rw.Status()

			if _, ok := skipSuccessPaths[r.URL.Path]; ok && status < 400 {
				return
			}

			args := []any{
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", status,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			}

			switch {
			case status >= 500:
				logger.Error("http request completed", args...)
			case status >= 400:
				logger.Warn("http request completed", args...)
			default:
				logger.Info("http request completed", args...)
			}
		})
	}
}
