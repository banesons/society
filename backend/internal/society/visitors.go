package society

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chs/backend/internal/auth"
)

// S09 Visitor domain errors
var (
	ErrVisitNotFound          = errors.New("visit not found")
	ErrInvalidVisitState      = errors.New("invalid visit state for requested operation")
	ErrVisitAlreadyCheckedIn  = errors.New("visit already checked in")
	ErrVisitNotCheckedIn      = errors.New("visit is not currently checked in")
	ErrVisitAlreadyCheckedOut = errors.New("visit already checked out")
	ErrPassNotFound           = errors.New("visitor pass not found")
	ErrPassExpired            = errors.New("visitor pass has expired")
	ErrPassRevoked            = errors.New("visitor pass has been revoked")
	ErrPassAlreadyConsumed    = errors.New("visitor pass has already been used")
	ErrGateDirectionMismatch  = errors.New("gate direction does not permit this movement")
	ErrDutySessionRequired    = errors.New("gatekeeper must hold an active duty session at target gate")
	ErrUnauthorizedResident   = errors.New("user is not an active occupant of the target unit")
	ErrInvalidPhoneFormat     = errors.New("invalid phone number format")
)

// S09 Audit Log Actions
const (
	ActionVisitRegistered          = "VISIT_REGISTERED"
	ActionWalkInRegistered         = "WALK_IN_REGISTERED"
	ActionVisitApproved            = "VISIT_APPROVED"
	ActionVisitRejected            = "VISIT_REJECTED"
	ActionVisitCancelled           = "VISIT_CANCELLED"
	ActionVisitorPassIssued        = "VISITOR_PASS_ISSUED"
	ActionVisitorPassRevoked       = "VISITOR_PASS_REVOKED"
	ActionVisitorCheckedIn         = "VISITOR_CHECKED_IN"
	ActionVisitorCheckedOut        = "VISITOR_CHECKED_OUT"
	ActionUnitVisitorPolicyUpdated = "UNIT_VISITOR_POLICY_UPDATED"
)

// ResidentUnit represents a unit accessible to an authorized resident member.
type ResidentUnit struct {
	ID                      uuid.UUID  `json:"id"`
	SocietyID               uuid.UUID  `json:"society_id"`
	BuildingID              uuid.UUID  `json:"building_id"`
	BuildingName            string     `json:"building_name"`
	WingID                  *uuid.UUID `json:"wing_id,omitempty"`
	WingCode                *string    `json:"wing_code,omitempty"`
	FloorNumber             int        `json:"floor_number"`
	UnitNumber              string     `json:"unit_number"`
	UnitIdentifier          string     `json:"unit_identifier"`
	UnitType                string     `json:"unit_type"`
	RelationshipType        string     `json:"relationship_type"`
	IsOccupant              bool       `json:"is_occupant"`
	RequiresVisitorApproval bool       `json:"requires_visitor_approval"`
}

// Visit represents a visitor's request and physical visit record.
type Visit struct {
	ID                 uuid.UUID  `json:"id"`
	SocietyID          uuid.UUID  `json:"society_id"`
	UnitID             uuid.UUID  `json:"unit_id"`
	UnitIdentifier     string     `json:"unit_identifier"`
	BuildingName       string     `json:"building_name"`
	ResidentID         *uuid.UUID `json:"resident_id,omitempty"`
	ResidentName       *string    `json:"resident_name,omitempty"`
	VisitorName        string     `json:"visitor_name"`
	VisitorPhone       string     `json:"visitor_phone"`
	VisitorPhoneMasked string     `json:"visitor_phone_masked"`
	Purpose            string     `json:"purpose"`
	VisitType          string     `json:"visit_type"` // EXPECTED, WALK_IN
	Status             string     `json:"status"`     // PENDING_APPROVAL, APPROVED, REJECTED, CHECKED_IN, CHECKED_OUT, CANCELLED, EXPIRED
	VehicleNumber      *string    `json:"vehicle_number,omitempty"`
	ExpectedArrival    *time.Time `json:"expected_arrival,omitempty"`
	ValidUntil         *time.Time `json:"valid_until,omitempty"`
	CheckInAt          *time.Time `json:"check_in_at,omitempty"`
	CheckOutAt         *time.Time `json:"check_out_at,omitempty"`
	EntryGateID        *uuid.UUID `json:"entry_gate_id,omitempty"`
	EntryGateName      *string    `json:"entry_gate_name,omitempty"`
	ExitGateID         *uuid.UUID `json:"exit_gate_id,omitempty"`
	ExitGateName       *string    `json:"exit_gate_name,omitempty"`
	RejectionReason    *string    `json:"rejection_reason,omitempty"`
	CreatedBy          uuid.UUID  `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// VisitorPass represents a verification pass for a visit.
type VisitorPass struct {
	ID          uuid.UUID  `json:"id"`
	SocietyID   uuid.UUID  `json:"society_id"`
	VisitID     uuid.UUID  `json:"visit_id"`
	PassCode    string     `json:"pass_code"`
	ValidFrom   time.Time  `json:"valid_from"`
	ValidUntil  time.Time  `json:"valid_until"`
	Status      string     `json:"status"` // ACTIVE, CONSUMED, REVOKED, EXPIRED
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   *uuid.UUID `json:"revoked_by,omitempty"`
	BearerToken string     `json:"bearer_token,omitempty"` // populated only on issuance
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// VisitApproval represents an audit record for an approval decision.
type VisitApproval struct {
	ID            uuid.UUID `json:"id"`
	SocietyID     uuid.UUID `json:"society_id"`
	VisitID       uuid.UUID `json:"visit_id"`
	Decision      string    `json:"decision"` // APPROVED, REJECTED
	DecidedBy     uuid.UUID `json:"decided_by"`
	DecidedByName string    `json:"decided_by_name,omitempty"`
	Notes         *string   `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// GateEvent represents an append-only entry or exit perimeter movement.
type GateEvent struct {
	ID            uuid.UUID  `json:"id"`
	SocietyID     uuid.UUID  `json:"society_id"`
	VisitID       uuid.UUID  `json:"visit_id"`
	GateID        uuid.UUID  `json:"gate_id"`
	GateName      string     `json:"gate_name"`
	SessionID     *uuid.UUID `json:"session_id,omitempty"`
	GatekeeperID  uuid.UUID  `json:"gatekeeper_id"`
	EventType     string     `json:"event_type"` // ENTRY, EXIT
	Direction     string     `json:"direction"`  // INGRESS, EGRESS
	VehicleNumber *string    `json:"vehicle_number,omitempty"`
	Notes         *string    `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// VisitorDashboardMetrics provides summary metrics.
type VisitorDashboardMetrics struct {
	ExpectedToday    int `json:"expected_today"`
	AwaitingApproval int `json:"awaiting_approval"`
	CurrentlyInside  int `json:"currently_inside"`
	CompletedToday   int `json:"completed_today"`
}

// RegisterExpectedVisitParams contains input parameters for an expected visitor.
type RegisterExpectedVisitParams struct {
	UnitID           uuid.UUID  `json:"unit_id"`
	VisitorName      string     `json:"visitor_name"`
	VisitorPhone     string     `json:"visitor_phone"`
	Purpose          string     `json:"purpose"`
	VehicleNumber    *string    `json:"vehicle_number,omitempty"`
	ExpectedArrival  *time.Time `json:"expected_arrival,omitempty"`
	ValidUntil       *time.Time `json:"valid_until,omitempty"`
	RequiresApproval bool       `json:"requires_approval"`
}

// RegisterWalkInVisitParams contains input parameters for a walk-in visitor.
type RegisterWalkInVisitParams struct {
	UnitID           uuid.UUID `json:"unit_id"`
	VisitorName      string    `json:"visitor_name"`
	VisitorPhone     string    `json:"visitor_phone"`
	Purpose          string    `json:"purpose"`
	VehicleNumber    *string   `json:"vehicle_number,omitempty"`
	RequiresApproval bool      `json:"requires_approval"`
}

// VisitFilter represents search & filter criteria.
type VisitFilter struct {
	Status    string
	VisitType string
	Search    string
	UnitID    *uuid.UUID
	Limit     int
	Offset    int
}

// Helper to generate a secure random pass code and token
func generatePassCredentials() (passCode string, bearerToken string, tokenHash string, err error) {
	// Short pass code: GP-XXXXXX
	const chars = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ" // exclude 0, 1, I, O for clarity
	var codeBuilder strings.Builder
	codeBuilder.WriteString("GP-")
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", "", "", err
		}
		codeBuilder.WriteByte(chars[num.Int64()])
	}
	passCode = codeBuilder.String()

	// High entropy bearer token: chs_pass_<32 random bytes as hex>
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", err
	}
	bearerToken = "chs_pass_" + hex.EncodeToString(tokenBytes)

	// SHA-256 hash
	hash := sha256.Sum256([]byte(bearerToken))
	tokenHash = hex.EncodeToString(hash[:])

	return passCode, bearerToken, tokenHash, nil
}

// RegisterExpectedVisit registers an expected visit and issues an optional initial pass.
func (r *Repository) RegisterExpectedVisit(
	ctx context.Context,
	societyID uuid.UUID,
	params RegisterExpectedVisitParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, *VisitorPass, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	normPhone, err := auth.NormalizePhone(params.VisitorPhone)
	if err != nil {
		return nil, nil, ErrInvalidPhoneFormat
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Validate target unit exists in target society
	var unitExists bool
	var unitIdentifier, buildingName string
	err = tx.QueryRow(ctx, `
		SELECT true, u.unit_identifier, b.name 
		FROM society_units u
		JOIN society_buildings b ON u.building_id = b.id
		WHERE u.id = $1 AND u.society_id = $2
	`, params.UnitID, societyID).Scan(&unitExists, &unitIdentifier, &buildingName)
	if err != nil || !unitExists {
		return nil, nil, ErrUnitNotFound
	}

	initialStatus := "APPROVED"
	if params.RequiresApproval {
		initialStatus = "PENDING_APPROVAL"
	}

	now := time.Now().UTC()
	expectedArrival := now
	if params.ExpectedArrival != nil {
		expectedArrival = *params.ExpectedArrival
	}
	validUntil := expectedArrival.Add(24 * time.Hour)
	if params.ValidUntil != nil {
		validUntil = *params.ValidUntil
	}

	visitID := uuid.New()
	var visit Visit
	err = tx.QueryRow(ctx, `
		INSERT INTO society_visits (
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'EXPECTED', $7, $8, $9, $10, $11, $12, $12
		)
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, visitID, societyID, params.UnitID, strings.TrimSpace(params.VisitorName), normPhone,
		strings.TrimSpace(params.Purpose), initialStatus, params.VehicleNumber, expectedArrival,
		validUntil, actorID, now).Scan(
		&visit.ID, &visit.SocietyID, &visit.UnitID, &visit.VisitorName, &visit.VisitorPhone,
		&visit.Purpose, &visit.VisitType, &visit.Status, &visit.VehicleNumber,
		&visit.ExpectedArrival, &visit.ValidUntil, &visit.CheckInAt, &visit.CheckOutAt,
		&visit.EntryGateID, &visit.ExitGateID, &visit.RejectionReason,
		&visit.CreatedBy, &visit.CreatedAt, &visit.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert visit: %w", err)
	}

	visit.UnitIdentifier = unitIdentifier
	visit.BuildingName = buildingName
	visit.VisitorPhoneMasked = auth.MaskPhone(visit.VisitorPhone)

	var pass *VisitorPass
	if initialStatus == "APPROVED" {
		passCode, bearerToken, tokenHash, err := generatePassCredentials()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate pass credentials: %w", err)
		}

		passID := uuid.New()
		pass = &VisitorPass{
			ID:          passID,
			SocietyID:   societyID,
			VisitID:     visitID,
			PassCode:    passCode,
			ValidFrom:   expectedArrival,
			ValidUntil:  validUntil,
			Status:      "ACTIVE",
			BearerToken: bearerToken,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO society_visitor_passes (
				id, society_id, visit_id, pass_code, token_hash, valid_from, valid_until, status, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $9
			)
		`, pass.ID, pass.SocietyID, pass.VisitID, pass.PassCode, tokenHash, pass.ValidFrom, pass.ValidUntil, pass.Status, now)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to insert visitor pass: %w", err)
		}
	}

	// Audit trail insertion in same transaction
	auditDetails := map[string]interface{}{
		"visit_id":      visit.ID.String(),
		"society_id":    societyID.String(),
		"unit_id":       params.UnitID.String(),
		"visitor_name":  visit.VisitorName,
		"visitor_phone": visit.VisitorPhoneMasked,
		"visit_type":    "EXPECTED",
		"status":        visit.Status,
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitRegistered, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &visit, pass, nil
}

// RegisterWalkInVisit registers a walk-in visitor at the gate.
func (r *Repository) RegisterWalkInVisit(
	ctx context.Context,
	societyID uuid.UUID,
	params RegisterWalkInVisitParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	normPhone, err := auth.NormalizePhone(params.VisitorPhone)
	if err != nil {
		return nil, ErrInvalidPhoneFormat
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var unitExists bool
	var unitIdentifier, buildingName string
	var requiresApproval bool
	err = tx.QueryRow(ctx, `
		SELECT true, u.unit_identifier, b.name, u.requires_visitor_approval
		FROM society_units u
		JOIN society_buildings b ON u.building_id = b.id
		WHERE u.id = $1 AND u.society_id = $2
	`, params.UnitID, societyID).Scan(&unitExists, &unitIdentifier, &buildingName, &requiresApproval)
	if err != nil || !unitExists {
		return nil, ErrUnitNotFound
	}

	initialStatus := "APPROVED"
	if requiresApproval {
		initialStatus = "PENDING_APPROVAL"
	}

	now := time.Now().UTC()
	validUntil := now.Add(12 * time.Hour)

	visitID := uuid.New()
	var visit Visit
	err = tx.QueryRow(ctx, `
		INSERT INTO society_visits (
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'WALK_IN', $7, $8, $9, $10, $11, $12, $12
		)
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, visitID, societyID, params.UnitID, strings.TrimSpace(params.VisitorName), normPhone,
		strings.TrimSpace(params.Purpose), initialStatus, params.VehicleNumber, now,
		validUntil, actorID, now).Scan(
		&visit.ID, &visit.SocietyID, &visit.UnitID, &visit.VisitorName, &visit.VisitorPhone,
		&visit.Purpose, &visit.VisitType, &visit.Status, &visit.VehicleNumber,
		&visit.ExpectedArrival, &visit.ValidUntil, &visit.CheckInAt, &visit.CheckOutAt,
		&visit.EntryGateID, &visit.ExitGateID, &visit.RejectionReason,
		&visit.CreatedBy, &visit.CreatedAt, &visit.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert walk-in visit: %w", err)
	}

	visit.UnitIdentifier = unitIdentifier
	visit.BuildingName = buildingName
	visit.VisitorPhoneMasked = auth.MaskPhone(visit.VisitorPhone)

	// If approval required, enqueue notification to unit occupants
	if initialStatus == "PENDING_APPROVAL" {
		occRows, qErr := tx.Query(ctx, `
			SELECT DISTINCT sm.user_id
			FROM unit_relationships ur
			JOIN society_members sm ON ur.member_id = sm.id
			WHERE ur.unit_id = $1 AND ur.society_id = $2
			  AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
			  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
		`, params.UnitID, societyID)
		if qErr == nil {
			for occRows.Next() {
				var residentUID uuid.UUID
				if err := occRows.Scan(&residentUID); err == nil {
					outboxPayload := map[string]interface{}{
						"recipient_user_id": residentUID.String(),
						"society_id":        societyID.String(),
						"visit_id":          visit.ID.String(),
						"visitor_name":      visit.VisitorName,
						"unit_number":       unitIdentifier,
						"gate_name":         "Security Gate",
					}
					_ = r.EnqueueNotificationOutbox(ctx, tx, "VISITOR_APPROVAL_REQUESTED", fmt.Sprintf("VISIT_REQ:%s:%s", visit.ID.String(), residentUID.String()), outboxPayload)
				}
			}
			occRows.Close()
		}
	}

	// Audit log
	auditDetails := map[string]interface{}{
		"visit_id":      visit.ID.String(),
		"society_id":    societyID.String(),
		"unit_id":       params.UnitID.String(),
		"visitor_name":  visit.VisitorName,
		"visitor_phone": visit.VisitorPhoneMasked,
		"visit_type":    "WALK_IN",
		"status":        visit.Status,
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionWalkInRegistered, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &visit, nil
}

// ListVisits retrieves visits matching criteria with bounded pagination.
func (r *Repository) ListVisits(ctx context.Context, societyID uuid.UUID, filter VisitFilter) ([]Visit, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, fmt.Sprintf("v.society_id = $%d", argIdx))
	args = append(args, societyID)
	argIdx++

	if filter.Status != "" && filter.Status != "ALL" {
		whereClauses = append(whereClauses, fmt.Sprintf("v.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.VisitType != "" && filter.VisitType != "ALL" {
		whereClauses = append(whereClauses, fmt.Sprintf("v.visit_type = $%d", argIdx))
		args = append(args, filter.VisitType)
		argIdx++
	}

	if filter.UnitID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("v.unit_id = $%d", argIdx))
		args = append(args, *filter.UnitID)
		argIdx++
	}

	if filter.Search != "" {
		searchTerm := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(v.visitor_name ILIKE $%d OR v.visitor_phone ILIKE $%d OR v.vehicle_number ILIKE $%d OR u.unit_identifier ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM society_visits v
		JOIN society_units u ON v.unit_id = u.id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count visits: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT 
			v.id, v.society_id, v.unit_id, u.unit_identifier, b.name AS building_name,
			v.resident_id, m.first_name || COALESCE(' ' || m.last_name, '') AS resident_name,
			v.visitor_name, v.visitor_phone, v.purpose, v.visit_type, v.status,
			v.vehicle_number, v.expected_arrival, v.valid_until, v.check_in_at, v.check_out_at,
			v.entry_gate_id, eg.name AS entry_gate_name,
			v.exit_gate_id, xg.name AS exit_gate_name,
			v.rejection_reason, v.created_by, v.created_at, v.updated_at
		FROM society_visits v
		JOIN society_units u ON v.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_members m ON v.resident_id = m.id
		LEFT JOIN society_gates eg ON v.entry_gate_id = eg.id
		LEFT JOIN society_gates xg ON v.exit_gate_id = xg.id
		WHERE %s
		ORDER BY v.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	argsWithPaging := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query visits: %w", err)
	}
	defer rows.Close()

	var visits []Visit
	for rows.Next() {
		var item Visit
		var resName, eGate, xGate *string
		err := rows.Scan(
			&item.ID, &item.SocietyID, &item.UnitID, &item.UnitIdentifier, &item.BuildingName,
			&item.ResidentID, &resName,
			&item.VisitorName, &item.VisitorPhone, &item.Purpose, &item.VisitType, &item.Status,
			&item.VehicleNumber, &item.ExpectedArrival, &item.ValidUntil, &item.CheckInAt, &item.CheckOutAt,
			&item.EntryGateID, &eGate,
			&item.ExitGateID, &xGate,
			&item.RejectionReason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan visit: %w", err)
		}
		item.ResidentName = resName
		item.EntryGateName = eGate
		item.ExitGateName = xGate
		item.VisitorPhoneMasked = auth.MaskPhone(item.VisitorPhone)
		visits = append(visits, item)
	}

	if visits == nil {
		visits = []Visit{}
	}

	return visits, totalCount, nil
}

// GetVisitDetails retrieves a visit along with associated passes, approvals, and events.
func (r *Repository) GetVisitDetails(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
) (*Visit, []VisitorPass, []VisitApproval, []GateEvent, error) {
	if r.pool == nil {
		return nil, nil, nil, nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT 
			v.id, v.society_id, v.unit_id, u.unit_identifier, b.name AS building_name,
			v.resident_id, m.first_name || COALESCE(' ' || m.last_name, '') AS resident_name,
			v.visitor_name, v.visitor_phone, v.purpose, v.visit_type, v.status,
			v.vehicle_number, v.expected_arrival, v.valid_until, v.check_in_at, v.check_out_at,
			v.entry_gate_id, eg.name AS entry_gate_name,
			v.exit_gate_id, xg.name AS exit_gate_name,
			v.rejection_reason, v.created_by, v.created_at, v.updated_at
		FROM society_visits v
		JOIN society_units u ON v.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_members m ON v.resident_id = m.id
		LEFT JOIN society_gates eg ON v.entry_gate_id = eg.id
		LEFT JOIN society_gates xg ON v.exit_gate_id = xg.id
		WHERE v.id = $1 AND v.society_id = $2
	`

	var item Visit
	var resName, eGate, xGate *string
	err := r.pool.QueryRow(ctx, query, visitID, societyID).Scan(
		&item.ID, &item.SocietyID, &item.UnitID, &item.UnitIdentifier, &item.BuildingName,
		&item.ResidentID, &resName,
		&item.VisitorName, &item.VisitorPhone, &item.Purpose, &item.VisitType, &item.Status,
		&item.VehicleNumber, &item.ExpectedArrival, &item.ValidUntil, &item.CheckInAt, &item.CheckOutAt,
		&item.EntryGateID, &eGate,
		&item.ExitGateID, &xGate,
		&item.RejectionReason, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil, ErrVisitNotFound
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to get visit: %w", err)
	}

	item.ResidentName = resName
	item.EntryGateName = eGate
	item.ExitGateName = xGate
	item.VisitorPhoneMasked = auth.MaskPhone(item.VisitorPhone)

	// Passes
	passRows, err := r.pool.Query(ctx, `
		SELECT id, society_id, visit_id, pass_code, valid_from, valid_until, status, consumed_at, revoked_at, revoked_by, created_at, updated_at
		FROM society_visitor_passes
		WHERE visit_id = $1 AND society_id = $2
		ORDER BY created_at DESC
	`, visitID, societyID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to query passes: %w", err)
	}
	defer passRows.Close()

	var passes []VisitorPass
	for passRows.Next() {
		var p VisitorPass
		_ = passRows.Scan(
			&p.ID, &p.SocietyID, &p.VisitID, &p.PassCode, &p.ValidFrom, &p.ValidUntil,
			&p.Status, &p.ConsumedAt, &p.RevokedAt, &p.RevokedBy, &p.CreatedAt, &p.UpdatedAt,
		)
		passes = append(passes, p)
	}
	if passes == nil {
		passes = []VisitorPass{}
	}

	// Approvals
	appRows, err := r.pool.Query(ctx, `
		SELECT a.id, a.society_id, a.visit_id, a.decision, a.decided_by, u.phone AS decided_by_phone, a.notes, a.created_at
		FROM society_visit_approvals a
		JOIN users u ON a.decided_by = u.id
		WHERE a.visit_id = $1 AND a.society_id = $2
		ORDER BY a.created_at DESC
	`, visitID, societyID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to query approvals: %w", err)
	}
	defer appRows.Close()

	var approvals []VisitApproval
	for appRows.Next() {
		var a VisitApproval
		var decPhone string
		_ = appRows.Scan(&a.ID, &a.SocietyID, &a.VisitID, &a.Decision, &a.DecidedBy, &decPhone, &a.Notes, &a.CreatedAt)
		a.DecidedByName = auth.MaskPhone(decPhone)
		approvals = append(approvals, a)
	}
	if approvals == nil {
		approvals = []VisitApproval{}
	}

	// Gate events
	evtRows, err := r.pool.Query(ctx, `
		SELECT e.id, e.society_id, e.visit_id, e.gate_id, g.name, e.session_id, e.gatekeeper_id, e.event_type, e.direction, e.vehicle_number, e.notes, e.created_at
		FROM society_gate_events e
		JOIN society_gates g ON e.gate_id = g.id
		WHERE e.visit_id = $1 AND e.society_id = $2
		ORDER BY e.created_at DESC
	`, visitID, societyID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to query gate events: %w", err)
	}
	defer evtRows.Close()

	var events []GateEvent
	for evtRows.Next() {
		var e GateEvent
		_ = evtRows.Scan(&e.ID, &e.SocietyID, &e.VisitID, &e.GateID, &e.GateName, &e.SessionID, &e.GatekeeperID, &e.EventType, &e.Direction, &e.VehicleNumber, &e.Notes, &e.CreatedAt)
		events = append(events, e)
	}
	if events == nil {
		events = []GateEvent{}
	}

	return &item, passes, approvals, events, nil
}

// ApproveVisit approves a pending visit.
func (r *Repository) ApproveVisit(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	notes *string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Row-level lock to ensure atomic state transition
	var currentStatus string
	var unitID uuid.UUID
	var visitorName, visitorPhone string
	err = tx.QueryRow(ctx, `
		SELECT status, unit_id, visitor_name, visitor_phone
		FROM society_visits 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &unitID, &visitorName, &visitorPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVisitNotFound
		}
		return nil, fmt.Errorf("failed to lock visit: %w", err)
	}

	if currentStatus != "PENDING_APPROVAL" {
		return nil, fmt.Errorf("%w: visit is in %s state, cannot approve", ErrInvalidVisitState, currentStatus)
	}

	now := time.Now().UTC()
	var updated Visit
	err = tx.QueryRow(ctx, `
		UPDATE society_visits
		SET status = 'APPROVED', updated_at = $1
		WHERE id = $2 AND society_id = $3
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, now, visitID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.UnitID, &updated.VisitorName, &updated.VisitorPhone,
		&updated.Purpose, &updated.VisitType, &updated.Status, &updated.VehicleNumber,
		&updated.ExpectedArrival, &updated.ValidUntil, &updated.CheckInAt, &updated.CheckOutAt,
		&updated.EntryGateID, &updated.ExitGateID, &updated.RejectionReason,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update visit: %w", err)
	}

	// Insert into society_visit_approvals
	_, err = tx.Exec(ctx, `
		INSERT INTO society_visit_approvals (
			id, society_id, visit_id, decision, decided_by, notes, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, 'APPROVED', $3, $4, $5
		)
	`, societyID, visitID, actorID, notes, now)
	if err != nil {
		return nil, fmt.Errorf("failed to record approval: %w", err)
	}

	// Audit trail
	auditDetails := map[string]interface{}{
		"visit_id":      visitID.String(),
		"society_id":    societyID.String(),
		"unit_id":       unitID.String(),
		"decision":      "APPROVED",
		"visitor_name":  visitorName,
		"visitor_phone": auth.MaskPhone(visitorPhone),
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitApproved, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	// Outbox notification for visit approval
	if updated.CreatedBy != uuid.Nil {
		outboxPayload := map[string]interface{}{
			"recipient_user_id": updated.CreatedBy.String(),
			"society_id":        societyID.String(),
			"visit_id":          visitID.String(),
			"visitor_name":      visitorName,
		}
		_ = r.EnqueueNotificationOutbox(ctx, tx, "VISIT_APPROVED", fmt.Sprintf("VISIT_APP:%s:%s", visitID.String(), updated.CreatedBy.String()), outboxPayload)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated.VisitorPhoneMasked = auth.MaskPhone(updated.VisitorPhone)
	return &updated, nil
}

// RejectVisit rejects a pending visit.
func (r *Repository) RejectVisit(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	reason *string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var unitID uuid.UUID
	var visitorName, visitorPhone string
	err = tx.QueryRow(ctx, `
		SELECT status, unit_id, visitor_name, visitor_phone
		FROM society_visits 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &unitID, &visitorName, &visitorPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVisitNotFound
		}
		return nil, fmt.Errorf("failed to lock visit: %w", err)
	}

	if currentStatus != "PENDING_APPROVAL" {
		return nil, fmt.Errorf("%w: visit is in %s state, cannot reject", ErrInvalidVisitState, currentStatus)
	}

	now := time.Now().UTC()
	var updated Visit
	err = tx.QueryRow(ctx, `
		UPDATE society_visits
		SET status = 'REJECTED', rejection_reason = $1, updated_at = $2
		WHERE id = $3 AND society_id = $4
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, reason, now, visitID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.UnitID, &updated.VisitorName, &updated.VisitorPhone,
		&updated.Purpose, &updated.VisitType, &updated.Status, &updated.VehicleNumber,
		&updated.ExpectedArrival, &updated.ValidUntil, &updated.CheckInAt, &updated.CheckOutAt,
		&updated.EntryGateID, &updated.ExitGateID, &updated.RejectionReason,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update visit: %w", err)
	}

	// Record in approvals table
	_, err = tx.Exec(ctx, `
		INSERT INTO society_visit_approvals (
			id, society_id, visit_id, decision, decided_by, notes, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, 'REJECTED', $3, $4, $5
		)
	`, societyID, visitID, actorID, reason, now)
	if err != nil {
		return nil, fmt.Errorf("failed to record rejection: %w", err)
	}

	// Audit trail
	auditDetails := map[string]interface{}{
		"visit_id":      visitID.String(),
		"society_id":    societyID.String(),
		"unit_id":       unitID.String(),
		"decision":      "REJECTED",
		"reason":        reason,
		"visitor_name":  visitorName,
		"visitor_phone": auth.MaskPhone(visitorPhone),
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitRejected, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	// Outbox notification for visit rejection
	if updated.CreatedBy != uuid.Nil {
		var reasonStr string
		if reason != nil {
			reasonStr = *reason
		}
		outboxPayload := map[string]interface{}{
			"recipient_user_id": updated.CreatedBy.String(),
			"society_id":        societyID.String(),
			"visit_id":          visitID.String(),
			"visitor_name":      visitorName,
			"rejection_reason":  reasonStr,
		}
		_ = r.EnqueueNotificationOutbox(ctx, tx, "VISIT_REJECTED", fmt.Sprintf("VISIT_REJ:%s:%s", visitID.String(), updated.CreatedBy.String()), outboxPayload)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated.VisitorPhoneMasked = auth.MaskPhone(updated.VisitorPhone)
	return &updated, nil
}

// CancelVisit cancels an unentered visit.
func (r *Repository) CancelVisit(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var unitID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT status, unit_id
		FROM society_visits 
		WHERE id = $1 AND society_id = $2 
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &unitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVisitNotFound
		}
		return nil, fmt.Errorf("failed to lock visit: %w", err)
	}

	if currentStatus != "PENDING_APPROVAL" && currentStatus != "APPROVED" {
		return nil, fmt.Errorf("%w: visit is in %s state, cannot cancel", ErrInvalidVisitState, currentStatus)
	}

	now := time.Now().UTC()
	var updated Visit
	err = tx.QueryRow(ctx, `
		UPDATE society_visits
		SET status = 'CANCELLED', updated_at = $1
		WHERE id = $2 AND society_id = $3
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, now, visitID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.UnitID, &updated.VisitorName, &updated.VisitorPhone,
		&updated.Purpose, &updated.VisitType, &updated.Status, &updated.VehicleNumber,
		&updated.ExpectedArrival, &updated.ValidUntil, &updated.CheckInAt, &updated.CheckOutAt,
		&updated.EntryGateID, &updated.ExitGateID, &updated.RejectionReason,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update visit: %w", err)
	}

	// Revoke active passes
	_, err = tx.Exec(ctx, `
		UPDATE society_visitor_passes
		SET status = 'REVOKED', revoked_at = $1, revoked_by = $2, updated_at = $1
		WHERE visit_id = $3 AND society_id = $4 AND status = 'ACTIVE'
	`, now, actorID, visitID, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke passes: %w", err)
	}

	// Audit trail
	auditDetails := map[string]interface{}{
		"visit_id":   visitID.String(),
		"society_id": societyID.String(),
		"unit_id":    unitID.String(),
		"status":     "CANCELLED",
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitCancelled, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated.VisitorPhoneMasked = auth.MaskPhone(updated.VisitorPhone)
	return &updated, nil
}

// IssueVisitorPass issues a pass token for an approved visit.
func (r *Repository) IssueVisitorPass(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	validFrom, validUntil time.Time,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*VisitorPass, string, error) {
	if r.pool == nil {
		return nil, "", errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM society_visits 
		WHERE id = $1 AND society_id = $2
	`, visitID, societyID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrVisitNotFound
		}
		return nil, "", fmt.Errorf("failed to check visit: %w", err)
	}

	if currentStatus != "APPROVED" {
		return nil, "", fmt.Errorf("%w: visit is not approved", ErrInvalidVisitState)
	}

	passCode, bearerToken, tokenHash, err := generatePassCredentials()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate pass credentials: %w", err)
	}

	now := time.Now().UTC()
	passID := uuid.New()
	pass := &VisitorPass{
		ID:          passID,
		SocietyID:   societyID,
		VisitID:     visitID,
		PassCode:    passCode,
		ValidFrom:   validFrom,
		ValidUntil:  validUntil,
		Status:      "ACTIVE",
		BearerToken: bearerToken,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO society_visitor_passes (
			id, society_id, visit_id, pass_code, token_hash, valid_from, valid_until, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $9
		)
	`, pass.ID, pass.SocietyID, pass.VisitID, pass.PassCode, tokenHash, pass.ValidFrom, pass.ValidUntil, pass.Status, now)
	if err != nil {
		return nil, "", fmt.Errorf("failed to insert visitor pass: %w", err)
	}

	// Audit trail
	auditDetails := map[string]interface{}{
		"pass_id":    pass.ID.String(),
		"visit_id":   visitID.String(),
		"society_id": societyID.String(),
		"pass_code":  passCode,
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitorPassIssued, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, "", fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return pass, bearerToken, nil
}

// VerifyVisitorPass verifies a presented bearer token or pass code.
func (r *Repository) VerifyVisitorPass(
	ctx context.Context,
	societyID uuid.UUID,
	tokenOrCode string,
) (*VisitorPass, *Visit, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	tokenOrCode = strings.TrimSpace(tokenOrCode)
	if tokenOrCode == "" {
		return nil, nil, ErrPassNotFound
	}

	var pass VisitorPass
	var visit Visit
	var uIdent, bName string

	var query string
	var arg interface{}
	if strings.HasPrefix(tokenOrCode, "chs_pass_") {
		hash := sha256.Sum256([]byte(tokenOrCode))
		query = `
			SELECT 
				p.id, p.society_id, p.visit_id, p.pass_code, p.valid_from, p.valid_until, 
				p.status, p.consumed_at, p.revoked_at, p.revoked_by, p.created_at, p.updated_at,
				v.id, v.unit_id, u.unit_identifier, b.name, v.visitor_name, v.visitor_phone, 
				v.purpose, v.visit_type, v.status, v.vehicle_number
			FROM society_visitor_passes p
			JOIN society_visits v ON p.visit_id = v.id
			JOIN society_units u ON v.unit_id = u.id
			JOIN society_buildings b ON u.building_id = b.id
			WHERE p.token_hash = $1 AND p.society_id = $2
		`
		arg = hex.EncodeToString(hash[:])
	} else {
		query = `
			SELECT 
				p.id, p.society_id, p.visit_id, p.pass_code, p.valid_from, p.valid_until, 
				p.status, p.consumed_at, p.revoked_at, p.revoked_by, p.created_at, p.updated_at,
				v.id, v.unit_id, u.unit_identifier, b.name, v.visitor_name, v.visitor_phone, 
				v.purpose, v.visit_type, v.status, v.vehicle_number
			FROM society_visitor_passes p
			JOIN society_visits v ON p.visit_id = v.id
			JOIN society_units u ON v.unit_id = u.id
			JOIN society_buildings b ON u.building_id = b.id
			WHERE UPPER(p.pass_code) = UPPER($1) AND p.society_id = $2
		`
		arg = tokenOrCode
	}

	err := r.pool.QueryRow(ctx, query, arg, societyID).Scan(
		&pass.ID, &pass.SocietyID, &pass.VisitID, &pass.PassCode, &pass.ValidFrom, &pass.ValidUntil,
		&pass.Status, &pass.ConsumedAt, &pass.RevokedAt, &pass.RevokedBy, &pass.CreatedAt, &pass.UpdatedAt,
		&visit.ID, &visit.UnitID, &uIdent, &bName, &visit.VisitorName, &visit.VisitorPhone,
		&visit.Purpose, &visit.VisitType, &visit.Status, &visit.VehicleNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrPassNotFound
		}
		return nil, nil, fmt.Errorf("failed to verify pass: %w", err)
	}

	visit.UnitIdentifier = uIdent
	visit.BuildingName = bName
	visit.VisitorPhoneMasked = auth.MaskPhone(visit.VisitorPhone)

	now := time.Now().UTC()
	if pass.Status == "REVOKED" {
		return &pass, &visit, ErrPassRevoked
	}
	if pass.Status == "CONSUMED" {
		return &pass, &visit, ErrPassAlreadyConsumed
	}
	if now.After(pass.ValidUntil) {
		return &pass, &visit, ErrPassExpired
	}

	return &pass, &visit, nil
}

// CheckInVisitor performs atomic entry validation and check-in.
func (r *Repository) CheckInVisitor(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	gateID uuid.UUID,
	vehicleNumber *string,
	notes *string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, *GateEvent, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Verify gate exists, is active, and direction allows ingress
	var gateName, directionality, opStatus, gateStatus string
	err = tx.QueryRow(ctx, `
		SELECT name, directionality, operational_status, status
		FROM society_gates
		WHERE id = $1 AND society_id = $2
	`, gateID, societyID).Scan(&gateName, &directionality, &opStatus, &gateStatus)
	if err != nil {
		return nil, nil, errors.New("invalid or foreign gate")
	}
	if gateStatus != "ACTIVE" || (opStatus != "ACTIVE" && opStatus != "EMERGENCY_OPEN") {
		return nil, nil, ErrGateInactive
	}
	if directionality == "EGRESS_ONLY" {
		return nil, nil, ErrGateDirectionMismatch
	}

	// 2. Verify gatekeeper active duty session (or retrieve session ID)
	var sessionID *uuid.UUID
	var sessID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM society_gate_sessions
		WHERE society_id = $1 AND user_id = $2 AND gate_id = $3 AND status = 'ACTIVE'
	`, societyID, actorID, gateID).Scan(&sessID)
	if err == nil {
		sessionID = &sessID
	} else if errors.Is(err, pgx.ErrNoRows) {
		var isGatekeeper bool
		_ = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM society_user_roles
				WHERE society_id = $1 AND user_id = $2 AND role = 'GATEKEEPER' AND status = 'ACTIVE'
			)
		`, societyID, actorID).Scan(&isGatekeeper)
		if isGatekeeper {
			return nil, nil, ErrDutySessionRequired
		}
	} else {
		return nil, nil, fmt.Errorf("failed to verify duty session: %w", err)
	}

	// 3. Atomically lock visit row
	var currentStatus, visitorName, visitorPhone string
	var unitID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT status, unit_id, visitor_name, visitor_phone
		FROM society_visits
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &unitID, &visitorName, &visitorPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrVisitNotFound
		}
		return nil, nil, fmt.Errorf("failed to lock visit: %w", err)
	}

	if currentStatus == "CHECKED_IN" {
		return nil, nil, ErrVisitAlreadyCheckedIn
	}
	if currentStatus != "APPROVED" {
		return nil, nil, fmt.Errorf("%w: visit is in %s state, must be APPROVED to check in", ErrInvalidVisitState, currentStatus)
	}

	now := time.Now().UTC()
	var updated Visit
	err = tx.QueryRow(ctx, `
		UPDATE society_visits
		SET status = 'CHECKED_IN', check_in_at = $1, entry_gate_id = $2,
		    vehicle_number = COALESCE($3, vehicle_number), updated_at = $1
		WHERE id = $4 AND society_id = $5
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, now, gateID, vehicleNumber, visitID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.UnitID, &updated.VisitorName, &updated.VisitorPhone,
		&updated.Purpose, &updated.VisitType, &updated.Status, &updated.VehicleNumber,
		&updated.ExpectedArrival, &updated.ValidUntil, &updated.CheckInAt, &updated.CheckOutAt,
		&updated.EntryGateID, &updated.ExitGateID, &updated.RejectionReason,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check in visit: %w", err)
	}

	// 4. Consume any active pass for this visit
	_, err = tx.Exec(ctx, `
		UPDATE society_visitor_passes
		SET status = 'CONSUMED', consumed_at = $1, updated_at = $1
		WHERE visit_id = $2 AND society_id = $3 AND status = 'ACTIVE'
	`, now, visitID, societyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to consume visitor pass: %w", err)
	}

	// 5. Append gate event
	eventID := uuid.New()
	event := &GateEvent{
		ID:            eventID,
		SocietyID:     societyID,
		VisitID:       visitID,
		GateID:        gateID,
		GateName:      gateName,
		SessionID:     sessionID,
		GatekeeperID:  actorID,
		EventType:     "ENTRY",
		Direction:     "INGRESS",
		VehicleNumber: vehicleNumber,
		Notes:         notes,
		CreatedAt:     now,
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO society_gate_events (
			id, society_id, visit_id, gate_id, session_id, gatekeeper_id, 
			event_type, direction, vehicle_number, notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`, event.ID, event.SocietyID, event.VisitID, event.GateID, event.SessionID,
		event.GatekeeperID, event.EventType, event.Direction, event.VehicleNumber,
		event.Notes, event.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to record gate event: %w", err)
	}

	// 6. Audit trail
	auditDetails := map[string]interface{}{
		"visit_id":       visitID.String(),
		"society_id":     societyID.String(),
		"gate_id":        gateID.String(),
		"gate_name":      gateName,
		"vehicle_number": vehicleNumber,
		"visitor_name":   visitorName,
		"visitor_phone":  auth.MaskPhone(visitorPhone),
		"event_type":     "ENTRY",
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitorCheckedIn, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated.VisitorPhoneMasked = auth.MaskPhone(updated.VisitorPhone)
	return &updated, event, nil
}

// CheckOutVisitor performs atomic departure validation and checkout.
func (r *Repository) CheckOutVisitor(
	ctx context.Context,
	societyID uuid.UUID,
	visitID uuid.UUID,
	gateID uuid.UUID,
	notes *string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*Visit, *GateEvent, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Verify gate exists, is active, and direction allows egress
	var gateName, directionality, opStatus, gateStatus string
	err = tx.QueryRow(ctx, `
		SELECT name, directionality, operational_status, status
		FROM society_gates
		WHERE id = $1 AND society_id = $2
	`, gateID, societyID).Scan(&gateName, &directionality, &opStatus, &gateStatus)
	if err != nil {
		return nil, nil, errors.New("invalid or foreign gate")
	}
	if gateStatus != "ACTIVE" || (opStatus != "ACTIVE" && opStatus != "EMERGENCY_OPEN") {
		return nil, nil, ErrGateInactive
	}
	if directionality == "INGRESS_ONLY" {
		return nil, nil, ErrGateDirectionMismatch
	}

	// 2. Active duty session
	var sessionID *uuid.UUID
	var sessID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM society_gate_sessions
		WHERE society_id = $1 AND user_id = $2 AND gate_id = $3 AND status = 'ACTIVE'
	`, societyID, actorID, gateID).Scan(&sessID)
	if err == nil {
		sessionID = &sessID
	} else if errors.Is(err, pgx.ErrNoRows) {
		var isGatekeeper bool
		_ = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM society_user_roles
				WHERE society_id = $1 AND user_id = $2 AND role = 'GATEKEEPER' AND status = 'ACTIVE'
			)
		`, societyID, actorID).Scan(&isGatekeeper)
		if isGatekeeper {
			return nil, nil, ErrDutySessionRequired
		}
	} else {
		return nil, nil, fmt.Errorf("failed to verify duty session: %w", err)
	}

	// 3. Atomically lock visit row
	var currentStatus, visitorName, visitorPhone string
	var unitID uuid.UUID
	var vehicleNumber *string
	err = tx.QueryRow(ctx, `
		SELECT status, unit_id, visitor_name, visitor_phone, vehicle_number
		FROM society_visits
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &unitID, &visitorName, &visitorPhone, &vehicleNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrVisitNotFound
		}
		return nil, nil, fmt.Errorf("failed to lock visit: %w", err)
	}

	if currentStatus == "CHECKED_OUT" {
		return nil, nil, ErrVisitAlreadyCheckedOut
	}
	if currentStatus != "CHECKED_IN" {
		return nil, nil, ErrVisitNotCheckedIn
	}

	now := time.Now().UTC()
	var updated Visit
	err = tx.QueryRow(ctx, `
		UPDATE society_visits
		SET status = 'CHECKED_OUT', check_out_at = $1, exit_gate_id = $2, updated_at = $1
		WHERE id = $3 AND society_id = $4
		RETURNING 
			id, society_id, unit_id, visitor_name, visitor_phone, purpose, 
			visit_type, status, vehicle_number, expected_arrival, valid_until, 
			check_in_at, check_out_at, entry_gate_id, exit_gate_id, rejection_reason, 
			created_by, created_at, updated_at
	`, now, gateID, visitID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.UnitID, &updated.VisitorName, &updated.VisitorPhone,
		&updated.Purpose, &updated.VisitType, &updated.Status, &updated.VehicleNumber,
		&updated.ExpectedArrival, &updated.ValidUntil, &updated.CheckInAt, &updated.CheckOutAt,
		&updated.EntryGateID, &updated.ExitGateID, &updated.RejectionReason,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check out visit: %w", err)
	}

	// 4. Append gate event
	eventID := uuid.New()
	event := &GateEvent{
		ID:            eventID,
		SocietyID:     societyID,
		VisitID:       visitID,
		GateID:        gateID,
		GateName:      gateName,
		SessionID:     sessionID,
		GatekeeperID:  actorID,
		EventType:     "EXIT",
		Direction:     "EGRESS",
		VehicleNumber: vehicleNumber,
		Notes:         notes,
		CreatedAt:     now,
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO society_gate_events (
			id, society_id, visit_id, gate_id, session_id, gatekeeper_id, 
			event_type, direction, vehicle_number, notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`, event.ID, event.SocietyID, event.VisitID, event.GateID, event.SessionID,
		event.GatekeeperID, event.EventType, event.Direction, event.VehicleNumber,
		event.Notes, event.CreatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to record gate event: %w", err)
	}

	// 5. Audit trail
	auditDetails := map[string]interface{}{
		"visit_id":      visitID.String(),
		"society_id":    societyID.String(),
		"gate_id":       gateID.String(),
		"gate_name":     gateName,
		"visitor_name":  visitorName,
		"visitor_phone": auth.MaskPhone(visitorPhone),
		"event_type":    "EXIT",
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionVisitorCheckedOut, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	updated.VisitorPhoneMasked = auth.MaskPhone(updated.VisitorPhone)
	return &updated, event, nil
}

// GetVisitorDashboardMetrics calculates authoritative operational counts.
func (r *Repository) GetVisitorDashboardMetrics(ctx context.Context, societyID uuid.UUID) (*VisitorDashboardMetrics, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var metrics VisitorDashboardMetrics

	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'APPROVED') AS expected_today,
			COUNT(*) FILTER (WHERE status = 'PENDING_APPROVAL') AS awaiting_approval,
			COUNT(*) FILTER (WHERE status = 'CHECKED_IN') AS currently_inside,
			COUNT(*) FILTER (WHERE status = 'CHECKED_OUT' AND check_out_at::date = CURRENT_DATE) AS completed_today
		FROM society_visits
		WHERE society_id = $1
	`
	err := r.pool.QueryRow(ctx, query, societyID).Scan(
		&metrics.ExpectedToday,
		&metrics.AwaitingApproval,
		&metrics.CurrentlyInside,
		&metrics.CompletedToday,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate dashboard metrics: %w", err)
	}

	return &metrics, nil
}

// ValidateUserUnitOccupancy checks if a user is an active occupant of the unit.
func (r *Repository) ValidateUserUnitOccupancy(
	ctx context.Context,
	societyID uuid.UUID,
	unitID uuid.UUID,
	userID uuid.UUID,
) (bool, *uuid.UUID, error) {
	if r.pool == nil {
		return false, nil, errors.New("database pool unavailable")
	}

	var memberID uuid.UUID
	query := `
		SELECT ur.member_id
		FROM unit_relationships ur
		JOIN society_members sm ON ur.member_id = sm.id
		WHERE ur.society_id = $1 
		  AND ur.unit_id = $2 
		  AND sm.user_id = $3 
		  AND ur.status = 'ACTIVE' 
		  AND sm.status = 'ACTIVE'
		  AND ur.is_occupant = true
		LIMIT 1
	`
	err := r.pool.QueryRow(ctx, query, societyID, unitID, userID).Scan(&memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &memberID, nil
}

// ValidateUserUnitAuthorization checks if a user is an active occupant or owner authorized to manage the unit.
func (r *Repository) ValidateUserUnitAuthorization(
	ctx context.Context,
	societyID uuid.UUID,
	unitID uuid.UUID,
	userID uuid.UUID,
) (bool, *uuid.UUID, error) {
	if r.pool == nil {
		return false, nil, errors.New("database pool unavailable")
	}

	var memberID uuid.UUID
	query := `
		SELECT ur.member_id
		FROM unit_relationships ur
		JOIN society_members sm ON ur.member_id = sm.id
		WHERE ur.society_id = $1 
		  AND ur.unit_id = $2 
		  AND sm.user_id = $3 
		  AND ur.status = 'ACTIVE' 
		  AND sm.status = 'ACTIVE'
		  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
		LIMIT 1
	`
	err := r.pool.QueryRow(ctx, query, societyID, unitID, userID).Scan(&memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &memberID, nil
}

// GetResidentUnits returns the units the authenticated resident is authorized to manage in this society.
func (r *Repository) GetResidentUnits(
	ctx context.Context,
	societyID uuid.UUID,
	userID uuid.UUID,
) ([]ResidentUnit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT u.id, u.society_id, u.building_id, b.name AS building_name,
		       u.wing_id, w.wing_code, u.floor_number, u.unit_number,
		       u.unit_identifier, u.unit_type, ur.relationship_type,
		       ur.is_occupant, u.requires_visitor_approval
		FROM unit_relationships ur
		JOIN society_members sm ON ur.member_id = sm.id
		JOIN society_units u ON ur.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_wings w ON u.wing_id = w.id
		WHERE ur.society_id = $1 
		  AND sm.user_id = $2 
		  AND ur.status = 'ACTIVE' 
		  AND sm.status = 'ACTIVE'
		  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
		ORDER BY b.name ASC, u.unit_identifier ASC
	`
	rows, err := r.pool.Query(ctx, query, societyID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query resident units: %w", err)
	}
	defer rows.Close()

	items := make([]ResidentUnit, 0)
	for rows.Next() {
		var item ResidentUnit
		if err := rows.Scan(
			&item.ID, &item.SocietyID, &item.BuildingID, &item.BuildingName,
			&item.WingID, &item.WingCode, &item.FloorNumber, &item.UnitNumber,
			&item.UnitIdentifier, &item.UnitType, &item.RelationshipType,
			&item.IsOccupant, &item.RequiresVisitorApproval,
		); err != nil {
			return nil, fmt.Errorf("failed to scan resident unit: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating resident units: %w", err)
	}

	return items, nil
}

// UpdateUnitVisitorPolicy updates the visitor approval preference for an authorized unit.
func (r *Repository) UpdateUnitVisitorPolicy(
	ctx context.Context,
	societyID uuid.UUID,
	unitID uuid.UUID,
	actorID uuid.UUID,
	requiresApproval bool,
	clientIP, userAgent string,
) (*ResidentUnit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	isAuthorized, _, err := r.ValidateUserUnitAuthorization(ctx, societyID, unitID, actorID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate unit authorization: %w", err)
	}
	if !isAuthorized {
		return nil, ErrUnauthorizedResident
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var unit ResidentUnit
	err = tx.QueryRow(ctx, `
		UPDATE society_units
		SET requires_visitor_approval = $1, updated_at = NOW()
		WHERE id = $2 AND society_id = $3
		RETURNING id, society_id, building_id, floor_number, unit_number, unit_identifier, unit_type, requires_visitor_approval
	`, requiresApproval, unitID, societyID).Scan(
		&unit.ID, &unit.SocietyID, &unit.BuildingID, &unit.FloorNumber,
		&unit.UnitNumber, &unit.UnitIdentifier, &unit.UnitType, &unit.RequiresVisitorApproval,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update unit visitor policy: %w", err)
	}

	_ = tx.QueryRow(ctx, "SELECT name FROM society_buildings WHERE id = $1", unit.BuildingID).Scan(&unit.BuildingName)

	auditDetails := map[string]interface{}{
		"unit_id":                   unitID.String(),
		"unit_identifier":           unit.UnitIdentifier,
		"society_id":                societyID.String(),
		"requires_visitor_approval": requiresApproval,
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`, actorID, ActionUnitVisitorPolicyUpdated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &unit, nil
}

// ListPendingVisitsForResident retrieves pending visits for units the caller is authorized to manage.
func (r *Repository) ListPendingVisitsForResident(
	ctx context.Context,
	societyID uuid.UUID,
	actorID uuid.UUID,
	callerRole string,
) ([]Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var query string
	var args []interface{}

	if callerRole == "SUPER_ADMIN" || callerRole == RoleSocietyAdmin {
		query = `
			SELECT v.id, v.society_id, v.unit_id, v.visitor_name, v.visitor_phone, 
			       v.purpose, v.visit_type, v.status, v.vehicle_number, v.expected_arrival, 
			       v.valid_until, v.check_in_at, v.check_out_at, v.entry_gate_id, v.exit_gate_id, 
			       v.rejection_reason, v.created_by, v.created_at, v.updated_at,
			       u.unit_identifier, b.name AS building_name
			FROM society_visits v
			JOIN society_units u ON v.unit_id = u.id
			JOIN society_buildings b ON u.building_id = b.id
			WHERE v.society_id = $1 AND v.status = 'PENDING_APPROVAL'
			ORDER BY v.created_at DESC
		`
		args = []interface{}{societyID}
	} else {
		query = `
			SELECT v.id, v.society_id, v.unit_id, v.visitor_name, v.visitor_phone, 
			       v.purpose, v.visit_type, v.status, v.vehicle_number, v.expected_arrival, 
			       v.valid_until, v.check_in_at, v.check_out_at, v.entry_gate_id, v.exit_gate_id, 
			       v.rejection_reason, v.created_by, v.created_at, v.updated_at,
			       u.unit_identifier, b.name AS building_name
			FROM society_visits v
			JOIN society_units u ON v.unit_id = u.id
			JOIN society_buildings b ON u.building_id = b.id
			WHERE v.society_id = $1 
			  AND v.status = 'PENDING_APPROVAL'
			  AND v.unit_id IN (
			      SELECT ur.unit_id
			      FROM unit_relationships ur
			      JOIN society_members sm ON ur.member_id = sm.id
			      WHERE ur.society_id = $1 
			        AND sm.user_id = $2 
			        AND ur.status = 'ACTIVE' 
			        AND sm.status = 'ACTIVE'
			        AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
			  )
			ORDER BY v.created_at DESC
		`
		args = []interface{}{societyID, actorID}
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending visits: %w", err)
	}
	defer rows.Close()

	items := make([]Visit, 0)
	for rows.Next() {
		var v Visit
		if err := rows.Scan(
			&v.ID, &v.SocietyID, &v.UnitID, &v.VisitorName, &v.VisitorPhone,
			&v.Purpose, &v.VisitType, &v.Status, &v.VehicleNumber, &v.ExpectedArrival,
			&v.ValidUntil, &v.CheckInAt, &v.CheckOutAt, &v.EntryGateID, &v.ExitGateID,
			&v.RejectionReason, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt,
			&v.UnitIdentifier, &v.BuildingName,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending visit: %w", err)
		}
		v.VisitorPhoneMasked = auth.MaskPhone(v.VisitorPhone)
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending visits: %w", err)
	}

	return items, nil
}
