package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter captures HTTP status code and size for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// StructuredLogger logs HTTP requests with timing, status, and request ID.
// It strictly avoids logging request payloads or PII.
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := GetRequestID(r.Context())

			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logAttrs := []any{
				slog.String("request_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.Int("bytes", rw.size),
			}

			if rw.status >= 500 {
				logger.Error("request completed with server error", logAttrs...)
			} else if rw.status >= 400 {
				logger.Warn("request completed with client error", logAttrs...)
			} else {
				logger.Info("request completed", logAttrs...)
			}
		})
	}
}
