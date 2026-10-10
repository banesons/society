package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"chs/backend/internal/auth"
)

var (
	ErrRoleAlreadyExists = errors.New("user already holds a platform role")
	ErrRoleNotFound      = errors.New("platform role not found")
	ErrCannotDisableSelf = errors.New("super admin cannot disable self")
	ErrInvalidRole       = errors.New("invalid platform role")
	ErrInvalidStatus     = errors.New("invalid platform status")
)

const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RoleAdmin      = "ADMIN"

	StatusActive   = "ACTIVE"
	StatusDisabled = "DISABLED"

	ActionSuperAdminBootstrapped = "SUPER_ADMIN_BOOTSTRAPPED"
	ActionAdminCreated           = "ADMIN_CREATED"
	ActionAdminDisabled          = "ADMIN_DISABLED"
	ActionAdminReactivated       = "ADMIN_REACTIVATED"
)

// PlatformRole represents a user's administrative assignment.
type PlatformRole struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// AuditLogEntry represents an immutable platform audit record.
type AuditLogEntry struct {
	ID           uuid.UUID              `json:"id"`
	ActorID      uuid.UUID              `json:"actor_id"`
	Action       string                 `json:"action"`
	TargetUserID *uuid.UUID             `json:"target_user_id,omitempty"`
	Details      map[string]interface{} `json:"details"`
	ClientIP     string                 `json:"client_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// Repository encapsulates database operations for platform administration.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new platform administration repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// BootstrapSuperAdmin idempotently establishes the initial SUPER_ADMIN identity.
// It normalizes the phone number, provisions the user if absent, and assigns SUPER_ADMIN
// without ever silently overwriting an existing assignment.
func (r *Repository) BootstrapSuperAdmin(ctx context.Context, rawPhone string) (*PlatformRole, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	phone, err := auth.NormalizePhone(rawPhone)
	if err != nil {
		return nil, fmt.Errorf("invalid super admin phone: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Ensure user exists in users table
	var userID uuid.UUID
	findUserQuery := `SELECT id FROM users WHERE phone = $1`
	err = tx.QueryRow(ctx, findUserQuery, phone).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			insertUserQuery := `
				INSERT INTO users (phone, status)
				VALUES ($1, 'ACTIVE')
				RETURNING id
			`
			if err := tx.QueryRow(ctx, insertUserQuery, phone).Scan(&userID); err != nil {
				return nil, fmt.Errorf("failed to provision user record: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to query user: %w", err)
		}
	}

	// 2. Check for existing role assignment (Never overwrite!)
	var existingRole PlatformRole
	checkRoleQuery := `
		SELECT id, user_id, role, status, created_by, created_at, updated_at
		FROM platform_roles
		WHERE user_id = $1
	`
	err = tx.QueryRow(ctx, checkRoleQuery, userID).Scan(
		&existingRole.ID,
		&existingRole.UserID,
		&existingRole.Role,
		&existingRole.Status,
		&existingRole.CreatedBy,
		&existingRole.CreatedAt,
		&existingRole.UpdatedAt,
	)
	if err == nil {
		// Already assigned - commit and return without overwrite
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit: %w", err)
		}
		return &existingRole, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to query platform role: %w", err)
	}

	// 3. Assign SUPER_ADMIN role
	var newRole PlatformRole
	insertRoleQuery := `
		INSERT INTO platform_roles (user_id, role, status, created_by)
		VALUES ($1, $2, $3, NULL)
		RETURNING id, user_id, role, status, created_by, created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertRoleQuery, userID, RoleSuperAdmin, StatusActive).Scan(
		&newRole.ID,
		&newRole.UserID,
		&newRole.Role,
		&newRole.Status,
		&newRole.CreatedBy,
		&newRole.CreatedAt,
		&newRole.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert platform role: %w", err)
	}

	// 4. Record audit log transactionally
	auditDetails, _ := json.Marshal(map[string]string{
		"phone":  auth.MaskPhone(phone),
		"role":   RoleSuperAdmin,
		"reason": "Initial platform bootstrap",
	})

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, $2, $3, $4::jsonb)
	`
	if _, err := tx.Exec(ctx, auditQuery, userID, ActionSuperAdminBootstrapped, userID, string(auditDetails)); err != nil {
		return nil, fmt.Errorf("failed to write bootstrap audit record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit bootstrap transaction: %w", err)
	}

	return &newRole, nil
}

// UpdateAdminStatus transactionally updates an admin's status, revokes active sessions if disabled,
// keeps the base user active, and records an immutable audit log.
func (r *Repository) UpdateAdminStatus(
	ctx context.Context,
	actorID, targetUserID uuid.UUID,
	newStatus, reason, clientIP, userAgent string,
) (*PlatformRole, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if actorID == targetUserID && newStatus == StatusDisabled {
		return nil, ErrCannotDisableSelf
	}

	if newStatus != StatusActive && newStatus != StatusDisabled {
		return nil, ErrInvalidStatus
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Fetch current role status
	var currentRole PlatformRole
	queryCurrent := `
		SELECT id, user_id, role, status, created_by, created_at, updated_at
		FROM platform_roles
		WHERE user_id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, queryCurrent, targetUserID).Scan(
		&currentRole.ID,
		&currentRole.UserID,
		&currentRole.Role,
		&currentRole.Status,
		&currentRole.CreatedBy,
		&currentRole.CreatedAt,
		&currentRole.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, fmt.Errorf("failed to query target role: %w", err)
	}

	prevStatus := currentRole.Status

	// 2. Update role status in platform_roles
	updateQuery := `
		UPDATE platform_roles
		SET status = $1, updated_at = NOW()
		WHERE user_id = $2
		RETURNING id, user_id, role, status, created_by, created_at, updated_at
	`
	var updatedRole PlatformRole
	err = tx.QueryRow(ctx, updateQuery, newStatus, targetUserID).Scan(
		&updatedRole.ID,
		&updatedRole.UserID,
		&updatedRole.Role,
		&updatedRole.Status,
		&updatedRole.CreatedBy,
		&updatedRole.CreatedAt,
		&updatedRole.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update platform role: %w", err)
	}

	// 3. If disabling, revoke all active sessions for target user immediately
	if newStatus == StatusDisabled {
		revokeSessionsQuery := `
			UPDATE user_sessions
			SET revoked_at = NOW()
			WHERE user_id = $1 AND revoked_at IS NULL
		`
		if _, err := tx.Exec(ctx, revokeSessionsQuery, targetUserID); err != nil {
			return nil, fmt.Errorf("failed to revoke active sessions: %w", err)
		}
	}

	// 4. Transactionally write immutable audit log
	action := ActionAdminDisabled
	if newStatus == StatusActive {
		action = ActionAdminReactivated
	}

	detailsMap := map[string]interface{}{
		"previous_status": prevStatus,
		"new_status":      newStatus,
		"role":            currentRole.Role,
		"reason":          reason,
	}
	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	if _, err := tx.Exec(ctx, auditQuery, actorID, action, targetUserID, string(detailsJSON), clientIP, userAgent); err != nil {
		return nil, fmt.Errorf("failed to insert audit record: %w", err)
	}

	// 5. Commit transaction (if audit failed, tx is rolled back)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &updatedRole, nil
}

// GetUserPlatformRole loads a user's active platform role.
func (r *Repository) GetUserPlatformRole(ctx context.Context, userID uuid.UUID) (*PlatformRole, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var role PlatformRole
	query := `
		SELECT id, user_id, role, status, created_by, created_at, updated_at
		FROM platform_roles
		WHERE user_id = $1
	`
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&role.ID,
		&role.UserID,
		&role.Role,
		&role.Status,
		&role.CreatedBy,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return &role, nil
}

// GetUserPlatformRoles returns the list of active platform role names for a user (for /auth/me).
func (r *Repository) GetUserPlatformRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var roles []string
	query := `
		SELECT role
		FROM platform_roles
		WHERE user_id = $1 AND status = 'ACTIVE'
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	if roles == nil {
		roles = []string{}
	}

	return roles, nil
}

// AdminListItem holds an admin record with masked contact information.
type AdminListItem struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Phone     string     `json:"phone"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateAdmin creates a new platform ADMIN account by phone number in a transaction.
func (r *Repository) CreateAdmin(ctx context.Context, actorID uuid.UUID, rawPhone, clientIP, userAgent string) (*AdminListItem, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	phone, err := auth.NormalizePhone(rawPhone)
	if err != nil {
		return nil, fmt.Errorf("invalid phone format: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Ensure user exists in users table (JIT provisioning)
	var targetUserID uuid.UUID
	findUserQuery := `SELECT id FROM users WHERE phone = $1`
	err = tx.QueryRow(ctx, findUserQuery, phone).Scan(&targetUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			insertUserQuery := `
				INSERT INTO users (phone, status)
				VALUES ($1, 'ACTIVE')
				RETURNING id
			`
			if err := tx.QueryRow(ctx, insertUserQuery, phone).Scan(&targetUserID); err != nil {
				return nil, fmt.Errorf("failed to provision user record: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to query user: %w", err)
		}
	}

	// 2. Check if user already holds a platform role (one role per user)
	var existingRole string
	checkRoleQuery := `SELECT role FROM platform_roles WHERE user_id = $1`
	err = tx.QueryRow(ctx, checkRoleQuery, targetUserID).Scan(&existingRole)
	if err == nil {
		return nil, ErrRoleAlreadyExists
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing role: %w", err)
	}

	// 3. Insert ADMIN role assignment
	var newRole PlatformRole
	insertRoleQuery := `
		INSERT INTO platform_roles (user_id, role, status, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, role, status, created_by, created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertRoleQuery, targetUserID, RoleAdmin, StatusActive, actorID).Scan(
		&newRole.ID,
		&newRole.UserID,
		&newRole.Role,
		&newRole.Status,
		&newRole.CreatedBy,
		&newRole.CreatedAt,
		&newRole.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert admin role: %w", err)
	}

	// 4. Transactionally write audit log
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"phone":  auth.MaskPhone(phone),
		"role":   RoleAdmin,
		"reason": "Created by Super Admin",
	})

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	if _, err := tx.Exec(ctx, auditQuery, actorID, ActionAdminCreated, targetUserID, string(detailsJSON), clientIP, userAgent); err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit admin creation transaction: %w", err)
	}

	return &AdminListItem{
		ID:        newRole.ID,
		UserID:    newRole.UserID,
		Phone:     auth.MaskPhone(phone),
		Role:      newRole.Role,
		Status:    newRole.Status,
		CreatedBy: newRole.CreatedBy,
		CreatedAt: newRole.CreatedAt,
		UpdatedAt: newRole.UpdatedAt,
	}, nil
}

// ListAdmins returns a paginated list of platform administrators.
func (r *Repository) ListAdmins(ctx context.Context, limit, offset int, roleFilter, statusFilter string) ([]AdminListItem, int, error) {
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

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM platform_roles pr
		WHERE ($1 = '' OR pr.role = $1)
		  AND ($2 = '' OR pr.status = $2)
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, roleFilter, statusFilter).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count admins: %w", err)
	}

	// Query page
	query := `
		SELECT pr.id, pr.user_id, u.phone, pr.role, pr.status, pr.created_by, pr.created_at, pr.updated_at
		FROM platform_roles pr
		JOIN users u ON pr.user_id = u.id
		WHERE ($1 = '' OR pr.role = $1)
		  AND ($2 = '' OR pr.status = $2)
		ORDER BY pr.created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, roleFilter, statusFilter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	var items []AdminListItem
	for rows.Next() {
		var item AdminListItem
		var rawPhone string
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&rawPhone,
			&item.Role,
			&item.Status,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan admin item: %w", err)
		}
		item.Phone = auth.MaskPhone(rawPhone)
		items = append(items, item)
	}

	if items == nil {
		items = []AdminListItem{}
	}

	return items, total, nil
}

// AuditLogItem represents an audit row for list queries.
type AuditLogItem struct {
	ID           uuid.UUID              `json:"id"`
	ActorID      uuid.UUID              `json:"actor_id"`
	Action       string                 `json:"action"`
	TargetUserID *uuid.UUID             `json:"target_user_id,omitempty"`
	Details      map[string]interface{} `json:"details"`
	ClientIP     string                 `json:"client_ip,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// ListAuditLogs returns a paginated list of audit events.
func (r *Repository) ListAuditLogs(ctx context.Context, limit, offset int, actionFilter string) ([]AuditLogItem, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	// Count total
	countQuery := `
		SELECT COUNT(*)
		FROM platform_audit_logs
		WHERE ($1 = '' OR action = $1)
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, actionFilter).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Query page
	query := `
		SELECT id, actor_id, action, target_user_id, details, COALESCE(client_ip, ''), COALESCE(user_agent, ''), created_at
		FROM platform_audit_logs
		WHERE ($1 = '' OR action = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, actionFilter, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var items []AuditLogItem
	for rows.Next() {
		var item AuditLogItem
		var rawDetails []byte
		if err := rows.Scan(
			&item.ID,
			&item.ActorID,
			&item.Action,
			&item.TargetUserID,
			&rawDetails,
			&item.ClientIP,
			&item.UserAgent,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		_ = json.Unmarshal(rawDetails, &item.Details)
		items = append(items, item)
	}

	if items == nil {
		items = []AuditLogItem{}
	}

	return items, total, nil
}

// GetRoleByID finds a platform role record by either its role ID or its user ID.
func (r *Repository) GetRoleByID(ctx context.Context, id uuid.UUID) (*PlatformRole, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var role PlatformRole
	query := `
		SELECT id, user_id, role, status, created_by, created_at, updated_at
		FROM platform_roles
		WHERE id = $1 OR user_id = $1
	`
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&role.ID,
		&role.UserID,
		&role.Role,
		&role.Status,
		&role.CreatedBy,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return &role, nil
}
