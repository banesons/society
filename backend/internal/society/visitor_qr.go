package society

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chs/backend/internal/auth"
)

// QR Domain Errors
var (
	ErrQRNotFound          = errors.New("registration QR code not found")
	ErrQRInactive          = errors.New("registration QR code is inactive or revoked")
	ErrVisitAlreadyDecided = errors.New("visitor registration has already been approved or rejected")
)

// Audit Log Actions for QR and Self-Registration
const (
	ActionGateQRGenerated     = "GATE_QR_GENERATED"
	ActionGateQRRevoked       = "GATE_QR_REVOKED"
	ActionGateQRRotated       = "GATE_QR_ROTATED"
	ActionVisitSelfRegistered = "VISIT_SELF_REGISTERED"
	ActionQRVisitAdmitted     = "QR_VISIT_ADMITTED"
)

// GateQR represents an administrator-managed QR registration code.
type GateQR struct {
	ID        uuid.UUID  `json:"id"`
	SocietyID uuid.UUID  `json:"society_id"`
	GateID    *uuid.UUID `json:"gate_id,omitempty"`
	GateName  *string    `json:"gate_name,omitempty"`
	Token     string     `json:"token"`
	Label     string     `json:"label"`
	Status    string     `json:"status"` // ACTIVE, INACTIVE, REVOKED
	CreatedBy uuid.UUID  `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// PublicQRContext represents the safe public context returned when a visitor scans a QR.
type PublicQRContext struct {
	Token       string     `json:"token"`
	SocietyID   uuid.UUID  `json:"society_id"`
	SocietyName string     `json:"society_name"`
	SocietyCode string     `json:"society_code"`
	GateID      *uuid.UUID `json:"gate_id,omitempty"`
	GateName    *string    `json:"gate_name,omitempty"`
	Status      string     `json:"status"`
}

// QRVisitorRegisterParams holds validated inputs for visitor self-registration.
type QRVisitorRegisterParams struct {
	QRToken           string     `json:"qr_token"`
	FullName          string     `json:"full_name"`
	Phone             string     `json:"phone"`
	Purpose           string     `json:"purpose"`
	ComingFrom        string     `json:"coming_from,omitempty"`
	VisitorPhotoURL   string     `json:"visitor_photo_url,omitempty"`
	UnitID            *uuid.UUID `json:"unit_id,omitempty"`
	VehicleNumber     string     `json:"vehicle_number,omitempty"`
	AuthenticatedUser *uuid.UUID `json:"-"`
}

// PublicVisitorPass represents the digital pass displayed to the visitor.
type PublicVisitorPass struct {
	PassReference      string     `json:"pass_reference"`
	VisitID            uuid.UUID  `json:"visit_id"`
	SocietyID          uuid.UUID  `json:"society_id"`
	VisitorName        string     `json:"visitor_name"`
	VisitorPhoneMasked string     `json:"visitor_phone_masked"`
	SocietyName        string     `json:"society_name"`
	GateName           *string    `json:"gate_name,omitempty"`
	UnitIdentifier     *string    `json:"unit_identifier,omitempty"`
	BuildingName       *string    `json:"building_name,omitempty"`
	Purpose            string     `json:"purpose"`
	ComingFrom         *string    `json:"coming_from,omitempty"`
	VisitorPhotoURL    *string    `json:"visitor_photo_url,omitempty"`
	Status             string     `json:"status"`
	RegisteredAt       time.Time  `json:"registered_at"`
	CheckInAt          *time.Time `json:"check_in_at,omitempty"`
}

// PendingQRVisit represents a pending visitor registration presented to gatekeepers.
type PendingQRVisit struct {
	ID                 uuid.UUID  `json:"id"`
	SocietyID          uuid.UUID  `json:"society_id"`
	PassReference      string     `json:"pass_reference"`
	VisitorName        string     `json:"visitor_name"`
	VisitorPhone       string     `json:"visitor_phone"`
	VisitorPhoneMasked string     `json:"visitor_phone_masked"`
	VisitorPhotoURL    *string    `json:"visitor_photo_url,omitempty"`
	Purpose            string     `json:"purpose"`
	ComingFrom         *string    `json:"coming_from,omitempty"`
	GateID             *uuid.UUID `json:"gate_id,omitempty"`
	GateName           *string    `json:"gate_name,omitempty"`
	UnitID             *uuid.UUID `json:"unit_id,omitempty"`
	UnitIdentifier     *string    `json:"unit_identifier,omitempty"`
	BuildingName       *string    `json:"building_name,omitempty"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
}

// generateOpaqueQRToken creates a high-entropy random token with prefix "chs_qr_".
func generateOpaqueQRToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "chs_qr_" + hex.EncodeToString(bytes), nil
}

// generatePassReference creates a 128-bit cryptographically secure, unpredictable visitor pass bearer reference (e.g., "VP-9B7C12...").
func generatePassReference() (string, error) {
	bytes := make([]byte, 16) // 16 bytes = 128 bits of cryptographic entropy
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure reference: %w", err)
	}
	return "VP-" + strings.ToUpper(hex.EncodeToString(bytes)), nil
}

// CreateGateQR generates a new visitor registration QR code for a gate or society.
func (r *Repository) CreateGateQR(ctx context.Context, societyID uuid.UUID, gateID *uuid.UUID, label string, actorID uuid.UUID) (*GateQR, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	label = strings.TrimSpace(label)
	if label == "" {
		label = "Entrance Visitor Registration"
	}

	token, err := generateOpaqueQRToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var qr GateQR
	qr.ID = uuid.New()
	qr.SocietyID = societyID
	qr.GateID = gateID
	qr.Token = token
	qr.Label = label
	qr.Status = "ACTIVE"
	qr.CreatedBy = actorID
	qr.CreatedAt = now
	qr.UpdatedAt = now

	query := `
		INSERT INTO society_gate_qrs (id, society_id, gate_id, token, label, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = r.pool.Exec(ctx, query, qr.ID, qr.SocietyID, qr.GateID, qr.Token, qr.Label, qr.Status, qr.CreatedBy, qr.CreatedAt, qr.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert gate QR: %w", err)
	}

	if qr.GateID != nil {
		var gName string
		_ = r.pool.QueryRow(ctx, `SELECT name FROM society_gates WHERE id = $1 AND society_id = $2`, qr.GateID, societyID).Scan(&gName)
		qr.GateName = &gName
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"qr_id":      qr.ID.String(),
		"society_id": societyID.String(),
		"gate_id":    gateID,
		"label":      label,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, $2, NULL, $3::jsonb)
	`, actorID, ActionGateQRGenerated, string(auditDetails))

	return &qr, nil
}

// ListGateQRs retrieves all gate registration QR configurations for a society.
func (r *Repository) ListGateQRs(ctx context.Context, societyID uuid.UUID, gateID *uuid.UUID) ([]GateQR, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT q.id, q.society_id, q.gate_id, g.name AS gate_name, q.token, q.label, q.status, q.created_by, q.created_at, q.updated_at
		FROM society_gate_qrs q
		LEFT JOIN society_gates g ON q.gate_id = g.id
		WHERE q.society_id = $1 AND ($2::uuid IS NULL OR q.gate_id = $2)
		ORDER BY q.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, societyID, gateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gate QRs: %w", err)
	}
	defer rows.Close()

	var list []GateQR
	for rows.Next() {
		var item GateQR
		var gName *string
		if err := rows.Scan(
			&item.ID, &item.SocietyID, &item.GateID, &gName,
			&item.Token, &item.Label, &item.Status, &item.CreatedBy,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan gate QR: %w", err)
		}
		item.GateName = gName
		list = append(list, item)
	}

	if list == nil {
		list = []GateQR{}
	}
	return list, nil
}

// RevokeGateQR marks a gate QR as revoked.
func (r *Repository) RevokeGateQR(ctx context.Context, societyID uuid.UUID, qrID uuid.UUID, actorID uuid.UUID) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	cmdTag, err := r.pool.Exec(ctx, `
		UPDATE society_gate_qrs
		SET status = 'REVOKED', updated_at = NOW()
		WHERE id = $1 AND society_id = $2 AND status != 'REVOKED'
	`, qrID, societyID)
	if err != nil {
		return fmt.Errorf("failed to revoke gate QR: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrQRNotFound
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"qr_id":      qrID.String(),
		"society_id": societyID.String(),
		"action":     "REVOKE",
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, $2, NULL, $3::jsonb)
	`, actorID, ActionGateQRRevoked, string(auditDetails))

	return nil
}

// RotateGateQR invalidates the previous QR code and issues a new active QR code for the same gate.
func (r *Repository) RotateGateQR(ctx context.Context, societyID uuid.UUID, qrID uuid.UUID, actorID uuid.UUID) (*GateQR, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var gateID *uuid.UUID
	var label string
	err := r.pool.QueryRow(ctx, `
		SELECT gate_id, label
		FROM society_gate_qrs
		WHERE id = $1 AND society_id = $2
	`, qrID, societyID).Scan(&gateID, &label)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQRNotFound
		}
		return nil, fmt.Errorf("failed to find QR to rotate: %w", err)
	}

	// Revoke old QR
	_, _ = r.pool.Exec(ctx, `
		UPDATE society_gate_qrs
		SET status = 'REVOKED', updated_at = NOW()
		WHERE id = $1 AND society_id = $2
	`, qrID, societyID)

	// Create new active QR
	newQR, err := r.CreateGateQR(ctx, societyID, gateID, label, actorID)
	if err != nil {
		return nil, fmt.Errorf("failed to create rotated QR: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"old_qr_id":  qrID.String(),
		"new_qr_id":  newQR.ID.String(),
		"society_id": societyID.String(),
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, $2, NULL, $3::jsonb)
	`, actorID, ActionGateQRRotated, string(auditDetails))

	return newQR, nil
}

// ResolvePublicGateQR looks up the public registration context from an opaque token.
// Does NOT return internal administrator fields.
func (r *Repository) ResolvePublicGateQR(ctx context.Context, token string) (*PublicQRContext, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrQRNotFound
	}

	query := `
		SELECT q.token, q.society_id, s.name AS society_name, s.society_code AS society_code,
		       q.gate_id, g.name AS gate_name, g.status AS gate_status, q.status AS qr_status
		FROM society_gate_qrs q
		JOIN societies s ON q.society_id = s.id
		LEFT JOIN society_gates g ON q.gate_id = g.id
		WHERE q.token = $1
	`
	var (
		pubCtx     PublicQRContext
		gateStatus *string
		qrStatus   string
	)

	err := r.pool.QueryRow(ctx, query, token).Scan(
		&pubCtx.Token, &pubCtx.SocietyID, &pubCtx.SocietyName, &pubCtx.SocietyCode,
		&pubCtx.GateID, &pubCtx.GateName, &gateStatus, &qrStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQRNotFound
		}
		return nil, fmt.Errorf("failed to resolve QR: %w", err)
	}

	if qrStatus != "ACTIVE" {
		return nil, ErrQRInactive
	}

	if gateStatus != nil && *gateStatus != "ACTIVE" {
		return nil, ErrQRInactive
	}

	pubCtx.Status = "ACTIVE"
	return &pubCtx, nil
}

// RegisterVisitorByQR creates a self-registered visitor record awaiting gatekeeper admission.
func (r *Repository) RegisterVisitorByQR(ctx context.Context, params QRVisitorRegisterParams) (*PublicVisitorPass, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// 1. Resolve QR token server-side
	pubCtx, err := r.ResolvePublicGateQR(ctx, params.QRToken)
	if err != nil {
		return nil, err
	}

	// 2. Validate input
	fullName := strings.TrimSpace(params.FullName)
	if fullName == "" {
		return nil, errors.New("visitor name is required")
	}

	normPhone, err := auth.NormalizePhone(params.Phone)
	if err != nil {
		return nil, ErrInvalidPhoneFormat
	}

	purpose := strings.TrimSpace(params.Purpose)
	if purpose == "" {
		purpose = "GUEST"
	}

	// 3. Prevent duplicate active registrations within 5 minutes for same phone & society
	var existingPass PublicVisitorPass
	var existingCreatedAt time.Time
	checkDupQuery := `
		SELECT v.id, v.society_id, v.pass_reference, v.visitor_name, v.visitor_phone,
		       s.name AS society_name, g.name AS gate_name, v.purpose, v.coming_from,
		       v.visitor_photo_url, v.status, v.created_at
		FROM society_visits v
		JOIN societies s ON v.society_id = s.id
		LEFT JOIN society_gates g ON v.entry_gate_id = g.id
		WHERE v.society_id = $1 AND v.visitor_phone = $2
		  AND v.status = 'REGISTERED_PENDING_GATE_VERIFICATION'
		  AND v.created_at > NOW() - INTERVAL '5 minutes'
		ORDER BY v.created_at DESC
		LIMIT 1
	`
	err = r.pool.QueryRow(ctx, checkDupQuery, pubCtx.SocietyID, normPhone).Scan(
		&existingPass.VisitID, &existingPass.SocietyID, &existingPass.PassReference,
		&existingPass.VisitorName, &existingPass.VisitorPhoneMasked,
		&existingPass.SocietyName, &existingPass.GateName, &existingPass.Purpose,
		&existingPass.ComingFrom, &existingPass.VisitorPhotoURL,
		&existingPass.Status, &existingCreatedAt,
	)
	if err == nil {
		existingPass.RegisteredAt = existingCreatedAt
		existingPass.VisitorPhoneMasked = auth.MaskVisitorPhone(existingPass.VisitorPhoneMasked)
		return &existingPass, nil
	}

	// 4. Begin transaction for atomic creation
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()

	// Do not auto-link visitor identity to a user account unless caller is authenticated
	var linkUserID *uuid.UUID
	var linkAt *time.Time
	if params.AuthenticatedUser != nil && *params.AuthenticatedUser != uuid.Nil {
		linkUserID = params.AuthenticatedUser
		linkAt = &now
	}

	// Fetch or create visitor identity
	var visitorIdentityID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO society_visitor_identities (
			id, society_id, phone, full_name, user_id, linked_at, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3,
			$4,
			$5,
			$6, $6
		)
		ON CONFLICT (society_id, phone) DO UPDATE
		SET full_name = COALESCE(NULLIF(society_visitor_identities.full_name, ''), EXCLUDED.full_name),
			user_id = COALESCE(society_visitor_identities.user_id, EXCLUDED.user_id),
			linked_at = COALESCE(society_visitor_identities.linked_at, EXCLUDED.linked_at),
			updated_at = EXCLUDED.updated_at
		RETURNING id
	`, pubCtx.SocietyID, normPhone, fullName, linkUserID, linkAt, now).Scan(&visitorIdentityID)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert visitor identity: %w", err)
	}

	// If unit_id is not provided, pick first active unit in society as reception/visitor unit
	unitID := params.UnitID
	if unitID == nil {
		var defaultUnitID uuid.UUID
		err := tx.QueryRow(ctx, `
			SELECT id FROM society_units
			WHERE society_id = $1
			ORDER BY unit_identifier ASC
			LIMIT 1
		`, pubCtx.SocietyID).Scan(&defaultUnitID)
		if err == nil {
			unitID = &defaultUnitID
		}
	}

	if unitID == nil {
		return nil, errors.New("no units configured in society for visitor routing")
	}

	// Generate pass reference with collision retry loop
	var passRef string
	for attempt := 0; attempt < 3; attempt++ {
		candidateRef, err := generatePassReference()
		if err != nil {
			return nil, err
		}
		var exists bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM society_visits WHERE pass_reference = $1)`, candidateRef).Scan(&exists)
		if !exists {
			passRef = candidateRef
			break
		}
	}
	if passRef == "" {
		return nil, errors.New("failed to generate unique pass reference")
	}

	// Lookup QR record ID
	var qrID *uuid.UUID
	var foundQRID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM society_gate_qrs WHERE token = $1`, params.QRToken).Scan(&foundQRID); err == nil && foundQRID != uuid.Nil {
		qrID = &foundQRID
	}

	visitID := uuid.New()
	var photoURL *string
	if strings.TrimSpace(params.VisitorPhotoURL) != "" {
		p := strings.TrimSpace(params.VisitorPhotoURL)
		photoURL = &p
	}
	var comingFrom *string
	if strings.TrimSpace(params.ComingFrom) != "" {
		cf := strings.TrimSpace(params.ComingFrom)
		comingFrom = &cf
	}
	var vehicleNumber *string
	if strings.TrimSpace(params.VehicleNumber) != "" {
		vn := strings.TrimSpace(params.VehicleNumber)
		vehicleNumber = &vn
	}

	insertVisitQuery := `
		INSERT INTO society_visits (
			id, society_id, unit_id, visitor_identity_id, visitor_name, visitor_phone,
			purpose, visit_type, status, vehicle_number, entry_gate_id,
			visitor_photo_url, coming_from, pass_reference, qr_id,
			created_by, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'QR_SELF_REGISTER', 'REGISTERED_PENDING_GATE_VERIFICATION', $8, $9, $10, $11, $12, $13, $14, $15, $15)
	`
	_, err = tx.Exec(ctx, insertVisitQuery,
		visitID, pubCtx.SocietyID, *unitID, visitorIdentityID, fullName, normPhone,
		purpose, vehicleNumber, pubCtx.GateID,
		photoURL, comingFrom, passRef, qrID,
		params.AuthenticatedUser, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create self-registered visit: %w", err)
	}

	// Record audit event
	var qrCreatedBy uuid.UUID
	_ = tx.QueryRow(ctx, `SELECT created_by FROM society_gate_qrs WHERE token = $1`, params.QRToken).Scan(&qrCreatedBy)
	actorForAudit := qrCreatedBy
	if params.AuthenticatedUser != nil && *params.AuthenticatedUser != uuid.Nil {
		actorForAudit = *params.AuthenticatedUser
	}

	if actorForAudit != uuid.Nil {
		auditDetails, _ := json.Marshal(map[string]interface{}{
			"visit_id":       visitID.String(),
			"society_id":     pubCtx.SocietyID.String(),
			"gate_id":        pubCtx.GateID,
			"pass_reference": passRef,
			"phone_masked":   auth.MaskVisitorPhone(normPhone),
			"purpose":        purpose,
		})
		_, _ = tx.Exec(ctx, `
			INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
			VALUES ($1, $2, NULL, $3::jsonb)
		`, actorForAudit, ActionVisitSelfRegistered, string(auditDetails))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit registration: %w", err)
	}

	pass := &PublicVisitorPass{
		PassReference:      passRef,
		VisitID:            visitID,
		SocietyID:          pubCtx.SocietyID,
		VisitorName:        fullName,
		VisitorPhoneMasked: auth.MaskVisitorPhone(normPhone),
		SocietyName:        pubCtx.SocietyName,
		GateName:           pubCtx.GateName,
		Purpose:            purpose,
		ComingFrom:         comingFrom,
		VisitorPhotoURL:    photoURL,
		Status:             "REGISTERED_PENDING_GATE_VERIFICATION",
		RegisteredAt:       now,
	}

	return pass, nil
}

// GetPublicVisitorPass retrieves status and pass details by pass_reference.
func (r *Repository) GetPublicVisitorPass(ctx context.Context, passReference string) (*PublicVisitorPass, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	passReference = strings.TrimSpace(passReference)
	if passReference == "" || !strings.HasPrefix(passReference, "VP-") || len(passReference) < 9 {
		return nil, ErrPassNotFound
	}

	query := `
		SELECT v.id, v.society_id, v.pass_reference, v.visitor_name, v.visitor_phone,
		       s.name AS society_name, g.name AS gate_name,
		       u.unit_identifier, b.name AS building_name,
		       v.purpose, v.coming_from, v.visitor_photo_url,
		       v.status, v.created_at, v.check_in_at
		FROM society_visits v
		JOIN societies s ON v.society_id = s.id
		LEFT JOIN society_gates g ON v.entry_gate_id = g.id
		LEFT JOIN society_units u ON v.unit_id = u.id
		LEFT JOIN society_buildings b ON u.building_id = b.id
		WHERE v.pass_reference = $1
		ORDER BY v.created_at DESC
		LIMIT 1
	`
	var (
		pass      PublicVisitorPass
		phoneRaw  string
		createdAt time.Time
	)
	err := r.pool.QueryRow(ctx, query, passReference).Scan(
		&pass.VisitID, &pass.SocietyID, &pass.PassReference, &pass.VisitorName, &phoneRaw,
		&pass.SocietyName, &pass.GateName,
		&pass.UnitIdentifier, &pass.BuildingName,
		&pass.Purpose, &pass.ComingFrom, &pass.VisitorPhotoURL,
		&pass.Status, &createdAt, &pass.CheckInAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPassNotFound
		}
		return nil, fmt.Errorf("failed to query pass: %w", err)
	}

	pass.RegisteredAt = createdAt
	pass.VisitorPhoneMasked = auth.MaskVisitorPhone(phoneRaw)
	return &pass, nil
}

// ListPendingQRRegistrations returns visits pending gatekeeper verification.
// For gatekeepers, visitor phone is strictly masked.
func (r *Repository) ListPendingQRRegistrations(ctx context.Context, societyID uuid.UUID, gateID *uuid.UUID, search string, limit int) ([]PendingQRVisit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, fmt.Sprintf("v.society_id = $%d", argIdx))
	args = append(args, societyID)
	argIdx++

	whereClauses = append(whereClauses, "v.status = 'REGISTERED_PENDING_GATE_VERIFICATION'")

	if gateID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(v.entry_gate_id = $%d OR v.entry_gate_id IS NULL)", argIdx))
		args = append(args, *gateID)
		argIdx++
	}

	search = strings.TrimSpace(search)
	if search != "" {
		searchTerm := "%" + search + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(v.visitor_name ILIKE $%d OR v.pass_reference ILIKE $%d OR v.visitor_phone ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, searchTerm)
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT v.id, v.society_id, v.pass_reference, v.visitor_name, v.visitor_phone,
		       v.visitor_photo_url, v.purpose, v.coming_from,
		       v.entry_gate_id, g.name AS gate_name,
		       v.unit_id, u.unit_identifier, b.name AS building_name,
		       v.status, v.created_at
		FROM society_visits v
		LEFT JOIN society_gates g ON v.entry_gate_id = g.id
		LEFT JOIN society_units u ON v.unit_id = u.id
		LEFT JOIN society_buildings b ON u.building_id = b.id
		WHERE %s
		ORDER BY v.created_at DESC
		LIMIT $%d
	`, strings.Join(whereClauses, " AND "), argIdx)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending QR registrations: %w", err)
	}
	defer rows.Close()

	var list []PendingQRVisit
	for rows.Next() {
		var item PendingQRVisit
		var passRef *string
		if err := rows.Scan(
			&item.ID, &item.SocietyID, &passRef, &item.VisitorName, &item.VisitorPhone,
			&item.VisitorPhotoURL, &item.Purpose, &item.ComingFrom,
			&item.GateID, &item.GateName,
			&item.UnitID, &item.UnitIdentifier, &item.BuildingName,
			&item.Status, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pending QR visit: %w", err)
		}
		if passRef != nil {
			item.PassReference = *passRef
		}
		item.VisitorPhoneMasked = auth.MaskVisitorPhone(item.VisitorPhone)
		item.VisitorPhone = item.VisitorPhoneMasked // Gatekeeper privacy enforcement: do not leak raw phone
		list = append(list, item)
	}

	if list == nil {
		list = []PendingQRVisit{}
	}
	return list, nil
}

// AdmitQRVisitor performs concurrency-safe gatekeeper approval or rejection of a self-registered visitor.
func (r *Repository) AdmitQRVisitor(ctx context.Context, societyID uuid.UUID, visitID uuid.UUID, gateID uuid.UUID, actorID uuid.UUID, decision string, rejectionReason string, clientIP, userAgent string) (*Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	decision = strings.ToUpper(strings.TrimSpace(decision))
	if decision != "APPROVE" && decision != "REJECT" {
		return nil, errors.New("decision must be APPROVE or REJECT")
	}

	// Begin serializable or row-locking transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify gate validity
	var gateDirection, gateStatus string
	err = tx.QueryRow(ctx, `
		SELECT directionality, status
		FROM society_gates
		WHERE id = $1 AND society_id = $2
	`, gateID, societyID).Scan(&gateDirection, &gateStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("gate not found")
		}
		return nil, fmt.Errorf("failed to query gate: %w", err)
	}

	if gateStatus != "ACTIVE" {
		return nil, errors.New("gate is inactive")
	}
	if decision == "APPROVE" && gateDirection == "EGRESS_ONLY" {
		return nil, ErrGateDirectionMismatch
	}

	// Acquire exclusive row lock on visit
	var currentStatus string
	var visitorName, visitorPhone string
	err = tx.QueryRow(ctx, `
		SELECT status, visitor_name, visitor_phone
		FROM society_visits
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, visitID, societyID).Scan(&currentStatus, &visitorName, &visitorPhone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVisitNotFound
		}
		return nil, fmt.Errorf("failed to lock visit record: %w", err)
	}

	if currentStatus != "REGISTERED_PENDING_GATE_VERIFICATION" {
		return nil, ErrVisitAlreadyDecided
	}

	now := time.Now().UTC()
	var updatedStatus string
	var reasonPtr *string

	if decision == "APPROVE" {
		updatedStatus = "CHECKED_IN"

		// Update visit to CHECKED_IN
		_, err = tx.Exec(ctx, `
			UPDATE society_visits
			SET status = $1, check_in_at = $2, entry_gate_id = $3, updated_at = $2
			WHERE id = $4 AND society_id = $5
		`, updatedStatus, now, gateID, visitID, societyID)
		if err != nil {
			return nil, fmt.Errorf("failed to admit visitor: %w", err)
		}

		// Check if actor has an active duty session on this gate
		var sessionID *uuid.UUID
		var activeSessID uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT id FROM society_gate_sessions
			WHERE society_id = $1 AND user_id = $2 AND gate_id = $3 AND status = 'ACTIVE'
			LIMIT 1
		`, societyID, actorID, gateID).Scan(&activeSessID); err == nil && activeSessID != uuid.Nil {
			sessionID = &activeSessID
		}

		// Insert gate entry event
		eventID := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO society_gate_events (
				id, society_id, gate_id, session_id, visit_id, gatekeeper_id, event_type, direction, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, 'ENTRY', 'INGRESS', $7)
		`, eventID, societyID, gateID, sessionID, visitID, actorID, now)
		if err != nil {
			return nil, fmt.Errorf("failed to record entry gate event: %w", err)
		}

		auditDetails, _ := json.Marshal(map[string]interface{}{
			"visit_id":     visitID.String(),
			"society_id":   societyID.String(),
			"gate_id":      gateID.String(),
			"visitor_name": visitorName,
			"decision":     "APPROVE",
		})
		_, _ = tx.Exec(ctx, `
			INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
			VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
		`, actorID, ActionQRVisitAdmitted, string(auditDetails), clientIP, userAgent)

	} else {
		updatedStatus = "REJECTED"
		cleanReason := strings.TrimSpace(rejectionReason)
		if cleanReason == "" {
			cleanReason = "Entry denied by gate security"
		}
		reasonPtr = &cleanReason

		_, err = tx.Exec(ctx, `
			UPDATE society_visits
			SET status = $1, rejection_reason = $2, updated_at = $3
			WHERE id = $4 AND society_id = $5
		`, updatedStatus, reasonPtr, now, visitID, societyID)
		if err != nil {
			return nil, fmt.Errorf("failed to reject visit: %w", err)
		}

		auditDetails, _ := json.Marshal(map[string]interface{}{
			"visit_id":         visitID.String(),
			"society_id":       societyID.String(),
			"gate_id":          gateID.String(),
			"visitor_name":     visitorName,
			"decision":         "REJECT",
			"rejection_reason": cleanReason,
		})
		_, _ = tx.Exec(ctx, `
			INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
			VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
		`, actorID, ActionVisitRejected, string(auditDetails), clientIP, userAgent)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit gatekeeper decision: %w", err)
	}

	// Fetch updated visit to return
	var v Visit
	var uIdent, bName string
	var eGate, xGate, resName *string
	query := `
		SELECT v.id, v.society_id, v.unit_id, u.unit_identifier, b.name AS building_name,
		       v.resident_id, m.first_name || COALESCE(' ' || m.last_name, '') AS resident_name,
		       v.visitor_name, v.visitor_phone, v.visitor_identity_id, v.purpose, v.visit_type, v.status,
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
	_ = r.pool.QueryRow(ctx, query, visitID, societyID).Scan(
		&v.ID, &v.SocietyID, &v.UnitID, &uIdent, &bName,
		&v.ResidentID, &resName,
		&v.VisitorName, &v.VisitorPhone, &v.VisitorIdentityID, &v.Purpose, &v.VisitType, &v.Status,
		&v.VehicleNumber, &v.ExpectedArrival, &v.ValidUntil, &v.CheckInAt, &v.CheckOutAt,
		&v.EntryGateID, &eGate,
		&v.ExitGateID, &xGate,
		&v.RejectionReason, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt,
	)
	v.UnitIdentifier = uIdent
	v.BuildingName = bName
	v.ResidentName = resName
	v.EntryGateName = eGate
	v.ExitGateName = xGate
	v.VisitorPhoneMasked = auth.MaskVisitorPhone(v.VisitorPhone)
	v.VisitorPhone = v.VisitorPhoneMasked

	return &v, nil
}

// ListVisitorSelfHistory returns all visits across societies for an authenticated user
// whose verified phone matches or who is linked to visitor identities.
func (r *Repository) ListVisitorSelfHistory(ctx context.Context, userID uuid.UUID, verifiedPhone string, limit int) ([]Visit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	normPhone, err := auth.NormalizePhone(verifiedPhone)
	if err != nil {
		return nil, ErrInvalidPhoneFormat
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT v.id, v.society_id, s.name AS society_name, v.unit_id, u.unit_identifier, b.name AS building_name,
		       v.visitor_name, v.visitor_phone, v.visitor_identity_id, v.purpose, v.visit_type, v.status,
		       v.vehicle_number, v.check_in_at, v.check_out_at,
		       v.entry_gate_id, eg.name AS entry_gate_name,
		       v.exit_gate_id, xg.name AS exit_gate_name,
		       v.pass_reference, v.visitor_photo_url, v.coming_from,
		       v.created_at, v.updated_at
		FROM society_visits v
		JOIN societies s ON v.society_id = s.id
		JOIN society_units u ON v.unit_id = u.id
		JOIN society_buildings b ON u.building_id = b.id
		LEFT JOIN society_gates eg ON v.entry_gate_id = eg.id
		LEFT JOIN society_gates xg ON v.exit_gate_id = xg.id
		WHERE (
			-- 1. Explicit verified ownership: The visit was created by or registered by the authenticated user
			v.created_by = $2
			OR (
				-- 2. Verified canonical linkage: matching phone AND (linked visitor identity OR unclaimed identity)
				v.visitor_phone = $1
				AND (
					v.visitor_identity_id IN (
						SELECT id FROM society_visitor_identities WHERE user_id = $2
					)
					OR NOT EXISTS (
						SELECT 1 FROM society_visitor_identities vi
						WHERE vi.phone = $1 AND vi.user_id IS NOT NULL AND vi.user_id != $2
					)
				)
				-- 3. Ambiguity exclusion: No other distinct names registered under this phone number
				AND (
					SELECT COUNT(DISTINCT LOWER(TRIM(v2.visitor_name)))
					FROM society_visits v2
					WHERE v2.visitor_phone = $1
				) = 1
				-- 4. Identity confirmation: Authenticated user has a non-empty canonical profile name matching the visit
				AND EXISTS (
					SELECT 1 FROM user_profiles up
					WHERE up.user_id = $2
					  AND TRIM(up.first_name) != ''
					  AND (
						LOWER(TRIM(v.visitor_name)) = LOWER(TRIM(up.first_name || ' ' || up.last_name))
						OR LOWER(TRIM(v.visitor_name)) = LOWER(TRIM(up.first_name))
					  )
				)
			)
		)
		ORDER BY v.created_at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, normPhone, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query visitor history: %w", err)
	}
	defer rows.Close()

	var visits []Visit
	for rows.Next() {
		var v Visit
		var socName string
		var eGate, xGate *string
		err := rows.Scan(
			&v.ID, &v.SocietyID, &socName, &v.UnitID, &v.UnitIdentifier, &v.BuildingName,
			&v.VisitorName, &v.VisitorPhone, &v.VisitorIdentityID, &v.Purpose, &v.VisitType, &v.Status,
			&v.VehicleNumber, &v.CheckInAt, &v.CheckOutAt,
			&v.EntryGateID, &eGate,
			&v.ExitGateID, &xGate,
			&v.PassReference, &v.VisitorPhotoURL, &v.ComingFrom,
			&v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan visit history: %w", err)
		}
		v.EntryGateName = eGate
		v.ExitGateName = xGate
		v.VisitorPhoneMasked = auth.MaskVisitorPhone(v.VisitorPhone)
		visits = append(visits, v)
	}

	if visits == nil {
		visits = []Visit{}
	}
	return visits, nil
}
