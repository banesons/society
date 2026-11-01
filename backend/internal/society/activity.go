package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserActivityCategory defines high-level event groupings for personal timelines.
type UserActivityCategory string

const (
	ActivityCategoryAll      UserActivityCategory = "ALL"
	ActivityCategoryAccount  UserActivityCategory = "ACCOUNT"
	ActivityCategoryVisitor  UserActivityCategory = "VISITOR"
	ActivityCategorySociety  UserActivityCategory = "SOCIETY"
	ActivityCategorySecurity UserActivityCategory = "SECURITY"
)

// UserActivityEvent represents a single chronological event on a user's personal timeline.
type UserActivityEvent struct {
	ID             uuid.UUID              `json:"id"`
	Category       string                 `json:"category"`
	Action         string                 `json:"action"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary"`
	DisplayTitle   string                 `json:"display_title"`
	DisplaySummary string                 `json:"display_summary"`
	Timestamp      time.Time              `json:"timestamp"`
	SocietyID      *uuid.UUID             `json:"society_id,omitempty"`
	SocietyName    *string                `json:"society_name,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UserActivityTimelineResponse represents a paginated activity timeline.
type UserActivityTimelineResponse struct {
	Events  []UserActivityEvent `json:"events"`
	Total   int                 `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
	HasMore bool                `json:"has_more"`
}

// GetUserActivityTimeline retrieves the authentic, authorized chronological activity history for a user.
func (r *Repository) GetUserActivityTimeline(
	ctx context.Context,
	userID uuid.UUID,
	category string,
	limit, offset int,
) (*UserActivityTimelineResponse, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var userPhone string
	_ = r.pool.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, userID).Scan(&userPhone)

	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var actionFilter string
	switch strings.ToUpper(strings.TrimSpace(category)) {
	case "ACCOUNT":
		actionFilter = "AND l.action IN ('PROFILE_UPDATED', 'PROFILE_PHOTO_UPLOADED', 'USER_REGISTERED', 'VISITOR_IDENTITY_LINKED')"
	case "VISITOR":
		actionFilter = "AND l.action IN ('VISIT_REGISTERED', 'WALK_IN_REGISTERED', 'VISIT_APPROVED', 'VISIT_REJECTED', 'VISIT_CANCELLED', 'VISITOR_PASS_ISSUED', 'VISITOR_PASS_REVOKED', 'VISITOR_CHECKED_IN', 'VISITOR_CHECKED_OUT', 'VISITOR_IDENTITY_LINKED')"
	case "SOCIETY":
		actionFilter = "AND l.action IN ('MEMBER_REGISTERED', 'UNIT_RELATIONSHIP_CREATED', 'MEMBER_DEPARTURE_CONFIRMED', 'UNIT_VISITOR_POLICY_UPDATED', 'SOCIETY_ADMIN_ASSIGNED', 'SOCIETY_ADMIN_REVOKED')"
	case "SECURITY":
		actionFilter = "AND l.action IN ('SESSION_ROTATED', 'USER_LOGOUT_ALL', 'SESSION_REVOKED')"
	default: // ALL
		actionFilter = `AND l.action IN (
			'PROFILE_UPDATED', 'PROFILE_PHOTO_UPLOADED', 'USER_REGISTERED',
			'VISIT_REGISTERED', 'WALK_IN_REGISTERED', 'VISIT_APPROVED', 'VISIT_REJECTED', 'VISIT_CANCELLED',
			'VISITOR_PASS_ISSUED', 'VISITOR_PASS_REVOKED', 'VISITOR_CHECKED_IN', 'VISITOR_CHECKED_OUT', 'VISITOR_IDENTITY_LINKED',
			'MEMBER_REGISTERED', 'UNIT_RELATIONSHIP_CREATED', 'MEMBER_DEPARTURE_CONFIRMED', 'UNIT_VISITOR_POLICY_UPDATED',
			'SOCIETY_ADMIN_ASSIGNED', 'SOCIETY_ADMIN_REVOKED',
			'SESSION_ROTATED', 'USER_LOGOUT_ALL', 'SESSION_REVOKED'
		)`
	}

	whereClause := fmt.Sprintf(`
		WHERE (
			l.actor_id = $1
			OR l.target_user_id = $1
			OR (l.details->>'visitor_phone' IS NOT NULL AND l.details->>'visitor_phone' = $2)
			OR (
				l.details->>'visit_id' IS NOT NULL 
				AND (l.details->>'visit_id')::text ~ '^[0-9a-fA-F-]{36}$'
				AND (l.details->>'visit_id')::uuid IN (
					SELECT v.id FROM society_visits v
					LEFT JOIN society_visitor_identities vi ON v.visitor_identity_id = vi.id
					WHERE vi.user_id = $1 
					   OR v.created_by = $1
					   OR v.unit_id IN (
					       SELECT ur.unit_id FROM unit_relationships ur
					       JOIN society_members sm ON ur.member_id = sm.id
					       WHERE sm.user_id = $1 AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
					   )
				)
			)
		)
		%s
	`, actionFilter)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM platform_audit_logs l
		%s
	`, whereClause)

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, userID, userPhone).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count user activity events: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT 
			l.id,
			l.action,
			l.details,
			l.society_id,
			s.name AS society_name,
			l.created_at
		FROM platform_audit_logs l
		LEFT JOIN societies s ON l.society_id = s.id
		%s
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT $3 OFFSET $4
	`, whereClause)

	rows, err := r.pool.Query(ctx, dataQuery, userID, userPhone, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query user activity events: %w", err)
	}
	defer rows.Close()

	var events []UserActivityEvent
	for rows.Next() {
		var id uuid.UUID
		var action string
		var rawDetails []byte
		var socID *uuid.UUID
		var socName *string
		var createdAt time.Time

		if err := rows.Scan(&id, &action, &rawDetails, &socID, &socName, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan activity event: %w", err)
		}

		var details map[string]interface{}
		if len(rawDetails) > 0 {
			_ = json.Unmarshal(rawDetails, &details)
		}
		if details == nil {
			details = make(map[string]interface{})
		}

		eventCategory, title, summary := resolveTimelineDisplay(action, details, socName)

		// Sanitize details: strip raw tokens, hashes, passwords
		delete(details, "token_hash")
		delete(details, "bearer_token")
		delete(details, "otp")

		events = append(events, UserActivityEvent{
			ID:             id,
			Category:       eventCategory,
			Action:         action,
			Title:          title,
			Summary:        summary,
			DisplayTitle:   title,
			DisplaySummary: summary,
			Timestamp:      createdAt,
			SocietyID:      socID,
			SocietyName:    socName,
			Metadata:       details,
		})
	}

	if events == nil {
		events = []UserActivityEvent{}
	}

	return &UserActivityTimelineResponse{
		Events:  events,
		Total:   totalCount,
		Limit:   limit,
		Offset:  offset,
		HasMore: (offset + len(events)) < totalCount,
	}, nil
}

func resolveTimelineDisplay(action string, details map[string]interface{}, societyName *string) (category, title, summary string) {
	socPrefix := ""
	if societyName != nil && *societyName != "" {
		socPrefix = *societyName + " · "
	}

	switch action {
	case "PROFILE_UPDATED":
		return "ACCOUNT", "Personal Profile Updated", "Account personal information was updated."
	case "PROFILE_PHOTO_UPLOADED":
		return "ACCOUNT", "Profile Photo Updated", "New personal avatar photo uploaded."
	case "USER_REGISTERED":
		return "ACCOUNT", "Account Registered", "Verified cryptographic phone identity established on CHS platform."

	case "USER_LOGIN":
		return "SECURITY", "Device Sign In", "Authenticated session established via verified OTP."
	case "USER_LOGOUT":
		return "SECURITY", "Device Sign Out", "Session terminated on this device."
	case "USER_LOGOUT_ALL":
		return "SECURITY", "Global Session Revocation", "All active sessions revoked across all devices."
	case "SESSION_ROTATED":
		return "SECURITY", "Session Rotated", "Oldest active device retired under 2-device limit."
	case "SESSION_REVOKED":
		return "SECURITY", "Device Access Revoked", "Active device session was terminated."

	case "VISITOR_IDENTITY_LINKED":
		return "VISITOR", "Visitor Identity Linked", "Historical society visits securely associated with verified account."
	case "VISIT_REGISTERED":
		name, _ := details["visitor_name"].(string)
		if name != "" {
			return "VISITOR", "Expected Visitor Pre-Registered", fmt.Sprintf("%sScheduled entry pass for %s.", socPrefix, name)
		}
		return "VISITOR", "Expected Visitor Pre-Registered", socPrefix + "Visitor pre-registration pass created."
	case "WALK_IN_REGISTERED":
		name, _ := details["visitor_name"].(string)
		if name != "" {
			return "VISITOR", "Walk-In Visitor Registered", fmt.Sprintf("%sGatekeeper registered entry for %s.", socPrefix, name)
		}
		return "VISITOR", "Walk-In Visitor Registered", socPrefix + "Walk-in visitor registration recorded at gate."
	case "VISIT_APPROVED":
		return "VISITOR", "Visitor Entry Approved", socPrefix + "Resident approved gate admission request."
	case "VISIT_REJECTED":
		return "VISITOR", "Visitor Entry Declined", socPrefix + "Resident declined gate admission request."
	case "VISIT_CANCELLED":
		return "VISITOR", "Visitor Pass Cancelled", socPrefix + "Scheduled visit was cancelled."
	case "VISITOR_PASS_ISSUED":
		return "VISITOR", "Visitor Pass Issued", socPrefix + "Secure entry verification pass generated."
	case "VISITOR_PASS_REVOKED":
		return "VISITOR", "Visitor Pass Revoked", socPrefix + "Visitor pass invalidated before entry."
	case "VISITOR_CHECKED_IN":
		gate, _ := details["gate_name"].(string)
		if gate != "" {
			return "VISITOR", "Visitor Checked In", fmt.Sprintf("%sArrived at %s.", socPrefix, gate)
		}
		return "VISITOR", "Visitor Checked In", socPrefix + "Physical arrival recorded at society gate."
	case "VISITOR_CHECKED_OUT":
		gate, _ := details["gate_name"].(string)
		if gate != "" {
			return "VISITOR", "Visitor Checked Out", fmt.Sprintf("%sDeparted through %s.", socPrefix, gate)
		}
		return "VISITOR", "Visitor Checked Out", socPrefix + "Physical departure recorded at society gate."

	case "UNIT_VISITOR_POLICY_UPDATED":
		reqApp, _ := details["requires_approval"].(bool)
		if reqApp {
			return "SOCIETY", "Visitor Approval Policy Updated", socPrefix + "Direct gate entry disabled; resident approval now required for all visitors."
		}
		return "SOCIETY", "Visitor Approval Policy Updated", socPrefix + "Direct gate entry enabled for registered visitors."
	case "MEMBER_REGISTERED":
		return "SOCIETY", "Society Member Registered", socPrefix + "Member record added to society directory."
	case "UNIT_RELATIONSHIP_CREATED":
		return "SOCIETY", "Unit Residency Linked", socPrefix + "Unit relationship association created."
	case "MEMBER_DEPARTURE_CONFIRMED":
		return "SOCIETY", "Unit Tenancy Terminated", socPrefix + "Resident departure recorded."
	case "SOCIETY_ADMIN_ASSIGNED":
		return "SOCIETY", "Society Role Assigned", socPrefix + "Administrative assignment granted."
	case "SOCIETY_ADMIN_REVOKED":
		return "SOCIETY", "Society Role Revoked", socPrefix + "Administrative assignment revoked."

	default:
		return "ACCOUNT", strings.ReplaceAll(action, "_", " "), socPrefix + "Operational event recorded."
	}
}
