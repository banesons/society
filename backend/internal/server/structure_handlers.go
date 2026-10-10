package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// ListBuildingsHandler handles GET /api/v1/societies/{id}/buildings.
func ListBuildingsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		buildings, err := repo.ListBuildings(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve buildings", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"buildings": buildings,
			"total":     len(buildings),
		}, reqID)
	}
}

// CreateBuildingHandler handles POST /api/v1/societies/{id}/buildings.
func CreateBuildingHandler(repo *society.Repository) http.HandlerFunc {
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

		var req society.CreateBuildingParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.BuildingCode = strings.ToUpper(strings.TrimSpace(req.BuildingCode))
		if req.Name == "" || req.BuildingCode == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Building name and code are required", reqID)
			return
		}
		if req.TotalFloors < 0 || req.TotalFloors > 100 {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Total floors must be between 0 and 100", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		building, err := repo.CreateBuilding(r.Context(), socID, req, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBuildingCodeExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A building with this code already exists in this society", reqID)
			case errors.Is(err, society.ErrBuildingNameExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A building with this name already exists in this society", reqID)
			case errors.Is(err, society.ErrInvalidFloorCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid floor count", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create building", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, building, reqID)
	}
}

// CreateWingHandler handles POST /api/v1/societies/{id}/buildings/{building_id}/wings.
func CreateWingHandler(repo *society.Repository) http.HandlerFunc {
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

		buildingIDStr := chi.URLParam(r, "building_id")
		buildingID, err := uuid.Parse(buildingIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid building identifier", reqID)
			return
		}

		var req society.CreateWingParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.WingCode = strings.ToUpper(strings.TrimSpace(req.WingCode))
		if req.Name == "" || req.WingCode == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Wing name and code are required", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		wing, err := repo.CreateWing(r.Context(), socID, buildingID, req, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBuildingNotFound):
				response.Error(w, http.StatusNotFound, "BUILDING_NOT_FOUND", "Referenced building does not exist in this society", reqID)
			case errors.Is(err, society.ErrWingCodeExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A wing with this code already exists in this building", reqID)
			case errors.Is(err, society.ErrWingNameExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A wing with this name already exists in this building", reqID)
			case errors.Is(err, society.ErrInvalidFloorCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid floor count", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create wing", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, wing, reqID)
	}
}

// PreviewUnitsHandler handles POST /api/v1/societies/{id}/units/preview.
func PreviewUnitsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		var req society.GenerateUnitsParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		preview, err := society.GenerateUnitIdentifiers(req)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrInvalidFloorCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Total floors must be between 1 and 100", reqID)
			case errors.Is(err, society.ErrInvalidUnitCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Units per floor must be between 1 and 50", reqID)
			case errors.Is(err, society.ErrBulkLimitExceeded):
				response.Error(w, http.StatusBadRequest, "LIMIT_EXCEEDED", "Cannot generate more than 1000 units in a single request", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, preview, reqID)
	}
}

// GenerateUnitsHandler handles POST /api/v1/societies/{id}/units/generate.
func GenerateUnitsHandler(repo *society.Repository) http.HandlerFunc {
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

		var req society.GenerateUnitsParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		count, err := repo.BulkGenerateUnits(r.Context(), socID, req, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrBuildingNotFound):
				response.Error(w, http.StatusNotFound, "BUILDING_NOT_FOUND", "Referenced building does not exist in this society", reqID)
			case errors.Is(err, society.ErrWingNotFound):
				response.Error(w, http.StatusNotFound, "WING_NOT_FOUND", "Referenced wing does not exist in this society", reqID)
			case errors.Is(err, society.ErrParentBuildingMismatch):
				response.Error(w, http.StatusBadRequest, "MISMATCH", "Referenced wing does not belong to the specified building", reqID)
			case errors.Is(err, society.ErrUnitIdentifierExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "One or more generated unit identifiers already exist in this society", reqID)
			case errors.Is(err, society.ErrInvalidFloorCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Total floors must be between 1 and 100", reqID)
			case errors.Is(err, society.ErrInvalidUnitCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Units per floor must be between 1 and 50", reqID)
			case errors.Is(err, society.ErrBulkLimitExceeded):
				response.Error(w, http.StatusBadRequest, "LIMIT_EXCEEDED", "Cannot generate more than 1000 units in a single request", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate units: "+err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"generated_count": count,
			"message":         "Units generated successfully",
		}, reqID)
	}
}

// ListUnitsHandler handles GET /api/v1/societies/{id}/units.
func ListUnitsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		filter := society.ListUnitsFilter{
			SocietyID: socID,
			Limit:     50,
			Offset:    0,
		}

		if bID := r.URL.Query().Get("building_id"); bID != "" {
			if id, err := uuid.Parse(bID); err == nil {
				filter.BuildingID = &id
			}
		}
		if wID := r.URL.Query().Get("wing_id"); wID != "" {
			if id, err := uuid.Parse(wID); err == nil {
				filter.WingID = &id
			}
		}
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				filter.Limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				filter.Offset = parsed
			}
		}

		units, total, err := repo.ListUnits(r.Context(), filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve units", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"units":  units,
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
		}, reqID)
	}
}

// ListGatesHandler handles GET /api/v1/societies/{id}/gates.
func ListGatesHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		gates, err := repo.ListGates(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve gates", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"gates": gates,
			"total": len(gates),
		}, reqID)
	}
}

// CreateGateHandler handles POST /api/v1/societies/{id}/gates.
func CreateGateHandler(repo *society.Repository) http.HandlerFunc {
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

		var req society.CreateGateParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.GateCode = strings.ToUpper(strings.TrimSpace(req.GateCode))
		if req.Name == "" || req.GateCode == "" {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Gate name and code are required", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		gate, err := repo.CreateGate(r.Context(), socID, req, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrGateCodeExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A gate with this code already exists in this society", reqID)
			case errors.Is(err, society.ErrGateNameExists):
				response.Error(w, http.StatusConflict, "CONFLICT", "A gate with this name already exists in this society", reqID)
			case errors.Is(err, society.ErrInvalidStaffCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Required staff per shift cannot be negative", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create gate", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, gate, reqID)
	}
}

// GetStaffingHandler handles GET /api/v1/societies/{id}/staffing.
func GetStaffingHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		sr, err := repo.GetStaffingRequirement(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve staffing requirements", reqID)
			return
		}

		response.Success(w, http.StatusOK, sr, reqID)
	}
}

// SetStaffingHandler handles PUT /api/v1/societies/{id}/staffing.
func SetStaffingHandler(repo *society.Repository) http.HandlerFunc {
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

		var req society.UpdateStaffingParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request body", reqID)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		sr, err := repo.SetStaffingRequirement(r.Context(), socID, req, actorID, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrInvalidShiftsPerDay):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Shifts per day must be between 1 and 4", reqID)
			case errors.Is(err, society.ErrInvalidStaffCount):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Total required headcount cannot be negative", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update staffing requirement", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, sr, reqID)
	}
}

// GetFormationSummaryHandler handles GET /api/v1/societies/{id}/formation-summary.
func GetFormationSummaryHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		summary, err := repo.GetFormationSummary(r.Context(), socID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate formation summary", reqID)
			return
		}

		response.Success(w, http.StatusOK, summary, reqID)
	}
}
