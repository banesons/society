package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Designation constants
const (
	DesignationCodeChairman        = "CHAIRMAN"
	DesignationCodeSecretary       = "SECRETARY"
	DesignationCodeTreasurer       = "TREASURER"
	DesignationCodeCommitteeMember = "COMMITTEE_MEMBER"
)

// Appointment statuses
const (
	AppointmentStatusActive  = "ACTIVE"
	AppointmentStatusEnded   = "ENDED"
	AppointmentStatusRevoked = "REVOKED"
)

// Audit actions for designations and committee appointments
const (
	ActionDesignationCreated          = "DESIGNATION_CREATED"
	ActionDesignationUpdated          = "DESIGNATION_UPDATED"
	ActionDesignationStatusChanged    = "DESIGNATION_STATUS_CHANGED"
	ActionCommitteeAppointmentCreated = "COMMITTEE_APPOINTMENT_CREATED"
	ActionCommitteeAppointmentEnded   = "COMMITTEE_APPOINTMENT_ENDED"
	ActionCommitteeAppointmentRevoked = "COMMITTEE_APPOINTMENT_REVOKED"
)

// Domain errors
var (
	ErrDesignationNotFound         = errors.New("designation not found in this society")
	ErrDesignationAlreadyExists    = errors.New("designation code already exists in this society")
	ErrDesignationInactive         = errors.New("cannot create appointment for an inactive designation")
	ErrDesignationAlreadyAppointed = errors.New("singleton designation already has an active appointment")
	ErrMemberAlreadyAppointed      = errors.New("member already holds this designation actively")
	ErrMemberNotActive             = errors.New("member is not in ACTIVE status in this society")
	ErrAppointmentNotFound         = errors.New("committee appointment not found in this society")
	ErrInvalidAppointmentDates     = errors.New("end date must be after or equal to start date")
	ErrInvalidAppointmentStatus    = errors.New("appointment is not in ACTIVE status")
	ErrInvalidDesignationCode      = errors.New("invalid designation code: must be 2-50 uppercase alphanumeric characters or underscores")
	ErrInvalidDesignationName      = errors.New("invalid designation name: must be between 2 and 100 characters")
	ErrConflictingDateRange        = errors.New("appointment date range conflicts with an existing appointment term")
	ErrMultiplePositionsDisallowed = errors.New("member cannot hold multiple designations concurrently")
)

var designationCodeRegex = regexp.MustCompile(`^[A-Z0-9_]{2,50}$`)

// SocietyDesignation represents a configurable committee position.
type SocietyDesignation struct {
	ID                               uuid.UUID `json:"id"`
	SocietyID                        uuid.UUID `json:"society_id"`
	Code                             string    `json:"code"`
	Name                             string    `json:"name"`
	Description                      *string   `json:"description,omitempty"`
	AllowsMultipleMembers            bool      `json:"allows_multiple_members"`
	AllowsMultiplePositionsPerMember bool      `json:"allows_multiple_positions_per_member"`
	IsActive                         bool      `json:"is_active"`
	ActiveAppointmentsCount          int       `json:"active_appointments_count"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`
}

// CommitteeAppointment represents an office tenure held by a society member.
type CommitteeAppointment struct {
	ID              uuid.UUID `json:"id"`
	SocietyID       uuid.UUID `json:"society_id"`
	DesignationID   uuid.UUID `json:"designation_id"`
	DesignationCode string    `json:"designation_code"`
	DesignationName string    `json:"designation_name"`
	MemberID        uuid.UUID `json:"member_id"`
	MemberName      string    `json:"member_name"`
	MemberPhone     string    `json:"member_phone"`
	StartDate       string    `json:"start_date"`
	ExpectedEndDate *string   `json:"expected_end_date,omitempty"`
	EndDate         *string   `json:"end_date,omitempty"`
	Status          string    `json:"status"`
	Notes           *string   `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// OfficeBearer represents a canonical executive office-bearer position and its current occupant.
type OfficeBearer struct {
	DesignationID     uuid.UUID  `json:"designation_id"`
	DesignationCode   string     `json:"designation_code"`
	DesignationName   string     `json:"designation_name"`
	MemberID          *uuid.UUID `json:"member_id,omitempty"`
	MemberName        *string    `json:"member_name,omitempty"`
	MemberPhoneMasked *string    `json:"member_phone_masked,omitempty"`
	StartDate         *string    `json:"start_date,omitempty"`
	ExpectedEndDate   *string    `json:"expected_end_date,omitempty"`
	Status            *string    `json:"status,omitempty"`
}

// CommitteeOverview summarizes current active positions in the society.
type CommitteeOverview struct {
	TotalDesignations   int                    `json:"total_designations"`
	ActiveDesignations  int                    `json:"active_designations"`
	ActiveAppointments  int                    `json:"active_appointments"`
	KeyPositionsFilled  map[string]bool        `json:"key_positions_filled"`
	CurrentAppointments []CommitteeAppointment `json:"current_appointments"`
	OfficeBearers       []OfficeBearer         `json:"office_bearers"`
}

// CreateDesignationParams holds arguments for creating a designation.
type CreateDesignationParams struct {
	Code                             string
	Name                             string
	Description                      *string
	AllowsMultipleMembers            bool
	AllowsMultiplePositionsPerMember bool
	ActorID                          uuid.UUID
	ClientIP                         string
	UserAgent                        string
}

// UpdateDesignationParams holds arguments for updating a designation.
type UpdateDesignationParams struct {
	Name                             string
	Description                      *string
	AllowsMultipleMembers            bool
	AllowsMultiplePositionsPerMember bool
	ActorID                          uuid.UUID
	ClientIP                         string
	UserAgent                        string
}

// CreateAppointmentParams holds arguments for creating an appointment.
type CreateAppointmentParams struct {
	DesignationID   uuid.UUID
	MemberID        uuid.UUID
	StartDate       string
	ExpectedEndDate *string
	Notes           *string
	ActorID         uuid.UUID
	ClientIP        string
	UserAgent       string
}

// EndAppointmentParams holds arguments for concluding or revoking an appointment.
type EndAppointmentParams struct {
	Action    string
	EndDate   string
	Notes     *string
	ActorID   uuid.UUID
	ClientIP  string
	UserAgent string
}

// EnsureDefaultDesignations ensures the standard defaults exist for a society.
func (r *Repository) EnsureDefaultDesignations(ctx context.Context, societyID uuid.UUID) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM society_designations WHERE society_id = $1", societyID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check designations: %w", err)
	}

	if count > 0 {
		return nil
	}

	defaults := []struct {
		code           string
		name           string
		description    string
		allowsMulti    bool
		allowsMultiPos bool
	}{
		{
			code:           DesignationCodeChairman,
			name:           "Chairman",
			description:    "Presiding officer and head of the managing committee",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           DesignationCodeSecretary,
			name:           "Secretary",
			description:    "Chief administrative officer responsible for executive management and records",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           DesignationCodeTreasurer,
			name:           "Treasurer",
			description:    "Financial officer responsible for accounts, collections, and disbursements",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           DesignationCodeCommitteeMember,
			name:           "Committee Member",
			description:    "Elected managing committee member representing society residents",
			allowsMulti:    true,
			allowsMultiPos: true,
		},
	}

	for _, d := range defaults {
		query := `
			INSERT INTO society_designations (
				society_id, code, name, description, allows_multiple_members, allows_multiple_positions_per_member, is_active, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
			ON CONFLICT (society_id, code) DO NOTHING
		`
		_, err := r.pool.Exec(ctx, query, societyID, d.code, d.name, d.description, d.allowsMulti, d.allowsMultiPos)
		if err != nil {
			return fmt.Errorf("failed to insert default designation %s: %w", d.code, err)
		}
	}

	return nil
}

// ListDesignations retrieves designations for a society, optionally filtered by status.
func (r *Repository) ListDesignations(ctx context.Context, societyID uuid.UUID, statusFilter string) ([]SocietyDesignation, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if err := r.EnsureDefaultDesignations(ctx, societyID); err != nil {
		return nil, err
	}

	query := `
		SELECT d.id, d.society_id, d.code, d.name, d.description,
		       d.allows_multiple_members, d.allows_multiple_positions_per_member,
		       d.is_active, d.created_at, d.updated_at,
		       COALESCE((
		           SELECT COUNT(*) FROM society_committee_appointments a 
		           WHERE a.designation_id = d.id AND a.status = 'ACTIVE'
		       ), 0) as active_appointments_count
		FROM society_designations d
		WHERE d.society_id = $1
	`
	args := []interface{}{societyID}

	switch strings.ToLower(statusFilter) {
	case "active":
		query += " AND d.is_active = true"
	case "inactive":
		query += " AND d.is_active = false"
	}

	query += " ORDER BY d.is_active DESC, d.code ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query designations: %w", err)
	}
	defer rows.Close()

	list := make([]SocietyDesignation, 0)
	for rows.Next() {
		var d SocietyDesignation
		err := rows.Scan(
			&d.ID, &d.SocietyID, &d.Code, &d.Name, &d.Description,
			&d.AllowsMultipleMembers, &d.AllowsMultiplePositionsPerMember,
			&d.IsActive, &d.CreatedAt, &d.UpdatedAt,
			&d.ActiveAppointmentsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan designation: %w", err)
		}
		list = append(list, d)
	}

	return list, nil
}

// CreateDesignation creates a new custom designation within a society.
func (r *Repository) CreateDesignation(ctx context.Context, societyID uuid.UUID, params CreateDesignationParams) (*SocietyDesignation, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedCode := strings.ToUpper(strings.TrimSpace(params.Code))
	if !designationCodeRegex.MatchString(trimmedCode) {
		return nil, ErrInvalidDesignationCode
	}

	trimmedName := strings.TrimSpace(params.Name)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return nil, ErrInvalidDesignationName
	}

	var trimmedDesc *string
	if params.Description != nil {
		val := strings.TrimSpace(*params.Description)
		if val != "" {
			trimmedDesc = &val
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	d := &SocietyDesignation{
		SocietyID:                        societyID,
		Code:                             trimmedCode,
		Name:                             trimmedName,
		Description:                      trimmedDesc,
		AllowsMultipleMembers:            params.AllowsMultipleMembers,
		AllowsMultiplePositionsPerMember: params.AllowsMultiplePositionsPerMember,
		IsActive:                         true,
	}

	query := `
		INSERT INTO society_designations (
			society_id, code, name, description, allows_multiple_members, allows_multiple_positions_per_member, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		d.SocietyID, d.Code, d.Name, d.Description,
		d.AllowsMultipleMembers, d.AllowsMultiplePositionsPerMember, d.IsActive,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDesignationAlreadyExists
		}
		return nil, fmt.Errorf("failed to insert designation: %w", err)
	}

	// Transactional audit log
	auditPayload, _ := json.Marshal(map[string]interface{}{
		"designation_id":          d.ID,
		"society_id":              d.SocietyID,
		"code":                    d.Code,
		"name":                    d.Name,
		"allows_multiple_members": d.AllowsMultipleMembers,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, params.ActorID, ActionDesignationCreated, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit designation creation: %w", err)
	}

	return d, nil
}

// UpdateDesignation updates display name, description, and cardinality settings of a designation.
func (r *Repository) UpdateDesignation(ctx context.Context, societyID, designationID uuid.UUID, params UpdateDesignationParams) (*SocietyDesignation, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedName := strings.TrimSpace(params.Name)
	if len(trimmedName) < 2 || len(trimmedName) > 100 {
		return nil, ErrInvalidDesignationName
	}

	var trimmedDesc *string
	if params.Description != nil {
		val := strings.TrimSpace(*params.Description)
		if val != "" {
			trimmedDesc = &val
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var d SocietyDesignation
	err = tx.QueryRow(ctx, `
		SELECT id, society_id, code, is_active FROM society_designations 
		WHERE id = $1 AND society_id = $2 FOR UPDATE
	`, designationID, societyID).Scan(&d.ID, &d.SocietyID, &d.Code, &d.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDesignationNotFound
		}
		return nil, fmt.Errorf("failed to lock designation: %w", err)
	}

	query := `
		UPDATE society_designations
		SET name = $1, description = $2, allows_multiple_members = $3,
		    allows_multiple_positions_per_member = $4, updated_at = NOW()
		WHERE id = $5 AND society_id = $6
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		trimmedName, trimmedDesc, params.AllowsMultipleMembers,
		params.AllowsMultiplePositionsPerMember, designationID, societyID,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update designation: %w", err)
	}

	d.Name = trimmedName
	d.Description = trimmedDesc
	d.AllowsMultipleMembers = params.AllowsMultipleMembers
	d.AllowsMultiplePositionsPerMember = params.AllowsMultiplePositionsPerMember

	// Audit log
	auditPayload, _ := json.Marshal(map[string]interface{}{
		"designation_id":          d.ID,
		"society_id":              d.SocietyID,
		"code":                    d.Code,
		"name":                    d.Name,
		"allows_multiple_members": d.AllowsMultipleMembers,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, params.ActorID, ActionDesignationUpdated, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit designation update: %w", err)
	}

	return &d, nil
}

// SetDesignationStatus activates or deactivates a designation.
func (r *Repository) SetDesignationStatus(ctx context.Context, societyID, designationID uuid.UUID, isActive bool, actorID uuid.UUID, clientIP, userAgent string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var code string
	err = tx.QueryRow(ctx, `
		SELECT code FROM society_designations 
		WHERE id = $1 AND society_id = $2 FOR UPDATE
	`, designationID, societyID).Scan(&code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDesignationNotFound
		}
		return fmt.Errorf("failed to lock designation: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE society_designations
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2 AND society_id = $3
	`, isActive, designationID, societyID)
	if err != nil {
		return fmt.Errorf("failed to update designation status: %w", err)
	}

	auditPayload, _ := json.Marshal(map[string]interface{}{
		"designation_id": designationID,
		"society_id":     societyID,
		"code":           code,
		"is_active":      isActive,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionDesignationStatusChanged, string(auditPayload), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// ListAppointments retrieves appointments in a society with filtering.
func (r *Repository) ListAppointments(ctx context.Context, societyID uuid.UUID, statusFilter string, memberID, designationID *uuid.UUID) ([]CommitteeAppointment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	baseWhere := "WHERE a.society_id = $1"
	args := []interface{}{societyID}
	argIdx := 2

	if statusFilter != "" && !strings.EqualFold(statusFilter, "all") {
		baseWhere += fmt.Sprintf(" AND a.status = $%d", argIdx)
		args = append(args, strings.ToUpper(statusFilter))
		argIdx++
	}

	if memberID != nil {
		baseWhere += fmt.Sprintf(" AND a.member_id = $%d", argIdx)
		args = append(args, *memberID)
		argIdx++
	}

	if designationID != nil {
		baseWhere += fmt.Sprintf(" AND a.designation_id = $%d", argIdx)
		args = append(args, *designationID)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.society_id, a.designation_id, d.code as designation_code, d.name as designation_name,
		       a.member_id, CONCAT(m.first_name, ' ', COALESCE(m.last_name, '')) as member_name,
		       m.phone as member_phone,
		       TO_CHAR(a.start_date, 'YYYY-MM-DD') as start_date,
		       CASE WHEN a.expected_end_date IS NOT NULL THEN TO_CHAR(a.expected_end_date, 'YYYY-MM-DD') ELSE NULL END as expected_end_date,
		       CASE WHEN a.end_date IS NOT NULL THEN TO_CHAR(a.end_date, 'YYYY-MM-DD') ELSE NULL END as end_date,
		       a.status, a.notes, a.created_at, a.updated_at
		FROM society_committee_appointments a
		JOIN society_designations d ON a.designation_id = d.id
		JOIN society_members m ON a.member_id = m.id
		%s
		ORDER BY 
			CASE a.status WHEN 'ACTIVE' THEN 1 WHEN 'ENDED' THEN 2 ELSE 3 END ASC,
			a.start_date DESC
	`, baseWhere)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query appointments: %w", err)
	}
	defer rows.Close()

	appointments := make([]CommitteeAppointment, 0)
	for rows.Next() {
		var apt CommitteeAppointment
		var rawPhone string
		err := rows.Scan(
			&apt.ID, &apt.SocietyID, &apt.DesignationID, &apt.DesignationCode, &apt.DesignationName,
			&apt.MemberID, &apt.MemberName, &rawPhone,
			&apt.StartDate, &apt.ExpectedEndDate, &apt.EndDate,
			&apt.Status, &apt.Notes, &apt.CreatedAt, &apt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan appointment: %w", err)
		}
		apt.MemberPhone = MaskPhone(rawPhone)
		appointments = append(appointments, apt)
	}

	return appointments, nil
}

// CreateAppointment appoints an active member to an active designation with concurrency protection.
func (r *Repository) CreateAppointment(ctx context.Context, societyID uuid.UUID, params CreateAppointmentParams) (*CommitteeAppointment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	parsedStartDate, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return nil, errors.New("invalid start date: must be YYYY-MM-DD")
	}

	var parsedExpectedEndDate *time.Time
	if params.ExpectedEndDate != nil && strings.TrimSpace(*params.ExpectedEndDate) != "" {
		t, err := time.Parse("2006-01-02", *params.ExpectedEndDate)
		if err != nil {
			return nil, errors.New("invalid expected end date: must be YYYY-MM-DD")
		}
		if t.Before(parsedStartDate) {
			return nil, ErrInvalidAppointmentDates
		}
		parsedExpectedEndDate = &t
	}

	var trimmedNotes *string
	if params.Notes != nil {
		val := strings.TrimSpace(*params.Notes)
		if val != "" {
			trimmedNotes = &val
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Lock designation row under transaction to enforce cardinality safely
	var desigCode, desigName string
	var allowsMulti, allowsMultiPos, isActive bool
	err = tx.QueryRow(ctx, `
		SELECT code, name, allows_multiple_members, allows_multiple_positions_per_member, is_active 
		FROM society_designations 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, params.DesignationID, societyID).Scan(&desigCode, &desigName, &allowsMulti, &allowsMultiPos, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDesignationNotFound
		}
		return nil, fmt.Errorf("failed to lock designation: %w", err)
	}

	if !isActive {
		return nil, ErrDesignationInactive
	}

	// 2. Lock and verify member
	var memberFirst, memberPhone, memberStatus string
	var memberLast *string
	err = tx.QueryRow(ctx, `
		SELECT first_name, last_name, phone, status 
		FROM society_members 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, params.MemberID, societyID).Scan(&memberFirst, &memberLast, &memberPhone, &memberStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to verify member: %w", err)
	}

	if memberStatus != MemberStatusActive {
		return nil, ErrMemberNotActive
	}

	// 3. Enforce Multiple Designations Per Member rule:
	// If the designation disallows multiple positions, member must have 0 other active appointments.
	// If the designation allows multiple positions, member must not hold any active appointment
	// whose designation disallows multiple positions.
	if !allowsMultiPos {
		var activeCount int
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM society_committee_appointments 
			WHERE member_id = $1 AND society_id = $2 AND status = 'ACTIVE'
		`, params.MemberID, societyID).Scan(&activeCount)
		if err != nil {
			return nil, fmt.Errorf("failed to check member active appointments: %w", err)
		}
		if activeCount > 0 {
			return nil, ErrMultiplePositionsDisallowed
		}
	} else {
		var exclusiveHeld int
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM society_committee_appointments a
			JOIN society_designations d ON a.designation_id = d.id
			WHERE a.member_id = $1 AND a.society_id = $2 AND a.status = 'ACTIVE'
			  AND d.allows_multiple_positions_per_member = FALSE
		`, params.MemberID, societyID).Scan(&exclusiveHeld)
		if err != nil {
			return nil, fmt.Errorf("failed to check exclusive designation hold: %w", err)
		}
		if exclusiveHeld > 0 {
			return nil, ErrMultiplePositionsDisallowed
		}
	}

	// 4. Singleton cardinality & date range overlap check:
	// A singleton designation permits at most 1 active occupant, and appointment terms must not overlap.
	if !allowsMulti {
		var activeCount int
		err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM society_committee_appointments 
			WHERE designation_id = $1 AND society_id = $2 AND status = 'ACTIVE'
		`, params.DesignationID, societyID).Scan(&activeCount)
		if err != nil {
			return nil, fmt.Errorf("failed to check active appointments count: %w", err)
		}
		if activeCount > 0 {
			return nil, ErrDesignationAlreadyAppointed
		}

		// Check historical appointments for conflicting date ranges on this singleton designation
		var singletonOverlap bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM society_committee_appointments
				WHERE designation_id = $1 AND society_id = $2 AND status IN ('ENDED', 'REVOKED')
				  AND start_date <= COALESCE($3, '9999-12-31'::date)
				  AND $4 <= COALESCE(end_date, expected_end_date, '9999-12-31'::date)
			)
		`, params.DesignationID, societyID, parsedExpectedEndDate, parsedStartDate).Scan(&singletonOverlap)
		if err != nil {
			return nil, fmt.Errorf("failed to check historical singleton overlap: %w", err)
		}
		if singletonOverlap {
			return nil, ErrConflictingDateRange
		}
	}

	// 5. Member-designation overlap check:
	// A member cannot hold the identical designation actively concurrently,
	// nor can they hold overlapping appointment terms for the same designation.
	var activeHolds bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM society_committee_appointments 
			WHERE designation_id = $1 AND member_id = $2 AND society_id = $3 AND status = 'ACTIVE'
		)
	`, params.DesignationID, params.MemberID, societyID).Scan(&activeHolds)
	if err != nil {
		return nil, fmt.Errorf("failed to check member designation overlap: %w", err)
	}
	if activeHolds {
		return nil, ErrMemberAlreadyAppointed
	}

	var historicalMemberOverlap bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM society_committee_appointments
			WHERE designation_id = $1 AND member_id = $2 AND society_id = $3 AND status IN ('ENDED', 'REVOKED')
			  AND start_date <= COALESCE($4, '9999-12-31'::date)
			  AND $5 <= COALESCE(end_date, expected_end_date, '9999-12-31'::date)
		)
	`, params.DesignationID, params.MemberID, societyID, parsedExpectedEndDate, parsedStartDate).Scan(&historicalMemberOverlap)
	if err != nil {
		return nil, fmt.Errorf("failed to check member historical overlap: %w", err)
	}
	if historicalMemberOverlap {
		return nil, ErrConflictingDateRange
	}

	// 5. Insert appointment
	apt := &CommitteeAppointment{
		SocietyID:       societyID,
		DesignationID:   params.DesignationID,
		DesignationCode: desigCode,
		DesignationName: desigName,
		MemberID:        params.MemberID,
		MemberName: strings.TrimSpace(memberFirst + " " + func() string {
			if memberLast != nil {
				return *memberLast
			}
			return ""
		}()),
		MemberPhone:     MaskPhone(memberPhone),
		StartDate:       params.StartDate,
		ExpectedEndDate: params.ExpectedEndDate,
		Status:          AppointmentStatusActive,
		Notes:           trimmedNotes,
	}

	query := `
		INSERT INTO society_committee_appointments (
			society_id, designation_id, member_id, start_date, expected_end_date, end_date, status, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, NULL, 'ACTIVE', $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		apt.SocietyID, apt.DesignationID, apt.MemberID,
		parsedStartDate, parsedExpectedEndDate, apt.Notes,
	).Scan(&apt.ID, &apt.CreatedAt, &apt.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrMemberAlreadyAppointed
		}
		return nil, fmt.Errorf("failed to insert appointment: %w", err)
	}

	// 6. Transactional audit log
	auditPayload, _ := json.Marshal(map[string]interface{}{
		"appointment_id":   apt.ID,
		"society_id":       societyID,
		"designation_id":   apt.DesignationID,
		"designation_code": apt.DesignationCode,
		"member_id":        apt.MemberID,
		"start_date":       apt.StartDate,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, params.ActorID, ActionCommitteeAppointmentCreated, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit appointment: %w", err)
	}

	return apt, nil
}

// EndAppointment concludes or revokes an active committee appointment.
func (r *Repository) EndAppointment(ctx context.Context, societyID, appointmentID uuid.UUID, params EndAppointmentParams) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	action := strings.ToUpper(strings.TrimSpace(params.Action))
	if action != AppointmentStatusEnded && action != AppointmentStatusRevoked {
		action = AppointmentStatusEnded
	}

	parsedEndDate, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return errors.New("invalid end date: must be YYYY-MM-DD")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var designationID, memberID uuid.UUID
	var startDate time.Time
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT designation_id, member_id, start_date, status 
		FROM society_committee_appointments 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, appointmentID, societyID).Scan(&designationID, &memberID, &startDate, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppointmentNotFound
		}
		return fmt.Errorf("failed to lock appointment: %w", err)
	}

	if currentStatus != AppointmentStatusActive {
		return ErrInvalidAppointmentStatus
	}

	if parsedEndDate.Before(startDate) {
		return ErrInvalidAppointmentDates
	}

	noteSuffix := ""
	if params.Notes != nil && strings.TrimSpace(*params.Notes) != "" {
		noteSuffix = " | " + strings.TrimSpace(*params.Notes)
	}

	query := `
		UPDATE society_committee_appointments
		SET status = $1, end_date = $2, notes = COALESCE(notes, '') || $3, updated_at = NOW()
		WHERE id = $4 AND society_id = $5
	`
	_, err = tx.Exec(ctx, query, action, parsedEndDate, noteSuffix, appointmentID, societyID)
	if err != nil {
		return fmt.Errorf("failed to update appointment: %w", err)
	}

	// Audit action
	auditAction := ActionCommitteeAppointmentEnded
	if action == AppointmentStatusRevoked {
		auditAction = ActionCommitteeAppointmentRevoked
	}

	auditPayload, _ := json.Marshal(map[string]interface{}{
		"appointment_id": appointmentID,
		"society_id":     societyID,
		"designation_id": designationID,
		"member_id":      memberID,
		"end_date":       params.EndDate,
		"action":         action,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, params.ActorID, auditAction, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// GetCommitteeOverview returns a high-level summary of the society's active committee.
func (r *Repository) GetCommitteeOverview(ctx context.Context, societyID uuid.UUID) (*CommitteeOverview, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if err := r.EnsureDefaultDesignations(ctx, societyID); err != nil {
		return nil, err
	}

	// Retrieve active appointments
	appointments, err := r.ListAppointments(ctx, societyID, "ACTIVE", nil, nil)
	if err != nil {
		return nil, err
	}

	if appointments == nil {
		appointments = make([]CommitteeAppointment, 0)
	}

	// Count designations
	var totalDesigs, activeDesigs int
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active = true)
		FROM society_designations
		WHERE society_id = $1
	`, societyID).Scan(&totalDesigs, &activeDesigs)
	if err != nil {
		return nil, fmt.Errorf("failed to count designations: %w", err)
	}

	keyPositions := map[string]bool{
		DesignationCodeChairman:  false,
		DesignationCodeSecretary: false,
		DesignationCodeTreasurer: false,
	}

	apptsByDesigCode := make(map[string]CommitteeAppointment)
	for _, apt := range appointments {
		if _, ok := keyPositions[apt.DesignationCode]; ok {
			keyPositions[apt.DesignationCode] = true
			apptsByDesigCode[apt.DesignationCode] = apt
		}
	}

	// Fetch canonical office bearer designations for this society
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name
		FROM society_designations
		WHERE society_id = $1 AND code IN ('CHAIRMAN', 'SECRETARY', 'TREASURER')
		ORDER BY 
			CASE code 
				WHEN 'CHAIRMAN' THEN 1 
				WHEN 'SECRETARY' THEN 2 
				WHEN 'TREASURER' THEN 3 
				ELSE 4 
			END ASC
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query office bearer designations: %w", err)
	}
	defer rows.Close()

	officeBearers := make([]OfficeBearer, 0, 3)
	for rows.Next() {
		var ob OfficeBearer
		if err := rows.Scan(&ob.DesignationID, &ob.DesignationCode, &ob.DesignationName); err != nil {
			return nil, fmt.Errorf("failed to scan office bearer designation: %w", err)
		}
		if apt, found := apptsByDesigCode[ob.DesignationCode]; found {
			ob.MemberID = &apt.MemberID
			ob.MemberName = &apt.MemberName
			ob.MemberPhoneMasked = &apt.MemberPhone
			ob.StartDate = &apt.StartDate
			ob.ExpectedEndDate = apt.ExpectedEndDate
			ob.Status = &apt.Status
		}
		officeBearers = append(officeBearers, ob)
	}

	return &CommitteeOverview{
		TotalDesignations:   totalDesigs,
		ActiveDesignations:  activeDesigs,
		ActiveAppointments:  len(appointments),
		KeyPositionsFilled:  keyPositions,
		CurrentAppointments: appointments,
		OfficeBearers:       officeBearers,
	}, nil
}
