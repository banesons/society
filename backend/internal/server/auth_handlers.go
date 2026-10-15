package server

import (
	"encoding/json"
	"errors"
	"net/http"

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

		// Safe, concurrency-aware identity linking for pre-registered member records
		if societyRepo != nil {
			_ = societyRepo.SafeLinkUserMembers(r.Context(), result.User.ID, result.User.Phone)
		}

		response.Success(w, http.StatusOK, result, nil)
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

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":                  user.ID,
			"phone":               auth.MaskPhone(user.Phone),
			"status":              user.Status,
			"platform_roles":      platformRoles,
			"society_assignments": societyAssignments,
			"user": map[string]interface{}{
				"id":     user.ID,
				"phone":  auth.MaskPhone(user.Phone),
				"status": user.Status,
			},
		}, nil)
	}
}
