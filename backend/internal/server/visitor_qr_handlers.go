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
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// Public QR Request DTOs
type PublicSendOTPRequest struct {
	Phone   string `json:"phone"`
	QRToken string `json:"qr_token"`
}

type PublicVerifyOTPRequest struct {
	Phone   string `json:"phone"`
	OTP     string `json:"otp"`
	QRToken string `json:"qr_token"`
}

type PublicVisitorRegisterRequest struct {
	QRToken           string     `json:"qr_token"`
	FullName          string     `json:"full_name"`
	Phone             string     `json:"phone"`
	VerificationToken string     `json:"verification_token"`
	Purpose           string     `json:"purpose"`
	PurposeOther      string     `json:"purpose_other,omitempty"`
	ComingFrom        string     `json:"coming_from,omitempty"`
	VisitorPhotoURL   string     `json:"visitor_photo_url,omitempty"`
	UnitID            *uuid.UUID `json:"unit_id,omitempty"`
	VehicleNumber     string     `json:"vehicle_number,omitempty"`
}

type AdmitQRVisitorRequest struct {
	Decision        string    `json:"decision"` // APPROVE, REJECT
	GateID          uuid.UUID `json:"gate_id"`
	RejectionReason string    `json:"rejection_reason,omitempty"`
}

type CreateGateQRRequest struct {
	GateID *uuid.UUID `json:"gate_id,omitempty"`
	Label  string     `json:"label"`
}

// ResolvePublicGateQRHandler resolves a public QR token to its safe display context.
func ResolvePublicGateQRHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		token := chi.URLParam(r, "token")

		if strings.TrimSpace(token) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_TOKEN", "Registration QR token is required", reqID)
			return
		}

		ctx, err := societyRepo.ResolvePublicGateQR(r.Context(), token)
		if err != nil {
			if errors.Is(err, society.ErrQRNotFound) {
				response.Error(w, http.StatusNotFound, "QR_NOT_FOUND", "Visitor registration point not found", reqID)
				return
			}
			if errors.Is(err, society.ErrQRInactive) {
				response.Error(w, http.StatusGone, "QR_INACTIVE", "This visitor registration QR is inactive or revoked", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve registration QR", reqID)
			return
		}

		response.Success(w, http.StatusOK, ctx, nil)
	}
}

// SendVisitorOTPHandler dispatches an OTP for public visitor phone verification.
func SendVisitorOTPHandler(authService *auth.Service, societyRepo *society.Repository, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		clientIP := r.RemoteAddr

		var req PublicSendOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		if strings.TrimSpace(req.Phone) == "" {
			response.Error(w, http.StatusBadRequest, "PHONE_REQUIRED", "Phone number is required", reqID)
			return
		}
		if strings.TrimSpace(req.QRToken) == "" {
			response.Error(w, http.StatusBadRequest, "QR_TOKEN_REQUIRED", "Registration QR token is required", reqID)
			return
		}

		// Ensure QR token is currently active
		_, err := societyRepo.ResolvePublicGateQR(r.Context(), req.QRToken)
		if err != nil {
			if errors.Is(err, society.ErrQRNotFound) {
				response.Error(w, http.StatusNotFound, "QR_NOT_FOUND", "Visitor registration point not found", reqID)
				return
			}
			response.Error(w, http.StatusGone, "QR_INACTIVE", "This visitor registration point is not active", reqID)
			return
		}

		resendSec, devOTP, err := authService.SendOTP(r.Context(), req.Phone, clientIP)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Please provide a valid Indian mobile number", reqID)
			case errors.Is(err, auth.ErrOTPCooldown):
				response.Error(w, http.StatusTooManyRequests, "OTP_COOLDOWN", err.Error(), reqID)
			case errors.Is(err, auth.ErrPhoneRateLimited), errors.Is(err, auth.ErrIPRateLimited):
				response.Error(w, http.StatusTooManyRequests, "OTP_RATE_LIMITED", err.Error(), reqID)
			default:
				response.Error(w, http.StatusBadRequest, "OTP_DISPATCH_FAILED", err.Error(), reqID)
			}
			return
		}

		data := map[string]interface{}{
			"message":              "Verification OTP sent successfully",
			"resend_after_seconds": resendSec,
		}
		if cfg != nil && cfg.Env != "production" && devOTP != "" {
			data["otp"] = devOTP
		}

		response.Success(w, http.StatusOK, data, nil)
	}
}

// VerifyVisitorOTPHandler verifies visitor OTP and returns an HMAC-signed phone verification token.
func VerifyVisitorOTPHandler(authService *auth.Service, societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		clientIP := r.RemoteAddr

		var req PublicVerifyOTPRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		if strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.OTP) == "" {
			response.Error(w, http.StatusBadRequest, "FIELDS_REQUIRED", "Phone and verification OTP are required", reqID)
			return
		}

		// Validate QR code is active
		_, err := societyRepo.ResolvePublicGateQR(r.Context(), req.QRToken)
		if err != nil {
			response.Error(w, http.StatusGone, "QR_INACTIVE", "Registration point is inactive or invalid", reqID)
			return
		}

		verificationToken, maskedPhone, err := authService.VerifyVisitorOTPWithContext(r.Context(), req.Phone, req.OTP, req.QRToken, clientIP)
		if err != nil {
			if errors.Is(err, auth.ErrOTPInvalid) {
				response.Error(w, http.StatusBadRequest, "INVALID_OTP", "Invalid verification code", reqID)
				return
			}
			if errors.Is(err, auth.ErrOTPExpired) {
				response.Error(w, http.StatusBadRequest, "OTP_EXPIRED", "Verification code has expired", reqID)
				return
			}
			if errors.Is(err, auth.ErrOTPAttemptsExceeded) {
				response.Error(w, http.StatusTooManyRequests, "ATTEMPTS_EXCEEDED", "Maximum verification attempts exceeded", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "VERIFICATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"verification_token": verificationToken,
			"phone_masked":       maskedPhone,
			"verified":           true,
		}, nil)
	}
}

// RegisterVisitorByQRHandler processes public visitor self-registration and returns a visitor pass.
func RegisterVisitorByQRHandler(societyRepo *society.Repository, authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		var req PublicVisitorRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid registration payload", reqID)
			return
		}

		if strings.TrimSpace(req.FullName) == "" {
			response.Error(w, http.StatusBadRequest, "NAME_REQUIRED", "Full name is required", reqID)
			return
		}
		if strings.TrimSpace(req.Phone) == "" {
			response.Error(w, http.StatusBadRequest, "PHONE_REQUIRED", "Phone number is required", reqID)
			return
		}
		if strings.TrimSpace(req.QRToken) == "" {
			response.Error(w, http.StatusBadRequest, "QR_TOKEN_REQUIRED", "Registration QR token is required", reqID)
			return
		}

		// Verify and atomically consume phone verification token (single-use, bound to QR context)
		valid, normPhone := authService.ConsumePhoneVerificationToken(req.Phone, req.QRToken, req.VerificationToken)
		if !valid {
			response.Error(w, http.StatusBadRequest, "PHONE_NOT_VERIFIED", "Phone verification is missing, expired, already used, or does not match phone number or QR context", reqID)
			return
		}

		// Enforce server-side photo size and format constraints
		photoURL := strings.TrimSpace(req.VisitorPhotoURL)
		if photoURL != "" {
			if len(photoURL) > 2500000 {
				response.Error(w, http.StatusBadRequest, "PHOTO_TOO_LARGE", "Visitor photo exceeds maximum allowed size (2.5MB)", reqID)
				return
			}
			isDataImage := strings.HasPrefix(photoURL, "data:image/jpeg;base64,") ||
				strings.HasPrefix(photoURL, "data:image/png;base64,") ||
				strings.HasPrefix(photoURL, "data:image/webp;base64,") ||
				strings.HasPrefix(photoURL, "data:image/jpg;base64,")
			isHttps := strings.HasPrefix(photoURL, "https://")
			if !isDataImage && !isHttps {
				response.Error(w, http.StatusBadRequest, "INVALID_PHOTO_FORMAT", "Visitor photo must be a valid JPEG/PNG/WebP image or secure HTTPS URL", reqID)
				return
			}
		}

		purpose := strings.TrimSpace(req.Purpose)
		if purpose == "OTHER" && strings.TrimSpace(req.PurposeOther) != "" {
			purpose = "OTHER: " + strings.TrimSpace(req.PurposeOther)
		} else if purpose == "" {
			purpose = "GUEST"
		}

		var authUserID *uuid.UUID
		if actorID, ok := middleware.GetUserID(r.Context()); ok && actorID != uuid.Nil {
			authUserID = &actorID
		}

		params := society.QRVisitorRegisterParams{
			QRToken:           req.QRToken,
			FullName:          strings.TrimSpace(req.FullName),
			Phone:             normPhone,
			Purpose:           purpose,
			ComingFrom:        strings.TrimSpace(req.ComingFrom),
			VisitorPhotoURL:   photoURL,
			UnitID:            req.UnitID,
			VehicleNumber:     strings.TrimSpace(req.VehicleNumber),
			AuthenticatedUser: authUserID,
		}

		pass, err := societyRepo.RegisterVisitorByQR(r.Context(), params)
		if err != nil {
			if errors.Is(err, society.ErrQRNotFound) {
				response.Error(w, http.StatusNotFound, "QR_NOT_FOUND", "Registration point not found", reqID)
				return
			}
			if errors.Is(err, society.ErrQRInactive) {
				response.Error(w, http.StatusGone, "QR_INACTIVE", "Registration point is inactive or revoked", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "REGISTRATION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, pass, nil)
	}
}

// GetPublicVisitorPassHandler allows visitors to retrieve status of their digital pass by reference.
func GetPublicVisitorPassHandler(societyRepo *society.Repository, authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		clientIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
		}

		if authService != nil && authService.RateLimiter() != nil {
			if err := authService.RateLimiter().CheckAndRecordPublicPassIP(clientIP); err != nil {
				response.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many pass lookup requests, please try again shortly", reqID)
				return
			}
		}

		passRef := chi.URLParam(r, "passReference")

		if strings.TrimSpace(passRef) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_REFERENCE", "Pass reference is required", reqID)
			return
		}

		pass, err := societyRepo.GetPublicVisitorPass(r.Context(), passRef)
		if err != nil {
			if errors.Is(err, society.ErrPassNotFound) {
				response.Error(w, http.StatusNotFound, "PASS_NOT_FOUND", "Visitor pass not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve pass details", reqID)
			return
		}

		response.Success(w, http.StatusOK, pass, nil)
	}
}

// ListPendingQRRegistrationsHandler returns visits pending gatekeeper approval.
func ListPendingQRRegistrationsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		var gateID *uuid.UUID
		if gStr := r.URL.Query().Get("gate_id"); gStr != "" {
			if gid, err := uuid.Parse(gStr); err == nil {
				gateID = &gid
			}
		}

		search := r.URL.Query().Get("search")
		limit := 50
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		list, err := societyRepo.ListPendingQRRegistrations(r.Context(), societyID, gateID, search, limit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query pending registrations", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"pending_registrations": list,
			"total":                 len(list),
		}, nil)
	}
}

// AdmitQRVisitorHandler allows gatekeepers to admit or reject self-registered visitors atomically.
func AdmitQRVisitorHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		visitID, err := uuid.Parse(chi.URLParam(r, "visitId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_VISIT_ID", "Invalid visit ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", reqID)
			return
		}

		var req AdmitQRVisitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid decision payload", reqID)
			return
		}

		if req.GateID == uuid.Nil {
			response.Error(w, http.StatusBadRequest, "GATE_REQUIRED", "Gate assignment is required for admission", reqID)
			return
		}

		visit, err := societyRepo.AdmitQRVisitor(r.Context(), societyID, visitID, req.GateID, actorID, req.Decision, req.RejectionReason, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrVisitAlreadyDecided) {
				response.Error(w, http.StatusConflict, "VISIT_ALREADY_DECIDED", "This visitor registration has already been approved or rejected", reqID)
				return
			}
			if errors.Is(err, society.ErrGateDirectionMismatch) {
				response.Error(w, http.StatusBadRequest, "GATE_DIRECTION_MISMATCH", "Target gate does not permit entry (egress only)", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "ADMISSION_FAILED", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit":   visit,
			"message": "Visitor admission decision recorded successfully",
		}, nil)
	}
}

// CreateGateQRHandler creates a gate visitor QR code.
func CreateGateQRHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", reqID)
			return
		}

		var req CreateGateQRRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Check if gateId is in URL
		if gStr := chi.URLParam(r, "gateId"); gStr != "" {
			if gid, err := uuid.Parse(gStr); err == nil {
				req.GateID = &gid
			}
		}

		qr, err := societyRepo.CreateGateQR(r.Context(), societyID, req.GateID, req.Label, actorID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate gate QR", reqID)
			return
		}

		response.Success(w, http.StatusCreated, qr, nil)
	}
}

// ListGateQRsHandler lists QR codes for a gate or society.
func ListGateQRsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		var gateID *uuid.UUID
		if gStr := chi.URLParam(r, "gateId"); gStr != "" {
			if gid, err := uuid.Parse(gStr); err == nil {
				gateID = &gid
			}
		}

		qrs, err := societyRepo.ListGateQRs(r.Context(), societyID, gateID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list gate QRs", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"gate_qrs": qrs,
		}, nil)
	}
}

// RevokeGateQRHandler revokes an active QR code.
func RevokeGateQRHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		qrID, err := uuid.Parse(chi.URLParam(r, "qrId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_QR_ID", "Invalid QR ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", reqID)
			return
		}

		if err := societyRepo.RevokeGateQR(r.Context(), societyID, qrID, actorID); err != nil {
			if errors.Is(err, society.ErrQRNotFound) {
				response.Error(w, http.StatusNotFound, "QR_NOT_FOUND", "Gate QR not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke gate QR", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]string{
			"message": "Gate QR code revoked successfully",
		}, nil)
	}
}

// RotateGateQRHandler invalidates existing QR and generates a fresh token.
func RotateGateQRHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			var err error
			societyID, err = uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				response.Error(w, http.StatusBadRequest, "INVALID_SOCIETY_ID", "Invalid society ID", reqID)
				return
			}
		}

		qrID, err := uuid.Parse(chi.URLParam(r, "qrId"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_QR_ID", "Invalid QR ID", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", reqID)
			return
		}

		newQR, err := societyRepo.RotateGateQR(r.Context(), societyID, qrID, actorID)
		if err != nil {
			if errors.Is(err, society.ErrQRNotFound) {
				response.Error(w, http.StatusNotFound, "QR_NOT_FOUND", "Gate QR not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rotate gate QR", reqID)
			return
		}

		response.Success(w, http.StatusOK, newQR, nil)
	}
}

// GetMyVisitorHistoryHandler returns visit history for the authenticated user across societies.
func GetMyVisitorHistoryHandler(societyRepo *society.Repository, authService *auth.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", reqID)
			return
		}

		// Fetch user's verified phone
		profile, err := authService.GetProfile(r.Context(), actorID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "User profile not found", reqID)
			return
		}

		limit := 50
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		visits, err := societyRepo.ListVisitorSelfHistory(r.Context(), actorID, profile.PhoneRaw, limit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query visit history", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visits": visits,
			"total":  len(visits),
		}, nil)
	}
}
