package middleware

import (
	"context"
	"net/http"

	"chs/backend/internal/platform"
	"chs/backend/internal/response"
)

const (
	PlatformRoleContextKey contextKey = "platform_role"
)

// RequireSuperAdmin middleware ensures the authenticated user possesses an active SUPER_ADMIN platform role.
func RequireSuperAdmin(repo *platform.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			role, err := repo.GetUserPlatformRole(r.Context(), userID)
			if err != nil || role == nil {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Administrative privileges required", reqID)
				return
			}

			if role.Status != platform.StatusActive {
				response.Error(w, http.StatusForbidden, "ACCOUNT_DISABLED", "Administrative account has been disabled", reqID)
				return
			}

			if role.Role != platform.RoleSuperAdmin {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Super Admin privileges required for this operation", reqID)
				return
			}

			ctx := context.WithValue(r.Context(), PlatformRoleContextKey, role.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePlatformRole ensures the authenticated user holds at least one active platform role from the allowed set.
func RequirePlatformRole(repo *platform.Repository, allowedRoles ...string) func(http.Handler) http.Handler {
	allowedMap := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedMap[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			role, err := repo.GetUserPlatformRole(r.Context(), userID)
			if err != nil || role == nil {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Administrative privileges required", reqID)
				return
			}

			if role.Status != platform.StatusActive {
				response.Error(w, http.StatusForbidden, "ACCOUNT_DISABLED", "Administrative account has been disabled", reqID)
				return
			}

			if !allowedMap[role.Role] {
				response.Error(w, http.StatusForbidden, "FORBIDDEN", "Insufficient platform administrative permissions", reqID)
				return
			}

			ctx := context.WithValue(r.Context(), PlatformRoleContextKey, role.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
