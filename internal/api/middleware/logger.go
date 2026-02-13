package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// StructuredLogger creates a middleware that logs HTTP requests with structured logging.
func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			wrapped := &responseWriter{ResponseWriter: w, status: 0}

			// Get request ID from middleware.RequestID
			reqID := middleware.GetReqID(r.Context())

			// Log request
			logger.Debug("HTTP request started",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", reqID,
			)

			// Process request
			defer func() {
				duration := time.Since(start)

				// Log response
				if wrapped.status >= 200 && wrapped.status < 400 {
					logger.Info("HTTP request completed",
						"method", r.Method,
						"path", r.URL.Path,
						"status", wrapped.status,
						"duration_ms", duration.Milliseconds(),
						"bytes", wrapped.bytes,
						"request_id", reqID,
					)
				} else {
					logger.Warn("HTTP request error",
						"method", r.Method,
						"path", r.URL.Path,
						"status", wrapped.status,
						"duration_ms", duration.Milliseconds(),
						"bytes", wrapped.bytes,
						"request_id", reqID,
					)
				}
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}
