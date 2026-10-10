package middleware

import (
	"context"
	"net/http"
	"regexp"

	"github.com/google/uuid"
)

type contextKey string

const (
	// RequestIDHeader is the canonical header for distributed request tracing.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for the request ID.
	RequestIDKey contextKey = "request_id"
)

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// RequestID ensures every HTTP request has a validated or newly generated request ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(RequestIDHeader)

		// Validate incoming request ID; generate new one if missing or malformed
		if reqID == "" || !validIDRegex.MatchString(reqID) {
			reqID = uuid.New().String()
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)

		// Set response header
		w.Header().Set(RequestIDHeader, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from the request context, or returns empty string.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
