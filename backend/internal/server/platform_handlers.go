package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/middleware"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
)

type CreateAdminRequest struct {
	Phone string `json:"phone"`
}

type UpdateAdminStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// ListAdminsHandler handles GET /api/v1/platform/admins.
func ListAdminsHandler(repo *platform.Repository) http.HandlerFunc {
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

		roleFilter := strings.TrimSpace(r.URL.Query().Get("role"))
		statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

		items, total, err := repo.ListAdmins(r.Context(), limit, offset, roleFilter, statusFilter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve platform administrators", reqID)
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

// CreateAdminHandler handles POST /api/v1/platform/admins.
func CreateAdminHandler(repo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var req CreateAdminRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		if strings.TrimSpace(req.Phone) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Phone number is required", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		item, err := repo.CreateAdmin(r.Context(), actorID, req.Phone, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, auth.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Please provide a valid Indian mobile number", reqID)
			case errors.Is(err, platform.ErrRoleAlreadyExists):
				response.Error(w, http.StatusConflict, "ADMIN_ALREADY_EXISTS", "User already holds a platform role", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create platform administrator", reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"id":         item.ID,
			"user_id":    item.UserID,
			"phone":      item.Phone,
			"role":       item.Role,
			"status":     item.Status,
			"created_at": item.CreatedAt,
		}, nil)
	}
}

// UpdateAdminStatusHandler handles PATCH /api/v1/platform/admins/{id}/status.
func UpdateAdminStatusHandler(repo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		rawID := chi.URLParam(r, "id")
		targetID, err := uuid.Parse(rawID)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid administrator identifier format", reqID)
			return
		}

		var req UpdateAdminStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
		if req.Status != platform.StatusActive && req.Status != platform.StatusDisabled {
			response.Error(w, http.StatusBadRequest, "INVALID_STATUS", "Status must be ACTIVE or DISABLED", reqID)
			return
		}

		// Look up target role by role ID or user ID
		targetRole, err := repo.GetRoleByID(r.Context(), targetID)
		if err != nil {
			if errors.Is(err, platform.ErrRoleNotFound) {
				response.Error(w, http.StatusNotFound, "ADMIN_NOT_FOUND", "Platform administrator not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query administrator", reqID)
			return
		}

		// Self-disable protection: actor cannot disable self
		if actorID == targetRole.UserID && req.Status == platform.StatusDisabled {
			response.Error(w, http.StatusBadRequest, "CANNOT_DISABLE_SELF", "A Super Admin cannot disable their own account", reqID)
			return
		}

		// Initial Super Admin protection: SUPER_ADMIN account cannot be disabled
		if targetRole.Role == platform.RoleSuperAdmin && req.Status == platform.StatusDisabled {
			response.Error(w, http.StatusBadRequest, "CANNOT_DISABLE_SELF", "A Super Admin cannot disable their own account", reqID)
			return
		}

		clientIP := extractIP(r)
		userAgent := r.UserAgent()

		updated, err := repo.UpdateAdminStatus(r.Context(), actorID, targetRole.UserID, req.Status, req.Reason, clientIP, userAgent)
		if err != nil {
			switch {
			case errors.Is(err, platform.ErrCannotDisableSelf):
				response.Error(w, http.StatusBadRequest, "CANNOT_DISABLE_SELF", "A Super Admin cannot disable their own account", reqID)
			case errors.Is(err, platform.ErrRoleNotFound):
				response.Error(w, http.StatusNotFound, "ADMIN_NOT_FOUND", "Platform administrator not found", reqID)
			case errors.Is(err, platform.ErrInvalidStatus):
				response.Error(w, http.StatusBadRequest, "INVALID_STATUS", "Status must be ACTIVE or DISABLED", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update administrator status", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":         updated.ID,
			"user_id":    updated.UserID,
			"role":       updated.Role,
			"status":     updated.Status,
			"updated_at": updated.UpdatedAt,
		}, nil)
	}
}

// ListAuditLogsHandler handles GET /api/v1/platform/audit-logs.
func ListAuditLogsHandler(repo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

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
			Page:   page,
			Limit:  limit,
			Action: strings.TrimSpace(r.URL.Query().Get("action")),
			Search: strings.TrimSpace(r.URL.Query().Get("search")),
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

		if socStr := strings.TrimSpace(r.URL.Query().Get("society_id")); socStr != "" {
			if parsed, err := uuid.Parse(socStr); err == nil {
				filters.SocietyID = &parsed
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

		items, total, err := repo.ListAuditLogsEnriched(r.Context(), filters)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve audit logs", reqID)
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

// GetPlatformAuditLogDetailHandler handles GET /api/v1/platform/audit-logs/{id}.
func GetPlatformAuditLogDetailHandler(repo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		idStr := chi.URLParam(r, "id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid audit log ID", reqID)
			return
		}

		item, err := repo.GetAuditLogByID(r.Context(), id, nil)
		if err != nil {
			if errors.Is(err, platform.ErrAuditLogNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Audit log event not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve audit log detail", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"item": item,
		}, nil)
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
