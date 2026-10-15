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
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/society"
)

type CreateAnnouncementRequest struct {
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	Category         string     `json:"category"`
	Priority         string     `json:"priority"`
	AudienceType     string     `json:"audience_type"`
	TargetBuildingID *uuid.UUID `json:"target_building_id,omitempty"`
	TargetWingID     *uuid.UUID `json:"target_wing_id,omitempty"`
	TargetUnitID     *uuid.UUID `json:"target_unit_id,omitempty"`
}

type UpdateAnnouncementRequest struct {
	Title            *string    `json:"title,omitempty"`
	Body             *string    `json:"body,omitempty"`
	Category         *string    `json:"category,omitempty"`
	Priority         *string    `json:"priority,omitempty"`
	AudienceType     *string    `json:"audience_type,omitempty"`
	TargetBuildingID *uuid.UUID `json:"target_building_id,omitempty"`
	TargetWingID     *uuid.UUID `json:"target_wing_id,omitempty"`
	TargetUnitID     *uuid.UUID `json:"target_unit_id,omitempty"`
}

// CreateAnnouncementHandler handles POST /api/v1/societies/{id}/announcements.
func CreateAnnouncementHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		var req CreateAnnouncementRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", reqID)
			return
		}

		params := society.CreateAnnouncementParams{
			Title:            req.Title,
			Body:             req.Body,
			Category:         req.Category,
			Priority:         req.Priority,
			AudienceType:     req.AudienceType,
			TargetBuildingID: req.TargetBuildingID,
			TargetWingID:     req.TargetWingID,
			TargetUnitID:     req.TargetUnitID,
		}

		item, err := repo.CreateAnnouncementDraft(r.Context(), societyID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAnnouncementTitleRequired):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Announcement title is required", reqID)
			case errors.Is(err, society.ErrAnnouncementBodyRequired):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Announcement body is required", reqID)
			case errors.Is(err, society.ErrAnnouncementTargetMismatch):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Audience target specification is invalid", reqID)
			case errors.Is(err, society.ErrAnnouncementCrossSocietyTarget):
				response.Error(w, http.StatusBadRequest, "CROSS_SOCIETY_TARGET", "Target resource does not belong to this society", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"announcement": item,
		}, nil)
	}
}

// ListAdminAnnouncementsHandler handles GET /api/v1/societies/{id}/announcements/admin.
func ListAdminAnnouncementsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
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

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		status := strings.TrimSpace(r.URL.Query().Get("status"))
		if strings.EqualFold(status, "ALL") {
			status = ""
		}

		category := strings.TrimSpace(r.URL.Query().Get("category"))
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		filter := society.ListAdminAnnouncementsFilter{
			Status:   status,
			Category: category,
			Search:   search,
			Limit:    limit,
			Offset:   offset,
		}

		result, err := repo.ListAdminAnnouncements(r.Context(), societyID, filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list announcements", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcements": result.Items,
			"total":         result.Total,
			"limit":         result.Limit,
			"offset":        result.Offset,
		}, nil)
	}
}

// UpdateAnnouncementHandler handles PUT /api/v1/societies/{id}/announcements/{announcementId}.
func UpdateAnnouncementHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		announcementIDStr := chi.URLParam(r, "announcementId")
		announcementID, err := uuid.Parse(announcementIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid announcement ID", reqID)
			return
		}

		var req UpdateAnnouncementRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body", reqID)
			return
		}

		params := society.UpdateAnnouncementParams{
			Title:            req.Title,
			Body:             req.Body,
			Category:         req.Category,
			Priority:         req.Priority,
			AudienceType:     req.AudienceType,
			TargetBuildingID: req.TargetBuildingID,
			TargetWingID:     req.TargetWingID,
			TargetUnitID:     req.TargetUnitID,
		}

		item, err := repo.UpdateAnnouncement(r.Context(), societyID, announcementID, params, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAnnouncementNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Announcement not found", reqID)
			case errors.Is(err, society.ErrAnnouncementInvalidLifecycle):
				response.Error(w, http.StatusBadRequest, "INVALID_LIFECYCLE", "Archived announcements cannot be modified", reqID)
			case errors.Is(err, society.ErrAnnouncementTitleRequired):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Announcement title is required", reqID)
			case errors.Is(err, society.ErrAnnouncementBodyRequired):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Announcement body is required", reqID)
			case errors.Is(err, society.ErrAnnouncementTargetMismatch):
				response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Audience target specification is invalid", reqID)
			case errors.Is(err, society.ErrAnnouncementCrossSocietyTarget):
				response.Error(w, http.StatusBadRequest, "CROSS_SOCIETY_TARGET", "Target resource does not belong to this society", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcement": item,
		}, nil)
	}
}

// PublishAnnouncementHandler handles POST /api/v1/societies/{id}/announcements/{announcementId}/publish.
func PublishAnnouncementHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		announcementIDStr := chi.URLParam(r, "announcementId")
		announcementID, err := uuid.Parse(announcementIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid announcement ID", reqID)
			return
		}

		item, err := repo.PublishAnnouncement(r.Context(), societyID, announcementID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAnnouncementNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Announcement not found", reqID)
			case errors.Is(err, society.ErrAnnouncementInvalidLifecycle):
				response.Error(w, http.StatusBadRequest, "INVALID_LIFECYCLE", "Only draft announcements can be published", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to publish announcement", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcement": item,
		}, nil)
	}
}

// ArchiveAnnouncementHandler handles POST /api/v1/societies/{id}/announcements/{announcementId}/archive.
func ArchiveAnnouncementHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		actorID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		announcementIDStr := chi.URLParam(r, "announcementId")
		announcementID, err := uuid.Parse(announcementIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid announcement ID", reqID)
			return
		}

		item, err := repo.ArchiveAnnouncement(r.Context(), societyID, announcementID, actorID, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAnnouncementNotFound):
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Announcement not found", reqID)
			case errors.Is(err, society.ErrAnnouncementInvalidLifecycle):
				response.Error(w, http.StatusBadRequest, "INVALID_LIFECYCLE", "Only published announcements can be archived", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive announcement", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcement": item,
		}, nil)
	}
}

// ListResidentAnnouncementFeedHandler handles GET /api/v1/societies/{id}/announcements/feed.
func ListResidentAnnouncementFeedHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
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

		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		category := strings.TrimSpace(r.URL.Query().Get("category"))
		unreadOnly := r.URL.Query().Get("unread_only") == "true"
		search := strings.TrimSpace(r.URL.Query().Get("search"))

		filter := society.ListResidentFeedFilter{
			Category:   category,
			UnreadOnly: unreadOnly,
			Search:     search,
			Limit:      limit,
			Offset:     offset,
		}

		result, err := repo.ListResidentAnnouncementFeed(r.Context(), societyID, userID, filter)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load announcement feed", reqID)
			return
		}

		unreadCount, _ := repo.GetUnreadAnnouncementCount(r.Context(), societyID, userID)

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcements": result.Items,
			"total":         result.Total,
			"unread_count":  unreadCount,
			"limit":         result.Limit,
			"offset":        result.Offset,
		}, nil)
	}
}

// GetAnnouncementDetailsHandler handles GET /api/v1/societies/{id}/announcements/{announcementId}.
func GetAnnouncementDetailsHandler(repo *society.Repository, platformRepo *platform.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		announcementIDStr := chi.URLParam(r, "announcementId")
		announcementID, err := uuid.Parse(announcementIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid announcement ID", reqID)
			return
		}

		// Determine if user has privileged administrative access
		isPrivileged := false
		if platformRepo != nil {
			pRole, err := platformRepo.GetUserPlatformRole(r.Context(), userID)
			if err == nil && pRole != nil && pRole.Status == platform.StatusActive && pRole.Role == platform.RoleSuperAdmin {
				isPrivileged = true
			}
		}
		if !isPrivileged {
			assignments, err := repo.ListUserSocietyAssignments(r.Context(), userID)
			if err == nil {
				for _, assign := range assignments {
					if assign.SocietyID == societyID && assign.Status == society.AssignmentStatusActive && assign.Role == society.RoleSocietyAdmin {
						isPrivileged = true
						break
					}
				}
			}
		}

		item, err := repo.GetAnnouncementByID(r.Context(), societyID, announcementID, userID, isPrivileged)
		if err != nil {
			if errors.Is(err, society.ErrAnnouncementNotFound) {
				response.Error(w, http.StatusNotFound, "NOT_FOUND", "Announcement not found or inaccessible", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve announcement", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcement": item,
		}, nil)
	}
}

// MarkAnnouncementReadHandler handles POST /api/v1/societies/{id}/announcements/{announcementId}/read.
func MarkAnnouncementReadHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		announcementIDStr := chi.URLParam(r, "announcementId")
		announcementID, err := uuid.Parse(announcementIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid announcement ID", reqID)
			return
		}

		err = repo.MarkAnnouncementRead(r.Context(), societyID, announcementID, userID)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrAnnouncementAccessDenied):
				response.Error(w, http.StatusForbidden, "ACCESS_DENIED", "Announcement is not accessible to your account", reqID)
			default:
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to mark announcement as read", reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"announcement_id": announcementID,
			"read":            true,
		}, nil)
	}
}

// GetUnreadAnnouncementCountHandler handles GET /api/v1/societies/{id}/announcements/unread-count.
func GetUnreadAnnouncementCountHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())

		userID, ok := middleware.GetUserID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication context", reqID)
			return
		}

		societyIDStr := chi.URLParam(r, "id")
		societyID, err := uuid.Parse(societyIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid society ID", reqID)
			return
		}

		count, err := repo.GetUnreadAnnouncementCount(r.Context(), societyID, userID)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get unread announcement count", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"unread_count": count,
		}, nil)
	}
}
