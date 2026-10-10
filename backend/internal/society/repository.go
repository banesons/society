package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"chs/backend/internal/auth"
)

var pincodeRegex = regexp.MustCompile(`^[1-9][0-9]{5}$`)

// Repository manages database persistence and lifecycle operations for housing societies.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Society repository instance.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// NormalizeCreateInput validates and standardizes parameters for a new society.
func NormalizeCreateInput(params CreateSocietyParams) (CreateSocietyParams, error) {
	name := strings.TrimSpace(params.Name)
	if len(name) < 3 || len(name) > 255 {
		return params, ErrInvalidName
	}

	addr1 := strings.TrimSpace(params.AddressLine1)
	locality := strings.TrimSpace(params.Locality)
	city := strings.TrimSpace(params.City)
	state := strings.TrimSpace(params.State)
	pincode := strings.TrimSpace(params.Pincode)

	if addr1 == "" || locality == "" || city == "" || state == "" {
		return params, ErrRequiredFieldMissing
	}

	if !pincodeRegex.MatchString(pincode) {
		return params, ErrInvalidPincode
	}

	var addr2 *string
	if params.AddressLine2 != nil {
		trimmed := strings.TrimSpace(*params.AddressLine2)
		if trimmed != "" {
			addr2 = &trimmed
		}
	}

	var regNum *string
	if params.RegistrationNumber != nil {
		trimmed := strings.TrimSpace(*params.RegistrationNumber)
		if trimmed != "" {
			regNum = &trimmed
		}
	}

	var email *string
	if params.ContactEmail != nil {
		trimmed := strings.TrimSpace(*params.ContactEmail)
		if trimmed != "" {
			email = &trimmed
		}
	}

	var phone *string
	if params.ContactPhone != nil {
		trimmed := strings.TrimSpace(*params.ContactPhone)
		if trimmed != "" {
			normalizedPhone, err := auth.NormalizePhone(trimmed)
			if err != nil {
				return params, fmt.Errorf("invalid contact phone: %w", err)
			}
			phone = &normalizedPhone
		}
	}

	return CreateSocietyParams{
		Name:               name,
		RegistrationNumber: regNum,
		AddressLine1:       addr1,
		AddressLine2:       addr2,
		Locality:           locality,
		City:               city,
		State:              state,
		Pincode:            pincode,
		ContactEmail:       email,
		ContactPhone:       phone,
	}, nil
}

// CreateSociety registers a new society with status PENDING, generating a collision-resistant code
// and atomically writing an audit log record.
func (r *Repository) CreateSociety(ctx context.Context, rawParams CreateSocietyParams, actorID uuid.UUID, clientIP, userAgent string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	params, err := NormalizeCreateInput(rawParams)
	if err != nil {
		return nil, err
	}

	const maxCollisionRetries = 5

	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		code, err := GenerateSocietyCode(params.City)
		if err != nil {
			return nil, err
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		insertQuery := `
			INSERT INTO societies (
				society_code, name, registration_number,
				address_line1, address_line2, locality, city, state, pincode,
				contact_email, contact_phone, status, created_by, updated_by
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'PENDING', $12, $12)
			RETURNING id, society_code, name, registration_number,
			          address_line1, address_line2, locality, city, state, pincode,
			          contact_email, contact_phone, status, created_by, updated_by,
			          created_at, updated_at
		`

		soc := &Society{}
		err = tx.QueryRow(ctx, insertQuery,
			code, params.Name, params.RegistrationNumber,
			params.AddressLine1, params.AddressLine2, params.Locality, params.City, params.State, params.Pincode,
			params.ContactEmail, params.ContactPhone, actorID,
		).Scan(
			&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
			&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
			&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
			&soc.CreatedAt, &soc.UpdatedAt,
		)

		if err != nil {
			tx.Rollback(ctx)
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				// Unique violation. If society_code, retry with a fresh code.
				if strings.Contains(pgErr.ConstraintName, "society_code") || strings.Contains(pgErr.ConstraintName, "code_unique") {
					continue
				}
				if strings.Contains(pgErr.ConstraintName, "reg_number") {
					return nil, ErrRegNumberExists
				}
			}
			return nil, fmt.Errorf("failed to insert society: %w", err)
		}

		// Insert audit record transactionally
		auditDetails := map[string]interface{}{
			"society_id":   soc.ID.String(),
			"society_code": soc.SocietyCode,
			"name":         soc.Name,
			"city":         soc.City,
			"state":        soc.State,
			"pincode":      soc.Pincode,
			"status":       soc.Status,
		}
		detailsJSON, err := json.Marshal(auditDetails)
		if err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("failed to marshal audit details: %w", err)
		}

		auditQuery := `
			INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
			VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
		`
		_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietyCreated, string(detailsJSON), clientIP, userAgent)
		if err != nil {
			tx.Rollback(ctx)
			return nil, fmt.Errorf("failed to record audit log: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return soc, nil
	}

	return nil, ErrMaxRetriesExceeded
}

// GetSocietyByID retrieves a single society by its primary UUID.
func (r *Repository) GetSocietyByID(ctx context.Context, id uuid.UUID) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_code, name, registration_number,
		       address_line1, address_line2, locality, city, state, pincode,
		       contact_email, contact_phone, status, created_by, updated_by,
		       created_at, updated_at
		FROM societies
		WHERE id = $1
	`

	soc := &Society{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to query society: %w", err)
	}

	return soc, nil
}

// GetSocietyByCode retrieves a single society by its human-readable code.
func (r *Repository) GetSocietyByCode(ctx context.Context, code string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_code, name, registration_number,
		       address_line1, address_line2, locality, city, state, pincode,
		       contact_email, contact_phone, status, created_by, updated_by,
		       created_at, updated_at
		FROM societies
		WHERE upper(society_code) = upper($1)
	`

	soc := &Society{}
	err := r.pool.QueryRow(ctx, query, code).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to query society by code: %w", err)
	}

	return soc, nil
}

// ListSocieties retrieves a paginated roster of societies with optional status and search filters.
func (r *Repository) ListSocieties(ctx context.Context, filter ListSocietiesFilter) ([]*Society, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	statusFilter := strings.TrimSpace(filter.Status)
	searchTerm := strings.TrimSpace(filter.Search)
	searchPattern := ""
	if searchTerm != "" {
		searchPattern = "%" + searchTerm + "%"
	}

	countQuery := `
		SELECT COUNT(*)
		FROM societies
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR name ILIKE $2 OR society_code ILIKE $2)
	`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, statusFilter, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count societies: %w", err)
	}

	query := `
		SELECT id, society_code, name, registration_number,
		       address_line1, address_line2, locality, city, state, pincode,
		       contact_email, contact_phone, status, created_by, updated_by,
		       created_at, updated_at
		FROM societies
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR name ILIKE $2 OR society_code ILIKE $2)
		ORDER BY created_at DESC, id ASC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, statusFilter, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query societies: %w", err)
	}
	defer rows.Close()

	items := make([]*Society, 0)
	for rows.Next() {
		soc := &Society{}
		err := rows.Scan(
			&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
			&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
			&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
			&soc.CreatedAt, &soc.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan society row: %w", err)
		}
		items = append(items, soc)
	}

	return items, total, nil
}

// UpdateSocietyProfile updates permitted profile fields for an existing society.
func (r *Repository) UpdateSocietyProfile(ctx context.Context, id uuid.UUID, rawParams UpdateSocietyParams, actorID uuid.UUID, clientIP, userAgent string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Validate inputs using NormalizeCreateInput semantics
	normParams, err := NormalizeCreateInput(CreateSocietyParams{
		Name:               rawParams.Name,
		RegistrationNumber: rawParams.RegistrationNumber,
		AddressLine1:       rawParams.AddressLine1,
		AddressLine2:       rawParams.AddressLine2,
		Locality:           rawParams.Locality,
		City:               rawParams.City,
		State:              rawParams.State,
		Pincode:            rawParams.Pincode,
		ContactEmail:       rawParams.ContactEmail,
		ContactPhone:       rawParams.ContactPhone,
	})
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify society exists with row lock
	var existingCode, existingStatus string
	lockQuery := `SELECT society_code, status FROM societies WHERE id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, lockQuery, id).Scan(&existingCode, &existingStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to lock society: %w", err)
	}

	updateQuery := `
		UPDATE societies
		SET name = $1,
		    registration_number = $2,
		    address_line1 = $3,
		    address_line2 = $4,
		    locality = $5,
		    city = $6,
		    state = $7,
		    pincode = $8,
		    contact_email = $9,
		    contact_phone = $10,
		    updated_by = $11,
		    updated_at = NOW()
		WHERE id = $12
		RETURNING id, society_code, name, registration_number,
		          address_line1, address_line2, locality, city, state, pincode,
		          contact_email, contact_phone, status, created_by, updated_by,
		          created_at, updated_at
	`

	soc := &Society{}
	err = tx.QueryRow(ctx, updateQuery,
		normParams.Name, normParams.RegistrationNumber,
		normParams.AddressLine1, normParams.AddressLine2, normParams.Locality,
		normParams.City, normParams.State, normParams.Pincode,
		normParams.ContactEmail, normParams.ContactPhone,
		actorID, id,
	).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update society: %w", err)
	}

	auditDetails := map[string]interface{}{
		"society_id":   soc.ID.String(),
		"society_code": soc.SocietyCode,
		"name":         soc.Name,
		"city":         soc.City,
		"state":        soc.State,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietyProfileUpdated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return soc, nil
}

// ActivateSociety transitions a society from PENDING to ACTIVE (or from SUSPENDED to ACTIVE).
func (r *Repository) ActivateSociety(ctx context.Context, id uuid.UUID, reason string, actorID uuid.UUID, clientIP, userAgent string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus, societyCode string
	lockQuery := `SELECT status, society_code FROM societies WHERE id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, lockQuery, id).Scan(&currentStatus, &societyCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to lock society: %w", err)
	}

	if currentStatus == StatusActive {
		return nil, ErrAlreadyInStatus
	}

	auditAction := ActionSocietyActivated
	if currentStatus == StatusSuspended {
		auditAction = ActionSocietyReactivated
	}

	updateQuery := `
		UPDATE societies
		SET status = 'ACTIVE',
		    updated_by = $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING id, society_code, name, registration_number,
		          address_line1, address_line2, locality, city, state, pincode,
		          contact_email, contact_phone, status, created_by, updated_by,
		          created_at, updated_at
	`

	soc := &Society{}
	err = tx.QueryRow(ctx, updateQuery, actorID, id).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to activate society: %w", err)
	}

	auditDetails := map[string]interface{}{
		"society_id":      soc.ID.String(),
		"society_code":    soc.SocietyCode,
		"previous_status": currentStatus,
		"new_status":      StatusActive,
		"reason":          strings.TrimSpace(reason),
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, auditAction, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return soc, nil
}

// ListSuspensionReasons retrieves active suspension reasons.
// If onlyManual is true, filters to reasons permitted for manual administration.
func (r *Repository) ListSuspensionReasons(ctx context.Context, onlyManual bool) ([]*SuspensionReason, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, code, label, description, category, allows_manual, allows_automated, is_active, created_at, updated_at
		FROM society_suspension_reasons
		WHERE is_active = TRUE
	`
	if onlyManual {
		query += ` AND allows_manual = TRUE`
	}
	query += ` ORDER BY id ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query suspension reasons: %w", err)
	}
	defer rows.Close()

	var reasons []*SuspensionReason
	for rows.Next() {
		sr := &SuspensionReason{}
		if err := rows.Scan(
			&sr.ID, &sr.Code, &sr.Label, &sr.Description, &sr.Category,
			&sr.AllowsManual, &sr.AllowsAutomated, &sr.IsActive,
			&sr.CreatedAt, &sr.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan suspension reason: %w", err)
		}
		reasons = append(reasons, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading suspension reason rows: %w", err)
	}

	return reasons, nil
}

// GetSuspensionReason retrieves a specific suspension reason record by code.
func (r *Repository) GetSuspensionReason(ctx context.Context, code string) (*SuspensionReason, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedCode := strings.TrimSpace(code)
	if trimmedCode == "" {
		return nil, ErrSuspensionReasonRequired
	}

	query := `
		SELECT id, code, label, description, category, allows_manual, allows_automated, is_active, created_at, updated_at
		FROM society_suspension_reasons
		WHERE upper(code) = upper($1)
	`
	sr := &SuspensionReason{}
	err := r.pool.QueryRow(ctx, query, trimmedCode).Scan(
		&sr.ID, &sr.Code, &sr.Label, &sr.Description, &sr.Category,
		&sr.AllowsManual, &sr.AllowsAutomated, &sr.IsActive,
		&sr.CreatedAt, &sr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidSuspensionReason
		}
		return nil, fmt.Errorf("failed to query suspension reason: %w", err)
	}

	return sr, nil
}

// GetSuspensionReasonByID retrieves a specific suspension reason record by its surrogate numeric ID.
func (r *Repository) GetSuspensionReasonByID(ctx context.Context, id int16) (*SuspensionReason, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if id <= 0 {
		return nil, ErrInvalidSuspensionReason
	}

	query := `
		SELECT id, code, label, description, category, allows_manual, allows_automated, is_active, created_at, updated_at
		FROM society_suspension_reasons
		WHERE id = $1
	`
	sr := &SuspensionReason{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sr.ID, &sr.Code, &sr.Label, &sr.Description, &sr.Category,
		&sr.AllowsManual, &sr.AllowsAutomated, &sr.IsActive,
		&sr.CreatedAt, &sr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidSuspensionReason
		}
		return nil, fmt.Errorf("failed to query suspension reason by id: %w", err)
	}

	return sr, nil
}

// SuspendSociety transitions an ACTIVE society to SUSPENDED.
// It requires a valid, active suspension reason from the catalogue that allows manual administrative use.
func (r *Repository) SuspendSociety(ctx context.Context, id uuid.UUID, params SuspendSocietyParams, actorID uuid.UUID, clientIP, userAgent string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var reason *SuspensionReason
	var err error
	if params.ReasonID != nil && *params.ReasonID > 0 {
		reason, err = r.GetSuspensionReasonByID(ctx, *params.ReasonID)
		if err != nil {
			return nil, err
		}
	} else if params.ReasonCode != nil && strings.TrimSpace(*params.ReasonCode) != "" {
		reason, err = r.GetSuspensionReason(ctx, *params.ReasonCode)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrSuspensionReasonRequired
	}

	if !reason.IsActive {
		return nil, ErrSuspensionReasonInactive
	}
	if !reason.AllowsManual {
		return nil, ErrManualSuspensionNotAllowed
	}

	var trimmedDetails *string
	if params.ReasonDetails != nil {
		d := strings.TrimSpace(*params.ReasonDetails)
		if len(d) > 1000 {
			return nil, ErrDetailsTooLong
		}
		if d != "" {
			trimmedDetails = &d
		}
	}

	if reason.Code == "OTHER_REVIEW_REQUIRED" && (trimmedDetails == nil || *trimmedDetails == "") {
		return nil, ErrSuspensionDetailsRequired
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus, societyCode string
	lockQuery := `SELECT status, society_code FROM societies WHERE id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, lockQuery, id).Scan(&currentStatus, &societyCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to lock society: %w", err)
	}

	if currentStatus == StatusSuspended {
		return nil, ErrAlreadyInStatus
	}
	if currentStatus == StatusPending {
		return nil, ErrInvalidStatusTransition
	}

	updateQuery := `
		UPDATE societies
		SET status = 'SUSPENDED',
		    updated_by = $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING id, society_code, name, registration_number,
		          address_line1, address_line2, locality, city, state, pincode,
		          contact_email, contact_phone, status, created_by, updated_by,
		          created_at, updated_at
	`

	soc := &Society{}
	err = tx.QueryRow(ctx, updateQuery, actorID, id).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to suspend society: %w", err)
	}

	auditDetails := map[string]interface{}{
		"society_id":            soc.ID.String(),
		"society_code":          soc.SocietyCode,
		"previous_status":       currentStatus,
		"new_status":            StatusSuspended,
		"reason_id":             reason.ID,
		"reason_code":           reason.Code,
		"reason_label_snapshot": reason.Label,
		"suspension_origin":     OriginManual,
	}
	if trimmedDetails != nil {
		auditDetails["reason_details"] = *trimmedDetails
	}

	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietySuspended, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return soc, nil
}

// ReactivateSociety transitions a SUSPENDED society to ACTIVE.
func (r *Repository) ReactivateSociety(ctx context.Context, id uuid.UUID, reason string, actorID uuid.UUID, clientIP, userAgent string) (*Society, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus, societyCode string
	lockQuery := `SELECT status, society_code FROM societies WHERE id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, lockQuery, id).Scan(&currentStatus, &societyCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to lock society: %w", err)
	}

	if currentStatus == StatusActive {
		return nil, ErrAlreadyInStatus
	}
	if currentStatus == StatusPending {
		return nil, ErrInvalidStatusTransition
	}

	updateQuery := `
		UPDATE societies
		SET status = 'ACTIVE',
		    updated_by = $1,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING id, society_code, name, registration_number,
		          address_line1, address_line2, locality, city, state, pincode,
		          contact_email, contact_phone, status, created_by, updated_by,
		          created_at, updated_at
	`

	soc := &Society{}
	err = tx.QueryRow(ctx, updateQuery, actorID, id).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to reactivate society: %w", err)
	}

	auditDetails := map[string]interface{}{
		"society_id":      soc.ID.String(),
		"society_code":    soc.SocietyCode,
		"previous_status": currentStatus,
		"new_status":      StatusActive,
		"reason":          strings.TrimSpace(reason),
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietyReactivated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return soc, nil
}

// AssignSocietyAdmin creates an active society administrator mapping and records a transactional audit log.
func (r *Repository) AssignSocietyAdmin(ctx context.Context, params AssignSocietyAdminParams, actorID uuid.UUID, clientIP, userAgent string) (*SocietyAdminAssignment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	role := params.Role
	if role == "" {
		role = RoleSocietyAdmin
	}
	if role != RoleSocietyAdmin {
		return nil, ErrInvalidRole
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify society exists
	var socID uuid.UUID
	var socCode, socName, socStatus string
	societyQuery := `SELECT id, society_code, name, status FROM societies WHERE id = $1`
	err = tx.QueryRow(ctx, societyQuery, params.SocietyID).Scan(&socID, &socCode, &socName, &socStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocietyNotFound
		}
		return nil, fmt.Errorf("failed to query society: %w", err)
	}

	// Resolve target user
	var targetUserID uuid.UUID
	var targetPhone, targetStatus string
	if params.Phone != "" {
		normPhone, err := auth.NormalizePhone(params.Phone)
		if err != nil {
			return nil, err
		}
		userQuery := `SELECT id, phone, status FROM users WHERE phone = $1`
		err = tx.QueryRow(ctx, userQuery, normPhone).Scan(&targetUserID, &targetPhone, &targetStatus)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, fmt.Errorf("failed to query user by phone: %w", err)
		}
	} else if params.UserID != nil {
		userQuery := `SELECT id, phone, status FROM users WHERE id = $1`
		err = tx.QueryRow(ctx, userQuery, *params.UserID).Scan(&targetUserID, &targetPhone, &targetStatus)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, fmt.Errorf("failed to query user by id: %w", err)
		}
	} else {
		return nil, ErrRequiredFieldMissing
	}

	// Insert active assignment into society_user_roles
	insertQuery := `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, $3, 'ACTIVE', $4)
		RETURNING id, society_id, user_id, role, status, assigned_by, revoked_by, revoked_at, created_at, updated_at
	`
	assignment := &SocietyAdminAssignment{}
	err = tx.QueryRow(ctx, insertQuery, socID, targetUserID, role, actorID).Scan(
		&assignment.ID, &assignment.SocietyID, &assignment.UserID,
		&assignment.Role, &assignment.Status, &assignment.AssignedBy,
		&assignment.RevokedBy, &assignment.RevokedAt,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrSocietyAdminAlreadyAssigned
		}
		return nil, fmt.Errorf("failed to insert society user role: %w", err)
	}

	// Record transactional audit log
	auditDetails := map[string]interface{}{
		"assignment_id":       assignment.ID.String(),
		"society_id":          socID.String(),
		"society_code":        socCode,
		"target_user_id":      targetUserID.String(),
		"target_phone_masked": auth.MaskPhone(targetPhone),
		"role":                assignment.Role,
		"status":              assignment.Status,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietyAdminAssigned, targetUserID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	assignment.UserPhoneMasked = auth.MaskPhone(targetPhone)
	return assignment, nil
}

// ListSocietyAdmins retrieves paginated administrators for a given society.
func (r *Repository) ListSocietyAdmins(ctx context.Context, filter ListSocietyAdminsFilter) (*SocietyAdminListResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Verify society exists
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM societies WHERE id = $1)`, filter.SocietyID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check society existence: %w", err)
	}
	if !exists {
		return nil, ErrSocietyNotFound
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(*)
		FROM society_user_roles
		WHERE society_id = $1 AND ($2 = '' OR status = $2)
	`
	var total int
	err = r.pool.QueryRow(ctx, countQuery, filter.SocietyID, filter.Status).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count society administrators: %w", err)
	}

	query := `
		SELECT sur.id, sur.society_id, sur.user_id, u.phone, sur.role, sur.status,
		       sur.assigned_by, sur.revoked_by, sur.revoked_at, sur.created_at, sur.updated_at
		FROM society_user_roles sur
		JOIN users u ON sur.user_id = u.id
		WHERE sur.society_id = $1 AND ($2 = '' OR sur.status = $2)
		ORDER BY sur.created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, filter.SocietyID, filter.Status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query society administrators: %w", err)
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
			return nil, fmt.Errorf("failed to scan society administrator: %w", err)
		}
		a.UserPhoneMasked = auth.MaskPhone(rawPhone)
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating society administrators: %w", err)
	}

	return &SocietyAdminListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// RevokeSocietyAdmin revokes an existing administrator assignment and writes an audit log transactionally.
func (r *Repository) RevokeSocietyAdmin(ctx context.Context, societyID, assignmentID, actorID uuid.UUID, clientIP, userAgent string) (*SocietyAdminAssignment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch current assignment with society and user information under lock
	selectQuery := `
		SELECT sur.id, sur.society_id, sur.user_id, u.phone, s.society_code, sur.role, sur.status
		FROM society_user_roles sur
		JOIN societies s ON sur.society_id = s.id
		JOIN users u ON sur.user_id = u.id
		WHERE sur.id = $1 AND sur.society_id = $2
		FOR UPDATE OF sur
	`
	var id, socID, userID uuid.UUID
	var rawPhone, socCode, role, status string
	err = tx.QueryRow(ctx, selectQuery, assignmentID, societyID).Scan(
		&id, &socID, &userID, &rawPhone, &socCode, &role, &status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssignmentNotFound
		}
		return nil, fmt.Errorf("failed to query assignment: %w", err)
	}

	if status == AssignmentStatusRevoked {
		return nil, ErrAssignmentAlreadyRevoked
	}

	// Update assignment to REVOKED
	updateQuery := `
		UPDATE society_user_roles
		SET status = 'REVOKED', revoked_by = $1, revoked_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND status = 'ACTIVE'
		RETURNING id, society_id, user_id, role, status, assigned_by, revoked_by, revoked_at, created_at, updated_at
	`
	assignment := &SocietyAdminAssignment{}
	err = tx.QueryRow(ctx, updateQuery, actorID, assignmentID).Scan(
		&assignment.ID, &assignment.SocietyID, &assignment.UserID,
		&assignment.Role, &assignment.Status, &assignment.AssignedBy,
		&assignment.RevokedBy, &assignment.RevokedAt,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke society administrator: %w", err)
	}

	// Transactional audit log
	auditDetails := map[string]interface{}{
		"assignment_id":       assignment.ID.String(),
		"society_id":          socID.String(),
		"society_code":        socCode,
		"target_user_id":      userID.String(),
		"target_phone_masked": auth.MaskPhone(rawPhone),
		"role":                assignment.Role,
		"status":              AssignmentStatusRevoked,
		"previous_status":     AssignmentStatusActive,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionSocietyAdminRevoked, userID, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	assignment.UserPhoneMasked = auth.MaskPhone(rawPhone)
	return assignment, nil
}

// ListUserSocietyAssignments returns all active society assignments for an authenticated user.
func (r *Repository) ListUserSocietyAssignments(ctx context.Context, userID uuid.UUID) ([]UserSocietyAssignment, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT s.id, s.society_code, s.name, sur.role, sur.status
		FROM society_user_roles sur
		JOIN societies s ON sur.society_id = s.id
		WHERE sur.user_id = $1 AND sur.status = 'ACTIVE'
		ORDER BY s.name ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user society assignments: %w", err)
	}
	defer rows.Close()

	items := make([]UserSocietyAssignment, 0)
	for rows.Next() {
		var a UserSocietyAssignment
		if err := rows.Scan(&a.SocietyID, &a.SocietyCode, &a.SocietyName, &a.Role, &a.Status); err != nil {
			return nil, fmt.Errorf("failed to scan user society assignment: %w", err)
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user society assignments: %w", err)
	}

	return items, nil
}

// VerifySocietyAdminAccess validates that target society exists, is ACTIVE, and the user has an ACTIVE assignment.
func (r *Repository) VerifySocietyAdminAccess(ctx context.Context, societyID, userID uuid.UUID, requiredRole string) (*Society, *SocietyAdminAssignment, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	// 1. Fetch society
	societyQuery := `
		SELECT id, society_code, name, registration_number,
		       address_line1, address_line2, locality, city, state, pincode,
		       contact_email, contact_phone, status, created_by, updated_by,
		       created_at, updated_at
		FROM societies
		WHERE id = $1
	`
	soc := &Society{}
	err := r.pool.QueryRow(ctx, societyQuery, societyID).Scan(
		&soc.ID, &soc.SocietyCode, &soc.Name, &soc.RegistrationNumber,
		&soc.AddressLine1, &soc.AddressLine2, &soc.Locality, &soc.City, &soc.State, &soc.Pincode,
		&soc.ContactEmail, &soc.ContactPhone, &soc.Status, &soc.CreatedBy, &soc.UpdatedBy,
		&soc.CreatedAt, &soc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrSocietyNotFound
		}
		return nil, nil, fmt.Errorf("failed to query society: %w", err)
	}

	if soc.Status == StatusPending {
		return nil, nil, ErrSocietyPending
	}
	if soc.Status == StatusSuspended {
		return nil, nil, ErrSocietySuspended
	}
	if soc.Status != StatusActive {
		return nil, nil, ErrSocietyAccessDenied
	}

	// 2. Fetch active assignment
	assignQuery := `
		SELECT id, society_id, user_id, role, status, assigned_by, revoked_by, revoked_at, created_at, updated_at
		FROM society_user_roles
		WHERE society_id = $1 AND user_id = $2 AND role = $3 AND status = 'ACTIVE'
	`
	assignment := &SocietyAdminAssignment{}
	err = r.pool.QueryRow(ctx, assignQuery, societyID, userID, requiredRole).Scan(
		&assignment.ID, &assignment.SocietyID, &assignment.UserID,
		&assignment.Role, &assignment.Status, &assignment.AssignedBy,
		&assignment.RevokedBy, &assignment.RevokedAt,
		&assignment.CreatedAt, &assignment.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrSocietyAccessDenied
		}
		return nil, nil, fmt.Errorf("failed to query society user role: %w", err)
	}

	return soc, assignment, nil
}
