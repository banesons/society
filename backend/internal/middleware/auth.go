package middleware

import (
	"context"
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

// Authenticate verifies the incoming Bearer token using TokenManager and sets user_id and session_id in context.
func Authenticate(tm *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				reqID := GetRequestID(r.Context())
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid Authorization header", reqID)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tm.ValidateAccessToken(tokenStr)
			if err != nil {
				reqID := GetRequestID(r.Context())
				code := "INVALID_TOKEN"
				if err == auth.ErrTokenExpired {
					code = "TOKEN_EXPIRED"
				}
				response.Error(w, http.StatusUnauthorized, code, err.Error(), reqID)
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				reqID := GetRequestID(r.Context())
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Malformed user identifier in token", reqID)
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
