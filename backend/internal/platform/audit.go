package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAuditLogNotFound = errors.New("audit log not found")
)

// AuditQueryFilters specifies parameters for filtering audit logs.
type AuditQueryFilters struct {
	Page         int
	Limit        int
	Search       string
	Action       string
	ActorID      *uuid.UUID
	TargetUserID *uuid.UUID
	SocietyID    *uuid.UUID
	StartDate    *time.Time
	EndDate      *time.Time
}

// EnrichedAuditLogItem is the comprehensive, sanitized, and labeled audit event contract.
type EnrichedAuditLogItem struct {
	ID            uuid.UUID              `json:"id"`
	ActorID       uuid.UUID              `json:"actor_id"`
	ActorName     string                 `json:"actor_name"`
	ActorPhone    string                 `json:"actor_phone"`
	Action        string                 `json:"action"`
	ActionLabel   string                 `json:"action_label"`
	TargetUserID  *uuid.UUID             `json:"target_user_id,omitempty"`
	TargetDisplay string                 `json:"target_display,omitempty"`
	SocietyID     *uuid.UUID             `json:"society_id,omitempty"`
	SocietyName   string                 `json:"society_name,omitempty"`
	SocietyCode   string                 `json:"society_code,omitempty"`
	Details       map[string]interface{} `json:"details"`
	ClientIP      string                 `json:"client_ip,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// AuditPagination represents pagination metadata.
type AuditPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

var sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|access_token|refresh_token|otp|pin|auth_header|authorization|service_role_key|session_secret|signed_url|private_key)`)

// MaskPhone masks phone numbers for audit display (e.g., +919820000101 -> +91 98*** ***01).
func MaskPhone(phone string) string {
	clean := strings.TrimSpace(phone)
	if clean == "" {
		return ""
	}
	digits := regexp.MustCompile(`\D`).ReplaceAllString(clean, "")
	if len(digits) == 12 && strings.HasPrefix(digits, "91") {
		return fmt.Sprintf("+91 %s*** ***%s", digits[2:4], digits[10:])
	}
	if len(digits) == 10 {
		return fmt.Sprintf("+91 %s*** ***%s", digits[:2], digits[8:])
	}
	if len(digits) > 4 {
		return fmt.Sprintf("%s***%s", digits[:2], digits[len(digits)-2:])
	}
	return clean
}

// SanitizeAuditDetails recursively redacts sensitive fields from audit details.
func SanitizeAuditDetails(raw map[string]interface{}) map[string]interface{} {
	if raw == nil {
		return map[string]interface{}{}
	}
	result := make(map[string]interface{}, len(raw))
	for k, v := range raw {
		if sensitiveKeyPattern.MatchString(k) {
			result[k] = "[REDACTED]"
			continue
		}

		// Mask phone fields if present in details
		if strings.Contains(strings.ToLower(k), "phone") {
			if s, ok := v.(string); ok {
				result[k] = MaskPhone(s)
				continue
			}
		}

		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = SanitizeAuditDetails(val)
		case []interface{}:
			result[k] = sanitizeSlice(val)
		default:
			result[k] = val
		}
	}
	return result
}

func sanitizeSlice(slice []interface{}) []interface{} {
	out := make([]interface{}, len(slice))
	for i, item := range slice {
		switch v := item.(type) {
		case map[string]interface{}:
			out[i] = SanitizeAuditDetails(v)
		case []interface{}:
			out[i] = sanitizeSlice(v)
		case string:
			if sensitiveKeyPattern.MatchString(v) {
				out[i] = "[REDACTED]"
			} else {
				out[i] = v
			}
		default:
			out[i] = v
		}
	}
	return out
}

// FormatActionLabel produces an editorial human-readable label for an action code.
func FormatActionLabel(action string) string {
	known := map[string]string{
		"SUPER_ADMIN_BOOTSTRAPPED":      "Super Admin Bootstrapped",
		"ADMIN_CREATED":                 "Platform Admin Created",
		"ADMIN_DISABLED":                "Platform Admin Disabled",
		"ADMIN_REACTIVATED":             "Platform Admin Reactivated",
		"TEST_AUDIT":                    "Test Audit Event",
		"SOCIETY_CREATED":               "Society Created",
		"SOCIETY_ACTIVATED":             "Society Activated",
		"SOCIETY_SUSPENDED":             "Society Suspended",
		"SOCIETY_REACTIVATED":           "Society Reactivated",
		"SOCIETY_PROFILE_UPDATED":       "Society Profile Updated",
		"SOCIETY_ADMIN_ASSIGNED":        "Society Admin Assigned",
		"SOCIETY_ADMIN_REVOKED":         "Society Admin Revoked",
		"BUILDING_CREATED":              "Building Created",
		"WING_CREATED":                  "Wing Created",
		"UNITS_GENERATED":               "Units Generated",
		"GATE_CREATED":                  "Gate Created",
		"GATE_UPDATED":                  "Gate Updated",
		"STAFFING_REQUIREMENTS_UPDATED": "Staffing Requirements Updated",
		"MEMBER_REGISTERED":             "Member Registered",
		"MEMBER_DEPARTED":               "Member Departed",
		"UNIT_RELATIONSHIP_CREATED":     "Unit Relationship Created",
		"UNIT_RELATIONSHIP_ENDED":       "Unit Relationship Ended",
		"DESIGNATION_CREATED":           "Designation Created",
		"DESIGNATION_UPDATED":           "Designation Updated",
		"DESIGNATION_STATUS_CHANGED":    "Designation Status Changed",
		"COMMITTEE_APPOINTMENT_CREATED": "Committee Appointment Created",
		"COMMITTEE_APPOINTMENT_ENDED":   "Committee Appointment Ended",
		"GATEKEEPER_ASSIGNED":           "Gatekeeper Assigned",
		"GATEKEEPER_REVOKED":            "Gatekeeper Revoked",
		"SHIFT_CREATED":                 "Shift Created",
		"GATE_DUTY_STARTED":             "Gate Duty Started",
		"GATE_DUTY_ENDED":               "Gate Duty Ended",
		"VISIT_REGISTERED":              "Visit Registered",
		"WALK_IN_REGISTERED":            "Walk-in Visit Registered",
		"VISIT_APPROVED":                "Visit Approved",
		"VISIT_REJECTED":                "Visit Rejected",
		"VISITOR_PASS_ISSUED":           "Visitor Pass Issued",
		"VISITOR_CHECKED_IN":            "Visitor Checked In",
		"VISITOR_CHECKED_OUT":           "Visitor Checked Out",
		"UNIT_VISITOR_POLICY_UPDATED":   "Unit Visitor Policy Updated",
		"ANNOUNCEMENT_CREATED":          "Announcement Created",
		"ANNOUNCEMENT_UPDATED":          "Announcement Updated",
		"ANNOUNCEMENT_PUBLISHED":        "Announcement Published",
		"ANNOUNCEMENT_ARCHIVED":         "Announcement Archived",
		"CHARGE_CATEGORY_CREATED":       "Charge Category Created",
		"CHARGE_CATEGORY_UPDATED":       "Charge Category Updated",
		"CHARGE_CONFIG_CREATED":         "Charge Config Created",
		"CHARGE_CONFIG_DEACTIVATED":     "Charge Config Deactivated",
		"BILLING_PERIOD_CREATED":        "Billing Period Created",
		"BILLS_GENERATED":               "Bills Generated",
		"BILL_VOIDED":                   "Bill Voided",
		"MAINTENANCE_PAYMENT_RECORDED":  "Maintenance Payment Recorded",
		"MAINTENANCE_PAYMENT_REVERSED":  "Maintenance Payment Reversed",
		"DOCUMENT_CATEGORY_CREATED":     "Document Category Created",
		"DOCUMENT_CREATED":              "Document Created",
		"DOCUMENT_VERSION_UPLOADED":     "Document Version Uploaded",
		"DOCUMENT_DOWNLOADED":           "Document Downloaded",
		"DOCUMENT_ARCHIVED":             "Document Archived",
		"DOCUMENT_RESTORED":             "Document Restored",
	}

	if label, ok := known[action]; ok {
		return label
	}

	// Fallback: title-case words separated by underscores
	parts := strings.Split(action, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

// ListAuditLogsEnriched returns a paginated, filtered, and sanitized slice of audit logs.
func (r *Repository) ListAuditLogsEnriched(ctx context.Context, filters AuditQueryFilters) ([]EnrichedAuditLogItem, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filters.SocietyID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(pal.society_id = $%d OR pal.details->>'society_id' = $%d::text)", argIdx, argIdx))
		args = append(args, *filters.SocietyID)
		argIdx++
	}

	if strings.TrimSpace(filters.Action) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("pal.action = $%d", argIdx))
		args = append(args, strings.TrimSpace(filters.Action))
		argIdx++
	}

	if filters.ActorID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("pal.actor_id = $%d", argIdx))
		args = append(args, *filters.ActorID)
		argIdx++
	}

	if filters.TargetUserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("pal.target_user_id = $%d", argIdx))
		args = append(args, *filters.TargetUserID)
		argIdx++
	}

	if filters.StartDate != nil && !filters.StartDate.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("pal.created_at >= $%d", argIdx))
		args = append(args, *filters.StartDate)
		argIdx++
	}

	if filters.EndDate != nil && !filters.EndDate.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("pal.created_at <= $%d", argIdx))
		args = append(args, *filters.EndDate)
		argIdx++
	}

	if strings.TrimSpace(filters.Search) != "" {
		pattern := "%" + strings.TrimSpace(filters.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf(`(
			pal.action ILIKE $%d OR
			s.name ILIKE $%d OR
			s.society_code ILIKE $%d OR
			u.phone ILIKE $%d OR
			TRIM(CONCAT(m.first_name, ' ', COALESCE(m.last_name, ''))) ILIKE $%d OR
			pal.details->>'title' ILIKE $%d OR
			pal.details->>'document_number' ILIKE $%d
		)`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, pattern)
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM platform_audit_logs pal
		LEFT JOIN users u ON pal.actor_id = u.id
		LEFT JOIN societies s ON (pal.society_id = s.id OR s.id::text = pal.details->>'society_id')
		LEFT JOIN society_members m ON (m.user_id = pal.actor_id AND (m.society_id = pal.society_id OR m.society_id::text = pal.details->>'society_id'))
		%s
	`, whereSQL)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count enriched audit logs: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			pal.id,
			pal.actor_id,
			COALESCE(u.phone, ''),
			COALESCE(TRIM(CONCAT(m.first_name, ' ', COALESCE(m.last_name, ''))), ''),
			pal.action,
			pal.target_user_id,
			COALESCE(tu.phone, ''),
			pal.society_id,
			pal.details->>'society_id',
			COALESCE(s.name, ''),
			COALESCE(s.society_code, ''),
			pal.details,
			COALESCE(pal.client_ip, ''),
			COALESCE(pal.user_agent, ''),
			pal.created_at
		FROM platform_audit_logs pal
		LEFT JOIN users u ON pal.actor_id = u.id
		LEFT JOIN users tu ON pal.target_user_id = tu.id
		LEFT JOIN societies s ON (pal.society_id = s.id OR s.id::text = pal.details->>'society_id')
		LEFT JOIN society_members m ON (m.user_id = pal.actor_id AND (m.society_id = pal.society_id OR m.society_id::text = pal.details->>'society_id'))
		%s
		ORDER BY pal.created_at DESC, pal.id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query enriched audit logs: %w", err)
	}
	defer rows.Close()

	var items []EnrichedAuditLogItem
	for rows.Next() {
		var item EnrichedAuditLogItem
		var actorPhone, actorName, targetPhone string
		var rawDetails []byte
		var colSocID *uuid.UUID
		var jsonSocID sql.NullString

		if err := rows.Scan(
			&item.ID,
			&item.ActorID,
			&actorPhone,
			&actorName,
			&item.Action,
			&item.TargetUserID,
			&targetPhone,
			&colSocID,
			&jsonSocID,
			&item.SocietyName,
			&item.SocietyCode,
			&rawDetails,
			&item.ClientIP,
			&item.UserAgent,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan enriched audit log: %w", err)
		}

		// Resolve society ID
		if colSocID != nil {
			item.SocietyID = colSocID
		} else if jsonSocID.Valid && jsonSocID.String != "" {
			if parsed, err := uuid.Parse(jsonSocID.String); err == nil {
				item.SocietyID = &parsed
			}
		}

		// Humanize action label
		item.ActionLabel = FormatActionLabel(item.Action)

		// Actor display
		if actorName != "" {
			item.ActorName = actorName
		} else if actorPhone != "" {
			item.ActorName = MaskPhone(actorPhone)
		} else {
			item.ActorName = "System"
		}
		item.ActorPhone = MaskPhone(actorPhone)

		// Target display
		if targetPhone != "" {
			item.TargetDisplay = MaskPhone(targetPhone)
		}

		// Parse & sanitize details
		var rawMap map[string]interface{}
		if len(rawDetails) > 0 {
			_ = json.Unmarshal(rawDetails, &rawMap)
		}
		item.Details = SanitizeAuditDetails(rawMap)

		// If target display wasn't user phone, check details for title, document_number, etc.
		if item.TargetDisplay == "" {
			if title, ok := item.Details["title"].(string); ok && title != "" {
				item.TargetDisplay = title
			} else if docNum, ok := item.Details["document_number"].(string); ok && docNum != "" {
				item.TargetDisplay = docNum
			} else if unitNumber, ok := item.Details["unit_number"].(string); ok && unitNumber != "" {
				item.TargetDisplay = "Unit " + unitNumber
			}
		}

		items = append(items, item)
	}

	if items == nil {
		items = []EnrichedAuditLogItem{}
	}

	return items, total, nil
}

// GetAuditLogByID returns a single enriched audit log by ID, optionally verifying society ownership.
func (r *Repository) GetAuditLogByID(ctx context.Context, id uuid.UUID, societyIDFilter *uuid.UUID) (*EnrichedAuditLogItem, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT
			pal.id,
			pal.actor_id,
			COALESCE(u.phone, ''),
			COALESCE(TRIM(CONCAT(m.first_name, ' ', COALESCE(m.last_name, ''))), ''),
			pal.action,
			pal.target_user_id,
			COALESCE(tu.phone, ''),
			pal.society_id,
			pal.details->>'society_id',
			COALESCE(s.name, ''),
			COALESCE(s.society_code, ''),
			pal.details,
			COALESCE(pal.client_ip, ''),
			COALESCE(pal.user_agent, ''),
			pal.created_at
		FROM platform_audit_logs pal
		LEFT JOIN users u ON pal.actor_id = u.id
		LEFT JOIN users tu ON pal.target_user_id = tu.id
		LEFT JOIN societies s ON (pal.society_id = s.id OR s.id::text = pal.details->>'society_id')
		LEFT JOIN society_members m ON (m.user_id = pal.actor_id AND (m.society_id = pal.society_id OR m.society_id::text = pal.details->>'society_id'))
		WHERE pal.id = $1
	`
	args := []interface{}{id}

	if societyIDFilter != nil {
		query += " AND (pal.society_id = $2 OR pal.details->>'society_id' = $2::text)"
		args = append(args, *societyIDFilter)
	}

	var item EnrichedAuditLogItem
	var actorPhone, actorName, targetPhone string
	var rawDetails []byte
	var colSocID *uuid.UUID
	var jsonSocID sql.NullString

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.ActorID,
		&actorPhone,
		&actorName,
		&item.Action,
		&item.TargetUserID,
		&targetPhone,
		&colSocID,
		&jsonSocID,
		&item.SocietyName,
		&item.SocietyCode,
		&rawDetails,
		&item.ClientIP,
		&item.UserAgent,
		&item.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAuditLogNotFound
		}
		return nil, fmt.Errorf("failed to get audit log by id: %w", err)
	}

	if colSocID != nil {
		item.SocietyID = colSocID
	} else if jsonSocID.Valid && jsonSocID.String != "" {
		if parsed, err := uuid.Parse(jsonSocID.String); err == nil {
			item.SocietyID = &parsed
		}
	}

	item.ActionLabel = FormatActionLabel(item.Action)

	if actorName != "" {
		item.ActorName = actorName
	} else if actorPhone != "" {
		item.ActorName = MaskPhone(actorPhone)
	} else {
		item.ActorName = "System"
	}
	item.ActorPhone = MaskPhone(actorPhone)

	if targetPhone != "" {
		item.TargetDisplay = MaskPhone(targetPhone)
	}

	var rawMap map[string]interface{}
	if len(rawDetails) > 0 {
		_ = json.Unmarshal(rawDetails, &rawMap)
	}
	item.Details = SanitizeAuditDetails(rawMap)

	if item.TargetDisplay == "" {
		if title, ok := item.Details["title"].(string); ok && title != "" {
			item.TargetDisplay = title
		} else if docNum, ok := item.Details["document_number"].(string); ok && docNum != "" {
			item.TargetDisplay = docNum
		}
	}

	return &item, nil
}
