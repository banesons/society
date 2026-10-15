package society

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chs/backend/internal/auth"
)

// Member lifecycle states
const (
	MemberStatusActive   = "ACTIVE"
	MemberStatusInactive = "INACTIVE"
	MemberStatusLeft     = "LEFT"
)

// Unit relationship types
const (
	RelTypePrimaryOwner       = "PRIMARY_OWNER"
	RelTypeSecondaryOwner     = "SECONDARY_OWNER"
	RelTypeOwnerFamily        = "OWNER_FAMILY"
	RelTypePrimaryTenant      = "PRIMARY_TENANT"
	RelTypeSecondaryTenant    = "SECONDARY_TENANT"
	RelTypeTenantFamily       = "TENANT_FAMILY"
	RelTypeAuthorizedOccupant = "AUTHORIZED_OCCUPANT"
)

// Relationship statuses
const (
	RelStatusActive = "ACTIVE"
	RelStatusEnded  = "ENDED"
)

// Member and relationship audit actions
const (
	ActionMemberRegistered    = "MEMBER_REGISTERED"
	ActionMemberUpdated       = "MEMBER_UPDATED"
	ActionMemberStatusChanged = "MEMBER_STATUS_CHANGED"
	ActionMemberDeparted      = "MEMBER_DEPARTED"
	ActionUnitRelCreated      = "UNIT_RELATIONSHIP_CREATED"
	ActionUnitRelEnded        = "UNIT_RELATIONSHIP_ENDED"
)

// Departure reason codes
const (
	DepartureReasonTenantMovedOut       = "TENANT_MOVED_OUT"
	DepartureReasonLeaseExpired         = "LEASE_EXPIRED"
	DepartureReasonOwnershipTransferred = "OWNERSHIP_TRANSFERRED"
	DepartureReasonMemberRelocated      = "MEMBER_RELOCATED"
	DepartureReasonFamilyMemberDeparted = "FAMILY_MEMBER_DEPARTED"
	DepartureReasonMembershipTerminated = "MEMBERSHIP_TERMINATED"
	DepartureReasonDeceased             = "DECEASED"
	DepartureReasonOther                = "OTHER"
)

var validDepartureReasons = map[string]bool{
	DepartureReasonTenantMovedOut:       true,
	DepartureReasonLeaseExpired:         true,
	DepartureReasonOwnershipTransferred: true,
	DepartureReasonMemberRelocated:      true,
	DepartureReasonFamilyMemberDeparted: true,
	DepartureReasonMembershipTerminated: true,
	DepartureReasonDeceased:             true,
	DepartureReasonOther:                true,
}

// Domain errors
var (
	ErrMemberNotFound            = errors.New("member not found")
	ErrMemberAlreadyExists       = errors.New("active member with this phone number already exists")
	ErrUserAlreadyLinked         = errors.New("user is already linked to a member in this society")
	ErrInvalidPhone              = errors.New("invalid E.164 phone number")
	ErrInvalidRelationshipType   = errors.New("invalid relationship type")
	ErrInvalidOwnershipShare     = errors.New("invalid ownership share: must be between 0.01 and 100.00 for owners, and total must not exceed 100%")
	ErrOwnershipShareNotAllowed  = errors.New("ownership share is only permitted for owner relationships")
	ErrOccupancyRequired         = errors.New("authorized occupants must have is_occupant set to true")
	ErrRelationshipAlreadyExists = errors.New("active identical relationship already exists for this member on this unit")
	ErrRelationshipNotFound      = errors.New("unit relationship not found")
	ErrInvalidDates              = errors.New("end date must be after or equal to start date")
	ErrPreviewStale              = errors.New("active relationships have changed since departure preview was generated")
	ErrUnitNotBelongToSociety    = errors.New("unit does not belong to specified society")
	ErrMemberNotBelongToSociety  = errors.New("member does not belong to specified society")
	ErrDepartureWorkflowRequired = errors.New("member departure requires the two-phase departure workflow")
	ErrInvalidDepartureReason    = errors.New("invalid or unsupported departure reason code")
	ErrReasonDetailsRequired     = errors.New("reason details are required when departure reason is OTHER")
	ErrReasonDetailsTooLong      = errors.New("reason details must not exceed 500 characters")
)

// NormalizePhone normalizes phone string to pure digits 91XXXXXXXXXX.
func NormalizePhone(raw string) (string, error) {
	norm, err := auth.NormalizePhone(raw)
	if err != nil {
		return "", ErrInvalidPhone
	}
	return norm, nil
}

// MaskPhone returns the phone number (full visibility per user request).
func MaskPhone(phone string) string {
	return auth.MaskPhone(phone)
}

// MaskEmail masks email address: ra****@example.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	user := parts[0]
	domain := parts[1]
	if len(user) <= 2 {
		return fmt.Sprintf("%s****@%s", user, domain)
	}
	return fmt.Sprintf("%s****@%s", user[:2], domain)
}

// SocietyMember represents a person in a society's member directory.
type SocietyMember struct {
	ID                     uuid.UUID  `json:"id"`
	SocietyID              uuid.UUID  `json:"society_id"`
	UserID                 *uuid.UUID `json:"user_id,omitempty"`
	FirstName              string     `json:"first_name"`
	LastName               *string    `json:"last_name,omitempty"`
	Phone                  string     `json:"phone"`
	Email                  *string    `json:"email,omitempty"`
	Status                 string     `json:"status"`
	EmergencyContactName   *string    `json:"emergency_contact_name,omitempty"`
	EmergencyContactPhone  *string    `json:"emergency_contact_phone,omitempty"`
	DepartureReasonCode    *string    `json:"departure_reason_code,omitempty"`
	DepartureReasonDetails *string    `json:"departure_reason_details,omitempty"`
	DepartedAt             *time.Time `json:"departed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// SocietyMemberItem is the formatted list item with data minimization (masked PII).
type SocietyMemberItem struct {
	ID                  uuid.UUID  `json:"id"`
	SocietyID           uuid.UUID  `json:"society_id"`
	UserID              *uuid.UUID `json:"user_id,omitempty"`
	FirstName           string     `json:"first_name"`
	LastName            *string    `json:"last_name,omitempty"`
	Phone               string     `json:"phone"`           // masked
	Email               *string    `json:"email,omitempty"` // masked
	Status              string     `json:"status"`
	IsRegisteredUser    bool       `json:"is_registered_user"`
	ActiveUnitsCount    int        `json:"active_units_count"`
	UnitNumbers         string     `json:"unit_numbers,omitempty"`
	DepartureReasonCode *string    `json:"departure_reason_code,omitempty"`
	DepartedAt          *time.Time `json:"departed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// UnitRelationship represents an association between a member and a unit.
type UnitRelationship struct {
	ID               uuid.UUID  `json:"id"`
	SocietyID        uuid.UUID  `json:"society_id"`
	UnitID           uuid.UUID  `json:"unit_id"`
	MemberID         uuid.UUID  `json:"member_id"`
	RelationshipType string     `json:"relationship_type"`
	IsOccupant       bool       `json:"is_occupant"`
	OwnershipShare   *float64   `json:"ownership_share,omitempty"`
	StartDate        time.Time  `json:"start_date"`
	EndDate          *time.Time `json:"end_date,omitempty"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// UnitRelationshipDetail represents rich detail including unit and member data.
type UnitRelationshipDetail struct {
	ID               uuid.UUID `json:"id"`
	SocietyID        uuid.UUID `json:"society_id"`
	UnitID           uuid.UUID `json:"unit_id"`
	MemberID         uuid.UUID `json:"member_id"`
	MemberName       string    `json:"member_name"`
	MemberPhone      string    `json:"member_phone"` // masked or unmasked depending on context
	UnitIdentifier   string    `json:"unit_identifier"`
	BuildingName     string    `json:"building_name"`
	WingCode         *string   `json:"wing_code,omitempty"`
	RelationshipType string    `json:"relationship_type"`
	IsOccupant       bool      `json:"is_occupant"`
	OwnershipShare   *float64  `json:"ownership_share,omitempty"`
	StartDate        string    `json:"start_date"`
	EndDate          *string   `json:"end_date,omitempty"`
	Status           string    `json:"status"`
	Notes            *string   `json:"notes,omitempty"`
}

// PreservedOwnershipItem holds details of an ownership relationship preserved during departure.
type PreservedOwnershipItem struct {
	RelationshipID   uuid.UUID `json:"relationship_id"`
	UnitIdentifier   string    `json:"unit_identifier"`
	BuildingName     string    `json:"building_name"`
	RelationshipType string    `json:"relationship_type"`
	OwnershipShare   *float64  `json:"ownership_share,omitempty"`
	Action           string    `json:"action"`
	Note             string    `json:"note"`
}

// TargetedTerminationItem holds details of an occupancy/tenancy slated for termination.
type TargetedTerminationItem struct {
	RelationshipID   uuid.UUID `json:"relationship_id"`
	UnitIdentifier   string    `json:"unit_identifier"`
	BuildingName     string    `json:"building_name"`
	RelationshipType string    `json:"relationship_type"`
	Action           string    `json:"action"`
	ProposedEndDate  string    `json:"proposed_end_date"`
}

// DeparturePreview holds the two-phase departure preview manifest.
type DeparturePreview struct {
	MemberID                        uuid.UUID                 `json:"member_id"`
	CurrentStatus                   string                    `json:"current_status"`
	VersionHash                     string                    `json:"version_hash"`
	PreservedOwnershipRelationships []PreservedOwnershipItem  `json:"preserved_ownership_relationships"`
	TargetedTerminations            []TargetedTerminationItem `json:"targeted_terminations"`
}

// RegisterMemberParams holds arguments for creating a society member.
type RegisterMemberParams struct {
	FirstName             string
	LastName              *string
	Phone                 string
	Email                 *string
	EmergencyContactName  *string
	EmergencyContactPhone *string
	ClientIP              string
	UserAgent             string
}

// RegisterMember registers a new person in a society's member directory.
func (r *Repository) RegisterMember(ctx context.Context, societyID, actorID uuid.UUID, params RegisterMemberParams) (*SocietyMember, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	normPhone, err := NormalizePhone(params.Phone)
	if err != nil {
		return nil, err
	}

	trimmedFirst := strings.TrimSpace(params.FirstName)
	if len(trimmedFirst) < 2 {
		return nil, errors.New("first name must be at least 2 characters")
	}

	var trimmedLast *string
	if params.LastName != nil {
		val := strings.TrimSpace(*params.LastName)
		if val != "" {
			trimmedLast = &val
		}
	}

	var trimmedEmail *string
	if params.Email != nil {
		val := strings.TrimSpace(*params.Email)
		if val != "" {
			trimmedEmail = &val
		}
	}

	var normEmergencyPhone *string
	if params.EmergencyContactPhone != nil {
		raw := strings.TrimSpace(*params.EmergencyContactPhone)
		if raw != "" {
			ep, err := NormalizePhone(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid emergency contact phone: %w", err)
			}
			normEmergencyPhone = &ep
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if an existing verified user exists with this phone
	var linkedUserID *uuid.UUID
	var foundUserID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT id FROM users WHERE phone = $1 LIMIT 1", normPhone).Scan(&foundUserID)
	if err == nil {
		// Check if this user is already linked in this society
		var existsInSociety bool
		_ = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM society_members WHERE society_id = $1 AND user_id = $2)", societyID, foundUserID).Scan(&existsInSociety)
		if !existsInSociety {
			linkedUserID = &foundUserID
		}
	}

	member := &SocietyMember{
		SocietyID:             societyID,
		UserID:                linkedUserID,
		FirstName:             trimmedFirst,
		LastName:              trimmedLast,
		Phone:                 normPhone,
		Email:                 trimmedEmail,
		Status:                MemberStatusActive,
		EmergencyContactName:  params.EmergencyContactName,
		EmergencyContactPhone: normEmergencyPhone,
	}

	query := `
		INSERT INTO society_members (
			society_id, user_id, first_name, last_name, phone, email, status,
			emergency_contact_name, emergency_contact_phone, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		member.SocietyID, member.UserID, member.FirstName, member.LastName,
		member.Phone, member.Email, member.Status,
		member.EmergencyContactName, member.EmergencyContactPhone,
	).Scan(&member.ID, &member.CreatedAt, &member.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "uq_society_members_active_phone") {
				return nil, ErrMemberAlreadyExists
			}
			if strings.Contains(pgErr.ConstraintName, "uq_society_members_user_per_society") {
				return nil, ErrUserAlreadyLinked
			}
		}
		return nil, fmt.Errorf("failed to insert society member: %w", err)
	}

	// Transactional audit log
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"member_id":   member.ID,
		"society_id":  societyID,
		"first_name":  member.FirstName,
		"phone_mask":  MaskPhone(member.Phone),
		"user_linked": member.UserID != nil,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionMemberRegistered, string(detailsJSON), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit member registration: %w", err)
	}

	return member, nil
}

// ListMembers lists society members with filtering, pagination, and data minimization.
func (r *Repository) ListMembers(ctx context.Context, societyID uuid.UUID, queryStr, status string, limit, offset int) ([]SocietyMemberItem, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	baseWhere := "WHERE m.society_id = $1"
	args := []interface{}{societyID}
	argIdx := 2

	if status != "" {
		baseWhere += fmt.Sprintf(" AND m.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if queryStr != "" {
		cleaned := strings.TrimSpace(queryStr)
		baseWhere += fmt.Sprintf(" AND (m.first_name ILIKE $%d OR m.last_name ILIKE $%d OR m.phone ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+cleaned+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM society_members m %s", baseWhere)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count members: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT m.id, m.society_id, m.user_id, m.first_name, m.last_name, m.phone, m.email, m.status, m.created_at,
		       (SELECT COUNT(*) FROM unit_relationships ur WHERE ur.member_id = m.id AND ur.status = 'ACTIVE') as active_units_count,
		       COALESCE((
		           SELECT string_agg(su.unit_identifier, ', ' ORDER BY su.unit_identifier)
		           FROM unit_relationships ur
		           JOIN society_units su ON su.id = ur.unit_id
		           WHERE ur.member_id = m.id AND ur.status = 'ACTIVE'
		       ), '') as unit_numbers,
		       m.departure_reason_code, m.departed_at
		FROM society_members m
		%s
		ORDER BY m.first_name ASC, m.last_name ASC NULLS LAST
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query members: %w", err)
	}
	defer rows.Close()

	items := make([]SocietyMemberItem, 0)
	for rows.Next() {
		var item SocietyMemberItem
		var rawPhone string
		var rawEmail *string
		err := rows.Scan(
			&item.ID, &item.SocietyID, &item.UserID,
			&item.FirstName, &item.LastName, &rawPhone, &rawEmail,
			&item.Status, &item.CreatedAt, &item.ActiveUnitsCount,
			&item.UnitNumbers,
			&item.DepartureReasonCode, &item.DepartedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan member: %w", err)
		}

		item.Phone = MaskPhone(rawPhone)
		if rawEmail != nil {
			masked := MaskEmail(*rawEmail)
			item.Email = &masked
		}
		item.IsRegisteredUser = item.UserID != nil
		items = append(items, item)
	}

	return items, total, nil
}

// GetMemberByID retrieves full member details and unit relationships.
func (r *Repository) GetMemberByID(ctx context.Context, societyID, memberID uuid.UUID) (*SocietyMember, []UnitRelationshipDetail, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_id, user_id, first_name, last_name, phone, email, status,
		       emergency_contact_name, emergency_contact_phone, created_at, updated_at,
		       departure_reason_code, departure_reason_details, departed_at
		FROM society_members
		WHERE id = $1 AND society_id = $2
	`
	var m SocietyMember
	err := r.pool.QueryRow(ctx, query, memberID, societyID).Scan(
		&m.ID, &m.SocietyID, &m.UserID, &m.FirstName, &m.LastName,
		&m.Phone, &m.Email, &m.Status,
		&m.EmergencyContactName, &m.EmergencyContactPhone,
		&m.CreatedAt, &m.UpdatedAt,
		&m.DepartureReasonCode, &m.DepartureReasonDetails, &m.DepartedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrMemberNotFound
		}
		return nil, nil, fmt.Errorf("failed to query member: %w", err)
	}

	relsQuery := `
		SELECT ur.id, ur.society_id, ur.unit_id, ur.member_id,
		       CONCAT(m.first_name, ' ', COALESCE(m.last_name, '')) as member_name,
		       m.phone as member_phone,
		       u.unit_identifier, b.name as building_name, w.wing_code,
		       ur.relationship_type, ur.is_occupant, ur.ownership_share,
		       TO_CHAR(ur.start_date, 'YYYY-MM-DD') as start_date,
		       CASE WHEN ur.end_date IS NOT NULL THEN TO_CHAR(ur.end_date, 'YYYY-MM-DD') ELSE NULL END as end_date,
		       ur.status, ur.notes
		FROM unit_relationships ur
		JOIN society_units u ON ur.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_wings w ON u.wing_id = w.id
		JOIN society_members m ON ur.member_id = m.id
		WHERE ur.member_id = $1 AND ur.society_id = $2
		ORDER BY ur.status ASC, ur.start_date DESC
	`
	rows, err := r.pool.Query(ctx, relsQuery, memberID, societyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query unit relationships: %w", err)
	}
	defer rows.Close()

	relationships := make([]UnitRelationshipDetail, 0)
	for rows.Next() {
		var rd UnitRelationshipDetail
		err := rows.Scan(
			&rd.ID, &rd.SocietyID, &rd.UnitID, &rd.MemberID,
			&rd.MemberName, &rd.MemberPhone,
			&rd.UnitIdentifier, &rd.BuildingName, &rd.WingCode,
			&rd.RelationshipType, &rd.IsOccupant, &rd.OwnershipShare,
			&rd.StartDate, &rd.EndDate, &rd.Status, &rd.Notes,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan relationship detail: %w", err)
		}
		relationships = append(relationships, rd)
	}

	return &m, relationships, nil
}

// UpdateMember updates allowed member contact details and administrative status.
func (r *Repository) UpdateMember(ctx context.Context, societyID, memberID, actorID uuid.UUID, firstName string, lastName, email *string, status *string, emergencyName, emergencyPhone *string, clientIP, userAgent string) (*SocietyMember, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedFirst := strings.TrimSpace(firstName)
	if trimmedFirst == "" {
		return nil, errors.New("first name cannot be empty")
	}

	if status != nil && *status == MemberStatusLeft {
		return nil, ErrDepartureWorkflowRequired
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current SocietyMember
	err = tx.QueryRow(ctx, `
		SELECT id, society_id, status FROM society_members 
		WHERE id = $1 AND society_id = $2 FOR UPDATE
	`, memberID, societyID).Scan(&current.ID, &current.SocietyID, &current.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to lock member: %w", err)
	}

	newStatus := current.Status
	if status != nil && (*status == MemberStatusActive || *status == MemberStatusInactive) {
		newStatus = *status
	}

	var normEmergencyPhone *string
	if emergencyPhone != nil {
		raw := strings.TrimSpace(*emergencyPhone)
		if raw != "" {
			ep, err := NormalizePhone(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid emergency contact phone: %w", err)
			}
			normEmergencyPhone = &ep
		}
	}

	updateQuery := `
		UPDATE society_members
		SET first_name = $1, last_name = $2, email = $3, status = $4,
		    emergency_contact_name = $5, emergency_contact_phone = $6, updated_at = NOW()
		WHERE id = $7 AND society_id = $8
		RETURNING id, society_id, user_id, first_name, last_name, phone, email, status,
		          emergency_contact_name, emergency_contact_phone, created_at, updated_at
	`
	var updated SocietyMember
	err = tx.QueryRow(ctx, updateQuery,
		trimmedFirst, lastName, email, newStatus,
		emergencyName, normEmergencyPhone, memberID, societyID,
	).Scan(
		&updated.ID, &updated.SocietyID, &updated.UserID, &updated.FirstName, &updated.LastName,
		&updated.Phone, &updated.Email, &updated.Status,
		&updated.EmergencyContactName, &updated.EmergencyContactPhone,
		&updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update member: %w", err)
	}

	// Audit logging
	action := ActionMemberUpdated
	if current.Status != updated.Status {
		action = ActionMemberStatusChanged
	}

	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"member_id":  memberID,
		"society_id": societyID,
		"old_status": current.Status,
		"new_status": updated.Status,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, action, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit member update: %w", err)
	}

	return &updated, nil
}

// computeRelationshipsVersionHash computes a deterministic SHA-256 hash of active relationship IDs and timestamps.
func computeRelationshipsVersionHash(rels []UnitRelationship) string {
	if len(rels) == 0 {
		return "empty_0000000000000000"
	}

	// Sort by ID for deterministic hashing
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].ID.String() < rels[j].ID.String()
	})

	h := sha256.New()
	for _, r := range rels {
		fmt.Fprintf(h, "%s:%s:%s:%t:%d;", r.ID.String(), r.RelationshipType, r.Status, r.IsOccupant, r.UpdatedAt.UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil))
}

// PreviewMemberDeparture generates the phase 1 preview manifest and version hash.
func (r *Repository) PreviewMemberDeparture(ctx context.Context, societyID, memberID uuid.UUID) (*DeparturePreview, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var status string
	err := r.pool.QueryRow(ctx, "SELECT status FROM society_members WHERE id = $1 AND society_id = $2", memberID, societyID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to resolve member: %w", err)
	}

	query := `
		SELECT ur.id, ur.society_id, ur.unit_id, ur.member_id, ur.relationship_type,
		       ur.is_occupant, ur.ownership_share, ur.start_date, ur.end_date, ur.status,
		       ur.notes, ur.created_at, ur.updated_at,
		       u.unit_identifier, b.name as building_name
		FROM unit_relationships ur
		JOIN society_units u ON ur.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		WHERE ur.member_id = $1 AND ur.society_id = $2 AND ur.status = 'ACTIVE'
		ORDER BY ur.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, memberID, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active relationships: %w", err)
	}
	defer rows.Close()

	activeRels := make([]UnitRelationship, 0)
	preserved := make([]PreservedOwnershipItem, 0)
	targeted := make([]TargetedTerminationItem, 0)

	todayStr := time.Now().UTC().Format("2006-01-02")

	for rows.Next() {
		var ur UnitRelationship
		var unitIdent, bldName string
		err := rows.Scan(
			&ur.ID, &ur.SocietyID, &ur.UnitID, &ur.MemberID, &ur.RelationshipType,
			&ur.IsOccupant, &ur.OwnershipShare, &ur.StartDate, &ur.EndDate, &ur.Status,
			&ur.Notes, &ur.CreatedAt, &ur.UpdatedAt,
			&unitIdent, &bldName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan relationship: %w", err)
		}
		activeRels = append(activeRels, ur)

		if ur.RelationshipType == RelTypePrimaryOwner || ur.RelationshipType == RelTypeSecondaryOwner {
			preserved = append(preserved, PreservedOwnershipItem{
				RelationshipID:   ur.ID,
				UnitIdentifier:   unitIdent,
				BuildingName:     bldName,
				RelationshipType: ur.RelationshipType,
				OwnershipShare:   ur.OwnershipShare,
				Action:           "PRESERVED_AS_ABSENTEE_OWNER",
				Note:             "Ownership title remains active; is_occupant will be set to false if previously true.",
			})
		} else {
			targeted = append(targeted, TargetedTerminationItem{
				RelationshipID:   ur.ID,
				UnitIdentifier:   unitIdent,
				BuildingName:     bldName,
				RelationshipType: ur.RelationshipType,
				Action:           "TERMINATED",
				ProposedEndDate:  todayStr,
			})
		}
	}

	versionHash := computeRelationshipsVersionHash(activeRels)

	return &DeparturePreview{
		MemberID:                        memberID,
		CurrentStatus:                   status,
		VersionHash:                     versionHash,
		PreservedOwnershipRelationships: preserved,
		TargetedTerminations:            targeted,
	}, nil
}

// ConfirmMemberDeparture executes the phase 2 atomic member departure with concurrency check.
func (r *Repository) ConfirmMemberDeparture(ctx context.Context, societyID, memberID, actorID uuid.UUID, expectedVersionHash string, endDate string, reasonCode string, reasonDetails *string, clientIP, userAgent string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	trimmedCode := strings.TrimSpace(reasonCode)
	if !validDepartureReasons[trimmedCode] {
		return ErrInvalidDepartureReason
	}

	var trimmedDetails *string
	if reasonDetails != nil {
		trimmed := strings.TrimSpace(*reasonDetails)
		if trimmed != "" {
			if len(trimmed) > 500 {
				return ErrReasonDetailsTooLong
			}
			trimmedDetails = &trimmed
		}
	}

	if trimmedCode == DepartureReasonOther {
		if trimmedDetails == nil || *trimmedDetails == "" {
			return ErrReasonDetailsRequired
		}
	}

	parsedEndDate, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return errors.New("invalid end date format: must be YYYY-MM-DD")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock member row
	var currentStatus string
	err = tx.QueryRow(ctx, "SELECT status FROM society_members WHERE id = $1 AND society_id = $2 FOR UPDATE", memberID, societyID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMemberNotFound
		}
		return fmt.Errorf("failed to lock member: %w", err)
	}

	// Query active relationships with lock
	query := `
		SELECT id, society_id, unit_id, member_id, relationship_type,
		       is_occupant, ownership_share, start_date, end_date, status,
		       notes, created_at, updated_at
		FROM unit_relationships
		WHERE member_id = $1 AND society_id = $2 AND status = 'ACTIVE'
		FOR UPDATE
	`
	rows, err := tx.Query(ctx, query, memberID, societyID)
	if err != nil {
		return fmt.Errorf("failed to lock active relationships: %w", err)
	}

	activeRels := make([]UnitRelationship, 0)
	for rows.Next() {
		var ur UnitRelationship
		err := rows.Scan(
			&ur.ID, &ur.SocietyID, &ur.UnitID, &ur.MemberID, &ur.RelationshipType,
			&ur.IsOccupant, &ur.OwnershipShare, &ur.StartDate, &ur.EndDate, &ur.Status,
			&ur.Notes, &ur.CreatedAt, &ur.UpdatedAt,
		)
		if err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan relationship: %w", err)
		}
		activeRels = append(activeRels, ur)
	}
	rows.Close()

	// Verify version hash to detect concurrent edits between preview and confirm
	currentHash := computeRelationshipsVersionHash(activeRels)
	if currentHash != expectedVersionHash {
		return ErrPreviewStale
	}

	departedAt := time.Now().UTC()

	// 1. Update member status to LEFT, persisting structured departure reason
	_, err = tx.Exec(ctx, `
		UPDATE society_members
		SET status = 'LEFT',
		    departure_reason_code = $1,
		    departure_reason_details = $2,
		    departed_at = $3,
		    updated_at = NOW()
		WHERE id = $4 AND society_id = $5
	`, trimmedCode, trimmedDetails, departedAt, memberID, societyID)
	if err != nil {
		return fmt.Errorf("failed to update member status to LEFT: %w", err)
	}

	terminatedCount := 0
	preservedCount := 0

	departureNote := "Ended on member departure: " + trimmedCode
	if trimmedDetails != nil && *trimmedDetails != "" {
		departureNote += " (" + *trimmedDetails + ")"
	}

	// 2. Process active relationships
	for _, ur := range activeRels {
		if ur.RelationshipType == RelTypePrimaryOwner || ur.RelationshipType == RelTypeSecondaryOwner {
			// Preserve ownership title, set is_occupant = false
			if ur.IsOccupant {
				_, err = tx.Exec(ctx, "UPDATE unit_relationships SET is_occupant = false, updated_at = NOW() WHERE id = $1", ur.ID)
				if err != nil {
					return fmt.Errorf("failed to update owner occupancy: %w", err)
				}
			}
			preservedCount++
		} else {
			// Terminate tenancy / family occupancy
			if parsedEndDate.Before(ur.StartDate) {
				return ErrInvalidDates
			}
			_, err = tx.Exec(ctx, `
				UPDATE unit_relationships
				SET status = 'ENDED', end_date = $1, notes = COALESCE(notes || ' | ', '') || $2, updated_at = NOW()
				WHERE id = $3
			`, parsedEndDate, departureNote, ur.ID)
			if err != nil {
				return fmt.Errorf("failed to end unit relationship: %w", err)
			}

			// Individual audit log per terminated occupancy
			termAudit := map[string]interface{}{
				"relationship_id":   ur.ID,
				"unit_id":           ur.UnitID,
				"member_id":         memberID,
				"relationship_type": ur.RelationshipType,
				"end_date":          endDate,
				"reason_code":       trimmedCode,
			}
			if trimmedDetails != nil {
				termAudit["reason_details"] = *trimmedDetails
			}
			detailsJSON, _ := json.Marshal(termAudit)
			_, err = tx.Exec(ctx, `
				INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
				VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
			`, actorID, ActionUnitRelEnded, string(detailsJSON), clientIP, userAgent)
			if err != nil {
				return fmt.Errorf("failed to log unit relationship end audit: %w", err)
			}

			terminatedCount++
		}
	}

	// 3. Conclude active committee appointments for departed member
	aptRows, err := tx.Query(ctx, `
		SELECT id, designation_id FROM society_committee_appointments
		WHERE member_id = $1 AND society_id = $2 AND status = 'ACTIVE'
		FOR UPDATE
	`, memberID, societyID)
	if err == nil {
		type termApt struct {
			id            uuid.UUID
			designationID uuid.UUID
		}
		var aptsToTerm []termApt
		for aptRows.Next() {
			var a termApt
			if scanErr := aptRows.Scan(&a.id, &a.designationID); scanErr == nil {
				aptsToTerm = append(aptsToTerm, a)
			}
		}
		aptRows.Close()

		for _, ta := range aptsToTerm {
			_, err = tx.Exec(ctx, `
				UPDATE society_committee_appointments
				SET status = 'ENDED', end_date = $1, notes = COALESCE(notes || ' | ', '') || $2, updated_at = NOW()
				WHERE id = $3
			`, parsedEndDate, departureNote, ta.id)
			if err != nil {
				return fmt.Errorf("failed to end committee appointment: %w", err)
			}

			aptAuditJSON, _ := json.Marshal(map[string]interface{}{
				"appointment_id": ta.id,
				"society_id":     societyID,
				"designation_id": ta.designationID,
				"member_id":      memberID,
				"end_date":       endDate,
				"action":         "ENDED",
			})
			_, err = tx.Exec(ctx, `
				INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
				VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
			`, actorID, ActionCommitteeAppointmentEnded, string(aptAuditJSON), clientIP, userAgent)
			if err != nil {
				return fmt.Errorf("failed to log appointment termination audit: %w", err)
			}
		}
	}

	// 4. Emit member departure audit log with structured reasons
	auditPayload := map[string]interface{}{
		"member_id":             memberID,
		"society_id":            societyID,
		"reason_code":           trimmedCode,
		"end_date":              endDate,
		"terminated_count":      terminatedCount,
		"preserved_owner_count": preservedCount,
	}
	if trimmedDetails != nil {
		auditPayload["reason_details"] = *trimmedDetails
	}
	detailsJSON, _ := json.Marshal(auditPayload)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionMemberDeparted, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to log member departure audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit member departure: %w", err)
	}

	return nil
}

// CreateUnitRelationshipParams holds parameters for creating a unit relationship.
type CreateUnitRelationshipParams struct {
	MemberID         uuid.UUID
	RelationshipType string
	IsOccupant       bool
	OwnershipShare   *float64
	StartDate        string
	Notes            *string
	ClientIP         string
	UserAgent        string
}

// CreateUnitRelationship associates a member with a unit with pessimistic unit row locking.
func (r *Repository) CreateUnitRelationship(ctx context.Context, societyID, unitID, actorID uuid.UUID, params CreateUnitRelationshipParams) (*UnitRelationship, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Validate relationship type
	switch params.RelationshipType {
	case RelTypePrimaryOwner, RelTypeSecondaryOwner, RelTypeOwnerFamily,
		RelTypePrimaryTenant, RelTypeSecondaryTenant, RelTypeTenantFamily,
		RelTypeAuthorizedOccupant:
		// Valid
	default:
		return nil, ErrInvalidRelationshipType
	}

	// Authorized occupant must be an occupant
	if params.RelationshipType == RelTypeAuthorizedOccupant && !params.IsOccupant {
		return nil, ErrOccupancyRequired
	}

	// Only owners can have ownership shares
	isOwner := params.RelationshipType == RelTypePrimaryOwner || params.RelationshipType == RelTypeSecondaryOwner
	if !isOwner && params.OwnershipShare != nil {
		return nil, ErrOwnershipShareNotAllowed
	}

	if isOwner && params.OwnershipShare != nil {
		if *params.OwnershipShare <= 0 || *params.OwnershipShare > 100.0 {
			return nil, ErrInvalidOwnershipShare
		}
	}

	parsedStartDate, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		parsedStartDate = time.Now().UTC()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Pessimistic unit row lock to serialize concurrent share writes on the same unit
	var foundUnitID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM society_units 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, unitID, societyID).Scan(&foundUnitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUnitNotFound
		}
		return nil, fmt.Errorf("failed to lock unit row: %w", err)
	}

	// Step 2: Validate member exists in same society
	var memberStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM society_members 
		WHERE id = $1 AND society_id = $2
	`, params.MemberID, societyID).Scan(&memberStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, fmt.Errorf("failed to verify member: %w", err)
	}

	// Step 3: Check aggregate ownership share under lock
	if isOwner && params.OwnershipShare != nil {
		var currentSum float64
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(ownership_share), 0)
			FROM unit_relationships
			WHERE unit_id = $1 AND society_id = $2 AND status = 'ACTIVE'
			  AND relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER')
		`, unitID, societyID).Scan(&currentSum)
		if err != nil {
			return nil, fmt.Errorf("failed to compute existing ownership sum: %w", err)
		}

		if (currentSum + *params.OwnershipShare) > 100.0001 {
			return nil, fmt.Errorf("%w: current total is %.2f%%, adding %.2f%% exceeds 100%%",
				ErrInvalidOwnershipShare, currentSum, *params.OwnershipShare)
		}
	}

	// Step 4: Insert unit relationship
	rel := &UnitRelationship{
		SocietyID:        societyID,
		UnitID:           unitID,
		MemberID:         params.MemberID,
		RelationshipType: params.RelationshipType,
		IsOccupant:       params.IsOccupant,
		OwnershipShare:   params.OwnershipShare,
		StartDate:        parsedStartDate,
		Status:           RelStatusActive,
		Notes:            params.Notes,
	}

	insertQuery := `
		INSERT INTO unit_relationships (
			society_id, unit_id, member_id, relationship_type, is_occupant,
			ownership_share, start_date, status, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertQuery,
		rel.SocietyID, rel.UnitID, rel.MemberID, rel.RelationshipType,
		rel.IsOccupant, rel.OwnershipShare, rel.StartDate, rel.Status, rel.Notes,
	).Scan(&rel.ID, &rel.CreatedAt, &rel.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "uq_unit_rel_active_duplicate") {
				return nil, ErrRelationshipAlreadyExists
			}
		}
		return nil, fmt.Errorf("failed to insert unit relationship: %w", err)
	}

	// Step 5: Transactional audit log
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"relationship_id":   rel.ID,
		"society_id":        societyID,
		"unit_id":           unitID,
		"member_id":         params.MemberID,
		"relationship_type": rel.RelationshipType,
		"is_occupant":       rel.IsOccupant,
		"ownership_share":   rel.OwnershipShare,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionUnitRelCreated, string(detailsJSON), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit unit relationship creation: %w", err)
	}

	return rel, nil
}

// EndUnitRelationship concludes an active relationship with an explicit end date.
func (r *Repository) EndUnitRelationship(ctx context.Context, societyID, unitID, relID, actorID uuid.UUID, endDate string, notes *string, clientIP, userAgent string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	parsedEndDate, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		parsedEndDate = time.Now().UTC()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var rel UnitRelationship
	query := `
		SELECT id, society_id, unit_id, member_id, relationship_type, start_date, status
		FROM unit_relationships
		WHERE id = $1 AND unit_id = $2 AND society_id = $3 FOR UPDATE
	`
	err = tx.QueryRow(ctx, query, relID, unitID, societyID).Scan(
		&rel.ID, &rel.SocietyID, &rel.UnitID, &rel.MemberID,
		&rel.RelationshipType, &rel.StartDate, &rel.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRelationshipNotFound
		}
		return fmt.Errorf("failed to lock unit relationship: %w", err)
	}

	if parsedEndDate.Before(rel.StartDate) {
		return ErrInvalidDates
	}

	updateQuery := `
		UPDATE unit_relationships
		SET status = 'ENDED', end_date = $1, notes = COALESCE($2, notes), updated_at = NOW()
		WHERE id = $3 AND unit_id = $4 AND society_id = $5
	`
	_, err = tx.Exec(ctx, updateQuery, parsedEndDate, notes, relID, unitID, societyID)
	if err != nil {
		return fmt.Errorf("failed to update unit relationship status: %w", err)
	}

	// Transactional audit log
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"relationship_id":   rel.ID,
		"society_id":        societyID,
		"unit_id":           unitID,
		"member_id":         rel.MemberID,
		"relationship_type": rel.RelationshipType,
		"end_date":          parsedEndDate.Format("2006-01-02"),
		"notes":             notes,
	})
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionUnitRelEnded, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit unit relationship termination: %w", err)
	}

	return nil
}

// ListUnitRelationships lists owners, tenants, and occupants for a unit.
func (r *Repository) ListUnitRelationships(ctx context.Context, societyID, unitID uuid.UUID, status string) ([]UnitRelationshipDetail, float64, bool, error) {
	if r.pool == nil {
		return nil, 0, false, errors.New("database pool unavailable")
	}

	// Verify unit exists in society
	var unitIdent string
	err := r.pool.QueryRow(ctx, "SELECT unit_identifier FROM society_units WHERE id = $1 AND society_id = $2", unitID, societyID).Scan(&unitIdent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, false, ErrUnitNotFound
		}
		return nil, 0, false, fmt.Errorf("failed to verify unit: %w", err)
	}

	baseWhere := "WHERE ur.unit_id = $1 AND ur.society_id = $2"
	args := []interface{}{unitID, societyID}
	if status != "" {
		baseWhere += " AND ur.status = $3"
		args = append(args, status)
	}

	query := fmt.Sprintf(`
		SELECT ur.id, ur.society_id, ur.unit_id, ur.member_id,
		       CONCAT(m.first_name, ' ', COALESCE(m.last_name, '')) as member_name,
		       m.phone as member_phone,
		       u.unit_identifier, b.name as building_name, w.wing_code,
		       ur.relationship_type, ur.is_occupant, ur.ownership_share,
		       TO_CHAR(ur.start_date, 'YYYY-MM-DD') as start_date,
		       CASE WHEN ur.end_date IS NOT NULL THEN TO_CHAR(ur.end_date, 'YYYY-MM-DD') ELSE NULL END as end_date,
		       ur.status, ur.notes
		FROM unit_relationships ur
		JOIN society_units u ON ur.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_wings w ON u.wing_id = w.id
		JOIN society_members m ON ur.member_id = m.id
		%s
		ORDER BY ur.status ASC, ur.start_date DESC
	`, baseWhere)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, false, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	details := make([]UnitRelationshipDetail, 0)
	var totalRecordedShare float64
	activeOccupantCount := 0

	for rows.Next() {
		var rd UnitRelationshipDetail
		var rawPhone string
		err := rows.Scan(
			&rd.ID, &rd.SocietyID, &rd.UnitID, &rd.MemberID,
			&rd.MemberName, &rawPhone,
			&rd.UnitIdentifier, &rd.BuildingName, &rd.WingCode,
			&rd.RelationshipType, &rd.IsOccupant, &rd.OwnershipShare,
			&rd.StartDate, &rd.EndDate, &rd.Status, &rd.Notes,
		)
		if err != nil {
			return nil, 0, false, fmt.Errorf("failed to scan relationship: %w", err)
		}

		rd.MemberPhone = MaskPhone(rawPhone)

		if rd.Status == RelStatusActive {
			if rd.OwnershipShare != nil {
				totalRecordedShare += *rd.OwnershipShare
			}
			if rd.IsOccupant {
				activeOccupantCount++
			}
		}

		details = append(details, rd)
	}

	isVacant := (activeOccupantCount == 0)

	return details, totalRecordedShare, isVacant, nil
}

// SafeLinkUserMembers safely links a verified user to their pre-registered member records across societies.
func (r *Repository) SafeLinkUserMembers(ctx context.Context, userID uuid.UUID, normalizedPhone string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Query candidate unlinked members for this phone under lock
	rows, err := tx.Query(ctx, `
		SELECT id, society_id 
		FROM society_members 
		WHERE phone = $1 AND user_id IS NULL AND status = 'ACTIVE' 
		FOR UPDATE
	`, normalizedPhone)
	if err != nil {
		return fmt.Errorf("failed to query candidate member records: %w", err)
	}

	type cand struct {
		id        uuid.UUID
		societyID uuid.UUID
	}
	cands := make([]cand, 0)
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.societyID); err == nil {
			cands = append(cands, c)
		}
	}
	rows.Close()

	if len(cands) == 0 {
		return nil
	}

	// Group by society
	bySoc := make(map[uuid.UUID][]uuid.UUID)
	for _, c := range cands {
		bySoc[c.societyID] = append(bySoc[c.societyID], c.id)
	}

	for socID, memberIDs := range bySoc {
		// Ambiguity check: if multiple candidate records match in the same society, fail closed
		if len(memberIDs) > 1 {
			continue // Skip linking for this society; administrator review required
		}

		// Check if user is already linked in this society
		var alreadyLinked bool
		_ = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM society_members WHERE society_id = $1 AND user_id = $2)", socID, userID).Scan(&alreadyLinked)
		if alreadyLinked {
			continue
		}

		// Safely link
		targetMemberID := memberIDs[0]
		_, err = tx.Exec(ctx, "UPDATE society_members SET user_id = $1, updated_at = NOW() WHERE id = $2 AND society_id = $3", userID, targetMemberID, socID)
		if err != nil {
			return fmt.Errorf("failed to link member identity: %w", err)
		}
	}

	return tx.Commit(ctx)
}
