package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"chs/backend/internal/auth"
)

// UpdateGate updates operational parameters of a physical gate.
func (r *Repository) UpdateGate(ctx context.Context, societyID, gateID uuid.UUID, params UpdateGateParams, actorID uuid.UUID, clientIP, userAgent string) (*Gate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify gate exists and belongs to society under lock
	var existing Gate
	queryExisting := `
		SELECT id, society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description, created_at, updated_at
		FROM society_gates
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, queryExisting, gateID, societyID).Scan(
		&existing.ID, &existing.SocietyID, &existing.Name, &existing.GateCode, &existing.GateType,
		&existing.Status, &existing.Directionality, &existing.OperationalStatus,
		&existing.RequiredStaffPerShift, &existing.Description, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGateNotFound
		}
		return nil, fmt.Errorf("failed to fetch gate: %w", err)
	}

	name := existing.Name
	if params.Name != nil && strings.TrimSpace(*params.Name) != "" {
		name = strings.TrimSpace(*params.Name)
	}

	directionality := existing.Directionality
	if params.Directionality != nil {
		dir := strings.ToUpper(strings.TrimSpace(*params.Directionality))
		if dir != "BIDIRECTIONAL" && dir != "INGRESS_ONLY" && dir != "EGRESS_ONLY" {
			return nil, errors.New("invalid directionality")
		}
		directionality = dir
	}

	opStatus := existing.OperationalStatus
	if params.OperationalStatus != nil {
		st := strings.ToUpper(strings.TrimSpace(*params.OperationalStatus))
		if st != "ACTIVE" && st != "INACTIVE" && st != "MAINTENANCE" && st != "EMERGENCY_OPEN" {
			return nil, errors.New("invalid operational status")
		}
		opStatus = st
	}

	staffCount := existing.RequiredStaffPerShift
	if params.RequiredStaffPerShift != nil {
		if *params.RequiredStaffPerShift < 0 {
			return nil, errors.New("required staff per shift cannot be negative")
		}
		staffCount = *params.RequiredStaffPerShift
	}

	description := existing.Description
	if params.Description != nil {
		description = params.Description
	}

	updateQuery := `
		UPDATE society_gates
		SET name = $1, directionality = $2, operational_status = $3, required_staff_per_shift = $4, description = $5, updated_at = NOW()
		WHERE id = $6 AND society_id = $7
		RETURNING id, society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description, created_at, updated_at
	`
	updated := &Gate{}
	err = tx.QueryRow(ctx, updateQuery, name, directionality, opStatus, staffCount, description, gateID, societyID).Scan(
		&updated.ID, &updated.SocietyID, &updated.Name, &updated.GateCode, &updated.GateType,
		&updated.Status, &updated.Directionality, &updated.OperationalStatus,
		&updated.RequiredStaffPerShift, &updated.Description, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update gate: %w", err)
	}

	auditDetails := map[string]interface{}{
		"gate_id":                  gateID.String(),
		"society_id":               societyID.String(),
		"name":                     updated.Name,
		"directionality":           updated.Directionality,
		"operational_status":       updated.OperationalStatus,
		"required_staff_per_shift": updated.RequiredStaffPerShift,
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionGateUpdated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return updated, nil
}

// ListGatekeepers retrieves assigned gatekeeper personnel for a society.
func (r *Repository) ListGatekeepers(ctx context.Context, societyID uuid.UUID, statusFilter string) ([]SocietyAdminAssignment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT sur.id, sur.society_id, sur.user_id, COALESCE(u.phone, ''), sur.role, sur.status,
		       sur.assigned_by, sur.revoked_by, sur.revoked_at, sur.created_at, sur.updated_at
		FROM society_user_roles sur
		JOIN users u ON u.id = sur.user_id
		WHERE sur.society_id = $1 AND sur.role = 'GATEKEEPER'
	`
	args := []interface{}{societyID}

	if statusFilter != "" && strings.ToUpper(statusFilter) != "ALL" {
		query += " AND sur.status = $2"
		args = append(args, strings.ToUpper(statusFilter))
	} else if statusFilter == "" {
		query += " AND sur.status = 'ACTIVE'"
	}

	query += " ORDER BY sur.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query gatekeepers: %w", err)
	}
	defer rows.Close()

	items := make([]SocietyAdminAssignment, 0)
	for rows.Next() {
		var a SocietyAdminAssignment
		var rawPhone string
		err := rows.Scan(
			&a.ID, &a.SocietyID, &a.UserID, &rawPhone, &a.Role, &a.Status,
			&a.AssignedBy, &a.RevokedBy, &a.RevokedAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gatekeeper: %w", err)
		}
		a.UserPhoneMasked = rawPhone
		items = append(items, a)
	}

	return items, nil
}

// AssignGatekeeper assigns a user as GATEKEEPER in the specified society.
func (r *Repository) AssignGatekeeper(ctx context.Context, societyID uuid.UUID, phone string, actorID uuid.UUID, clientIP, userAgent string) (*SocietyAdminAssignment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	normPhone, err := auth.NormalizePhone(phone)
	if err != nil {
		return nil, errors.New("invalid phone format: " + err.Error())
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Resolve or create user by phone
	var targetUserID uuid.UUID
	userQuery := `SELECT id FROM users WHERE phone = $1`
	err = tx.QueryRow(ctx, userQuery, normPhone).Scan(&targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			createUserQuery := `
				INSERT INTO users (phone, status)
				VALUES ($1, 'ACTIVE')
				RETURNING id
			`
			err = tx.QueryRow(ctx, createUserQuery, normPhone).Scan(&targetUserID)
			if err != nil {
				return nil, fmt.Errorf("failed to provision user: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to resolve user: %w", err)
		}
	}

	// 2. Check if user already holds active GATEKEEPER role
	var activeCount int
	checkQuery := `
		SELECT COUNT(*)
		FROM society_user_roles
		WHERE society_id = $1 AND user_id = $2 AND role = 'GATEKEEPER' AND status = 'ACTIVE'
	`
	err = tx.QueryRow(ctx, checkQuery, societyID, targetUserID).Scan(&activeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing role: %w", err)
	}
	if activeCount > 0 {
		return nil, ErrGatekeeperAlreadyAssigned
	}

	// 3. Insert assignment
	insertQuery := `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)
		RETURNING id, society_id, user_id, role, status, assigned_by, revoked_by, revoked_at, created_at, updated_at
	`
	a := &SocietyAdminAssignment{}
	err = tx.QueryRow(ctx, insertQuery, societyID, targetUserID, actorID).Scan(
		&a.ID, &a.SocietyID, &a.UserID, &a.Role, &a.Status,
		&a.AssignedBy, &a.RevokedBy, &a.RevokedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert gatekeeper role: %w", err)
	}
	a.UserPhoneMasked = normPhone

	// 4. Audit log
	details := map[string]interface{}{
		"assignment_id": a.ID.String(),
		"society_id":    societyID.String(),
		"user_id":       targetUserID.String(),
		"phone":         normPhone,
		"role":          "GATEKEEPER",
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionGatekeeperAssigned, targetUserID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return a, nil
}

// RevokeGatekeeper revokes a gatekeeper assignment and closes active duty sessions.
func (r *Repository) RevokeGatekeeper(ctx context.Context, societyID, assignmentID uuid.UUID, actorID uuid.UUID, clientIP, userAgent string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var targetUserID uuid.UUID
	updateQuery := `
		UPDATE society_user_roles
		SET status = 'REVOKED', revoked_by = $1, revoked_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND society_id = $3 AND role = 'GATEKEEPER' AND status = 'ACTIVE'
		RETURNING user_id
	`
	err = tx.QueryRow(ctx, updateQuery, actorID, assignmentID, societyID).Scan(&targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGatekeeperNotFound
		}
		return fmt.Errorf("failed to revoke gatekeeper: %w", err)
	}

	// Close any active duty sessions for this guard in this society
	sessionCloseQuery := `
		UPDATE society_gate_sessions
		SET status = 'ENDED', ended_at = NOW(), handover_notes = 'Session terminated upon role revocation', updated_at = NOW()
		WHERE society_id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`
	_, err = tx.Exec(ctx, sessionCloseQuery, societyID, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to terminate active sessions: %w", err)
	}

	details := map[string]interface{}{
		"assignment_id": assignmentID.String(),
		"society_id":    societyID.String(),
		"target_user":   targetUserID.String(),
		"role":          "GATEKEEPER",
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionGatekeeperRevoked, targetUserID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListGateShifts returns all shifts for a society.
func (r *Repository) ListGateShifts(ctx context.Context, societyID uuid.UUID) ([]GateShift, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_id, name, start_time::text, end_time::text, is_active, created_at, updated_at
		FROM society_gate_shifts
		WHERE society_id = $1
		ORDER BY start_time ASC
	`
	rows, err := r.pool.Query(ctx, query, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shifts: %w", err)
	}
	defer rows.Close()

	items := make([]GateShift, 0)
	for rows.Next() {
		var s GateShift
		err := rows.Scan(
			&s.ID, &s.SocietyID, &s.Name, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shift: %w", err)
		}
		items = append(items, s)
	}

	return items, nil
}

// CreateGateShift defines a recurring shift template for a society.
func (r *Repository) CreateGateShift(ctx context.Context, societyID uuid.UUID, params CreateGateShiftParams, actorID uuid.UUID, clientIP, userAgent string) (*GateShift, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("shift name cannot be blank")
	}

	startTime := strings.TrimSpace(params.StartTime)
	endTime := strings.TrimSpace(params.EndTime)
	if startTime == "" || endTime == "" {
		return nil, errors.New("start_time and end_time are required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	insertQuery := `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, $2, $3::time, $4::time, TRUE)
		RETURNING id, society_id, name, start_time::text, end_time::text, is_active, created_at, updated_at
	`
	s := &GateShift{}
	err = tx.QueryRow(ctx, insertQuery, societyID, name, startTime, endTime).Scan(
		&s.ID, &s.SocietyID, &s.Name, &s.StartTime, &s.EndTime, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrShiftNameExists
		}
		return nil, fmt.Errorf("failed to insert shift: %w", err)
	}

	details := map[string]interface{}{
		"shift_id":   s.ID.String(),
		"society_id": societyID.String(),
		"name":       s.Name,
		"start_time": s.StartTime,
		"end_time":   s.EndTime,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionShiftCreated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s, nil
}

// ListDutySessions retrieves duty sessions with optional gate and status filters.
func (r *Repository) ListDutySessions(ctx context.Context, societyID uuid.UUID, gateID *uuid.UUID, statusFilter string) ([]GateDutySession, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT gs.id, gs.society_id, gs.gate_id, g.name AS gate_name, g.gate_code,
		       gs.user_id, COALESCE(u.phone, '') AS user_phone,
		       gs.shift_id, s.name AS shift_name,
		       gs.started_at, gs.ended_at, gs.status, gs.handover_notes,
		       gs.created_at, gs.updated_at
		FROM society_gate_sessions gs
		JOIN society_gates g ON g.id = gs.gate_id
		JOIN users u ON u.id = gs.user_id
		LEFT JOIN society_gate_shifts s ON s.id = gs.shift_id
		WHERE gs.society_id = $1
	`
	args := []interface{}{societyID}

	if gateID != nil && *gateID != uuid.Nil {
		query += fmt.Sprintf(" AND gs.gate_id = $%d", len(args)+1)
		args = append(args, *gateID)
	}

	if statusFilter != "" && strings.ToUpper(statusFilter) != "ALL" {
		query += fmt.Sprintf(" AND gs.status = $%d", len(args)+1)
		args = append(args, strings.ToUpper(statusFilter))
	} else if statusFilter == "" {
		query += " AND gs.status = 'ACTIVE'"
	}

	query += " ORDER BY gs.started_at DESC LIMIT 100"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query duty sessions: %w", err)
	}
	defer rows.Close()

	items := make([]GateDutySession, 0)
	for rows.Next() {
		var ds GateDutySession
		err := rows.Scan(
			&ds.ID, &ds.SocietyID, &ds.GateID, &ds.GateName, &ds.GateCode,
			&ds.UserID, &ds.UserPhone,
			&ds.ShiftID, &ds.ShiftName,
			&ds.StartedAt, &ds.EndedAt, &ds.Status, &ds.HandoverNotes,
			&ds.CreatedAt, &ds.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan duty session: %w", err)
		}
		items = append(items, ds)
	}

	return items, nil
}

// CheckInGateDuty starts an active duty session for a guard at a gate.
func (r *Repository) CheckInGateDuty(ctx context.Context, societyID, gateID, userID uuid.UUID, shiftID *uuid.UUID, clientIP, userAgent string) (*GateDutySession, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Verify gate exists, is active, and is open operationally
	var gateStatus, opStatus, gateName, gateCode string
	gateQuery := `
		SELECT name, gate_code, status, operational_status
		FROM society_gates
		WHERE id = $1 AND society_id = $2
	`
	err = tx.QueryRow(ctx, gateQuery, gateID, societyID).Scan(&gateName, &gateCode, &gateStatus, &opStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGateNotFound
		}
		return nil, fmt.Errorf("failed to resolve gate: %w", err)
	}
	if gateStatus != "ACTIVE" || opStatus == "INACTIVE" {
		return nil, ErrGateInactive
	}

	// 2. Check if guard already holds an active session in this society
	var activeCount int
	checkSession := `
		SELECT COUNT(*)
		FROM society_gate_sessions
		WHERE society_id = $1 AND user_id = $2 AND status = 'ACTIVE'
	`
	err = tx.QueryRow(ctx, checkSession, societyID, userID).Scan(&activeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing sessions: %w", err)
	}
	if activeCount > 0 {
		return nil, ErrSessionAlreadyActive
	}

	// 3. Insert new duty session
	insertQuery := `
		INSERT INTO society_gate_sessions (society_id, gate_id, user_id, shift_id, started_at, status)
		VALUES ($1, $2, $3, $4, NOW(), 'ACTIVE')
		RETURNING id, society_id, gate_id, user_id, shift_id, started_at, ended_at, status, handover_notes, created_at, updated_at
	`
	ds := &GateDutySession{
		GateName: gateName,
		GateCode: gateCode,
	}
	err = tx.QueryRow(ctx, insertQuery, societyID, gateID, userID, shiftID).Scan(
		&ds.ID, &ds.SocietyID, &ds.GateID, &ds.UserID, &ds.ShiftID,
		&ds.StartedAt, &ds.EndedAt, &ds.Status, &ds.HandoverNotes,
		&ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSessionAlreadyActive
		}
		return nil, fmt.Errorf("failed to insert duty session: %w", err)
	}

	// Fetch user phone
	_ = tx.QueryRow(ctx, `SELECT COALESCE(phone, '') FROM users WHERE id = $1`, userID).Scan(&ds.UserPhone)

	// Fetch shift name if shift_id provided
	if shiftID != nil {
		_ = tx.QueryRow(ctx, `SELECT name FROM society_gate_shifts WHERE id = $1`, *shiftID).Scan(&ds.ShiftName)
	}

	// 4. Audit log
	details := map[string]interface{}{
		"session_id": ds.ID.String(),
		"society_id": societyID.String(),
		"gate_id":    gateID.String(),
		"user_id":    userID.String(),
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, userID, ActionGateDutyStarted, userID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ds, nil
}

// CheckOutGateDuty concludes an active duty session for a guard at a gate.
func (r *Repository) CheckOutGateDuty(ctx context.Context, societyID, gateID, userID uuid.UUID, handoverNotes *string, clientIP, userAgent string) (*GateDutySession, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := `
		UPDATE society_gate_sessions
		SET status = 'ENDED', ended_at = NOW(), handover_notes = $1, updated_at = NOW()
		WHERE society_id = $2 AND gate_id = $3 AND user_id = $4 AND status = 'ACTIVE'
		RETURNING id, society_id, gate_id, user_id, shift_id, started_at, ended_at, status, handover_notes, created_at, updated_at
	`
	ds := &GateDutySession{}
	err = tx.QueryRow(ctx, updateQuery, handoverNotes, societyID, gateID, userID).Scan(
		&ds.ID, &ds.SocietyID, &ds.GateID, &ds.UserID, &ds.ShiftID,
		&ds.StartedAt, &ds.EndedAt, &ds.Status, &ds.HandoverNotes,
		&ds.CreatedAt, &ds.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to close duty session: %w", err)
	}

	// Fetch gate details
	_ = tx.QueryRow(ctx, `SELECT name, gate_code FROM society_gates WHERE id = $1`, ds.GateID).Scan(&ds.GateName, &ds.GateCode)
	// Fetch user phone
	_ = tx.QueryRow(ctx, `SELECT COALESCE(phone, '') FROM users WHERE id = $1`, userID).Scan(&ds.UserPhone)

	details := map[string]interface{}{
		"session_id":     ds.ID.String(),
		"society_id":     societyID.String(),
		"gate_id":        gateID.String(),
		"user_id":        userID.String(),
		"handover_notes": handoverNotes,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, userID, ActionGateDutyEnded, userID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return ds, nil
}
