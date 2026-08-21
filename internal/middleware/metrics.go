package middleware

import (
	"net/http"
	"strconv"
	"time"

	"CrudTutorialProject/internal/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Metrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			route := r.Pattern
			if route == "" {
				route = "unknown"
			}

			status := strconv.Itoa(recorder.status)
			duration := time.Since(start).Seconds()

			m.HTTPRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration)
		})
	}
}
