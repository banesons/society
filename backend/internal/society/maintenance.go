package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateChargeCategory creates a new society-scoped charge category.
func (r *Repository) CreateChargeCategory(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateChargeCategoryParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceChargeCategory, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("category name is required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check for unique name within society
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_charge_categories
			WHERE society_id = $1 AND LOWER(TRIM(name)) = LOWER($2)
		)
	`, societyID, name).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check category existence: %w", err)
	}
	if exists {
		return nil, ErrChargeCategoryNameExists
	}

	cat := MaintenanceChargeCategory{
		SocietyID: societyID,
		Name:      name,
		Status:    "ACTIVE",
	}
	if params.Description != nil && strings.TrimSpace(*params.Description) != "" {
		desc := strings.TrimSpace(*params.Description)
		cat.Description = &desc
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_charge_categories (society_id, name, description, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, societyID, cat.Name, cat.Description, cat.Status).Scan(&cat.ID, &cat.CreatedAt, &cat.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert charge category: %w", err)
	}

	// Transactional Audit Log
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":  societyID,
		"category_id": cat.ID,
		"name":        cat.Name,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionChargeCategoryCreated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &cat, nil
}

// UpdateChargeCategory updates category name, description, or status.
func (r *Repository) UpdateChargeCategory(
	ctx context.Context,
	societyID uuid.UUID,
	categoryID uuid.UUID,
	params UpdateChargeCategoryParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceChargeCategory, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("category name is required")
	}

	status := strings.ToUpper(strings.TrimSpace(params.Status))
	if status != "ACTIVE" && status != "INACTIVE" {
		status = "ACTIVE"
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check name collision with another category
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_charge_categories
			WHERE society_id = $1 AND id != $2 AND LOWER(TRIM(name)) = LOWER($3)
		)
	`, societyID, categoryID, name).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check category name uniqueness: %w", err)
	}
	if exists {
		return nil, ErrChargeCategoryNameExists
	}

	var cat MaintenanceChargeCategory
	err = tx.QueryRow(ctx, `
		UPDATE maintenance_charge_categories
		SET name = $1, description = $2, status = $3, updated_at = NOW()
		WHERE id = $4 AND society_id = $5
		RETURNING id, society_id, name, description, status, created_at, updated_at
	`, name, params.Description, status, categoryID, societyID).Scan(
		&cat.ID, &cat.SocietyID, &cat.Name, &cat.Description, &cat.Status, &cat.CreatedAt, &cat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChargeCategoryNotFound
		}
		return nil, fmt.Errorf("update charge category: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":  societyID,
		"category_id": cat.ID,
		"name":        cat.Name,
		"status":      cat.Status,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionChargeCategoryUpdated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &cat, nil
}

// ListChargeCategories lists charge categories for a society.
func (r *Repository) ListChargeCategories(
	ctx context.Context,
	societyID uuid.UUID,
	statusFilter string,
) ([]MaintenanceChargeCategory, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_id, name, description, status, created_at, updated_at
		FROM maintenance_charge_categories
		WHERE society_id = $1
	`
	args := []interface{}{societyID}

	if statusFilter != "" && !strings.EqualFold(statusFilter, "ALL") {
		query += " AND status = $2"
		args = append(args, strings.ToUpper(statusFilter))
	}

	query += " ORDER BY name ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var categories []MaintenanceChargeCategory
	for rows.Next() {
		var c MaintenanceChargeCategory
		if err := rows.Scan(&c.ID, &c.SocietyID, &c.Name, &c.Description, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, c)
	}

	if categories == nil {
		categories = []MaintenanceChargeCategory{}
	}
	return categories, nil
}

// CreateChargeConfig creates a new calculation rule.
func (r *Repository) CreateChargeConfig(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateChargeConfigParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceChargeConfig, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("charge configuration name is required")
	}
	if params.Amount <= 0 {
		return nil, ErrInvalidAmount
	}

	chargeType := strings.ToUpper(strings.TrimSpace(params.ChargeType))
	if chargeType != ChargeTypeFlatRecurring && chargeType != ChargeTypeUnitTypeRecurring && chargeType != ChargeTypeOneOff {
		chargeType = ChargeTypeFlatRecurring
	}

	billingFreq := strings.ToUpper(strings.TrimSpace(params.BillingFrequency))
	if billingFreq != FrequencyMonthly && billingFreq != FrequencyQuarterly && billingFreq != FrequencyAnnually && billingFreq != FrequencyOneOff {
		billingFreq = FrequencyMonthly
	}

	if params.EffectiveUntil != nil && params.EffectiveUntil.Before(params.EffectiveFrom) {
		return nil, errors.New("effective_until cannot be before effective_from")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Validate category exists in this society and is ACTIVE
	var catName string
	var catStatus string
	err = tx.QueryRow(ctx, `
		SELECT name, status FROM maintenance_charge_categories
		WHERE id = $1 AND society_id = $2
	`, params.CategoryID, societyID).Scan(&catName, &catStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChargeCategoryNotFound
		}
		return nil, fmt.Errorf("validate category: %w", err)
	}
	if catStatus != "ACTIVE" {
		return nil, errors.New("cannot associate charge with inactive category")
	}

	cfg := MaintenanceChargeConfig{
		SocietyID:          societyID,
		CategoryID:         params.CategoryID,
		CategoryName:       catName,
		Name:               name,
		ChargeType:         chargeType,
		ApplicableUnitType: params.ApplicableUnitType,
		Amount:             params.Amount,
		BillingFrequency:   billingFreq,
		EffectiveFrom:      params.EffectiveFrom,
		EffectiveUntil:     params.EffectiveUntil,
		Status:             "ACTIVE",
		CreatedBy:          actorID,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_charge_configs (
			society_id, category_id, name, charge_type, applicable_unit_type,
			amount, billing_frequency, effective_from, effective_until, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, societyID, cfg.CategoryID, cfg.Name, cfg.ChargeType, cfg.ApplicableUnitType,
		cfg.Amount, cfg.BillingFrequency, cfg.EffectiveFrom, cfg.EffectiveUntil, cfg.Status, cfg.CreatedBy,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert charge config: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":  societyID,
		"config_id":   cfg.ID,
		"category_id": cfg.CategoryID,
		"name":        cfg.Name,
		"amount":      cfg.Amount,
		"charge_type": cfg.ChargeType,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionChargeConfigCreated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &cfg, nil
}

// DeactivateChargeConfig deactivates a charge configuration.
func (r *Repository) DeactivateChargeConfig(
	ctx context.Context,
	societyID uuid.UUID,
	configID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		UPDATE maintenance_charge_configs
		SET status = 'INACTIVE', updated_at = NOW()
		WHERE id = $1 AND society_id = $2
	`, configID, societyID)
	if err != nil {
		return fmt.Errorf("update charge config status: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrChargeConfigNotFound
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id": societyID,
		"config_id":  configID,
		"status":     "INACTIVE",
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionChargeConfigDeactivated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("audit log: %w", err)
	}

	return tx.Commit(ctx)
}

// ListChargeConfigs lists charge configurations for a society.
func (r *Repository) ListChargeConfigs(
	ctx context.Context,
	societyID uuid.UUID,
	statusFilter string,
) ([]MaintenanceChargeConfig, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT cfg.id, cfg.society_id, cfg.category_id, cat.name as category_name,
		       cfg.name, cfg.charge_type, cfg.applicable_unit_type, cfg.amount,
		       cfg.billing_frequency, cfg.effective_from, cfg.effective_until,
		       cfg.status, cfg.created_by, cfg.created_at, cfg.updated_at
		FROM maintenance_charge_configs cfg
		JOIN maintenance_charge_categories cat ON cfg.category_id = cat.id AND cfg.society_id = cat.society_id
		WHERE cfg.society_id = $1
	`
	args := []interface{}{societyID}

	if statusFilter != "" && !strings.EqualFold(statusFilter, "ALL") {
		query += " AND cfg.status = $2"
		args = append(args, strings.ToUpper(statusFilter))
	}

	query += " ORDER BY cfg.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query charge configs: %w", err)
	}
	defer rows.Close()

	var configs []MaintenanceChargeConfig
	for rows.Next() {
		var c MaintenanceChargeConfig
		if err := rows.Scan(
			&c.ID, &c.SocietyID, &c.CategoryID, &c.CategoryName,
			&c.Name, &c.ChargeType, &c.ApplicableUnitType, &c.Amount,
			&c.BillingFrequency, &c.EffectiveFrom, &c.EffectiveUntil,
			&c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan charge config: %w", err)
		}
		configs = append(configs, c)
	}

	if configs == nil {
		configs = []MaintenanceChargeConfig{}
	}
	return configs, nil
}

// CreateBillingPeriod creates a new billing window.
func (r *Repository) CreateBillingPeriod(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateBillingPeriodParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillingPeriod, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	periodName := strings.TrimSpace(params.PeriodName)
	if periodName == "" {
		return nil, errors.New("period name is required")
	}

	if params.PeriodEnd.Before(params.PeriodStart) || params.DueDate.Before(params.PeriodStart) {
		return nil, ErrInvalidBillingDates
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check unique dates
	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_billing_periods
			WHERE society_id = $1 AND period_start = $2 AND period_end = $3
		)
	`, societyID, params.PeriodStart, params.PeriodEnd).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check period date collision: %w", err)
	}
	if exists {
		return nil, ErrBillingPeriodOverlap
	}

	period := MaintenanceBillingPeriod{
		SocietyID:   societyID,
		PeriodName:  periodName,
		PeriodStart: params.PeriodStart,
		PeriodEnd:   params.PeriodEnd,
		DueDate:     params.DueDate,
		Status:      "ACTIVE",
		CreatedBy:   actorID,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_billing_periods (
			society_id, period_name, period_start, period_end, due_date, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, societyID, period.PeriodName, period.PeriodStart, period.PeriodEnd, period.DueDate, period.Status, period.CreatedBy).Scan(
		&period.ID, &period.CreatedAt, &period.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert billing period: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":   societyID,
		"period_id":    period.ID,
		"period_name":  period.PeriodName,
		"period_start": period.PeriodStart,
		"period_end":   period.PeriodEnd,
		"due_date":     period.DueDate,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionBillingPeriodCreated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &period, nil
}

// ListBillingPeriods lists all billing periods for a society.
func (r *Repository) ListBillingPeriods(ctx context.Context, societyID uuid.UUID) ([]MaintenanceBillingPeriod, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, society_id, period_name, period_start, period_end, due_date, status, created_by, created_at, updated_at
		FROM maintenance_billing_periods
		WHERE society_id = $1
		ORDER BY period_start DESC
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("query billing periods: %w", err)
	}
	defer rows.Close()

	var periods []MaintenanceBillingPeriod
	for rows.Next() {
		var p MaintenanceBillingPeriod
		if err := rows.Scan(
			&p.ID, &p.SocietyID, &p.PeriodName, &p.PeriodStart, &p.PeriodEnd, &p.DueDate,
			&p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan billing period: %w", err)
		}
		periods = append(periods, p)
	}

	if periods == nil {
		periods = []MaintenanceBillingPeriod{}
	}
	return periods, nil
}

// GetBillingPeriod retrieves a single billing period.
func (r *Repository) GetBillingPeriod(ctx context.Context, societyID, periodID uuid.UUID) (*MaintenanceBillingPeriod, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var p MaintenanceBillingPeriod
	err := r.pool.QueryRow(ctx, `
		SELECT id, society_id, period_name, period_start, period_end, due_date, status, created_by, created_at, updated_at
		FROM maintenance_billing_periods
		WHERE id = $1 AND society_id = $2
	`, periodID, societyID).Scan(
		&p.ID, &p.SocietyID, &p.PeriodName, &p.PeriodStart, &p.PeriodEnd, &p.DueDate,
		&p.Status, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingPeriodNotFound
		}
		return nil, fmt.Errorf("get billing period: %w", err)
	}

	return &p, nil
}

// PreviewBillGeneration previews the scope and charges for a generation run.
func (r *Repository) PreviewBillGeneration(
	ctx context.Context,
	societyID, periodID uuid.UUID,
) (*BillGenerationPreview, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	period, err := r.GetBillingPeriod(ctx, societyID, periodID)
	if err != nil {
		return nil, err
	}

	// Count total active units in society
	var totalUnits int
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM society_units
		WHERE society_id = $1 AND status = 'ACTIVE'
	`, societyID).Scan(&totalUnits)
	if err != nil {
		return nil, fmt.Errorf("count active units: %w", err)
	}

	// Count units already billed for this period
	var alreadyBilled int
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM maintenance_bills
		WHERE society_id = $1 AND billing_period_id = $2 AND status != 'VOIDED'
	`, societyID, periodID).Scan(&alreadyBilled)
	if err != nil {
		return nil, fmt.Errorf("count already billed units: %w", err)
	}

	eligibleUnits := totalUnits - alreadyBilled
	if eligibleUnits < 0 {
		eligibleUnits = 0
	}

	// Fetch applicable active configs overlapping period
	configs, err := r.ListChargeConfigs(ctx, societyID, "ACTIVE")
	if err != nil {
		return nil, fmt.Errorf("list configs: %w", err)
	}

	var applicable []MaintenanceChargeConfig
	var projectedRatePerUnit float64

	for _, c := range configs {
		// Verify effective window overlap
		if c.EffectiveFrom.After(period.PeriodEnd) {
			continue
		}
		if c.EffectiveUntil != nil && c.EffectiveUntil.Before(period.PeriodStart) {
			continue
		}
		applicable = append(applicable, c)
		projectedRatePerUnit += c.Amount
	}

	projectedTotal := projectedRatePerUnit * float64(eligibleUnits)
	// Round to 2 decimals
	projectedTotal = math.Round(projectedTotal*100) / 100

	if applicable == nil {
		applicable = []MaintenanceChargeConfig{}
	}

	return &BillGenerationPreview{
		PeriodID:             period.ID,
		PeriodName:           period.PeriodName,
		TotalUnits:           totalUnits,
		AlreadyBilledUnits:   alreadyBilled,
		EligibleUnits:        eligibleUnits,
		ProjectedTotalAmount: projectedTotal,
		ApplicableConfigs:    applicable,
	}, nil
}

// GenerateBills generates maintenance bills for all unbilled active units in the period.
func (r *Repository) GenerateBills(
	ctx context.Context,
	societyID, periodID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*BillGenerationResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch and lock period
	var period MaintenanceBillingPeriod
	err = tx.QueryRow(ctx, `
		SELECT id, society_id, period_name, period_start, period_end, due_date, status
		FROM maintenance_billing_periods
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, periodID, societyID).Scan(
		&period.ID, &period.SocietyID, &period.PeriodName, &period.PeriodStart, &period.PeriodEnd, &period.DueDate, &period.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingPeriodNotFound
		}
		return nil, fmt.Errorf("get period: %w", err)
	}

	if period.Status == "CLOSED" {
		return nil, errors.New("cannot generate bills for a closed billing period")
	}

	// Fetch society code for human-readable bill numbers
	var societyCode string
	err = tx.QueryRow(ctx, `SELECT society_code FROM societies WHERE id = $1`, societyID).Scan(&societyCode)
	if err != nil {
		return nil, fmt.Errorf("get society code: %w", err)
	}
	codeParts := strings.Split(societyCode, "-")
	shortCode := societyCode
	if len(codeParts) >= 2 {
		shortCode = codeParts[1]
	}

	// Fetch applicable active configs overlapping period
	type configWithCat struct {
		id                 uuid.UUID
		categoryID         uuid.UUID
		categoryName       string
		name               string
		chargeType         string
		applicableUnitType *string
		amount             float64
	}

	cfgRows, err := tx.Query(ctx, `
		SELECT cfg.id, cfg.category_id, cat.name, cfg.name, cfg.charge_type, cfg.applicable_unit_type, cfg.amount
		FROM maintenance_charge_configs cfg
		JOIN maintenance_charge_categories cat ON cfg.category_id = cat.id AND cfg.society_id = cat.society_id
		WHERE cfg.society_id = $1 
		  AND cfg.status = 'ACTIVE'
		  AND cat.status = 'ACTIVE'
		  AND cfg.effective_from <= $2
		  AND (cfg.effective_until IS NULL OR cfg.effective_until >= $3)
		ORDER BY cfg.name ASC
	`, societyID, period.PeriodEnd, period.PeriodStart)
	if err != nil {
		return nil, fmt.Errorf("query applicable configs: %w", err)
	}
	defer cfgRows.Close()

	var activeConfigs []configWithCat
	for cfgRows.Next() {
		var c configWithCat
		if err := cfgRows.Scan(&c.id, &c.categoryID, &c.categoryName, &c.name, &c.chargeType, &c.applicableUnitType, &c.amount); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		activeConfigs = append(activeConfigs, c)
	}
	cfgRows.Close()

	if len(activeConfigs) == 0 {
		return nil, ErrNoEligibleCharges
	}

	// Fetch active units
	type unitRecord struct {
		id             uuid.UUID
		unitIdentifier string
		unitType       string
	}

	unitRows, err := tx.Query(ctx, `
		SELECT id, unit_identifier, unit_type
		FROM society_units
		WHERE society_id = $1 AND status = 'ACTIVE'
		ORDER BY unit_identifier ASC
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("query active units: %w", err)
	}
	defer unitRows.Close()

	var units []unitRecord
	for unitRows.Next() {
		var u unitRecord
		if err := unitRows.Scan(&u.id, &u.unitIdentifier, &u.unitType); err != nil {
			return nil, fmt.Errorf("scan unit: %w", err)
		}
		units = append(units, u)
	}
	unitRows.Close()

	// Fetch units that already have an active/issued bill for this period
	existingBills := make(map[uuid.UUID]bool)
	exRows, err := tx.Query(ctx, `
		SELECT unit_id FROM maintenance_bills
		WHERE society_id = $1 AND billing_period_id = $2 AND status != 'VOIDED'
	`, societyID, periodID)
	if err != nil {
		return nil, fmt.Errorf("query existing bills: %w", err)
	}
	defer exRows.Close()

	for exRows.Next() {
		var uID uuid.UUID
		if err := exRows.Scan(&uID); err == nil {
			existingBills[uID] = true
		}
	}
	exRows.Close()

	billsCreated := 0
	unitsSkipped := 0
	var totalBilledAmount float64

	monthYearPrefix := period.PeriodStart.Format("200601")

	for _, unit := range units {
		if existingBills[unit.id] {
			unitsSkipped++
			continue
		}

		// Calculate matching charges for this unit
		type lineItemDraft struct {
			configID     uuid.UUID
			categoryID   uuid.UUID
			categoryName string
			description  string
			rate         float64
			amount       float64
		}

		var items []lineItemDraft
		var billTotal float64

		for _, cfg := range activeConfigs {
			applies := false
			switch cfg.chargeType {
			case ChargeTypeFlatRecurring, ChargeTypeOneOff:
				applies = true
			case ChargeTypeUnitTypeRecurring:
				if cfg.applicableUnitType == nil || *cfg.applicableUnitType == unit.unitType {
					applies = true
				}
			}

			if applies {
				exactAmount := math.Round(cfg.amount*100) / 100
				items = append(items, lineItemDraft{
					configID:     cfg.id,
					categoryID:   cfg.categoryID,
					categoryName: cfg.categoryName,
					description:  cfg.name,
					rate:         exactAmount,
					amount:       exactAmount,
				})
				billTotal += exactAmount
			}
		}

		if len(items) == 0 {
			// No charges match this unit type
			unitsSkipped++
			continue
		}

		billTotal = math.Round(billTotal*100) / 100

		// Sequence-backed, concurrency-safe bill number
		var billSeq int64
		err = tx.QueryRow(ctx, "SELECT nextval('maintenance_bill_seq')").Scan(&billSeq)
		if err != nil {
			return nil, fmt.Errorf("nextval bill sequence: %w", err)
		}
		sanitizedUnitID := strings.ReplaceAll(unit.unitIdentifier, " ", "-")
		billNumber := fmt.Sprintf("BILL-%s-%s-%s-%04d", shortCode, monthYearPrefix, sanitizedUnitID, billSeq%10000)

		var billID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO maintenance_bills (
				society_id, unit_id, billing_period_id, bill_number, issue_date, due_date,
				currency, subtotal, total_amount, status, created_by
			) VALUES ($1, $2, $3, $4, CURRENT_DATE, $5, 'INR', $6, $7, 'ISSUED', $8)
			RETURNING id
		`, societyID, unit.id, periodID, billNumber, period.DueDate, billTotal, billTotal, actorID).Scan(&billID)
		if err != nil {
			return nil, fmt.Errorf("insert maintenance bill for unit %s: %w", unit.unitIdentifier, err)
		}

		// Insert line items
		for _, item := range items {
			_, err = tx.Exec(ctx, `
				INSERT INTO maintenance_bill_items (
					bill_id, society_id, charge_category_id, category_name,
					charge_config_id, description, rate, quantity, amount
				) VALUES ($1, $2, $3, $4, $5, $6, $7, 1.00, $8)
			`, billID, societyID, item.categoryID, item.categoryName, item.configID, item.description, item.rate, item.amount)
			if err != nil {
				return nil, fmt.Errorf("insert line item: %w", err)
			}
		}

		// Enqueue notification outbox event for unit owners/occupants
		ownerRows, qErr := tx.Query(ctx, `
			SELECT DISTINCT sm.user_id, COALESCE(sm.email, '')
			FROM unit_relationships ur
			JOIN society_members sm ON ur.member_id = sm.id
			WHERE ur.unit_id = $1 AND ur.society_id = $2
			  AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
			  AND sm.user_id IS NOT NULL
			  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
		`, unit.id, societyID)
		if qErr == nil {
			for ownerRows.Next() {
				var ownerUID uuid.UUID
				var ownerEmail string
				if err := ownerRows.Scan(&ownerUID, &ownerEmail); err == nil {
					outboxPayload := map[string]interface{}{
						"recipient_user_id": ownerUID.String(),
						"recipient_email":   ownerEmail,
						"society_id":        societyID.String(),
						"bill_id":           billID.String(),
						"bill_number":       billNumber,
						"period_name":       period.PeriodName,
						"due_date":          period.DueDate.Format("2006-01-02"),
						"amount":            fmt.Sprintf("%.2f", billTotal),
					}
					_ = r.EnqueueNotificationOutbox(ctx, tx, "BILL_ISSUED", fmt.Sprintf("BILL_ISS:%s:%s", billID.String(), ownerUID.String()), outboxPayload)
				}
			}
			ownerRows.Close()
		}

		billsCreated++
		totalBilledAmount += billTotal
	}

	totalBilledAmount = math.Round(totalBilledAmount*100) / 100

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":          societyID,
		"period_id":           periodID,
		"bills_created":       billsCreated,
		"units_skipped":       unitsSkipped,
		"total_billed_amount": totalBilledAmount,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionBillsGenerated, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &BillGenerationResult{
		PeriodID:             periodID,
		BillsCreated:         billsCreated,
		UnitsSkippedExisting: unitsSkipped,
		TotalBilledAmount:    totalBilledAmount,
	}, nil
}

// ListBills lists bills in the admin bills register with pagination and filters.
func (r *Repository) ListBills(
	ctx context.Context,
	societyID uuid.UUID,
	filter ListBillsFilter,
) (*ListBillsResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
		FROM maintenance_bills b
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		JOIN society_buildings bld ON u.building_id = bld.id AND u.society_id = bld.society_id
		JOIN maintenance_billing_periods bp ON b.billing_period_id = bp.id AND b.society_id = bp.society_id
		WHERE b.society_id = $1
	`
	args := []interface{}{societyID}
	argIdx := 2

	if filter.PeriodID != nil {
		baseQuery += fmt.Sprintf(" AND b.billing_period_id = $%d", argIdx)
		args = append(args, *filter.PeriodID)
		argIdx++
	}

	if filter.UnitID != nil {
		baseQuery += fmt.Sprintf(" AND b.unit_id = $%d", argIdx)
		args = append(args, *filter.UnitID)
		argIdx++
	}

	if filter.Status != "" && !strings.EqualFold(filter.Status, "ALL") {
		st := strings.ToUpper(filter.Status)
		if st == "OVERDUE" {
			baseQuery += " AND b.status = 'ISSUED' AND b.due_date < CURRENT_DATE"
		} else if st == "ISSUED" {
			baseQuery += " AND b.status = 'ISSUED' AND b.due_date >= CURRENT_DATE"
		} else {
			baseQuery += fmt.Sprintf(" AND b.status = $%d", argIdx)
			args = append(args, st)
			argIdx++
		}
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (u.unit_identifier ILIKE $%d OR b.bill_number ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count bills: %w", err)
	}

	selectQuery := `
		SELECT b.id, b.society_id, b.unit_id, u.unit_identifier, bld.name as building_name, u.floor_number,
		       b.billing_period_id, bp.period_name, b.bill_number, b.issue_date, b.due_date,
		       b.currency, b.subtotal, b.total_amount, b.status,
		       CASE 
		           WHEN b.status = 'VOID' THEN 'VOID'
		           WHEN b.status = 'ISSUED' AND b.due_date < CURRENT_DATE THEN 'OVERDUE'
		           ELSE b.status
		       END as derived_status,
		       b.created_by, b.voided_by, b.voided_at, b.void_reason, b.created_at, b.updated_at
	` + baseQuery + fmt.Sprintf(" ORDER BY b.due_date DESC, b.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query bills: %w", err)
	}
	defer rows.Close()

	var bills []MaintenanceBill
	for rows.Next() {
		var b MaintenanceBill
		if err := rows.Scan(
			&b.ID, &b.SocietyID, &b.UnitID, &b.UnitIdentifier, &b.BuildingName, &b.FloorNumber,
			&b.BillingPeriodID, &b.PeriodName, &b.BillNumber, &b.IssueDate, &b.DueDate,
			&b.Currency, &b.Subtotal, &b.TotalAmount, &b.Status, &b.DerivedStatus,
			&b.CreatedBy, &b.VoidedBy, &b.VoidedAt, &b.VoidReason, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bill: %w", err)
		}
		bills = append(bills, b)
	}

	if bills == nil {
		bills = []MaintenanceBill{}
	}

	return &ListBillsResult{
		Items:  bills,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetBillDetails retrieves full bill details and line items.
func (r *Repository) GetBillDetails(
	ctx context.Context,
	societyID, billID uuid.UUID,
) (*MaintenanceBill, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var b MaintenanceBill
	err := r.pool.QueryRow(ctx, `
		SELECT b.id, b.society_id, b.unit_id, u.unit_identifier, bld.name as building_name, u.floor_number,
		       b.billing_period_id, bp.period_name, b.bill_number, b.issue_date, b.due_date,
		       b.currency, b.subtotal, b.total_amount, b.status,
		       CASE 
		           WHEN b.status = 'VOID' THEN 'VOID'
		           WHEN b.status = 'ISSUED' AND b.due_date < CURRENT_DATE THEN 'OVERDUE'
		           ELSE b.status
		       END as derived_status,
		       b.created_by, b.voided_by, b.voided_at, b.void_reason, b.created_at, b.updated_at
		FROM maintenance_bills b
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		JOIN society_buildings bld ON u.building_id = bld.id AND u.society_id = bld.society_id
		JOIN maintenance_billing_periods bp ON b.billing_period_id = bp.id AND b.society_id = bp.society_id
		WHERE b.id = $1 AND b.society_id = $2
	`, billID, societyID).Scan(
		&b.ID, &b.SocietyID, &b.UnitID, &b.UnitIdentifier, &b.BuildingName, &b.FloorNumber,
		&b.BillingPeriodID, &b.PeriodName, &b.BillNumber, &b.IssueDate, &b.DueDate,
		&b.Currency, &b.Subtotal, &b.TotalAmount, &b.Status, &b.DerivedStatus,
		&b.CreatedBy, &b.VoidedBy, &b.VoidedAt, &b.VoidReason, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("get bill: %w", err)
	}

	// Fetch line items
	itemRows, err := r.pool.Query(ctx, `
		SELECT id, bill_id, society_id, charge_category_id, category_name,
		       charge_config_id, description, rate, quantity, amount, created_at
		FROM maintenance_bill_items
		WHERE bill_id = $1 AND society_id = $2
		ORDER BY created_at ASC
	`, billID, societyID)
	if err != nil {
		return nil, fmt.Errorf("query line items: %w", err)
	}
	defer itemRows.Close()

	var items []MaintenanceBillItem
	for itemRows.Next() {
		var it MaintenanceBillItem
		if err := itemRows.Scan(
			&it.ID, &it.BillID, &it.SocietyID, &it.ChargeCategoryID, &it.CategoryName,
			&it.ChargeConfigID, &it.Description, &it.Rate, &it.Quantity, &it.Amount, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan line item: %w", err)
		}
		items = append(items, it)
	}

	if items == nil {
		items = []MaintenanceBillItem{}
	}
	b.LineItems = items
	b.Items = items

	return &b, nil
}

// VoidBill voids an issued bill with an audit trail.
func (r *Repository) VoidBill(
	ctx context.Context,
	societyID, billID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBill, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, errors.New("void reason is required")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM maintenance_bills
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, billID, societyID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("check bill: %w", err)
	}

	if currentStatus == "VOID" {
		return nil, ErrBillAlreadyVoid
	}

	// Guard against voiding a bill with active payment allocations
	var activeAllocCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM maintenance_payment_allocations
		WHERE bill_id = $1 AND society_id = $2 AND status = 'ACTIVE'
	`, billID, societyID).Scan(&activeAllocCount)
	if err != nil {
		return nil, fmt.Errorf("check active allocations: %w", err)
	}
	if activeAllocCount > 0 {
		return nil, ErrBillHasAllocations
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_bills
		SET status = 'VOID', voided_by = $1, voided_at = $2, void_reason = $3, updated_at = NOW()
		WHERE id = $4 AND society_id = $5
	`, actorID, now, trimmedReason, billID, societyID)
	if err != nil {
		return nil, fmt.Errorf("void bill: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id": societyID,
		"bill_id":    billID,
		"reason":     trimmedReason,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionBillVoided, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetBillDetails(ctx, societyID, billID)
}

// ListResidentBills lists bills for units the resident is actively authorized for.
func (r *Repository) ListResidentBills(
	ctx context.Context,
	societyID, userID uuid.UUID,
	limit, offset int,
) (*ListBillsResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
		FROM maintenance_bills b
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		JOIN society_buildings bld ON u.building_id = bld.id AND u.society_id = bld.society_id
		JOIN maintenance_billing_periods bp ON b.billing_period_id = bp.id AND b.society_id = bp.society_id
		JOIN unit_relationships ur ON ur.unit_id = u.id AND ur.society_id = b.society_id
		JOIN society_members sm ON ur.member_id = sm.id AND sm.society_id = b.society_id
		WHERE b.society_id = $1
		  AND sm.user_id = $2
		  AND ur.status = 'ACTIVE'
		  AND sm.status = 'ACTIVE'
		  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
	`

	var total int
	countQuery := "SELECT COUNT(DISTINCT b.id) " + baseQuery
	if err := r.pool.QueryRow(ctx, countQuery, societyID, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count resident bills: %w", err)
	}

	selectQuery := `
		SELECT DISTINCT b.id, b.society_id, b.unit_id, u.unit_identifier, bld.name as building_name, u.floor_number,
		       b.billing_period_id, bp.period_name, b.bill_number, b.issue_date, b.due_date,
		       b.currency, b.subtotal, b.total_amount, b.status,
		       CASE 
		           WHEN b.status = 'VOID' THEN 'VOID'
		           WHEN b.status = 'ISSUED' AND b.due_date < CURRENT_DATE THEN 'OVERDUE'
		           ELSE b.status
		       END as derived_status,
		       b.created_by, b.voided_by, b.voided_at, b.void_reason, b.created_at, b.updated_at
	` + baseQuery + " ORDER BY b.due_date DESC, b.created_at DESC LIMIT $3 OFFSET $4"

	rows, err := r.pool.Query(ctx, selectQuery, societyID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query resident bills: %w", err)
	}
	defer rows.Close()

	var bills []MaintenanceBill
	for rows.Next() {
		var b MaintenanceBill
		if err := rows.Scan(
			&b.ID, &b.SocietyID, &b.UnitID, &b.UnitIdentifier, &b.BuildingName, &b.FloorNumber,
			&b.BillingPeriodID, &b.PeriodName, &b.BillNumber, &b.IssueDate, &b.DueDate,
			&b.Currency, &b.Subtotal, &b.TotalAmount, &b.Status, &b.DerivedStatus,
			&b.CreatedBy, &b.VoidedBy, &b.VoidedAt, &b.VoidReason, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan resident bill: %w", err)
		}
		bills = append(bills, b)
	}

	if bills == nil {
		bills = []MaintenanceBill{}
	}

	return &ListBillsResult{
		Items:  bills,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetResidentBillDetails gets details of a bill if the resident is authorized for the unit.
func (r *Repository) GetResidentBillDetails(
	ctx context.Context,
	societyID, billID, userID uuid.UUID,
) (*MaintenanceBill, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// First verify the bill exists in the society and get its unit_id
	var unitID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT unit_id FROM maintenance_bills
		WHERE id = $1 AND society_id = $2
	`, billID, societyID).Scan(&unitID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("find bill: %w", err)
	}

	// Verify authorization for that unit
	isAuth, _, err := r.ValidateUserUnitAuthorization(ctx, societyID, unitID, userID)
	if err != nil {
		return nil, fmt.Errorf("validate unit authorization: %w", err)
	}
	if !isAuth {
		return nil, errors.New("unauthorized to access this unit's maintenance bill")
	}

	return r.GetBillDetails(ctx, societyID, billID)
}

// RecordPayment records an offline payment received and allocates it against a bill.
func (r *Repository) RecordPayment(
	ctx context.Context,
	societyID uuid.UUID,
	params RecordPaymentParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*RecordPaymentResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Validate payment method
	switch params.PaymentMethod {
	case PaymentMethodCash, PaymentMethodCheque, PaymentMethodBankTransfer, PaymentMethodUPIOffline, PaymentMethodOther:
		// Valid
	default:
		return nil, ErrInvalidPaymentMethod
	}

	exactAmount := math.Round(params.Amount*100) / 100
	if exactAmount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Check idempotency if key provided
	if params.IdempotencyKey != nil && strings.TrimSpace(*params.IdempotencyKey) != "" {
		trimmedKey := strings.TrimSpace(*params.IdempotencyKey)
		var existingPayID uuid.UUID
		var existingAmount float64
		var existingMethod string
		err := r.pool.QueryRow(ctx, `
			SELECT id, amount, payment_method FROM maintenance_payments
			WHERE society_id = $1 AND idempotency_key = $2
		`, societyID, trimmedKey).Scan(&existingPayID, &existingAmount, &existingMethod)
		if err == nil {
			// Idempotency key already exists: verify payload consistency
			if math.Abs(existingAmount-exactAmount) > 0.001 || existingMethod != params.PaymentMethod {
				return nil, ErrIdempotencyConflict
			}
			// Fetch existing payment, allocation, and receipt
			return r.getExistingPaymentResult(ctx, societyID, existingPayID)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check idempotency key: %w", err)
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch and lock bill
	var billTotal float64
	var billStatus string
	var unitID uuid.UUID
	var billNumber string
	err = tx.QueryRow(ctx, `
		SELECT total_amount, status, unit_id, bill_number FROM maintenance_bills
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, params.BillID, societyID).Scan(&billTotal, &billStatus, &unitID, &billNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("fetch bill: %w", err)
	}

	if billStatus == "VOID" {
		return nil, ErrBillCannotBePaid
	}

	// Calculate current outstanding balance
	var alreadyAllocated float64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(allocated_amount), 0)
		FROM maintenance_payment_allocations
		WHERE bill_id = $1 AND society_id = $2 AND status = 'ACTIVE'
	`, params.BillID, societyID).Scan(&alreadyAllocated)
	if err != nil {
		return nil, fmt.Errorf("calculate allocated total: %w", err)
	}

	outstanding := math.Round((billTotal-alreadyAllocated)*100) / 100
	if exactAmount > outstanding {
		return nil, ErrAllocationExceedsBalance
	}

	// Fetch society code and short code
	var societyCode, societyName string
	err = tx.QueryRow(ctx, `SELECT society_code, name FROM societies WHERE id = $1`, societyID).Scan(&societyCode, &societyName)
	if err != nil {
		return nil, fmt.Errorf("get society info: %w", err)
	}
	codeParts := strings.Split(societyCode, "-")
	shortCode := societyCode
	if len(codeParts) >= 2 {
		shortCode = codeParts[1]
	}

	now := time.Now()
	monthPrefix := now.Format("200601")

	// Allocate sequence-backed payment reference
	var paySeq int64
	err = tx.QueryRow(ctx, "SELECT nextval('maintenance_payment_seq')").Scan(&paySeq)
	if err != nil {
		return nil, fmt.Errorf("nextval payment sequence: %w", err)
	}
	paymentRef := fmt.Sprintf("PAY-%s-%s-%05d", shortCode, monthPrefix, paySeq)

	// Insert maintenance_payments
	var paymentID uuid.UUID
	payDate := params.PaymentDate
	if payDate.IsZero() {
		payDate = now
	}

	var idemKey *string
	if params.IdempotencyKey != nil && strings.TrimSpace(*params.IdempotencyKey) != "" {
		k := strings.TrimSpace(*params.IdempotencyKey)
		idemKey = &k
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_payments (
			society_id, payment_reference, payment_method, amount, currency,
			payment_date, external_reference, status, recorded_by, payer_name,
			notes, idempotency_key
		) VALUES ($1, $2, $3, $4, 'INR', $5, $6, 'RECORDED', $7, $8, $9, $10)
		RETURNING id
	`, societyID, paymentRef, params.PaymentMethod, exactAmount, payDate, params.ExternalReference,
		actorID, params.PayerName, params.Notes, idemKey).Scan(&paymentID)
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}

	// Insert maintenance_payment_allocations
	var allocID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_payment_allocations (
			society_id, payment_id, bill_id, allocated_amount, status, allocated_by
		) VALUES ($1, $2, $3, $4, 'ACTIVE', $5)
		RETURNING id
	`, societyID, paymentID, params.BillID, exactAmount, actorID).Scan(&allocID)
	if err != nil {
		return nil, fmt.Errorf("insert allocation: %w", err)
	}

	// Allocate sequence-backed receipt number
	var recSeq int64
	err = tx.QueryRow(ctx, "SELECT nextval('maintenance_receipt_seq')").Scan(&recSeq)
	if err != nil {
		return nil, fmt.Errorf("nextval receipt sequence: %w", err)
	}
	receiptNumber := fmt.Sprintf("REC-%s-%s-%05d", shortCode, monthPrefix, recSeq)

	var receiptID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_receipts (
			society_id, payment_id, receipt_number, receipt_date, total_amount,
			currency, issued_by, status
		) VALUES ($1, $2, $3, $4, $5, 'INR', $6, 'ISSUED')
		RETURNING id
	`, societyID, paymentID, receiptNumber, payDate, exactAmount, actorID).Scan(&receiptID)
	if err != nil {
		return nil, fmt.Errorf("insert receipt: %w", err)
	}

	// Transactional audit logging
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":        societyID,
		"payment_id":        paymentID,
		"payment_reference": paymentRef,
		"receipt_id":        receiptID,
		"receipt_number":    receiptNumber,
		"bill_id":           params.BillID,
		"bill_number":       billNumber,
		"amount":            exactAmount,
		"payment_method":    params.PaymentMethod,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionPaymentRecorded, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}

	// Enqueue notification outbox event for payment recorded
	payerRows, qErr := tx.Query(ctx, `
		SELECT DISTINCT sm.user_id, COALESCE(sm.email, '')
		FROM maintenance_bills b
		JOIN unit_relationships ur ON ur.unit_id = b.unit_id AND ur.society_id = b.society_id
		JOIN society_members sm ON ur.member_id = sm.id
		WHERE b.id = $1 AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
		  AND sm.user_id IS NOT NULL
		  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
	`, params.BillID)
	if qErr == nil {
		for payerRows.Next() {
			var payerUID uuid.UUID
			var payerEmail string
			if err := payerRows.Scan(&payerUID, &payerEmail); err == nil {
				outboxPayload := map[string]interface{}{
					"recipient_user_id": payerUID.String(),
					"recipient_email":   payerEmail,
					"society_id":        societyID.String(),
					"payment_id":        paymentID.String(),
					"receipt_number":    receiptNumber,
					"amount":            fmt.Sprintf("%.2f", exactAmount),
				}
				_ = r.EnqueueNotificationOutbox(ctx, tx, "PAYMENT_RECORDED", fmt.Sprintf("PAY_REC:%s:%s", paymentID.String(), payerUID.String()), outboxPayload)
			}
		}
		payerRows.Close()
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	remaining := math.Round((outstanding-exactAmount)*100) / 100

	return &RecordPaymentResult{
		Payment: MaintenancePayment{
			ID:                paymentID,
			SocietyID:         societyID,
			PaymentReference:  paymentRef,
			PaymentMethod:     params.PaymentMethod,
			Amount:            exactAmount,
			Currency:          "INR",
			PaymentDate:       payDate,
			ExternalReference: params.ExternalReference,
			Status:            PaymentStatusRecorded,
			RecordedBy:        actorID,
			PayerName:         params.PayerName,
			Notes:             params.Notes,
			IdempotencyKey:    idemKey,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		Receipt: MaintenanceReceipt{
			ID:                receiptID,
			SocietyID:         societyID,
			PaymentID:         paymentID,
			ReceiptNumber:     receiptNumber,
			ReceiptDate:       payDate,
			TotalAmount:       exactAmount,
			Currency:          "INR",
			IssuedBy:          actorID,
			Status:            ReceiptStatusIssued,
			CreatedAt:         now,
			SocietyName:       societyName,
			SocietyCode:       societyCode,
			PaymentReference:  paymentRef,
			PaymentMethod:     params.PaymentMethod,
			ExternalReference: params.ExternalReference,
			PayerName:         params.PayerName,
			BillNumber:        billNumber,
		},
		Allocation: MaintenancePaymentAllocation{
			ID:              allocID,
			SocietyID:       societyID,
			PaymentID:       paymentID,
			BillID:          params.BillID,
			AllocatedAmount: exactAmount,
			Status:          AllocationStatusActive,
			AllocatedBy:     actorID,
			CreatedAt:       now,
		},
		RemainingBalance: remaining,
	}, nil
}

func (r *Repository) getExistingPaymentResult(ctx context.Context, societyID, paymentID uuid.UUID) (*RecordPaymentResult, error) {
	var p MaintenancePayment
	err := r.pool.QueryRow(ctx, `
		SELECT id, society_id, payment_reference, payment_method, amount, currency,
		       payment_date, external_reference, status, recorded_by, payer_name,
		       notes, idempotency_key, created_at, updated_at
		FROM maintenance_payments
		WHERE id = $1 AND society_id = $2
	`, paymentID, societyID).Scan(
		&p.ID, &p.SocietyID, &p.PaymentReference, &p.PaymentMethod, &p.Amount, &p.Currency,
		&p.PaymentDate, &p.ExternalReference, &p.Status, &p.RecordedBy, &p.PayerName,
		&p.Notes, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch existing payment: %w", err)
	}

	var rec MaintenanceReceipt
	err = r.pool.QueryRow(ctx, `
		SELECT id, society_id, payment_id, receipt_number, receipt_date, total_amount,
		       currency, issued_by, status, created_at
		FROM maintenance_receipts
		WHERE payment_id = $1 AND society_id = $2
	`, paymentID, societyID).Scan(
		&rec.ID, &rec.SocietyID, &rec.PaymentID, &rec.ReceiptNumber, &rec.ReceiptDate,
		&rec.TotalAmount, &rec.Currency, &rec.IssuedBy, &rec.Status, &rec.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch existing receipt: %w", err)
	}

	var alloc MaintenancePaymentAllocation
	err = r.pool.QueryRow(ctx, `
		SELECT id, society_id, payment_id, bill_id, allocated_amount, status, allocated_by, created_at
		FROM maintenance_payment_allocations
		WHERE payment_id = $1 AND society_id = $2
		LIMIT 1
	`, paymentID, societyID).Scan(
		&alloc.ID, &alloc.SocietyID, &alloc.PaymentID, &alloc.BillID, &alloc.AllocatedAmount,
		&alloc.Status, &alloc.AllocatedBy, &alloc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch existing allocation: %w", err)
	}

	// Calculate remaining balance on bill
	var billTotal, totalAlloc float64
	_ = r.pool.QueryRow(ctx, `SELECT total_amount FROM maintenance_bills WHERE id = $1`, alloc.BillID).Scan(&billTotal)
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(allocated_amount), 0) FROM maintenance_payment_allocations
		WHERE bill_id = $1 AND society_id = $2 AND status = 'ACTIVE'
	`, alloc.BillID, societyID).Scan(&totalAlloc)

	rem := math.Round((billTotal-totalAlloc)*100) / 100

	return &RecordPaymentResult{
		Payment:          p,
		Receipt:          rec,
		Allocation:       alloc,
		RemainingBalance: rem,
	}, nil
}

// ReversePayment reverses a recorded payment and voids its active allocations and receipt.
func (r *Repository) ReversePayment(
	ctx context.Context,
	societyID, paymentID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return ErrReversalReasonRequired
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	var amount float64
	var paymentRef string
	err = tx.QueryRow(ctx, `
		SELECT status, amount, payment_reference FROM maintenance_payments
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, paymentID, societyID).Scan(&currentStatus, &amount, &paymentRef)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPaymentNotFound
		}
		return fmt.Errorf("lock payment: %w", err)
	}

	if currentStatus == PaymentStatusReversed {
		return ErrPaymentAlreadyReversed
	}

	now := time.Now()

	// Update payment
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_payments
		SET status = 'REVERSED', reversed_at = $1, reversed_by = $2, reversal_reason = $3, updated_at = NOW()
		WHERE id = $4 AND society_id = $5
	`, now, actorID, trimmedReason, paymentID, societyID)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	// Update allocations
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_payment_allocations
		SET status = 'REVERSED', reversed_at = $1
		WHERE payment_id = $2 AND society_id = $3 AND status = 'ACTIVE'
	`, now, paymentID, societyID)
	if err != nil {
		return fmt.Errorf("update allocations: %w", err)
	}

	// Update receipts
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_receipts
		SET status = 'REVERSED', reversed_at = $1
		WHERE payment_id = $2 AND society_id = $3 AND status = 'ISSUED'
	`, now, paymentID, societyID)
	if err != nil {
		return fmt.Errorf("update receipts: %w", err)
	}

	// Transactional audit log
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"society_id":        societyID,
		"payment_id":        paymentID,
		"payment_reference": paymentRef,
		"amount":            amount,
		"reason":            trimmedReason,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionPaymentReversed, string(auditDetails), clientIP, userAgent)
	if err != nil {
		return fmt.Errorf("audit log: %w", err)
	}

	// Enqueue notification outbox event for payment reversed
	revRows, qErr := tx.Query(ctx, `
		SELECT DISTINCT sm.user_id, COALESCE(sm.email, ''), COALESCE(mr.receipt_number, '')
		FROM maintenance_payments mp
		JOIN maintenance_payment_allocations mpa ON mpa.payment_id = mp.id
		JOIN maintenance_bills b ON b.id = mpa.bill_id
		JOIN unit_relationships ur ON ur.unit_id = b.unit_id AND ur.society_id = b.society_id
		JOIN society_members sm ON ur.member_id = sm.id
		LEFT JOIN maintenance_receipts mr ON mr.payment_id = mp.id
		WHERE mp.id = $1 AND mp.society_id = $2
		  AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
		  AND sm.user_id IS NOT NULL
		  AND (ur.is_occupant = true OR ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER'))
	`, paymentID, societyID)
	if qErr == nil {
		for revRows.Next() {
			var revUID uuid.UUID
			var revEmail, receiptNo string
			if err := revRows.Scan(&revUID, &revEmail, &receiptNo); err == nil {
				outboxPayload := map[string]interface{}{
					"recipient_user_id": revUID.String(),
					"recipient_email":   revEmail,
					"society_id":        societyID.String(),
					"payment_id":        paymentID.String(),
					"receipt_number":    receiptNo,
					"reason":            trimmedReason,
				}
				_ = r.EnqueueNotificationOutbox(ctx, tx, "PAYMENT_REVERSED", fmt.Sprintf("PAY_REV:%s:%s", paymentID.String(), revUID.String()), outboxPayload)
			}
		}
		revRows.Close()
	}

	return tx.Commit(ctx)
}

// ListCollectionRegister lists all bills and their live collection and balance statuses.
func (r *Repository) ListCollectionRegister(
	ctx context.Context,
	societyID uuid.UUID,
	filter ListCollectionRegisterFilter,
) (*ListCollectionRegisterResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
		FROM maintenance_bills b
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		JOIN maintenance_billing_periods bp ON b.billing_period_id = bp.id AND b.society_id = bp.society_id
		LEFT JOIN (
			SELECT bill_id,
			       COALESCE(SUM(allocated_amount), 0) AS total_paid,
			       MAX(created_at) AS last_payment_date
			FROM maintenance_payment_allocations
			WHERE society_id = $1 AND status = 'ACTIVE'
			GROUP BY bill_id
		) alloc ON b.id = alloc.bill_id
		WHERE b.society_id = $1
	`
	args := []interface{}{societyID}
	argIdx := 2

	if filter.PeriodID != nil {
		baseQuery += fmt.Sprintf(" AND b.billing_period_id = $%d", argIdx)
		args = append(args, *filter.PeriodID)
		argIdx++
	}

	if filter.Search != "" {
		baseQuery += fmt.Sprintf(" AND (u.unit_identifier ILIKE $%d OR b.bill_number ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	if filter.Status != "" {
		switch strings.ToUpper(filter.Status) {
		case "PAID":
			baseQuery += " AND b.status != 'VOID' AND COALESCE(alloc.total_paid, 0) >= b.total_amount"
		case "PARTIALLY_PAID":
			baseQuery += " AND b.status != 'VOID' AND COALESCE(alloc.total_paid, 0) > 0 AND COALESCE(alloc.total_paid, 0) < b.total_amount"
		case "UNPAID":
			baseQuery += " AND b.status != 'VOID' AND COALESCE(alloc.total_paid, 0) = 0"
		case "VOIDED", "VOID":
			baseQuery += " AND b.status = 'VOID'"
		}
	}

	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count collections: %w", err)
	}

	selectQuery := `
		SELECT b.id, b.bill_number, b.unit_id, u.unit_identifier,
		       b.billing_period_id, bp.period_name, b.due_date,
		       b.total_amount,
		       COALESCE(alloc.total_paid, 0) AS total_paid,
		       CASE WHEN b.status = 'VOID' THEN 0.00
		            ELSE GREATEST(0.00, b.total_amount - COALESCE(alloc.total_paid, 0))
		       END AS outstanding_balance,
		       b.status AS bill_status,
		       CASE WHEN b.status = 'VOID' THEN 'VOID'
		            WHEN CURRENT_DATE > b.due_date THEN 'OVERDUE'
		            ELSE b.status
		       END AS derived_status,
		       CASE WHEN b.status = 'VOID' THEN 'VOIDED'
		            WHEN COALESCE(alloc.total_paid, 0) >= b.total_amount THEN 'PAID'
		            WHEN COALESCE(alloc.total_paid, 0) > 0 THEN 'PARTIALLY_PAID'
		            ELSE 'UNPAID'
		       END AS collection_status,
		       alloc.last_payment_date
	` + baseQuery + fmt.Sprintf(" ORDER BY u.unit_identifier ASC, b.created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query collections register: %w", err)
	}
	defer rows.Close()

	var items []CollectionRegisterItem
	for rows.Next() {
		var it CollectionRegisterItem
		if err := rows.Scan(
			&it.BillID, &it.BillNumber, &it.UnitID, &it.UnitIdentifier,
			&it.PeriodID, &it.PeriodName, &it.DueDate,
			&it.TotalAmount, &it.TotalPaid, &it.OutstandingBalance,
			&it.BillStatus, &it.DerivedStatus, &it.CollectionStatus,
			&it.LastPaymentDate,
		); err != nil {
			return nil, fmt.Errorf("scan collection item: %w", err)
		}
		items = append(items, it)
	}

	if items == nil {
		items = []CollectionRegisterItem{}
	}

	return &ListCollectionRegisterResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetCollectionSummary calculates society collection metrics.
func (r *Repository) GetCollectionSummary(
	ctx context.Context,
	societyID uuid.UUID,
	periodID *uuid.UUID,
) (*CollectionSummary, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		WITH bill_summary AS (
			SELECT b.id, b.total_amount, b.status,
			       COALESCE(SUM(alloc.allocated_amount), 0) AS paid_amount
			FROM maintenance_bills b
			LEFT JOIN maintenance_payment_allocations alloc
			       ON b.id = alloc.bill_id AND b.society_id = alloc.society_id AND alloc.status = 'ACTIVE'
			WHERE b.society_id = $1
			  AND ($2::uuid IS NULL OR b.billing_period_id = $2::uuid)
			GROUP BY b.id, b.total_amount, b.status
		)
		SELECT 
			COALESCE(SUM(CASE WHEN status != 'VOID' THEN total_amount ELSE 0 END), 0) AS total_billed,
			COALESCE(SUM(CASE WHEN status != 'VOID' THEN paid_amount ELSE 0 END), 0) AS total_collected,
			COALESCE(SUM(CASE WHEN status != 'VOID' THEN GREATEST(0.00, total_amount - paid_amount) ELSE 0 END), 0) AS total_outstanding,
			COALESCE(COUNT(CASE WHEN status != 'VOID' AND paid_amount = 0 THEN 1 END), 0) AS unpaid_count,
			COALESCE(COUNT(CASE WHEN status != 'VOID' AND paid_amount > 0 AND paid_amount < total_amount THEN 1 END), 0) AS partially_paid_count,
			COALESCE(COUNT(CASE WHEN status != 'VOID' AND paid_amount >= total_amount THEN 1 END), 0) AS paid_count
		FROM bill_summary
	`
	var sum CollectionSummary
	err := r.pool.QueryRow(ctx, query, societyID, periodID).Scan(
		&sum.TotalBilled, &sum.TotalCollected, &sum.TotalOutstanding,
		&sum.UnpaidBillsCount, &sum.PartiallyPaidBillsCount, &sum.PaidBillsCount,
	)
	if err != nil {
		return nil, fmt.Errorf("calculate collection summary: %w", err)
	}

	// Calculate reversed payments
	revQuery := `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM maintenance_payments
		WHERE society_id = $1 AND status = 'REVERSED'
	`
	_ = r.pool.QueryRow(ctx, revQuery, societyID).Scan(&sum.ReversedPaymentsCount, &sum.ReversedAmount)

	sum.TotalBilled = math.Round(sum.TotalBilled*100) / 100
	sum.TotalCollected = math.Round(sum.TotalCollected*100) / 100
	sum.TotalOutstanding = math.Round(sum.TotalOutstanding*100) / 100
	sum.ReversedAmount = math.Round(sum.ReversedAmount*100) / 100

	return &sum, nil
}

// GetReceiptDetails retrieves printable receipt details.
func (r *Repository) GetReceiptDetails(
	ctx context.Context,
	societyID, receiptID uuid.UUID,
) (*MaintenanceReceipt, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT r.id, r.society_id, r.payment_id, r.receipt_number, r.receipt_date,
		       r.total_amount, r.currency, r.issued_by, r.status, r.reversed_at, r.created_at,
		       s.name AS society_name, s.society_code,
		       p.payment_reference, p.payment_method, p.external_reference, p.payer_name,
		       b.bill_number, u.unit_identifier
		FROM maintenance_receipts r
		JOIN societies s ON r.society_id = s.id
		JOIN maintenance_payments p ON r.payment_id = p.id AND r.society_id = p.society_id
		LEFT JOIN maintenance_payment_allocations a ON p.id = a.payment_id AND p.society_id = a.society_id
		LEFT JOIN maintenance_bills b ON a.bill_id = b.id AND a.society_id = b.society_id
		LEFT JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		WHERE r.id = $1 AND r.society_id = $2
		LIMIT 1
	`
	var rec MaintenanceReceipt
	err := r.pool.QueryRow(ctx, query, receiptID, societyID).Scan(
		&rec.ID, &rec.SocietyID, &rec.PaymentID, &rec.ReceiptNumber, &rec.ReceiptDate,
		&rec.TotalAmount, &rec.Currency, &rec.IssuedBy, &rec.Status, &rec.ReversedAt, &rec.CreatedAt,
		&rec.SocietyName, &rec.SocietyCode,
		&rec.PaymentReference, &rec.PaymentMethod, &rec.ExternalReference, &rec.PayerName,
		&rec.BillNumber, &rec.UnitIdentifier,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReceiptNotFound
		}
		return nil, fmt.Errorf("fetch receipt details: %w", err)
	}

	return &rec, nil
}

// ListResidentPayments retrieves payment records for units authorized to the resident.
func (r *Repository) ListResidentPayments(
	ctx context.Context,
	societyID, userID uuid.UUID,
	limit, offset int,
) ([]ResidentPaymentItem, int, error) {
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

	baseQuery := `
		FROM maintenance_payments p
		JOIN maintenance_payment_allocations a ON p.id = a.payment_id AND p.society_id = a.society_id
		JOIN maintenance_bills b ON a.bill_id = b.id AND a.society_id = b.society_id
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		JOIN unit_relationships ur ON u.id = ur.unit_id AND u.society_id = ur.society_id
		JOIN society_members sm ON ur.member_id = sm.id AND ur.society_id = sm.society_id
		LEFT JOIN maintenance_receipts rec ON p.id = rec.payment_id AND p.society_id = rec.society_id
		WHERE p.society_id = $1
		  AND sm.user_id = $2
		  AND ur.status = 'ACTIVE'
	`

	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(DISTINCT p.id) "+baseQuery, societyID, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count resident payments: %w", err)
	}

	selectQuery := `
		SELECT DISTINCT ON (p.created_at, p.id)
		       p.id AS payment_id, p.payment_reference,
		       rec.id AS receipt_id, COALESCE(rec.receipt_number, '') AS receipt_number,
		       b.id AS bill_id, b.bill_number, u.unit_identifier,
		       p.amount, p.payment_method, p.payment_date, p.status, p.created_at
	` + baseQuery + fmt.Sprintf(" ORDER BY p.created_at DESC, p.id LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, societyID, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("query resident payments: %w", err)
	}
	defer rows.Close()

	var items []ResidentPaymentItem
	for rows.Next() {
		var it ResidentPaymentItem
		if err := rows.Scan(
			&it.PaymentID, &it.PaymentReference,
			&it.ReceiptID, &it.ReceiptNumber,
			&it.BillID, &it.BillNumber, &it.UnitIdentifier,
			&it.Amount, &it.PaymentMethod, &it.PaymentDate, &it.Status, &it.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan resident payment item: %w", err)
		}
		items = append(items, it)
	}

	if items == nil {
		items = []ResidentPaymentItem{}
	}

	return items, total, nil
}

// GetResidentReceipt returns a receipt if the resident is authorized for the linked unit.
func (r *Repository) GetResidentReceipt(
	ctx context.Context,
	societyID, receiptID, userID uuid.UUID,
) (*MaintenanceReceipt, error) {
	rec, err := r.GetReceiptDetails(ctx, societyID, receiptID)
	if err != nil {
		return nil, err
	}

	// Verify authorization for the unit associated with this receipt
	var unitID uuid.UUID
	err = r.pool.QueryRow(ctx, `
		SELECT b.unit_id
		FROM maintenance_receipts rec
		JOIN maintenance_payments p ON rec.payment_id = p.id AND rec.society_id = p.society_id
		JOIN maintenance_payment_allocations a ON p.id = a.payment_id AND p.society_id = a.society_id
		JOIN maintenance_bills b ON a.bill_id = b.id AND a.society_id = b.society_id
		WHERE rec.id = $1 AND rec.society_id = $2
		LIMIT 1
	`, receiptID, societyID).Scan(&unitID)
	if err != nil {
		return nil, ErrReceiptNotFound
	}

	isAuth, _, err := r.ValidateUserUnitAuthorization(ctx, societyID, unitID, userID)
	if err != nil {
		return nil, fmt.Errorf("verify unit authorization: %w", err)
	}
	if !isAuth {
		return nil, errors.New("unauthorized to view receipt for this unit")
	}

	return rec, nil
}
