package api

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs each request with method, path, status, remote, and duration.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		log.Printf(
			"INFO: %s %s [%d] from %s in %v",
			r.Method, r.URL.Path, rw.statusCode, r.RemoteAddr, duration,
		)
	})
}
