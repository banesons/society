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

// ListMembersHandler handles GET /api/v1/societies/{id}/members.
func ListMembersHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
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

		query := r.URL.Query().Get("q")
		status := r.URL.Query().Get("status")

		members, total, err := repo.ListMembers(r.Context(), socID, query, status, limit, offset)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve members", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"members": members,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
		}, reqID)
	}
}

// RegisterMemberRequest holds JSON payload for member registration.
type RegisterMemberRequest struct {
	FirstName             string  `json:"first_name"`
	LastName              *string `json:"last_name,omitempty"`
	Phone                 string  `json:"phone"`
	Email                 *string `json:"email,omitempty"`
	EmergencyContactName  *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string `json:"emergency_contact_phone,omitempty"`
}

// RegisterMemberHandler handles POST /api/v1/societies/{id}/members.
func RegisterMemberHandler(repo *society.Repository) http.HandlerFunc {
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

		var req RegisterMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request payload", reqID)
			return
		}

		member, err := repo.RegisterMember(r.Context(), socID, actorID, society.RegisterMemberParams{
			FirstName:             req.FirstName,
			LastName:              req.LastName,
			Phone:                 req.Phone,
			Email:                 req.Email,
			EmergencyContactName:  req.EmergencyContactName,
			EmergencyContactPhone: req.EmergencyContactPhone,
			ClientIP:              r.RemoteAddr,
			UserAgent:             r.UserAgent(),
		})

		if err != nil {
			switch {
			case errors.Is(err, society.ErrInvalidPhone):
				response.Error(w, http.StatusBadRequest, "INVALID_PHONE", "Phone must be a valid 10-digit mobile number", reqID)
			case errors.Is(err, society.ErrMemberAlreadyExists):
				response.Error(w, http.StatusConflict, "MEMBER_ALREADY_EXISTS", "An active member with this phone number already exists in this society", reqID)
			case errors.Is(err, society.ErrUserAlreadyLinked):
				response.Error(w, http.StatusConflict, "USER_ALREADY_LINKED", "This user account is already linked to another member in this society", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"member": member,
		}, reqID)
	}
}

// GetMemberHandler handles GET /api/v1/societies/{id}/members/{memberId}.
func GetMemberHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid member identifier", reqID)
			return
		}

		member, relationships, err := repo.GetMemberByID(r.Context(), socID, memberID)
		if err != nil {
			if errors.Is(err, society.ErrMemberNotFound) {
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve member", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"member":        member,
			"relationships": relationships,
		}, reqID)
	}
}

// UpdateMemberRequest holds JSON payload for updating member details.
type UpdateMemberRequest struct {
	FirstName             string  `json:"first_name"`
	LastName              *string `json:"last_name,omitempty"`
	Email                 *string `json:"email,omitempty"`
	Status                *string `json:"status,omitempty"`
	EmergencyContactName  *string `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone *string `json:"emergency_contact_phone,omitempty"`
}

// UpdateMemberHandler handles PATCH /api/v1/societies/{id}/members/{memberId}.
func UpdateMemberHandler(repo *society.Repository) http.HandlerFunc {
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

		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid member identifier", reqID)
			return
		}

		var req UpdateMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request payload", reqID)
			return
		}

		member, err := repo.UpdateMember(
			r.Context(), socID, memberID, actorID,
			req.FirstName, req.LastName, req.Email, req.Status,
			req.EmergencyContactName, req.EmergencyContactPhone,
			r.RemoteAddr, r.UserAgent(),
		)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrMemberNotFound):
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
			case errors.Is(err, society.ErrDepartureWorkflowRequired):
				response.Error(w, http.StatusBadRequest, "DEPARTURE_WORKFLOW_REQUIRED", "Member departure must use the two-phase departure workflow", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"member": member,
		}, reqID)
	}
}

// PreviewMemberDepartureHandler handles POST /api/v1/societies/{id}/members/{memberId}/departure/preview.
func PreviewMemberDepartureHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid member identifier", reqID)
			return
		}

		preview, err := repo.PreviewMemberDeparture(r.Context(), socID, memberID)
		if err != nil {
			if errors.Is(err, society.ErrMemberNotFound) {
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate departure preview", reqID)
			return
		}

		response.Success(w, http.StatusOK, preview, reqID)
	}
}

// ConfirmMemberDepartureRequest holds payload for confirming departure.
type ConfirmMemberDepartureRequest struct {
	ExpectedVersionHash string  `json:"expected_version_hash"`
	EndDate             string  `json:"end_date"`
	ReasonCode          string  `json:"reason_code"`
	ReasonDetails       *string `json:"reason_details,omitempty"`
}

// ConfirmMemberDepartureHandler handles POST /api/v1/societies/{id}/members/{memberId}/departure/confirm.
func ConfirmMemberDepartureHandler(repo *society.Repository) http.HandlerFunc {
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

		memberIDStr := chi.URLParam(r, "memberId")
		memberID, err := uuid.Parse(memberIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid member identifier", reqID)
			return
		}

		var req ConfirmMemberDepartureRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request payload", reqID)
			return
		}

		if req.ExpectedVersionHash == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "expected_version_hash is required", reqID)
			return
		}
		if req.EndDate == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "end_date is required", reqID)
			return
		}
		if strings.TrimSpace(req.ReasonCode) == "" {
			response.Error(w, http.StatusBadRequest, "INVALID_DEPARTURE_REASON", "reason_code is required", reqID)
			return
		}

		err = repo.ConfirmMemberDeparture(
			r.Context(), socID, memberID, actorID,
			req.ExpectedVersionHash, req.EndDate, req.ReasonCode, req.ReasonDetails,
			r.RemoteAddr, r.UserAgent(),
		)
		if err != nil {
			switch {
			case errors.Is(err, society.ErrMemberNotFound):
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
			case errors.Is(err, society.ErrPreviewStale):
				response.Error(w, http.StatusConflict, "PREVIEW_STALE", "Active relationships changed since preview was generated. Please refresh preview.", reqID)
			case errors.Is(err, society.ErrInvalidDates):
				response.Error(w, http.StatusBadRequest, "INVALID_DATES", "End date must be after or equal to relationship start date", reqID)
			case errors.Is(err, society.ErrInvalidDepartureReason):
				response.Error(w, http.StatusBadRequest, "INVALID_DEPARTURE_REASON", "Invalid or unsupported departure reason code", reqID)
			case errors.Is(err, society.ErrReasonDetailsRequired):
				response.Error(w, http.StatusBadRequest, "REASON_DETAILS_REQUIRED", "reason_details is required when reason_code is OTHER", reqID)
			case errors.Is(err, society.ErrReasonDetailsTooLong):
				response.Error(w, http.StatusBadRequest, "REASON_DETAILS_TOO_LONG", "reason_details must not exceed 500 characters", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Member departed successfully; occupancies ended and ownerships preserved",
		}, reqID)
	}
}

// ListUnitRelationshipsHandler handles GET /api/v1/societies/{id}/units/{unitId}/relationships.
func ListUnitRelationshipsHandler(repo *society.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		socID, ok := middleware.GetSocietyID(r.Context())
		if !ok {
			response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing society context", reqID)
			return
		}

		unitIDStr := chi.URLParam(r, "unitId")
		unitID, err := uuid.Parse(unitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit identifier", reqID)
			return
		}

		status := r.URL.Query().Get("status")

		relationships, totalShare, isVacant, err := repo.ListUnitRelationships(r.Context(), socID, unitID, status)
		if err != nil {
			if errors.Is(err, society.ErrUnitNotFound) {
				response.Error(w, http.StatusNotFound, "UNIT_NOT_FOUND", "Unit not found in this society", reqID)
				return
			}
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve unit relationships", reqID)
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"unit_id":                        unitID,
			"relationships":                  relationships,
			"total_ownership_share_recorded": totalShare,
			"is_vacant":                      isVacant,
		}, reqID)
	}
}

// CreateUnitRelationshipRequest holds payload for associating a member with a unit.
type CreateUnitRelationshipRequest struct {
	MemberID         uuid.UUID `json:"member_id"`
	RelationshipType string    `json:"relationship_type"`
	IsOccupant       bool      `json:"is_occupant"`
	OwnershipShare   *float64  `json:"ownership_share,omitempty"`
	StartDate        string    `json:"start_date"`
	Notes            *string   `json:"notes,omitempty"`
}

// CreateUnitRelationshipHandler handles POST /api/v1/societies/{id}/units/{unitId}/relationships.
func CreateUnitRelationshipHandler(repo *society.Repository) http.HandlerFunc {
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

		unitIDStr := chi.URLParam(r, "unitId")
		unitID, err := uuid.Parse(unitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit identifier", reqID)
			return
		}

		var req CreateUnitRelationshipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request payload", reqID)
			return
		}

		rel, err := repo.CreateUnitRelationship(r.Context(), socID, unitID, actorID, society.CreateUnitRelationshipParams{
			MemberID:         req.MemberID,
			RelationshipType: req.RelationshipType,
			IsOccupant:       req.IsOccupant,
			OwnershipShare:   req.OwnershipShare,
			StartDate:        req.StartDate,
			Notes:            req.Notes,
			ClientIP:         r.RemoteAddr,
			UserAgent:        r.UserAgent(),
		})

		if err != nil {
			switch {
			case errors.Is(err, society.ErrUnitNotFound):
				response.Error(w, http.StatusNotFound, "UNIT_NOT_FOUND", "Unit not found in this society", reqID)
			case errors.Is(err, society.ErrMemberNotFound):
				response.Error(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found in this society", reqID)
			case errors.Is(err, society.ErrRelationshipAlreadyExists):
				response.Error(w, http.StatusConflict, "RELATIONSHIP_ALREADY_EXISTS", "Active identical relationship already exists for this member on this unit", reqID)
			case errors.Is(err, society.ErrInvalidRelationshipType):
				response.Error(w, http.StatusBadRequest, "INVALID_RELATIONSHIP_TYPE", "Invalid relationship type", reqID)
			case errors.Is(err, society.ErrOwnershipShareNotAllowed):
				response.Error(w, http.StatusBadRequest, "INVALID_OWNERSHIP_SHARE", "Ownership share is only permitted for owner relationships", reqID)
			case errors.Is(err, society.ErrOccupancyRequired):
				response.Error(w, http.StatusBadRequest, "OCCUPANCY_REQUIRED", "Authorized occupants must have is_occupant set to true", reqID)
			default:
				if strings.Contains(err.Error(), "invalid ownership share") {
					response.Error(w, http.StatusBadRequest, "INVALID_OWNERSHIP_SHARE", err.Error(), reqID)
					return
				}
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusCreated, map[string]interface{}{
			"relationship": rel,
		}, reqID)
	}
}

// EndUnitRelationshipRequest holds payload for ending a relationship.
type EndUnitRelationshipRequest struct {
	EndDate string  `json:"end_date"`
	Notes   *string `json:"notes,omitempty"`
}

// EndUnitRelationshipHandler handles POST /api/v1/societies/{id}/units/{unitId}/relationships/{relationshipId}/end.
func EndUnitRelationshipHandler(repo *society.Repository) http.HandlerFunc {
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

		unitIDStr := chi.URLParam(r, "unitId")
		unitID, err := uuid.Parse(unitIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid unit identifier", reqID)
			return
		}

		relIDStr := chi.URLParam(r, "relationshipId")
		relID, err := uuid.Parse(relIDStr)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid relationship identifier", reqID)
			return
		}

		var req EndUnitRelationshipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Malformed request payload", reqID)
			return
		}

		if req.EndDate == "" {
			response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "end_date is required", reqID)
			return
		}

		err = repo.EndUnitRelationship(r.Context(), socID, unitID, relID, actorID, req.EndDate, req.Notes, r.RemoteAddr, r.UserAgent())
		if err != nil {
			switch {
			case errors.Is(err, society.ErrRelationshipNotFound):
				response.Error(w, http.StatusNotFound, "RELATIONSHIP_NOT_FOUND", "Relationship not found on this unit", reqID)
			case errors.Is(err, society.ErrInvalidDates):
				response.Error(w, http.StatusBadRequest, "INVALID_DATES", "End date must be after or equal to relationship start date", reqID)
			default:
				response.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), reqID)
			}
			return
		}

		response.Success(w, http.StatusOK, map[string]interface{}{
			"message": "Unit relationship ended successfully",
		}, reqID)
	}
}
