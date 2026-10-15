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

// UpdateGateHandler handles PATCH /api/v1/societies/{id}/gates/{gateId}
func UpdateGateHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		gateIDStr := chi.URLParam(r, "gateId")
		gateID, err := uuid.Parse(gateIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid gate identifier", reqID)
			return
		}

		var params society.UpdateGateParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		updatedGate, err := repo.UpdateGate(r.Context(), societyID, gateID, params, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrGateNotFound) {
				response.Error(w, http.StatusNotFound, "GATE_NOT_FOUND", "Gate not found in target society", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"gate": updatedGate,
		}, reqID)
	}
}

// ListGatekeepersHandler handles GET /api/v1/societies/{id}/gatekeepers
func ListGatekeepersHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		statusFilter := r.URL.Query().Get("status")
		gatekeepers, err := repo.ListGatekeepers(r.Context(), societyID, statusFilter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve gatekeepers", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"gatekeepers": gatekeepers,
		}, reqID)
	}
}

// AssignGatekeeperHandler handles POST /api/v1/societies/{id}/gatekeepers
func AssignGatekeeperHandler(repo *society.Repository) http.HandlerFunc {
	type RequestBody struct {
		Phone string `json:"phone"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var body RequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		if strings.TrimSpace(body.Phone) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Phone number is required", reqID)
			return
		}

		assignment, err := repo.AssignGatekeeper(r.Context(), societyID, body.Phone, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrGatekeeperAlreadyAssigned):
				response.Error(w, http.StatusConflict, "GATEKEEPER_ALREADY_ASSIGNED", "User already holds an active GATEKEEPER assignment for this society", reqID)
			case strings.Contains(err.Error(), "invalid phone format"):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", err.Error(), reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign gatekeeper", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"gatekeeper": assignment,
		}, reqID)
	}
}

// RevokeGatekeeperHandler handles DELETE /api/v1/societies/{id}/gatekeepers/{assignmentId}
func RevokeGatekeeperHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		assignmentIDStr := chi.URLParam(r, "assignmentId")
		assignmentID, err := uuid.Parse(assignmentIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid assignment identifier", reqID)
			return
		}

		err = repo.RevokeGatekeeper(r.Context(), societyID, assignmentID, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrGatekeeperNotFound) {
				response.Error(w, http.StatusNotFound, "GATEKEEPER_NOT_FOUND", "Active gatekeeper assignment not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke gatekeeper", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Gatekeeper access revoked successfully",
		}, reqID)
	}
}

// ListGateShiftsHandler handles GET /api/v1/societies/{id}/shifts
func ListGateShiftsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		shifts, err := repo.ListGateShifts(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve shifts", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"shifts": shifts,
		}, reqID)
	}
}

// CreateGateShiftHandler handles POST /api/v1/societies/{id}/shifts
func CreateGateShiftHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		actorID, _ := middleware.GetUserID(r.Context())
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var params society.CreateGateShiftParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed request body", reqID)
			return
		}

		shift, err := repo.CreateGateShift(r.Context(), societyID, params, actorID, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrShiftNameExists) {
				response.Error(w, http.StatusConflict, "SHIFT_NAME_EXISTS", "A shift with this name already exists", reqID)
				return
			}
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"shift": shift,
		}, reqID)
	}
}

// ListDutySessionsHandler handles GET /api/v1/societies/{id}/duty-sessions
func ListDutySessionsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		var gateID *uuid.UUID
		if gIDStr := r.URL.Query().Get("gate_id"); gIDStr != "" {
			if parsed, err := uuid.Parse(gIDStr); err == nil {
				gateID = &parsed
			}
		}

		statusFilter := r.URL.Query().Get("status")
		sessions, err := repo.ListDutySessions(r.Context(), societyID, gateID, statusFilter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve duty sessions", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
		}, reqID)
	}
}

// CheckInDutyHandler handles POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-in
func CheckInDutyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		gateIDStr := chi.URLParam(r, "gateId")
		gateID, err := uuid.Parse(gateIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid gate identifier", reqID)
			return
		}

		var params society.CheckInDutyParams
		_ = json.NewDecoder(r.Body).Decode(&params)

		session, err := repo.CheckInGateDuty(r.Context(), societyID, gateID, userID, params.ShiftID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrGateNotFound):
				response.Error(w, http.StatusNotFound, "GATE_NOT_FOUND", "Gate not found in society", reqID)
			case errors.Is(err, society.ErrGateInactive):
				response.Error(w, http.StatusBadRequest, "GATE_INACTIVE", "Gate is inactive or closed for operations", reqID)
			case errors.Is(err, society.ErrSessionAlreadyActive):
				response.Error(w, http.StatusConflict, "SESSION_ALREADY_ACTIVE", "Guard is already checked into an active duty session", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check into duty", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"session": session,
		}, reqID)
	}
}

// CheckOutDutyHandler handles POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-out
func CheckOutDutyHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing user context", reqID)
			return
		}
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		gateIDStr := chi.URLParam(r, "gateId")
		gateID, err := uuid.Parse(gateIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid gate identifier", reqID)
			return
		}

		var params society.CheckOutDutyParams
		_ = json.NewDecoder(r.Body).Decode(&params)

		session, err := repo.CheckOutGateDuty(r.Context(), societyID, gateID, userID, params.HandoverNotes, clientIP, userAgent)
		if err != nil {
			if errors.Is(err, society.ErrSessionNotFound) {
				response.Error(w, http.StatusNotFound, "SESSION_NOT_FOUND", "No active duty session found for this user at this gate", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to conclude duty session", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"session": session,
		}, reqID)
	}
}
