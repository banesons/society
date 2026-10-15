package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chs/backend/internal/middleware"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListSocietyAuditLogsHandler handles GET /api/v1/societies/{id}/audit-logs.
func ListSocietyAuditLogsHandler(platformRepo *platform.Repository, societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		socIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(socIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		if limit > 100 {
			limit = 100
		}

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		} else if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				page = (parsed / limit) + 1
			}
		}

		filters := platform.AuditQueryFilters{
			Page:      page,
			Limit:     limit,
			SocietyID: &societyID, // STRICT TENANT ISOLATION
			Action:    strings.TrimSpace(r.URL.Query().Get("action")),
			Search:    strings.TrimSpace(r.URL.Query().Get("search")),
		}

		if actorStr := strings.TrimSpace(r.URL.Query().Get("actor_id")); actorStr != "" {
			if parsed, err := uuid.Parse(actorStr); err == nil {
				filters.ActorID = &parsed
			}
		}

		if targetStr := strings.TrimSpace(r.URL.Query().Get("target_user_id")); targetStr != "" {
			if parsed, err := uuid.Parse(targetStr); err == nil {
				filters.TargetUserID = &parsed
			}
		}

		if startStr := strings.TrimSpace(r.URL.Query().Get("start_date")); startStr != "" {
			if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
				filters.StartDate = &parsed
			} else if parsedDate, err := time.Parse("2006-01-02", startStr); err == nil {
				filters.StartDate = &parsedDate
			}
		}

		if endStr := strings.TrimSpace(r.URL.Query().Get("end_date")); endStr != "" {
			if parsed, err := time.Parse(time.RFC3339, endStr); err == nil {
				filters.EndDate = &parsed
			} else if parsedDate, err := time.Parse("2006-01-02", endStr); err == nil {
				endOfDay := parsedDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				filters.EndDate = &endOfDay
			}
		}

		items, total, err := platformRepo.ListAuditLogsEnriched(r.Context(), filters)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve society audit logs", reqID)
			return
		}

		totalPages := 1
		if total > 0 {
			totalPages = (total + limit - 1) / limit
		}

		offset := (page - 1) * limit

		response.Success(w, http.StatusOK, map[string]interface{}{
			"items":  items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total_items": total,
				"total_pages": totalPages,
			},
		}, nil)
	}
}

// GetSocietyAuditLogDetailHandler handles GET /api/v1/societies/{id}/audit-logs/{event_id}.
func GetSocietyAuditLogDetailHandler(platformRepo *platform.Repository, societyRepo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		socIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(socIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		eventIDStr := chi.URLParam(r, "event_id")
		eventID, err := uuid.Parse(eventIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid audit log ID", reqID)
			return
		}

		item, err := platformRepo.GetAuditLogByID(r.Context(), eventID, &societyID)
		if err != nil {
			if errors.Is(err, platform.ErrAuditLogNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Society audit log event not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve society audit log detail", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"item": item,
		}, nil)
	}
}
