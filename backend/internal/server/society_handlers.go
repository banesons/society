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
	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

type CreateSocietyRequest struct {
	Name               string  `json:"name"`
	RegistrationNumber *string `json:"registration_number,omitempty"`
	AddressLine1       string  `json:"address_line1"`
	AddressLine2       *string `json:"address_line2,omitempty"`
	Locality           string  `json:"locality"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Pincode            string  `json:"pincode"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
}

type UpdateSocietyRequest struct {
	Name               string  `json:"name"`
	RegistrationNumber *string `json:"registration_number,omitempty"`
	AddressLine1       string  `json:"address_line1"`
	AddressLine2       *string `json:"address_line2,omitempty"`
	Locality           string  `json:"locality"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Pincode            string  `json:"pincode"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
}

type StatusActionRequest struct {
	Reason string `json:"reason"`
}

// ListSocietiesHandler handles GET /api/v1/platform/societies.
func ListSocietiesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
		searchTerm := strings.TrimSpace(r.URL.Query().Get("search"))

		items, total, err := repo.ListSocieties(r.Context(), society.ListSocietiesFilter{
			Status: statusFilter,
			Search: searchTerm,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve societies", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"items":  items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		}, nil)
	}
}

// CreateSocietyHandler handles POST /api/v1/platform/societies.
func CreateSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req CreateSocietyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		item, err := repo.CreateSociety(r.Context(), society.CreateSocietyParams{
			Name:               req.Name,
			RegistrationNumber: req.RegistrationNumber,
			AddressLine1:       req.AddressLine1,
			AddressLine2:       req.AddressLine2,
			Locality:           req.Locality,
			City:               req.City,
			State:              req.State,
			Pincode:            req.Pincode,
			ContactEmail:       req.ContactEmail,
			ContactPhone:       req.ContactPhone,
		}, actorID, clientIP, userAgent)

		if err != nil {
			switch {
			case errors.Is(err, society.ErrInvalidName):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Society name must be between 3 and 255 characters", reqID)
			case errors.Is(err, society.ErrRequiredFieldMissing):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Address line 1, locality, city, and state are required", reqID)
			case errors.Is(err, society.ErrInvalidPincode):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "PIN code must be a valid 6-digit Indian postal code", reqID)
			case errors.Is(err, society.ErrInvalidCity):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "City name cannot be converted to a valid city code", reqID)
			case errors.Is(err, society.ErrRegNumberExists):
				response.Error(w, http.StatusConflict, "REG_NUMBER_EXISTS", "Registration number already exists in this state", reqID)
			case errors.Is(err, society.ErrMaxRetriesExceeded):
				response.Error(w, http.StatusInternalServerError, "CODE_GENERATION_FAILED", "Unable to generate unique society code, please retry", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create society", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, item, nil)
	}
}

// GetSocietyHandler handles GET /api/v1/platform/societies/{id}.
func GetSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		idParam := chi.URLParam(r, "id")

		var soc *society.Society
		var err error

		if idUUID, parseErr := uuid.Parse(idParam); parseErr == nil {
			soc, err = repo.GetSocietyByID(r.Context(), idUUID)
		} else {
			soc, err = repo.GetSocietyByCode(r.Context(), idParam)
		}

		if err != nil {
			if errors.Is(err, society.ErrSocietyNotFound) {
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve society", reqID)
			return
		}

		response.Success(w, http.StatusOK, soc, nil)
	}
}

// UpdateSocietyHandler handles PATCH /api/v1/platform/societies/{id}.
func UpdateSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		idUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		var req UpdateSocietyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		soc, err := repo.UpdateSocietyProfile(r.Context(), idUUID, society.UpdateSocietyParams{
			Name:               req.Name,
			RegistrationNumber: req.RegistrationNumber,
			AddressLine1:       req.AddressLine1,
			AddressLine2:       req.AddressLine2,
			Locality:           req.Locality,
			City:               req.City,
			State:              req.State,
			Pincode:            req.Pincode,
			ContactEmail:       req.ContactEmail,
			ContactPhone:       req.ContactPhone,
		}, actorID, clientIP, userAgent)

		if err != nil {
			switch {
			case errors.Is(err, society.ErrSocietyNotFound):
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
			case errors.Is(err, society.ErrInvalidName):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Society name must be between 3 and 255 characters", reqID)
			case errors.Is(err, society.ErrRequiredFieldMissing):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Address line 1, locality, city, and state are required", reqID)
			case errors.Is(err, society.ErrInvalidPincode):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "PIN code must be a valid 6-digit Indian postal code", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update society profile", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, soc, nil)
	}
}

// ActivateSocietyHandler handles POST /api/v1/platform/societies/{id}/activate.
func ActivateSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		idUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		var req StatusActionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		soc, err := repo.ActivateSociety(r.Context(), idUUID, req.Reason, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrSocietyNotFound):
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
			case errors.Is(err, society.ErrAlreadyInStatus):
				response.Error(w, http.StatusConflict, "ALREADY_IN_STATUS", "Society is already in ACTIVE status", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to activate society", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, soc, nil)
	}
}

// SuspendSocietyRequest represents payload for suspending a society.
type SuspendSocietyRequest struct {
	ReasonID      *int16  `json:"reason_id,omitempty"`
	ReasonCode    *string `json:"reason_code,omitempty"`
	ReasonDetails *string `json:"reason_details,omitempty"`
}

// ListSuspensionReasonsHandler handles GET /api/v1/platform/society-suspension-reasons.
func ListSuspensionReasonsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		reasons, err := repo.ListSuspensionReasons(r.Context(), true)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve suspension reasons", reqID)
			return
		}

		response.Success(w, http.StatusOK, reasons, nil)
	}
}

// SuspendSocietyHandler handles POST /api/v1/platform/societies/{id}/suspend.
func SuspendSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		idUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		var req SuspendSocietyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		params := society.SuspendSocietyParams{
			ReasonID:      req.ReasonID,
			ReasonCode:    req.ReasonCode,
			ReasonDetails: req.ReasonDetails,
		}

		soc, err := repo.SuspendSociety(r.Context(), idUUID, params, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrSocietyNotFound):
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
			case errors.Is(err, society.ErrAlreadyInStatus):
				response.Error(w, http.StatusConflict, "ALREADY_IN_STATUS", "Society is already in SUSPENDED status", reqID)
			case errors.Is(err, society.ErrInvalidStatusTransition):
				response.Error(w, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", "Cannot suspend a society in PENDING status", reqID)
			case errors.Is(err, society.ErrSuspensionReasonRequired):
				response.Error(w, http.StatusBadRequest, "SUSPENDED_REASON_REQUIRED", "Suspension reason code is required", reqID)
			case errors.Is(err, society.ErrInvalidSuspensionReason):
				response.Error(w, http.StatusBadRequest, "INVALID_SUSPENSION_REASON", "Invalid suspension reason code", reqID)
			case errors.Is(err, society.ErrSuspensionReasonInactive):
				response.Error(w, http.StatusBadRequest, "SUSPENDED_REASON_INACTIVE", "Selected suspension reason is inactive", reqID)
			case errors.Is(err, society.ErrManualSuspensionNotAllowed):
				response.Error(w, http.StatusBadRequest, "MANUAL_SUSPENSION_NOT_ALLOWED", "Selected reason is not permitted for manual suspension", reqID)
			case errors.Is(err, society.ErrSuspensionDetailsRequired):
				response.Error(w, http.StatusBadRequest, "SUSPENDED_DETAILS_REQUIRED", "Additional details are required for this reason", reqID)
			case errors.Is(err, society.ErrDetailsTooLong):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Additional details must not exceed 1000 characters", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to suspend society", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, soc, nil)
	}
}

// ReactivateSocietyHandler handles POST /api/v1/platform/societies/{id}/reactivate.
func ReactivateSocietyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		idUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		var req StatusActionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		soc, err := repo.ReactivateSociety(r.Context(), idUUID, req.Reason, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrSocietyNotFound):
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
			case errors.Is(err, society.ErrAlreadyInStatus):
				response.Error(w, http.StatusConflict, "ALREADY_IN_STATUS", "Society is already in ACTIVE status", reqID)
			case errors.Is(err, society.ErrInvalidStatusTransition):
				response.Error(w, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", "Cannot reactivate a society in PENDING status", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reactivate society", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, soc, nil)
	}
}

// AssignSocietyAdminRequest holds parameters for mapping an administrator to a society.
type AssignSocietyAdminRequest struct {
	Phone  string     `json:"phone,omitempty"`
	UserID *uuid.UUID `json:"user_id,omitempty"`
	Role   string     `json:"role,omitempty"`
}

// AssignSocietyAdminHandler handles POST /api/v1/platform/societies/{id}/admins.
func AssignSocietyAdminHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		socUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		var req AssignSocietyAdminRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", reqID)
			return
		}

		if req.Phone == "" && req.UserID == nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Phone number or user_id is required", reqID)
			return
		}

		role := req.Role
		if role == "" {
			role = society.RoleSocietyAdmin
		}
		if role != society.RoleSocietyAdmin {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Only SOC_ADMIN role is supported", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		assignment, err := repo.AssignSocietyAdmin(r.Context(), society.AssignSocietyAdminParams{
			SocietyID: socUUID,
			Phone:     req.Phone,
			UserID:    req.UserID,
			Role:      role,
		}, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrSocietyNotFound):
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
			case errors.Is(err, society.ErrUserNotFound):
				response.Error(w, http.StatusNotFound, "USER_NOT_FOUND", "No registered user found with the specified phone number", reqID)
			case errors.Is(err, society.ErrSocietyAdminAlreadyAssigned):
				response.Error(w, http.StatusConflict, "SOCIETY_ADMIN_ALREADY_ASSIGNED", "User already holds an active SOC_ADMIN assignment for this society", reqID)
			case errors.Is(err, auth.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Invalid phone number format", reqID)
			case errors.Is(err, society.ErrInvalidRole):
				response.Error(w, http.StatusBadRequest, "INVALID_ROLE", "Invalid role specified", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign society administrator", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, assignment, nil)
	}
}

// ListSocietyAdminsHandler handles GET /api/v1/platform/societies/{id}/admins.
func ListSocietyAdminsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		idParam := chi.URLParam(r, "id")
		socUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
		limit := 20
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}
		offset := 0
		if oStr := r.URL.Query().Get("offset"); oStr != "" {
			if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
				offset = o
			}
		}

		result, err := repo.ListSocietyAdmins(r.Context(), society.ListSocietyAdminsFilter{
			SocietyID: socUUID,
			Status:    status,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			if errors.Is(err, society.ErrSocietyNotFound) {
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list society administrators", reqID)
			return
		}

		response.Success(w, http.StatusOK, result, nil)
	}
}

// RevokeSocietyAdminHandler handles DELETE /api/v1/platform/societies/{id}/admins/{assignment_id}.
func RevokeSocietyAdminHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idParam := chi.URLParam(r, "id")
		socUUID, err := uuid.Parse(idParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid society UUID", reqID)
			return
		}

		assignParam := chi.URLParam(r, "assignment_id")
		assignUUID, err := uuid.Parse(assignParam)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid assignment UUID", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		assignment, err := repo.RevokeSocietyAdmin(r.Context(), socUUID, assignUUID, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAssignmentNotFound):
				response.Error(w, http.StatusNotFound, "ASSIGNMENT_NOT_FOUND", "Assignment does not exist for this society", reqID)
			case errors.Is(err, society.ErrAssignmentAlreadyRevoked):
				response.Error(w, http.StatusConflict, "ALREADY_REVOKED", "Assignment has already been revoked", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke society administrator", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, assignment, nil)
	}
}

// GetSocietyProfileHandler handles GET /api/v1/societies/{id}/profile for an authenticated and verified SOC_ADMIN.
func GetSocietyProfileHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing verified society context", reqID)
			return
		}

		soc, err := repo.GetSocietyByID(r.Context(), socID)
		if err != nil {
			if errors.Is(err, society.ErrSocietyNotFound) {
				response.Error(w, http.StatusNotFound, "SOCIETY_NOT_FOUND", "Society not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve society profile", reqID)
			return
		}

		profile := map[string]interface{}{
			"id":            soc.ID,
			"society_code":  soc.SocietyCode,
			"name":          soc.Name,
			"status":        soc.Status,
			"address_line1": soc.AddressLine1,
			"address_line2": soc.AddressLine2,
			"locality":      soc.Locality,
			"city":          soc.City,
			"state":         soc.State,
			"pincode":       soc.Pincode,
			"contact_email": soc.ContactEmail,
			"contact_phone": soc.ContactPhone,
		}

		response.Success(w, http.StatusOK, profile, nil)
	}
}
