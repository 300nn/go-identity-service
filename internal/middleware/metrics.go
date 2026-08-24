package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/300nn/go-identity-service/internal/metrics"
)

func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			route := r.Pattern
			if route == "" {
				route = "unknown"
			}

			status := strconv.Itoa(rw.Status())
			duration := time.Since(start).Seconds()

			m.HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
		})
	}
}
