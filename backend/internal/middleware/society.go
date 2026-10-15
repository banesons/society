package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

const (
	SocietyIDContextKey   contextKey = "society_id"
	SocietyRoleContextKey contextKey = "society_role"
)

// RequireSocietyAccess validates that the target society exists, is in ACTIVE status,
// and that the authenticated caller is either a platform SUPER_ADMIN or holds an ACTIVE role assignment for that society.
func RequireSocietyAccess(role string, societyRepo *society.Repository, platformRepo *platform.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			societyIDStr := chi.URLParam(r, "id")
			if societyIDStr == "" {
				societyIDStr = chi.URLParam(r, "societyID")
			}

			societyID, err := uuid.Parse(societyIDStr)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society identifier", reqID)
				return
			}

			headerSocID := r.Header.Get("X-Society-ID")
			if headerSocID != "" {
				parsedHeaderID, err := uuid.Parse(headerSocID)
				if err != nil || parsedHeaderID != societyID {
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Client society context mismatch", reqID)
					return
				}
			}

			if societyRepo == nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
				return
			}

			// 1. Verify target society exists and check its operational status
			soc, err := societyRepo.GetSocietyByID(r.Context(), societyID)
			if err != nil {
				if errors.Is(err, society.ErrSocietyNotFound) {
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
					return
				}
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve society", reqID)
				return
			}

			if soc.Status == society.StatusPending {
				response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				return
			}
			if soc.Status == society.StatusSuspended {
				response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				return
			}
			if soc.Status != society.StatusActive {
				response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Target society is not active", reqID)
				return
			}

			// 2. Check if caller is platform SUPER_ADMIN
			isSuperAdmin := false
			if platformRepo != nil {
				pRole, err := platformRepo.GetUserPlatformRole(r.Context(), userID)
				if err == nil && pRole != nil && pRole.Status == platform.StatusActive && pRole.Role == platform.RoleSuperAdmin {
					isSuperAdmin = true
				}
			}

			if isSuperAdmin {
				ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
				ctx = context.WithValue(ctx, SocietyRoleContextKey, "SUPER_ADMIN")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 3. Fallback to society-scoped assignment check
			_, _, err = societyRepo.VerifySocietyAdminAccess(r.Context(), societyID, userID, role)
			if err != nil {
				switch {
				case errors.Is(err, society.ErrSocietyNotFound):
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
				case errors.Is(err, society.ErrSocietyPending):
					response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				case errors.Is(err, society.ErrSocietySuspended):
					response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				case errors.Is(err, society.ErrSocietyAccessDenied):
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "You do not have active administrative access to this society", reqID)
				default:
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify society authorization", reqID)
				}
				return
			}

			ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
			ctx = context.WithValue(ctx, SocietyRoleContextKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAnySocietyRoleAccess validates that target society exists and caller has SUPER_ADMIN or ANY of the allowed society roles.
func RequireAnySocietyRoleAccess(societyRepo *society.Repository, platformRepo *platform.Repository, allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			societyIDStr := chi.URLParam(r, "id")
			if societyIDStr == "" {
				societyIDStr = chi.URLParam(r, "societyID")
			}

			societyID, err := uuid.Parse(societyIDStr)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society identifier", reqID)
				return
			}

			headerSocID := r.Header.Get("X-Society-ID")
			if headerSocID != "" {
				parsedHeaderID, err := uuid.Parse(headerSocID)
				if err != nil || parsedHeaderID != societyID {
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Client society context mismatch", reqID)
					return
				}
			}

			if societyRepo == nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
				return
			}

			// 1. Verify target society exists and check its operational status
			soc, err := societyRepo.GetSocietyByID(r.Context(), societyID)
			if err != nil {
				if errors.Is(err, society.ErrSocietyNotFound) {
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
					return
				}
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve society", reqID)
				return
			}

			if soc.Status == society.StatusPending {
				response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				return
			}
			if soc.Status == society.StatusSuspended {
				response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				return
			}
			if soc.Status != society.StatusActive {
				response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Target society is not active", reqID)
				return
			}

			// 2. Check if caller is platform SUPER_ADMIN
			if platformRepo != nil {
				pRole, err := platformRepo.GetUserPlatformRole(r.Context(), userID)
				if err == nil && pRole != nil && pRole.Status == platform.StatusActive && pRole.Role == platform.RoleSuperAdmin {
					ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
					ctx = context.WithValue(ctx, SocietyRoleContextKey, "SUPER_ADMIN")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 3. Check society roles
			_, assignment, err := societyRepo.VerifySocietyRolesAccess(r.Context(), societyID, userID, allowedRoles...)
			if err != nil {
				switch {
				case errors.Is(err, society.ErrSocietyNotFound):
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
				case errors.Is(err, society.ErrSocietyPending):
					response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				case errors.Is(err, society.ErrSocietySuspended):
					response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				case errors.Is(err, society.ErrSocietyAccessDenied):
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "You do not have active access to this society", reqID)
				default:
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify society authorization", reqID)
				}
				return
			}

			ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
			ctx = context.WithValue(ctx, SocietyRoleContextKey, assignment.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSocietyRole validates that the target society exists, is in ACTIVE status,
// and that the authenticated caller holds an ACTIVE role assignment for that society.
// Society context is extracted canonically from the URL parameter.
func RequireSocietyRole(role string, repo *society.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			societyIDStr := chi.URLParam(r, "id")
			if societyIDStr == "" {
				societyIDStr = chi.URLParam(r, "societyID")
			}

			societyID, err := uuid.Parse(societyIDStr)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society identifier", reqID)
				return
			}

			// Validate client-asserted header if provided to prevent header forgery (AC-08)
			headerSocID := r.Header.Get("X-Society-ID")
			if headerSocID != "" {
				parsedHeaderID, err := uuid.Parse(headerSocID)
				if err != nil || parsedHeaderID != societyID {
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Client society context mismatch", reqID)
					return
				}
			}

			if repo == nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
				return
			}

			_, _, err = repo.VerifySocietyAdminAccess(r.Context(), societyID, userID, role)
			if err != nil {
				switch {
				case errors.Is(err, society.ErrSocietyNotFound):
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
				case errors.Is(err, society.ErrSocietyPending):
					response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				case errors.Is(err, society.ErrSocietySuspended):
					response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				case errors.Is(err, society.ErrSocietyAccessDenied):
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "You do not have active administrative access to this society", reqID)
				default:
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify society authorization", reqID)
				}
				return
			}

			ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
			ctx = context.WithValue(ctx, SocietyRoleContextKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSocietyMemberOrAdminAccess validates that target society exists and caller is SUPER_ADMIN, SOC_ADMIN, GATEKEEPER, or an active RESIDENT member.
func RequireSocietyMemberOrAdminAccess(societyRepo *society.Repository, platformRepo *platform.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := GetRequestID(r.Context())

			userID, ok := GetUserID(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
				return
			}

			societyIDStr := chi.URLParam(r, "id")
			if societyIDStr == "" {
				societyIDStr = chi.URLParam(r, "societyID")
			}

			societyID, err := uuid.Parse(societyIDStr)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society identifier", reqID)
				return
			}

			headerSocID := r.Header.Get("X-Society-ID")
			if headerSocID != "" {
				parsedHeaderID, err := uuid.Parse(headerSocID)
				if err != nil || parsedHeaderID != societyID {
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "Client society context mismatch", reqID)
					return
				}
			}

			if societyRepo == nil {
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
				return
			}

			// 1. Check platform SUPER_ADMIN
			if platformRepo != nil {
				pRole, err := platformRepo.GetUserPlatformRole(r.Context(), userID)
				if err == nil && pRole != nil && pRole.Status == platform.StatusActive && pRole.Role == platform.RoleSuperAdmin {
					ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
					ctx = context.WithValue(ctx, SocietyRoleContextKey, "SUPER_ADMIN")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 2. Check society membership or assigned role
			soc, role, err := societyRepo.VerifyActiveSocietyMembership(r.Context(), societyID, userID)
			if err != nil {
				switch {
				case errors.Is(err, society.ErrSocietyNotFound):
					response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Target society does not exist", reqID)
				case errors.Is(err, society.ErrSocietyPending):
					response.Error(w, http.StatusForbidden, "SOCIETY_PENDING", "Target society has not been activated", reqID)
				case errors.Is(err, society.ErrSocietySuspended):
					response.Error(w, http.StatusForbidden, "SOCIETY_SUSPENDED", "Target society is on administrative hold", reqID)
				case errors.Is(err, society.ErrSocietyAccessDenied):
					response.Error(w, http.StatusForbidden, "SOCIETY_ACCESS_DENIED", "You do not have active access to this society", reqID)
				default:
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to verify society authorization", reqID)
				}
				return
			}
			_ = soc

			ctx := context.WithValue(r.Context(), SocietyIDContextKey, societyID)
			ctx = context.WithValue(ctx, SocietyRoleContextKey, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSocietyID extracts the verified society UUID from request context.
func GetSocietyID(ctx context.Context) (uuid.UUID, bool) {
	if ctx == nil {
		return uuid.Nil, false
	}
	id, ok := ctx.Value(SocietyIDContextKey).(uuid.UUID)
	return id, ok
}

// GetSocietyRole extracts the verified society role string from request context.
func GetSocietyRole(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	role, ok := ctx.Value(SocietyRoleContextKey).(string)
	return role, ok
}
