package server

import (
	"net/http"

	"chs/backend/internal/middleware"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

// GetPlatformDashboardHandler handles GET /api/v1/platform/dashboard/summary
func GetPlatformDashboardHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		if societyRepo == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
			return
		}

		summary, err := societyRepo.GetPlatformDashboardSummary(r.Context())
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve platform dashboard summary", reqID)
			return
		}

		response.Success(w, http.StatusOK, summary, reqID)
	}
}

// GetSocietyDashboardHandler handles GET /api/v1/societies/{id}/dashboard/summary
func GetSocietyDashboardHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		if societyRepo == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
			return
		}

		summary, err := societyRepo.GetSocietyDashboardSummary(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve society dashboard summary", reqID)
			return
		}

		response.Success(w, http.StatusOK, summary, reqID)
	}
}

// GetGateDashboardHandler handles GET /api/v1/societies/{id}/gate/dashboard
func GetGateDashboardHandler(societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		societyID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Missing society context", reqID)
			return
		}

		if societyRepo == nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Society repository unavailable", reqID)
			return
		}

		summary, err := societyRepo.GetGateDashboardSummary(r.Context(), societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve gate dashboard summary", reqID)
			return
		}

		response.Success(w, http.StatusOK, summary, reqID)
	}
}
