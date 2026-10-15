package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// CreateDesignationRequest payload for creating a custom designation.
type CreateDesignationRequest struct {
	Code                             string  `json:"code"`
	Name                             string  `json:"name"`
	Description                      *string `json:"description,omitempty"`
	AllowsMultipleMembers            bool    `json:"allows_multiple_members"`
	AllowsMultiplePositionsPerMember bool    `json:"allows_multiple_positions_per_member"`
}

// UpdateDesignationRequest payload for updating an existing designation.
type UpdateDesignationRequest struct {
	Name                             string  `json:"name"`
	Description                      *string `json:"description,omitempty"`
	AllowsMultipleMembers            bool    `json:"allows_multiple_members"`
	AllowsMultiplePositionsPerMember bool    `json:"allows_multiple_positions_per_member"`
}

// SetDesignationStatusRequest payload for toggling active/inactive status.
type SetDesignationStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// CreateAppointmentRequest payload for appointing a member to a designation.
type CreateAppointmentRequest struct {
	DesignationID   string  `json:"designation_id"`
	MemberID        string  `json:"member_id"`
	StartDate       string  `json:"start_date"`
	ExpectedEndDate *string `json:"expected_end_date,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

// EndAppointmentRequest payload for concluding or revoking an appointment.
type EndAppointmentRequest struct {
	Action  string  `json:"action"` // ENDED, REVOKED
	EndDate string  `json:"end_date"`
	Notes   *string `json:"notes,omitempty"`
}

// ListDesignationsHandler handles GET /api/v1/societies/{id}/designations.
func ListDesignationsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		statusFilter := r.URL.Query().Get("status")
		designations, err := repo.ListDesignations(r.Context(), socID, statusFilter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"designations": designations,
		}, reqID)
	}
}

// CreateDesignationHandler handles POST /api/v1/societies/{id}/designations.
func CreateDesignationHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req CreateDesignationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request payload", reqID)
			return
		}

		params := society.CreateDesignationParams{
			Code:                             req.Code,
			Name:                             req.Name,
			Description:                      req.Description,
			AllowsMultipleMembers:            req.AllowsMultipleMembers,
			AllowsMultiplePositionsPerMember: req.AllowsMultiplePositionsPerMember,
			ActorID:                          actorID,
			ClientIP:                         r.RemoteAddr,
			UserAgent:                        r.UserAgent(),
		}

		created, err := repo.CreateDesignation(r.Context(), socID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrInvalidDesignationCode):
				response.Error(w, http.StatusBadRequest, "INVALID_DESIGNATION_CODE", err.Error(), reqID)
			case errors.Is(err, society.ErrInvalidDesignationName):
				response.Error(w, http.StatusBadRequest, "INVALID_DESIGNATION_NAME", err.Error(), reqID)
			case errors.Is(err, society.ErrDesignationAlreadyExists):
				response.Error(w, http.StatusConflict, "DESIGNATION_ALREADY_EXISTS", "A designation with this code already exists in this society", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"designation": created,
		}, reqID)
	}
}

// UpdateDesignationHandler handles PATCH /api/v1/societies/{id}/designations/{designationId}.
func UpdateDesignationHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		designationIDStr := chi.URLParam(r, "designationId")
		designationID, err := uuid.Parse(designationIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid designation identifier", reqID)
			return
		}

		var req UpdateDesignationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request payload", reqID)
			return
		}

		params := society.UpdateDesignationParams{
			Name:                             req.Name,
			Description:                      req.Description,
			AllowsMultipleMembers:            req.AllowsMultipleMembers,
			AllowsMultiplePositionsPerMember: req.AllowsMultiplePositionsPerMember,
			ActorID:                          actorID,
			ClientIP:                         r.RemoteAddr,
			UserAgent:                        r.UserAgent(),
		}

		updated, err := repo.UpdateDesignation(r.Context(), socID, designationID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrDesignationNotFound):
				response.Error(w, http.StatusNotFound, "DESIGNATION_NOT_FOUND", "Designation not found in this society", reqID)
			case errors.Is(err, society.ErrInvalidDesignationName):
				response.Error(w, http.StatusBadRequest, "INVALID_DESIGNATION_NAME", err.Error(), reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"designation": updated,
		}, reqID)
	}
}

// SetDesignationStatusHandler handles PATCH /api/v1/societies/{id}/designations/{designationId}/status.
func SetDesignationStatusHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		designationIDStr := chi.URLParam(r, "designationId")
		designationID, err := uuid.Parse(designationIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid designation identifier", reqID)
			return
		}

		var req SetDesignationStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request payload", reqID)
			return
		}

		err = repo.SetDesignationStatus(r.Context(), socID, designationID, req.IsActive, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			if errors.Is(err, society.ErrDesignationNotFound) {
				response.Error(w, http.StatusNotFound, "DESIGNATION_NOT_FOUND", "Designation not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message":   "Designation status updated successfully",
			"is_active": req.IsActive,
		}, reqID)
	}
}

// ListAppointmentsHandler handles GET /api/v1/societies/{id}/appointments.
func ListAppointmentsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		statusFilter := r.URL.Query().Get("status")
		var memberID, designationID *uuid.UUID

		if memStr := r.URL.Query().Get("member_id"); memStr != "" {
			if id, err := uuid.Parse(memStr); err == nil {
				memberID = &id
			}
		}
		if desStr := r.URL.Query().Get("designation_id"); desStr != "" {
			if id, err := uuid.Parse(desStr); err == nil {
				designationID = &id
			}
		}

		appointments, err := repo.ListAppointments(r.Context(), socID, statusFilter, memberID, designationID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"appointments": appointments,
		}, reqID)
	}
}

// CreateAppointmentHandler handles POST /api/v1/societies/{id}/appointments.
func CreateAppointmentHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req CreateAppointmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request payload", reqID)
			return
		}

		desigID, err := uuid.Parse(req.DesignationID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid designation identifier", reqID)
			return
		}

		memID, err := uuid.Parse(req.MemberID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid member identifier", reqID)
			return
		}

		if strings.TrimSpace(req.StartDate) == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "start_date is required", reqID)
			return
		}

		params := society.CreateAppointmentParams{
			DesignationID:   desigID,
			MemberID:        memID,
			StartDate:       req.StartDate,
			ExpectedEndDate: req.ExpectedEndDate,
			Notes:           req.Notes,
			ActorID:         actorID,
			ClientIP:        r.RemoteAddr,
			UserAgent:       r.UserAgent(),
		}

		created, err := repo.CreateAppointment(r.Context(), socID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrDesignationNotFound):
				response.Error(w, http.StatusNotFound, "DESIGNATION_NOT_FOUND", "Designation not found in this society", reqID)
			case errors.Is(err, society.ErrMemberNotFound):
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
			case errors.Is(err, society.ErrDesignationInactive):
				response.Error(w, http.StatusBadRequest, "DESIGNATION_INACTIVE", "Cannot make appointments to an inactive designation", reqID)
			case errors.Is(err, society.ErrMemberNotActive):
				response.Error(w, http.StatusBadRequest, "MEMBER_NOT_ACTIVE", "Only active society members may be appointed", reqID)
			case errors.Is(err, society.ErrDesignationAlreadyAppointed):
				response.Error(w, http.StatusConflict, "DESIGNATION_ALREADY_APPOINTED", "This designation is already held by an active office-bearer", reqID)
			case errors.Is(err, society.ErrMemberAlreadyAppointed):
				response.Error(w, http.StatusConflict, "MEMBER_ALREADY_APPOINTED", "Member already holds this designation actively", reqID)
			case errors.Is(err, society.ErrInvalidAppointmentDates):
				response.Error(w, http.StatusBadRequest, "INVALID_DATES", "Expected end date must be after or equal to start date", reqID)
			case errors.Is(err, society.ErrConflictingDateRange):
				response.Error(w, http.StatusConflict, "CONFLICTING_DATE_RANGE", "Appointment date range conflicts with an existing appointment term", reqID)
			case errors.Is(err, society.ErrMultiplePositionsDisallowed):
				response.Error(w, http.StatusConflict, "MULTIPLE_DESIGNATIONS_DISALLOWED", "Member cannot hold multiple designations simultaneously", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"appointment": created,
		}, reqID)
	}
}

// EndAppointmentHandler handles POST /api/v1/societies/{id}/appointments/{appointmentId}/end.
func EndAppointmentHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		aptIDStr := chi.URLParam(r, "appointmentId")
		aptID, err := uuid.Parse(aptIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid appointment identifier", reqID)
			return
		}

		var req EndAppointmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request payload", reqID)
			return
		}

		if strings.TrimSpace(req.EndDate) == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "end_date is required", reqID)
			return
		}

		params := society.EndAppointmentParams{
			Action:    req.Action,
			EndDate:   req.EndDate,
			Notes:     req.Notes,
			ActorID:   actorID,
			ClientIP:  r.RemoteAddr,
			UserAgent: r.UserAgent(),
		}

		err = repo.EndAppointment(r.Context(), socID, aptID, params)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAppointmentNotFound):
				response.Error(w, http.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found in this society", reqID)
			case errors.Is(err, society.ErrInvalidAppointmentStatus):
				response.Error(w, http.StatusBadRequest, "APPOINTMENT_NOT_ACTIVE", "Appointment is not currently active", reqID)
			case errors.Is(err, society.ErrInvalidAppointmentDates):
				response.Error(w, http.StatusBadRequest, "INVALID_DATES", "End date must be after or equal to start date", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Appointment concluded successfully",
		}, reqID)
	}
}

// GetCommitteeOverviewHandler handles GET /api/v1/societies/{id}/committee/overview.
func GetCommitteeOverviewHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		overview, err := repo.GetCommitteeOverview(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, overview, reqID)
	}
}
