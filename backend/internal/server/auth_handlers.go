package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/middleware"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

type SendOTPRequest struct {
	Phone string `json:"phone"`
}

type VerifyOTPRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// SendOTPHandler handles POST /api/v1/auth/otp/send.
func SendOTPHandler(svc *auth.Service, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		var req SendOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		clientIP := r.RemoteAddr
		resendAfter, otpCode, err := svc.SendOTP(r.Context(), req.Phone, clientIP)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Please provide a valid Indian mobile number", reqID)
			case errors.Is(err, auth.ErrOTPCooldown):
				response.Error(w, http.StatusTooManyRequests, "OTP_COOLDOWN", err.Error(), reqID)
			case errors.Is(err, auth.ErrPhoneRateLimited), errors.Is(err, auth.ErrIPRateLimited):
				response.Error(w, http.StatusTooManyRequests, "OTP_RATE_LIMITED", err.Error(), reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to dispatch verification code", reqID)
			}
			return
		}

		data := map[string]interface{}{
			"resend_after_seconds": resendAfter,
		}
		if cfg != nil && cfg.Env != "production" {
			data["otp"] = otpCode
		}

		response.Success(w, http.StatusOK, data, nil)
	}
}

// VerifyOTPHandler handles POST /api/v1/auth/otp/verify.
func VerifyOTPHandler(svc *auth.Service, platformRepo *platform.Repository, societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		var req VerifyOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		if req.Phone == "" || req.OTP == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Phone and OTP are required", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		result, err := svc.VerifyOTP(r.Context(), req.Phone, req.OTP, userAgent, clientIP)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Invalid phone number format", reqID)
			case errors.Is(err, auth.ErrOTPInvalid):
				response.Error(w, http.StatusUnauthorized, "OTP_INVALID", "Incorrect verification code", reqID)
			case errors.Is(err, auth.ErrOTPExpired):
				response.Error(w, http.StatusGone, "OTP_EXPIRED", "Verification code has expired", reqID)
			case errors.Is(err, auth.ErrOTPAttemptsExceeded):
				response.Error(w, http.StatusTooManyRequests, "OTP_ATTEMPTS_EXCEEDED", "Maximum verification attempts exceeded", reqID)
			case errors.Is(err, auth.ErrVerifyIPRateLimited):
				response.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many failed verification attempts, please try again later", reqID)
			case errors.Is(err, auth.ErrSessionLimitReached):
				response.Error(w, http.StatusConflict, "SESSION_LIMIT_REACHED", "Maximum 2 active devices reached. Please log out from another device.", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication failed", reqID)
			}
			return
		}

		if platformRepo != nil {
			if roles, err := platformRepo.GetUserPlatformRoles(r.Context(), result.User.ID); err == nil && roles != nil {
				result.User.PlatformRoles = roles
			}
		}

		// Safe, concurrency-aware identity linking for pre-registered member records and visitor identities
		societyAssignments := []society.UserSocietyAssignment{}
		if societyRepo != nil {
			normPhone, _ := auth.NormalizePhone(req.Phone)
			_ = societyRepo.SafeLinkUserMembers(r.Context(), result.User.ID, normPhone)
			_, _ = societyRepo.SafeLinkVisitorIdentities(r.Context(), result.User.ID, normPhone)
			if assignments, err := societyRepo.ListUserSocietyAssignments(r.Context(), result.User.ID); err == nil && assignments != nil {
				societyAssignments = assignments
			}
		}

		var prof *auth.CanonicalUserProfile
		if p, err := svc.GetProfile(r.Context(), result.User.ID); err == nil && p != nil {
			prof = p
		}
		var userFirstName, userLastName, userDisplayName string
		var userEmail, userPhoto *string
		if prof != nil {
			userFirstName = prof.FirstName
			userLastName = prof.LastName
			userDisplayName = prof.DisplayName
			userEmail = prof.Email
			userPhoto = prof.ProfilePhotoURL
		} else {
			userDisplayName = auth.ResolveDisplayName("", "", result.User.Phone)
		}

		responsePayload := map[string]interface{}{
			"access_token":     result.AccessToken,
			"refresh_token":    result.RefreshToken,
			"expires_in":       result.ExpiresIn,
			"session_replaced": result.SessionReplaced,
			"user": map[string]interface{}{
				"id":                  result.User.ID,
				"phone":               result.User.Phone,
				"first_name":          userFirstName,
				"last_name":           userLastName,
				"display_name":        userDisplayName,
				"email":               userEmail,
				"profile_photo_url":   userPhoto,
				"status":              result.User.Status,
				"platform_roles":      result.User.PlatformRoles,
				"society_assignments": societyAssignments,
				"created_at":          result.User.CreatedAt,
				"updated_at":          result.User.UpdatedAt,
			},
			"society_assignments": societyAssignments,
		}

		response.Success(w, http.StatusOK, responsePayload, nil)
	}
}

// RefreshTokenHandler handles POST /api/v1/auth/refresh.
func RefreshTokenHandler(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		var req RefreshTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_TOKEN", "Valid refresh token required", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		result, err := svc.RefreshToken(r.Context(), req.RefreshToken, userAgent, clientIP)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidToken):
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token", reqID)
			case errors.Is(err, auth.ErrTokenExpired):
				response.Error(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token has expired", reqID)
			case errors.Is(err, auth.ErrSessionRevoked):
				response.Error(w, http.StatusUnauthorized, "SESSION_REVOKED", "Session was previously revoked", reqID)
			case errors.Is(err, auth.ErrRefreshTokenReused):
				response.Error(w, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED", "Suspicious token reuse detected; session revoked", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh session", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, result, nil)
	}
}

// LogoutHandler handles POST /api/v1/auth/logout.
func LogoutHandler(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID, ok := middleware.GetSessionID(r.Context())
		if !ok {
			reqID := middleware.GetRequestID(r.Context())
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing session context", reqID)
			return
		}

		if err := svc.Logout(r.Context(), sessionID); err != nil {
			reqID := middleware.GetRequestID(r.Context())
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Logout failed", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]string{
			"message": "Logged out successfully",
		}, nil)
	}
}

// LogoutAllHandler handles POST /api/v1/auth/logout-all.
func LogoutAllHandler(svc *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			reqID := middleware.GetRequestID(r.Context())
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user context", reqID)
			return
		}

		if err := svc.LogoutAll(r.Context(), userID); err != nil {
			reqID := middleware.GetRequestID(r.Context())
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Global logout failed", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]string{
			"message": "All sessions revoked successfully",
		}, nil)
	}
}

// MeHandler handles GET /api/v1/auth/me.
func MeHandler(svc *auth.Service, platformRepo *platform.Repository, societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication", reqID)
			return
		}

		user, err := svc.GetUser(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "User record not found", reqID)
			return
		}

		platformRoles := []string{}
		if platformRepo != nil {
			if roles, err := platformRepo.GetUserPlatformRoles(r.Context(), userID); err == nil && roles != nil {
				platformRoles = roles
			}
		}

		societyAssignments := []society.UserSocietyAssignment{}
		if societyRepo != nil {
			if assignments, err := societyRepo.ListUserSocietyAssignments(r.Context(), userID); err == nil && assignments != nil {
				societyAssignments = assignments
			}
		}

		var prof *auth.CanonicalUserProfile
		if svc != nil {
			if p, err := svc.GetProfile(r.Context(), userID); err == nil && p != nil {
				prof = p
			}
		}
		var userFirstName, userLastName, userDisplayName string
		var userEmail, userPhoto *string
		if prof != nil {
			userFirstName = prof.FirstName
			userLastName = prof.LastName
			userDisplayName = prof.DisplayName
			userEmail = prof.Email
			userPhoto = prof.ProfilePhotoURL
		} else {
			userDisplayName = auth.ResolveDisplayName("", "", user.Phone)
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":                  user.ID,
			"phone":               auth.MaskPhone(user.Phone),
			"first_name":          userFirstName,
			"last_name":           userLastName,
			"display_name":        userDisplayName,
			"email":               userEmail,
			"profile_photo_url":   userPhoto,
			"status":              user.Status,
			"platform_roles":      platformRoles,
			"society_assignments": societyAssignments,
			"user": map[string]interface{}{
				"id":                user.ID,
				"phone":             auth.MaskPhone(user.Phone),
				"first_name":        userFirstName,
				"last_name":         userLastName,
				"display_name":      userDisplayName,
				"email":             userEmail,
				"profile_photo_url": userPhoto,
				"status":            user.Status,
			},
		}, nil)
	}
}

// GetProfileHandler handles GET /api/v1/auth/profile.
func GetProfileHandler(authService *auth.Service, societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication", reqID)
			return
		}

		if authService == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication service unavailable", reqID)
			return
		}

		profile, err := authService.GetProfile(r.Context(), userID)
		if err != nil {
			if errors.Is(err, auth.ErrUserNotFound) {
				response.Error(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "User profile not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve user profile", reqID)
			return
		}

		platformRoles := []string{}
		if platformRepo != nil {
			if roles, err := platformRepo.GetUserPlatformRoles(r.Context(), userID); err == nil && roles != nil {
				platformRoles = roles
			}
		}

		societyAssignments := []society.UserSocietyAssignment{}
		if societyRepo != nil {
			if assignments, err := societyRepo.ListUserSocietyAssignments(r.Context(), userID); err == nil && assignments != nil {
				societyAssignments = assignments
			}
		}

		emailStr := ""
		if profile.Email != nil {
			emailStr = *profile.Email
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":                  profile.UserID,
			"phone":               profile.Phone,
			"phone_raw":           profile.PhoneRaw,
			"first_name":          profile.FirstName,
			"last_name":           profile.LastName,
			"display_name":        profile.DisplayName,
			"email":               emailStr,
			"profile_photo_url":   profile.ProfilePhotoURL,
			"status":              profile.Status,
			"platform_roles":      platformRoles,
			"society_assignments": societyAssignments,
		}, nil)
	}
}

// UpdateProfileRequest defines the body for PATCH /api/v1/auth/profile.
type UpdateProfileRequest struct {
	FirstName       string  `json:"first_name"`
	LastName        *string `json:"last_name"`
	Email           *string `json:"email"`
	ProfilePhotoURL *string `json:"profile_photo_url,omitempty"`
}

// UpdateProfileHandler handles PATCH /api/v1/auth/profile.
func UpdateProfileHandler(authService *auth.Service, societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication", reqID)
			return
		}

		if authService == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Authentication service unavailable", reqID)
			return
		}

		var req UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		trimmedFirst := strings.TrimSpace(req.FirstName)
		if trimmedFirst == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "First name cannot be empty", reqID)
			return
		}

		updated, err := authService.UpdateProfile(r.Context(), userID, auth.UpdateProfileParams{
			FirstName:       trimmedFirst,
			LastName:        req.LastName,
			Email:           req.Email,
			ProfilePhotoURL: req.ProfilePhotoURL,
		})
		if err != nil {
			response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			return
		}

		platformRoles := []string{}
		if platformRepo != nil {
			if roles, err := platformRepo.GetUserPlatformRoles(r.Context(), userID); err == nil && roles != nil {
				platformRoles = roles
			}
		}

		societyAssignments := []society.UserSocietyAssignment{}
		if societyRepo != nil {
			if assignments, err := societyRepo.ListUserSocietyAssignments(r.Context(), userID); err == nil && assignments != nil {
				societyAssignments = assignments
			}
		}

		emailStr := ""
		if updated.Email != nil {
			emailStr = *updated.Email
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":                  updated.UserID,
			"phone":               updated.Phone,
			"phone_raw":           updated.PhoneRaw,
			"first_name":          updated.FirstName,
			"last_name":           updated.LastName,
			"display_name":        updated.DisplayName,
			"email":               emailStr,
			"profile_photo_url":   updated.ProfilePhotoURL,
			"status":              updated.Status,
			"platform_roles":      platformRoles,
			"society_assignments": societyAssignments,
		}, nil)
	}
}

// GetUserActivityTimelineHandler handles GET /api/v1/auth/profile/activity
func GetUserActivityTimelineHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		if societyRepo == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
			return
		}

		category := r.URL.Query().Get("category")
		limit := 30
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil {
				limit = l
			}
		}

		offset := 0
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil {
				offset = o
			}
		}

		timeline, err := societyRepo.GetUserActivityTimeline(r.Context(), userID, category, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve activity timeline", reqID)
			return
		}

		response.Success(w, http.StatusOK, timeline, reqID)
	}
}

// ListUserSessionsHandler handles GET /api/v1/auth/sessions.
// Returns only the authenticated user's currently active sessions (max 2), excluding raw IP addresses.
func ListUserSessionsHandler(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		currentSessionID, _ := middleware.GetSessionID(r.Context())

		sessions, err := authService.ListActiveSessions(r.Context(), userID, currentSessionID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve active sessions", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"sessions":            sessions,
			"max_active_sessions": 2,
		}, reqID)
	}
}

// RevokeUserSessionHandler handles DELETE /api/v1/auth/sessions/{id}.
// Allows the authenticated user to revoke an active session that belongs to them.
func RevokeUserSessionHandler(authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		targetSessionID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_SESSION_ID", "Invalid session ID format", reqID)
			return
		}

		if err := authService.RevokeSession(r.Context(), userID, targetSessionID); err != nil {
			if errors.Is(err, auth.ErrSessionNotFound) {
				response.Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "Active session not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke session", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]string{
			"message": "Session revoked successfully",
		}, reqID)
	}
}
