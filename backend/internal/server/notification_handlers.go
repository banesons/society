package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"chs/backend/internal/middleware"
	"chs/backend/internal/notification"
	"chs/backend/internal/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListNotificationsHandler handles GET /api/v1/notifications.
func ListNotificationsHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
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

		unreadOnly := r.URL.Query().Get("unread_only") == "true" || r.URL.Query().Get("unread") == "true"

		filters := notification.NotificationListFilters{
			Page:       page,
			Limit:      limit,
			UnreadOnly: unreadOnly,
		}

		if socStr := strings.TrimSpace(r.URL.Query().Get("society_id")); socStr != "" {
			if parsed, err := uuid.Parse(socStr); err == nil {
				filters.SocietyID = &parsed
			}
		}

		items, total, err := repo.ListNotifications(r.Context(), userID, filters)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list notifications", reqID)
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

// GetUnreadNotificationCountHandler handles GET /api/v1/notifications/unread-count.
func GetUnreadNotificationCountHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var societyID *uuid.UUID
		if socStr := strings.TrimSpace(r.URL.Query().Get("society_id")); socStr != "" {
			if parsed, err := uuid.Parse(socStr); err == nil {
				societyID = &parsed
			}
		}

		count, err := repo.GetUnreadCount(r.Context(), userID, societyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get unread notification count", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"unread_count": count,
		}, nil)
	}
}

// MarkNotificationReadHandler handles PATCH /api/v1/notifications/{id}/read.
func MarkNotificationReadHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		idStr := chi.URLParam(r, "id")
		notificationID, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid notification identifier", reqID)
			return
		}

		readAt, err := repo.MarkAsRead(r.Context(), userID, notificationID)
		if err != nil {
			if errors.Is(err, notification.ErrNotificationNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Notification not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark notification read", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"id":      notificationID,
			"read_at": readAt,
		}, nil)
	}
}

// MarkAllNotificationsReadHandler handles POST /api/v1/notifications/mark-all-read.
func MarkAllNotificationsReadHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var payload struct {
			SocietyID *uuid.UUID `json:"society_id"`
		}
		if r.Body != nil && r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}

		count, err := repo.MarkAllAsRead(r.Context(), userID, payload.SocietyID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark notifications read", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"marked_count": count,
		}, nil)
	}
}

// RegisterDeviceHandler handles POST /api/v1/notifications/devices.
func RegisterDeviceHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		var params notification.RegisterDeviceParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request payload", reqID)
			return
		}

		if strings.TrimSpace(params.DeviceID) == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "device_id is required", reqID)
			return
		}
		if strings.TrimSpace(params.FCMToken) == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "fcm_token is required", reqID)
			return
		}

		userAgent := r.UserAgent()
		clientIP := extractIP(r)

		dev, err := repo.RegisterDevice(r.Context(), userID, params, userAgent, clientIP)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to register device", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"device": dev,
		}, nil)
	}
}

// DeactivateDeviceHandler handles DELETE /api/v1/notifications/devices/{device_id}.
func DeactivateDeviceHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		deviceID := chi.URLParam(r, "device_id")
		if strings.TrimSpace(deviceID) == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "device_id is required", reqID)
			return
		}

		err := repo.DeactivateDevice(r.Context(), userID, deviceID)
		if err != nil {
			if errors.Is(err, notification.ErrDeviceNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Active device registration not found", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to deactivate device", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"device_id": deviceID,
			"status":    notification.DeviceStatusInactive,
		}, nil)
	}
}

// ListUserDevicesHandler handles GET /api/v1/notifications/devices.
func ListUserDevicesHandler(repo *notification.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		devices, err := repo.ListUserDevices(r.Context(), userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list user devices", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"devices": devices,
		}, nil)
	}
}
