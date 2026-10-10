package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/response"
)

const (
	UserIDContextKey    contextKey = "user_id"
	SessionIDContextKey contextKey = "session_id"
)

// Authenticate verifies the incoming Bearer token using TokenManager,
// enforces active server-side session validation using SessionValidator,
// and sets user_id and session_id in the request context.
func Authenticate(tm *auth.TokenManager, validator auth.SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid Authorization header", reqID)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tm.ValidateAccessToken(tokenStr)
			if err != nil {
				code := "INVALID_TOKEN"
				if err == auth.ErrTokenExpired {
					code = "TOKEN_EXPIRED"
				}
				response.Error(w, http.StatusUnauthorized, code, err.Error(), reqID)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Malformed user identifier in token", reqID)
				return
			}

			// Fail closed: enforce server-side active session validation
			if validator == nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Session validator unavailable", reqID)
				return
			}

			if err := validator.ValidateSession(r.Context(), userID, claims.SessionID); err != nil {
				switch {
				case errors.Is(err, auth.ErrSessionRevoked):
					response.Error(w, http.StatusUnauthorized, "SESSION_REVOKED", "Session has been revoked", reqID)
				case errors.Is(err, auth.ErrSessionExpired):
					response.Error(w, http.StatusUnauthorized, "SESSION_EXPIRED", "Session has expired", reqID)
				case errors.Is(err, auth.ErrSessionNotFound), errors.Is(err, auth.ErrSessionMismatch):
					response.Error(w, http.StatusUnauthorized, "SESSION_INVALID", "Invalid or unknown session", reqID)
				default:
					// Database query failure: Fail closed with 500 INTERNAL_ERROR
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify session status", reqID)
				}
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			ctx = context.WithValue(ctx, SessionIDContextKey, claims.SessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the authenticated user's UUID from the request context.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	id, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return id, ok
}

// GetSessionID extracts the active session UUID from the request context.
func GetSessionID(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	id, ok := ctx.Value(SessionIDContextKey).(uuid.UUID)
	return id, ok
}
