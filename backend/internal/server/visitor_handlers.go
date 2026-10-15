package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// RegisterExpectedVisitHandler handles POST /api/v1/societies/{id}/visitors/expected
func RegisterExpectedVisitHandler(societyRepo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var params society.RegisterExpectedVisitParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		if params.UnitID == uuid.Nil || params.VisitorName == "" || params.VisitorPhone == "" || params.Purpose == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "unit_id, visitor_name, visitor_phone, and purpose are required", reqID)
			return
		}

		// Authorization: SOC_ADMIN or resident occupant of the target unit
		callerRole, _ := middleware.GetSocietyRole(r.Context())
		if callerRole != "SUPER_ADMIN" && callerRole != society.RoleSocietyAdmin {
			isOccupant, _, err := societyRepo.ValidateUserUnitOccupancy(r.Context(), societyID, params.UnitID, actorID)
			if err != nil || !isOccupant {
				response.Error(w, http.StatusForbidden, "OCCUPANT_UNAUTHORIZED", "You are not authorized to register visits for this unit", reqID)
				return
			}
		}

		visit, pass, err := societyRepo.RegisterExpectedVisit(r.Context(), societyID, params, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrUnitNotFound) {
				response.Error(w, http.StatusNotFound, "UNIT_NOT_FOUND", "Target unit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidPhoneFormat) {
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Invalid visitor phone number", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		resData := map[string]interface{}{
			"visit": visit,
		}
		if pass != nil {
			resData["pass"] = pass
		}

		response.Success(w, http.StatusCreated, resData, reqID)
	}
}

// RegisterWalkInVisitHandler handles POST /api/v1/societies/{id}/visitors/walk-in
func RegisterWalkInVisitHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var params society.RegisterWalkInVisitParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		if params.UnitID == uuid.Nil || params.VisitorName == "" || params.VisitorPhone == "" || params.Purpose == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "unit_id, visitor_name, visitor_phone, and purpose are required", reqID)
			return
		}

		visit, err := societyRepo.RegisterWalkInVisit(r.Context(), societyID, params, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrUnitNotFound) {
				response.Error(w, http.StatusNotFound, "UNIT_NOT_FOUND", "Target unit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidPhoneFormat) {
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Invalid visitor phone number", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"visit": visit,
		}, reqID)
	}
}

// ListVisitsHandler handles GET /api/v1/societies/{id}/visitors
func ListVisitsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		filter := society.VisitFilter{
			Status:    r.URL.Query().Get("status"),
			VisitType: r.URL.Query().Get("visit_type"),
			Search:    r.URL.Query().Get("search"),
		}

		if unitStr := r.URL.Query().Get("unit_id"); unitStr != "" {
			if parsedUnit, err := uuid.Parse(unitStr); err == nil {
				filter.UnitID = &parsedUnit
			}
		}

		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil {
				filter.Limit = l
			}
		}
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil {
				filter.Offset = o
			}
		}

		visits, total, err := societyRepo.ListVisits(r.Context(), societyID, filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve visits", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visits": visits,
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
		}, reqID)
	}
}

// GetVisitorDashboardHandler handles GET /api/v1/societies/{id}/visitors/dashboard
func GetVisitorDashboardHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		metrics, err := societyRepo.GetVisitorDashboardMetrics(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate dashboard metrics", reqID)
			return
		}

		response.Success(w, http.StatusOK, metrics, reqID)
	}
}

// GetVisitDetailsHandler handles GET /api/v1/societies/{id}/visitors/{visitId}
func GetVisitDetailsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		visit, passes, approvals, events, err := societyRepo.GetVisitDetails(r.Context(), societyID, visitID)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found in society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get visit details", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit":     visit,
			"passes":    passes,
			"approvals": approvals,
			"events":    events,
		}, reqID)
	}
}

// ApproveVisitHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/approve
func ApproveVisitHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		var body struct {
			Notes *string `json:"notes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		// Authorization: SOC_ADMIN or resident occupant of the visit's unit
		callerRole, _ := middleware.GetSocietyRole(r.Context())
		if callerRole == society.RoleGatekeeper {
			response.Error(w, http.StatusForbidden, "GATEKEEPER_UNAUTHORIZED", "Gatekeepers are not permitted to approve or reject visits", reqID)
			return
		}
		if callerRole != "SUPER_ADMIN" && callerRole != society.RoleSocietyAdmin {
			visit, _, _, _, err := societyRepo.GetVisitDetails(r.Context(), societyID, visitID)
			if err != nil {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
				return
			}
			isOccupant, _, err := societyRepo.ValidateUserUnitAuthorization(r.Context(), societyID, visit.UnitID, actorID)
			if err != nil || !isOccupant {
				response.Error(w, http.StatusForbidden, "OCCUPANT_UNAUTHORIZED", "Only authorized unit occupants or owners can approve this visit", reqID)
				return
			}
		}

		updated, err := societyRepo.ApproveVisit(r.Context(), societyID, visitID, body.Notes, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidVisitState) {
				response.Error(w, http.StatusConflict, "INVALID_STATE", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit": updated,
		}, reqID)
	}
}

// RejectVisitHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/reject
func RejectVisitHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		var body struct {
			Reason *string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		callerRole, _ := middleware.GetSocietyRole(r.Context())
		if callerRole == society.RoleGatekeeper {
			response.Error(w, http.StatusForbidden, "GATEKEEPER_UNAUTHORIZED", "Gatekeepers are not permitted to approve or reject visits", reqID)
			return
		}
		if callerRole != "SUPER_ADMIN" && callerRole != society.RoleSocietyAdmin {
			visit, _, _, _, err := societyRepo.GetVisitDetails(r.Context(), societyID, visitID)
			if err != nil {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
				return
			}
			isOccupant, _, err := societyRepo.ValidateUserUnitAuthorization(r.Context(), societyID, visit.UnitID, actorID)
			if err != nil || !isOccupant {
				response.Error(w, http.StatusForbidden, "OCCUPANT_UNAUTHORIZED", "Only authorized unit occupants or owners can reject this visit", reqID)
				return
			}
		}

		updated, err := societyRepo.RejectVisit(r.Context(), societyID, visitID, body.Reason, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidVisitState) {
				response.Error(w, http.StatusConflict, "INVALID_STATE", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit": updated,
		}, reqID)
	}
}

// CancelVisitHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/cancel
func CancelVisitHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		callerRole, _ := middleware.GetSocietyRole(r.Context())
		if callerRole != "SUPER_ADMIN" && callerRole != society.RoleSocietyAdmin {
			visit, _, _, _, err := societyRepo.GetVisitDetails(r.Context(), societyID, visitID)
			if err != nil {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
				return
			}
			if visit.CreatedBy != actorID {
				isOccupant, _, err := societyRepo.ValidateUserUnitOccupancy(r.Context(), societyID, visit.UnitID, actorID)
				if err != nil || !isOccupant {
					response.Error(w, http.StatusForbidden, "OCCUPANT_UNAUTHORIZED", "Not authorized to cancel this visit", reqID)
					return
				}
			}
		}

		updated, err := societyRepo.CancelVisit(r.Context(), societyID, visitID, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found in society", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidVisitState) {
				response.Error(w, http.StatusConflict, "INVALID_STATE", err.Error(), reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit": updated,
		}, reqID)
	}
}

// IssueVisitorPassHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/passes
func IssueVisitorPassHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		var body struct {
			ValidFrom  *time.Time `json:"valid_from"`
			ValidUntil *time.Time `json:"valid_until"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		now := time.Now().UTC()
		validFrom := now
		if body.ValidFrom != nil {
			validFrom = *body.ValidFrom
		}
		validUntil := validFrom.Add(24 * time.Hour)
		if body.ValidUntil != nil {
			validUntil = *body.ValidUntil
		}

		pass, bearerToken, err := societyRepo.IssueVisitorPass(r.Context(), societyID, visitID, validFrom, validUntil, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrVisitNotFound) {
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
				return
			}
			if errors.Is(err, society.ErrInvalidVisitState) {
				response.Error(w, http.StatusConflict, "VISIT_NOT_APPROVED", "Cannot issue pass for unapproved visit", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"pass":         pass,
			"bearer_token": bearerToken,
		}, reqID)
	}
}

// VerifyVisitorPassHandler handles POST /api/v1/societies/{id}/visitors/passes/verify
func VerifyVisitorPassHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var body struct {
			TokenOrCode string `json:"token_or_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TokenOrCode == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "token_or_code is required", reqID)
			return
		}

		pass, visit, err := societyRepo.VerifyVisitorPass(r.Context(), societyID, body.TokenOrCode)
		if err != nil {
			if errors.Is(err, society.ErrPassNotFound) {
				response.Error(w, http.StatusNotFound, "PASS_NOT_FOUND", "Pass not found", reqID)
				return
			}
			if errors.Is(err, society.ErrPassRevoked) {
				response.Error(w, http.StatusConflict, "PASS_REVOKED", "Visitor pass has been revoked", reqID)
				return
			}
			if errors.Is(err, society.ErrPassAlreadyConsumed) {
				response.Error(w, http.StatusConflict, "PASS_CONSUMED", "Visitor pass has already been used", reqID)
				return
			}
			if errors.Is(err, society.ErrPassExpired) {
				response.Error(w, http.StatusConflict, "PASS_EXPIRED", "Visitor pass has expired", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"pass":  pass,
			"visit": visit,
		}, reqID)
	}
}

// CheckInVisitorHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/check-in
func CheckInVisitorHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		var body struct {
			GateID        uuid.UUID `json:"gate_id"`
			VehicleNumber *string   `json:"vehicle_number"`
			Notes         *string   `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GateID == uuid.Nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "gate_id is required", reqID)
			return
		}

		updatedVisit, event, err := societyRepo.CheckInVisitor(
			r.Context(), societyID, visitID, body.GateID, body.VehicleNumber, body.Notes,
			actorID, clientIP, userAgent,
		)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrVisitNotFound):
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
			case errors.Is(err, society.ErrVisitAlreadyCheckedIn):
				response.Error(w, http.StatusConflict, "VISIT_ALREADY_CHECKED_IN", "Visitor is already checked in", reqID)
			case errors.Is(err, society.ErrInvalidVisitState):
				response.Error(w, http.StatusConflict, "VISIT_NOT_APPROVED", err.Error(), reqID)
			case errors.Is(err, society.ErrDutySessionRequired):
				response.Error(w, http.StatusForbidden, "DUTY_SESSION_REQUIRED", "Active duty session required at target gate", reqID)
			case errors.Is(err, society.ErrGateInactive):
				response.Error(w, http.StatusBadRequest, "GATE_INACTIVE", "Target gate is inactive or maintenance", reqID)
			case errors.Is(err, society.ErrGateDirectionMismatch):
				response.Error(w, http.StatusBadRequest, "GATE_DIRECTION_MISMATCH", "Gate does not allow inbound entry", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit": updatedVisit,
			"event": event,
		}, reqID)
	}
}

// CheckOutVisitorHandler handles POST /api/v1/societies/{id}/visitors/{visitId}/check-out
func CheckOutVisitorHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		visitIDStr := chi.URLParam(r, "visitId")
		visitID, err := uuid.Parse(visitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid visit identifier", reqID)
			return
		}

		var body struct {
			GateID uuid.UUID `json:"gate_id"`
			Notes  *string   `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.GateID == uuid.Nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "gate_id is required", reqID)
			return
		}

		updatedVisit, event, err := societyRepo.CheckOutVisitor(
			r.Context(), societyID, visitID, body.GateID, body.Notes,
			actorID, clientIP, userAgent,
		)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrVisitNotFound):
				response.Error(w, http.StatusNotFound, "VISIT_NOT_FOUND", "Visit not found", reqID)
			case errors.Is(err, society.ErrVisitAlreadyCheckedOut):
				response.Error(w, http.StatusConflict, "VISIT_ALREADY_CHECKED_OUT", "Visitor has already checked out", reqID)
			case errors.Is(err, society.ErrVisitNotCheckedIn):
				response.Error(w, http.StatusConflict, "VISIT_NOT_CHECKED_IN", "Visitor is not currently checked in", reqID)
			case errors.Is(err, society.ErrDutySessionRequired):
				response.Error(w, http.StatusForbidden, "DUTY_SESSION_REQUIRED", "Active duty session required at target gate", reqID)
			case errors.Is(err, society.ErrGateInactive):
				response.Error(w, http.StatusBadRequest, "GATE_INACTIVE", "Target gate is inactive or maintenance", reqID)
			case errors.Is(err, society.ErrGateDirectionMismatch):
				response.Error(w, http.StatusBadRequest, "GATE_DIRECTION_MISMATCH", "Gate does not allow outbound departure", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visit": updatedVisit,
			"event": event,
		}, reqID)
	}
}

// GetResidentUnitsHandler handles GET /api/v1/societies/{id}/units/my
func GetResidentUnitsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		units, err := societyRepo.GetResidentUnits(r.Context(), societyID, actorID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve authorized units", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"units": units,
		}, reqID)
	}
}

// UpdateUnitVisitorPolicyHandler handles PATCH /api/v1/societies/{id}/units/{unitId}/visitor-policy
func UpdateUnitVisitorPolicyHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		unitIDStr := chi.URLParam(r, "unitId")
		unitID, err := uuid.Parse(unitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit identifier", reqID)
			return
		}

		callerRole, _ := middleware.GetSocietyRole(r.Context())
		if callerRole == society.RoleGatekeeper {
			response.Error(w, http.StatusForbidden, "GATEKEEPER_UNAUTHORIZED", "Gatekeepers cannot modify resident visitor policies", reqID)
			return
		}

		var body struct {
			RequiresVisitorApproval bool `json:"requires_visitor_approval"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		unit, err := societyRepo.UpdateUnitVisitorPolicy(r.Context(), societyID, unitID, actorID, body.RequiresVisitorApproval, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrUnauthorizedResident) {
				response.Error(w, http.StatusForbidden, "OCCUPANT_UNAUTHORIZED", "Only authorized unit occupants or owners can modify this preference", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"unit": unit,
		}, reqID)
	}
}

// ListPendingVisitsHandler handles GET /api/v1/societies/{id}/visitors/pending
func ListPendingVisitsHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		callerRole, _ := middleware.GetSocietyRole(r.Context())
		visits, err := societyRepo.ListPendingVisitsForResident(r.Context(), societyID, actorID, callerRole)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve pending visits", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"visits": visits,
		}, reqID)
	}
}
