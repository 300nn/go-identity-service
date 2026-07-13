package middleware

import (
	"CrudTutorialProject/internal/response"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				requestID := GetRequestID(r.Context())

				logger.Error("panic recovered",
					"request_id", requestID,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				response.SendError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
