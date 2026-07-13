package middleware

import "net/http"

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(status int) {
	if rw.wroteHeader {
		return
	}

	rw.ResponseWriter.WriteHeader(status)
	rw.status = status
	rw.wroteHeader = true
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(data)
}

func (rw *responseWriter) Status() int {
	return rw.status
}
