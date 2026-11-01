package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrScheduleNotFound       = errors.New("billing schedule not found")
	ErrScheduleAlreadyRunning  = errors.New("billing schedule is currently being processed by another worker")
	ErrScheduleDisabled        = errors.New("cannot execute disabled billing schedule")
	ErrInvalidRecurrenceType   = errors.New("invalid recurrence type: must be MONTHLY, QUARTERLY, or ANNUAL")
	ErrInvalidDayOfMonth       = errors.New("invalid day of month: must be between 1 and 31")
	ErrInvalidDueDays          = errors.New("invalid due days: must be between 1 and 90")
	ErrInvalidTimezone         = errors.New("invalid timezone identifier")
	ErrDuplicateScheduleName   = errors.New("a schedule with this name already exists in this society")
)

type BillingSchedule struct {
	ID             uuid.UUID              `json:"id"`
	SocietyID      uuid.UUID              `json:"society_id"`
	Name           string                 `json:"name"`
	RecurrenceType string                 `json:"recurrence_type"` // MONTHLY, QUARTERLY, ANNUAL
	TemplateID     uuid.UUID              `json:"template_id"`
	TemplateName   string                 `json:"template_name,omitempty"`
	SelectionMode  string                 `json:"selection_mode"` // ALL_UNITS, SELECTED_UNITS
	FilterCriteria map[string]interface{} `json:"filter_criteria"`
	DayOfMonth     int                    `json:"day_of_month"`
	DueDays        int                    `json:"due_days"`
	Timezone       string                 `json:"timezone"`
	AutoFinalize   bool                   `json:"auto_finalize"`
	NextRunAt      time.Time              `json:"next_run_at"`
	LastRunAt      *time.Time             `json:"last_run_at,omitempty"`
	LastBatchID    *uuid.UUID             `json:"last_batch_id,omitempty"`
	Status         string                 `json:"status"` // ACTIVE, PAUSED, DISABLED
	LockedBy       *string                `json:"locked_by,omitempty"`
	LockedUntil    *time.Time             `json:"locked_until,omitempty"`
	CreatedBy      uuid.UUID              `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type ScheduleExecution struct {
	ID              uuid.UUID  `json:"id"`
	ScheduleID      uuid.UUID  `json:"schedule_id"`
	SocietyID       uuid.UUID  `json:"society_id"`
	BillingPeriodID *uuid.UUID `json:"billing_period_id,omitempty"`
	PeriodName      string     `json:"period_name,omitempty"`
	BatchID         *uuid.UUID `json:"batch_id,omitempty"`
	BatchNumber     string     `json:"batch_number,omitempty"`
	IdempotencyKey  string     `json:"idempotency_key"`
	TriggerSource   string     `json:"trigger_source"` // SCHEDULED, MANUAL
	Status          string     `json:"status"`         // CLAIMED, PROCESSING, SUCCESS, FAILED, SKIPPED
	AttemptCount    int        `json:"attempt_count"`
	ErrorCode       *string    `json:"error_code,omitempty"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreateBillingScheduleParams struct {
	Name           string                 `json:"name"`
	RecurrenceType string                 `json:"recurrence_type"`
	TemplateID     uuid.UUID              `json:"template_id"`
	SelectionMode  string                 `json:"selection_mode"`
	FilterCriteria map[string]interface{} `json:"filter_criteria"`
	DayOfMonth     int                    `json:"day_of_month"`
	DueDays        int                    `json:"due_days"`
	Timezone       string                 `json:"timezone"`
	AutoFinalize   bool                   `json:"auto_finalize"`
	FirstRunAt     *time.Time             `json:"first_run_at,omitempty"`
}

type UpdateBillingScheduleParams struct {
	Name           *string                 `json:"name,omitempty"`
	TemplateID     *uuid.UUID              `json:"template_id,omitempty"`
	SelectionMode  *string                 `json:"selection_mode,omitempty"`
	FilterCriteria map[string]interface{}  `json:"filter_criteria,omitempty"`
	DayOfMonth     *int                    `json:"day_of_month,omitempty"`
	DueDays        *int                    `json:"due_days,omitempty"`
	Timezone       *string                 `json:"timezone,omitempty"`
	AutoFinalize   *bool                   `json:"auto_finalize,omitempty"`
}

// ValidateTimezone ensures the string is a valid IANA timezone location.
func ValidateTimezone(tz string) (*time.Location, error) {
	if strings.TrimSpace(tz) == "" {
		return time.LoadLocation("Asia/Kolkata")
	}
	loc, err := time.LoadLocation(strings.TrimSpace(tz))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidTimezone, tz)
	}
	return loc, nil
}

// CalculateBillingPeriodDates computes the intended start, end, due date, and period name
// for a schedule execution based on recurrence type, day of month, and timezone.
func CalculateBillingPeriodDates(runTime time.Time, recurrenceType string, dueDays int, loc *time.Location) (periodStart, periodEnd, dueDate time.Time, periodName string) {
	if loc == nil {
		loc = time.Local
	}
	t := runTime.In(loc)
	year, month, _ := t.Date()

	switch strings.ToUpper(recurrenceType) {
	case "QUARTERLY":
		qStartMonth := time.Month(1 + ((int(month)-1)/3)*3)
		periodStart = time.Date(year, qStartMonth, 1, 0, 0, 0, 0, loc)
		// Last day of quarter
		periodEnd = time.Date(year, qStartMonth+3, 0, 23, 59, 59, 0, loc)
		qNum := ((int(month) - 1) / 3) + 1
		periodName = fmt.Sprintf("Q%d %d Maintenance", qNum, year)

	case "ANNUAL":
		periodStart = time.Date(year, 1, 1, 0, 0, 0, 0, loc)
		periodEnd = time.Date(year+1, 1, 0, 23, 59, 59, 0, loc)
		periodName = fmt.Sprintf("FY %d Maintenance", year)

	case "MONTHLY":
		fallthrough
	default:
		periodStart = time.Date(year, month, 1, 0, 0, 0, 0, loc)
		// Last day of current month safely calculated by day 0 of month+1
		periodEnd = time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
		periodName = fmt.Sprintf("%s %d Maintenance", month.String(), year)
	}

	if dueDays <= 0 {
		dueDays = 15
	}
	dueDate = periodStart.AddDate(0, 0, dueDays)
	return
}

// CalculateNextRunAt computes the next execution time for a schedule following the given runTime.
func CalculateNextRunAt(fromTime time.Time, recurrenceType string, dayOfMonth int, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	t := fromTime.In(loc)
	year, month, _ := t.Date()

	var nextYear int
	var nextMonth time.Month

	switch strings.ToUpper(recurrenceType) {
	case "QUARTERLY":
		m := int(month) + 3
		nextYear = year + (m-1)/12
		nextMonth = time.Month(((m - 1) % 12) + 1)
	case "ANNUAL":
		nextYear = year + 1
		nextMonth = month
	case "MONTHLY":
		fallthrough
	default:
		m := int(month) + 1
		nextYear = year + (m-1)/12
		nextMonth = time.Month(((m - 1) % 12) + 1)
	}

	// Safely clamp day of month to the days present in target month
	// e.g. Feb 29/28, Apr 30
	lastDayOfMonth := time.Date(nextYear, nextMonth+1, 0, 0, 0, 0, 0, loc).Day()
	targetDay := dayOfMonth
	if targetDay <= 0 {
		targetDay = 1
	}
	if targetDay > lastDayOfMonth {
		targetDay = lastDayOfMonth
	}

	return time.Date(nextYear, nextMonth, targetDay, 9, 0, 0, 0, loc) // 09:00 AM local time
}

// CreateBillingSchedule persists a new recurring billing schedule for a society.
func (r *Repository) CreateBillingSchedule(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateBillingScheduleParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("schedule name is required")
	}

	recType := strings.ToUpper(strings.TrimSpace(params.RecurrenceType))
	if recType != "MONTHLY" && recType != "QUARTERLY" && recType != "ANNUAL" {
		return nil, ErrInvalidRecurrenceType
	}

	if params.DayOfMonth < 1 || params.DayOfMonth > 31 {
		return nil, ErrInvalidDayOfMonth
	}

	if params.DueDays < 1 || params.DueDays > 90 {
		return nil, ErrInvalidDueDays
	}

	loc, err := ValidateTimezone(params.Timezone)
	if err != nil {
		return nil, err
	}
	tzName := loc.String()

	selMode := strings.ToUpper(strings.TrimSpace(params.SelectionMode))
	if selMode != "ALL_UNITS" && selMode != "SELECTED_UNITS" {
		selMode = "ALL_UNITS"
	}

	// Verify template exists in society
	var tplName string
	err = r.pool.QueryRow(ctx, `
		SELECT name FROM maintenance_bill_templates WHERE id = $1 AND society_id = $2
	`, params.TemplateID, societyID).Scan(&tplName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("verify template: %w", err)
	}

	var nextRunAt time.Time
	if params.FirstRunAt != nil && !params.FirstRunAt.IsZero() {
		nextRunAt = params.FirstRunAt.In(loc)
	} else {
		// Calculate next run time
		now := time.Now().In(loc)
		lastDayOfThisMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		targetDay := params.DayOfMonth
		if targetDay > lastDayOfThisMonth {
			targetDay = lastDayOfThisMonth
		}
		candidate := time.Date(now.Year(), now.Month(), targetDay, 9, 0, 0, 0, loc)
		if candidate.After(now) {
			nextRunAt = candidate
		} else {
			nextRunAt = CalculateNextRunAt(now, recType, params.DayOfMonth, loc)
		}
	}

	filterJSON, _ := json.Marshal(params.FilterCriteria)

	var s BillingSchedule
	err = r.pool.QueryRow(ctx, `
		INSERT INTO maintenance_billing_schedules (
			society_id, name, recurrence_type, template_id, selection_mode,
			filter_criteria, day_of_month, due_days, timezone, auto_finalize,
			next_run_at, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, 'ACTIVE', $12
		)
		RETURNING id, created_at, updated_at
	`, societyID, name, recType, params.TemplateID, selMode,
		string(filterJSON), params.DayOfMonth, params.DueDays, tzName,
		params.AutoFinalize, nextRunAt, actorID,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_maint_schedules_name") {
			return nil, ErrDuplicateScheduleName
		}
		return nil, fmt.Errorf("insert billing schedule: %w", err)
	}

	s.SocietyID = societyID
	s.Name = name
	s.RecurrenceType = recType
	s.TemplateID = params.TemplateID
	s.TemplateName = tplName
	s.SelectionMode = selMode
	s.FilterCriteria = params.FilterCriteria
	s.DayOfMonth = params.DayOfMonth
	s.DueDays = params.DueDays
	s.Timezone = tzName
	s.AutoFinalize = params.AutoFinalize
	s.NextRunAt = nextRunAt
	s.Status = "ACTIVE"
	s.CreatedBy = actorID

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"schedule_id":     s.ID,
		"name":            s.Name,
		"recurrence_type": s.RecurrenceType,
		"next_run_at":     s.NextRunAt,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_SCHEDULE_CREATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return &s, nil
}

// ListBillingSchedules lists all recurring schedules for a society.
func (r *Repository) ListBillingSchedules(ctx context.Context, societyID uuid.UUID) ([]BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.society_id, s.name, s.recurrence_type, s.template_id,
		       t.name as template_name, s.selection_mode, s.filter_criteria,
		       s.day_of_month, s.due_days, s.timezone, s.auto_finalize,
		       s.next_run_at, s.last_run_at, s.last_batch_id, s.status,
		       s.locked_by, s.locked_until, s.created_by, s.created_at, s.updated_at
		FROM maintenance_billing_schedules s
		JOIN maintenance_bill_templates t ON s.template_id = t.id AND s.society_id = t.society_id
		WHERE s.society_id = $1
		ORDER BY s.created_at DESC
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("list billing schedules: %w", err)
	}
	defer rows.Close()

	var schedules []BillingSchedule
	for rows.Next() {
		var s BillingSchedule
		var filterJSON []byte
		if err := rows.Scan(
			&s.ID, &s.SocietyID, &s.Name, &s.RecurrenceType, &s.TemplateID,
			&s.TemplateName, &s.SelectionMode, &filterJSON,
			&s.DayOfMonth, &s.DueDays, &s.Timezone, &s.AutoFinalize,
			&s.NextRunAt, &s.LastRunAt, &s.LastBatchID, &s.Status,
			&s.LockedBy, &s.LockedUntil, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan billing schedule: %w", err)
		}
		if len(filterJSON) > 0 {
			_ = json.Unmarshal(filterJSON, &s.FilterCriteria)
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// GetBillingSchedule returns a single schedule by ID.
func (r *Repository) GetBillingSchedule(ctx context.Context, societyID, scheduleID uuid.UUID) (*BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var s BillingSchedule
	var filterJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.society_id, s.name, s.recurrence_type, s.template_id,
		       t.name as template_name, s.selection_mode, s.filter_criteria,
		       s.day_of_month, s.due_days, s.timezone, s.auto_finalize,
		       s.next_run_at, s.last_run_at, s.last_batch_id, s.status,
		       s.locked_by, s.locked_until, s.created_by, s.created_at, s.updated_at
		FROM maintenance_billing_schedules s
		JOIN maintenance_bill_templates t ON s.template_id = t.id AND s.society_id = t.society_id
		WHERE s.id = $1 AND s.society_id = $2
	`, scheduleID, societyID).Scan(
		&s.ID, &s.SocietyID, &s.Name, &s.RecurrenceType, &s.TemplateID,
		&s.TemplateName, &s.SelectionMode, &filterJSON,
		&s.DayOfMonth, &s.DueDays, &s.Timezone, &s.AutoFinalize,
		&s.NextRunAt, &s.LastRunAt, &s.LastBatchID, &s.Status,
		&s.LockedBy, &s.LockedUntil, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("get billing schedule: %w", err)
	}
	if len(filterJSON) > 0 {
		_ = json.Unmarshal(filterJSON, &s.FilterCriteria)
	}
	return &s, nil
}

// UpdateBillingSchedule updates mutable properties of an existing schedule.
func (r *Repository) UpdateBillingSchedule(
	ctx context.Context,
	societyID, scheduleID uuid.UUID,
	params UpdateBillingScheduleParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	existing, err := r.GetBillingSchedule(ctx, societyID, scheduleID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil && strings.TrimSpace(*params.Name) != "" {
		existing.Name = strings.TrimSpace(*params.Name)
	}
	if params.TemplateID != nil {
		var tplName string
		if err := r.pool.QueryRow(ctx, `
			SELECT name FROM maintenance_bill_templates WHERE id = $1 AND society_id = $2
		`, *params.TemplateID, societyID).Scan(&tplName); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrTemplateNotFound
			}
			return nil, fmt.Errorf("verify template: %w", err)
		}
		existing.TemplateID = *params.TemplateID
		existing.TemplateName = tplName
	}
	if params.SelectionMode != nil {
		sm := strings.ToUpper(strings.TrimSpace(*params.SelectionMode))
		if sm == "ALL_UNITS" || sm == "SELECTED_UNITS" {
			existing.SelectionMode = sm
		}
	}
	if params.FilterCriteria != nil {
		existing.FilterCriteria = params.FilterCriteria
	}
	if params.DayOfMonth != nil {
		if *params.DayOfMonth < 1 || *params.DayOfMonth > 31 {
			return nil, ErrInvalidDayOfMonth
		}
		existing.DayOfMonth = *params.DayOfMonth
	}
	if params.DueDays != nil {
		if *params.DueDays < 1 || *params.DueDays > 90 {
			return nil, ErrInvalidDueDays
		}
		existing.DueDays = *params.DueDays
	}
	if params.Timezone != nil {
		loc, tzErr := ValidateTimezone(*params.Timezone)
		if tzErr != nil {
			return nil, tzErr
		}
		existing.Timezone = loc.String()
	}
	if params.AutoFinalize != nil {
		existing.AutoFinalize = *params.AutoFinalize
	}

	filterJSON, _ := json.Marshal(existing.FilterCriteria)

	_, err = r.pool.Exec(ctx, `
		UPDATE maintenance_billing_schedules
		SET name = $1,
		    template_id = $2,
		    selection_mode = $3,
		    filter_criteria = $4::jsonb,
		    day_of_month = $5,
		    due_days = $6,
		    timezone = $7,
		    auto_finalize = $8,
		    updated_at = NOW()
		WHERE id = $9 AND society_id = $10
	`, existing.Name, existing.TemplateID, existing.SelectionMode,
		string(filterJSON), existing.DayOfMonth, existing.DueDays,
		existing.Timezone, existing.AutoFinalize, scheduleID, societyID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_maint_schedules_name") {
			return nil, ErrDuplicateScheduleName
		}
		return nil, fmt.Errorf("update billing schedule: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"schedule_id": scheduleID,
		"name":        existing.Name,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_SCHEDULE_UPDATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return r.GetBillingSchedule(ctx, societyID, scheduleID)
}

// SetBillingScheduleStatus modifies the status of a schedule (ACTIVE, PAUSED, DISABLED).
func (r *Repository) SetBillingScheduleStatus(
	ctx context.Context,
	societyID, scheduleID uuid.UUID,
	newStatus string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	status := strings.ToUpper(strings.TrimSpace(newStatus))
	if status != "ACTIVE" && status != "PAUSED" && status != "DISABLED" {
		return nil, errors.New("invalid status: must be ACTIVE, PAUSED, or DISABLED")
	}

	existing, err := r.GetBillingSchedule(ctx, societyID, scheduleID)
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation(existing.Timezone)
	if loc == nil {
		loc = time.Local
	}

	// If resuming from PAUSED/DISABLED and next_run_at is in the past, recalculate next_run_at
	nextRun := existing.NextRunAt
	if status == "ACTIVE" && nextRun.Before(time.Now()) {
		nextRun = CalculateNextRunAt(time.Now().In(loc), existing.RecurrenceType, existing.DayOfMonth, loc)
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE maintenance_billing_schedules
		SET status = $1,
		    next_run_at = $2,
		    updated_at = NOW()
		WHERE id = $3 AND society_id = $4
	`, status, nextRun, scheduleID, societyID)
	if err != nil {
		return nil, fmt.Errorf("update schedule status: %w", err)
	}

	auditAction := fmt.Sprintf("BILLING_SCHEDULE_%s", status)
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"schedule_id": scheduleID,
		"new_status":  status,
		"next_run_at": nextRun,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, auditAction, string(auditDetails), clientIP, userAgent)

	return r.GetBillingSchedule(ctx, societyID, scheduleID)
}

// ClaimDueSchedules locks and claims up to limit schedules due for execution.
func (r *Repository) ClaimDueSchedules(ctx context.Context, workerID string, limit int) ([]BillingSchedule, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	if limit <= 0 {
		limit = 10
	}

	query := `
		WITH claimable AS (
			SELECT id
			FROM maintenance_billing_schedules
			WHERE status = 'ACTIVE'
			  AND next_run_at <= NOW()
			  AND (locked_until IS NULL OR locked_until < NOW())
			ORDER BY next_run_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE maintenance_billing_schedules s
		SET locked_by = $2,
		    locked_until = NOW() + INTERVAL '5 minutes',
		    updated_at = NOW()
		FROM claimable c
		WHERE s.id = c.id
		RETURNING s.id, s.society_id, s.name, s.recurrence_type, s.template_id,
		          s.selection_mode, s.filter_criteria, s.day_of_month, s.due_days,
		          s.timezone, s.auto_finalize, s.next_run_at, s.last_run_at,
		          s.last_batch_id, s.status, s.locked_by, s.locked_until,
		          s.created_by, s.created_at, s.updated_at
	`

	rows, err := r.pool.Query(ctx, query, limit, workerID)
	if err != nil {
		return nil, fmt.Errorf("claim due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []BillingSchedule
	for rows.Next() {
		var s BillingSchedule
		var filterJSON []byte
		if err := rows.Scan(
			&s.ID, &s.SocietyID, &s.Name, &s.RecurrenceType, &s.TemplateID,
			&s.SelectionMode, &filterJSON, &s.DayOfMonth, &s.DueDays,
			&s.Timezone, &s.AutoFinalize, &s.NextRunAt, &s.LastRunAt,
			&s.LastBatchID, &s.Status, &s.LockedBy, &s.LockedUntil,
			&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed schedule: %w", err)
		}
		if len(filterJSON) > 0 {
			_ = json.Unmarshal(filterJSON, &s.FilterCriteria)
		}
		schedules = append(schedules, s)
	}
	return schedules, nil
}

// ExecuteScheduleRun coordinates the idempotent creation and execution of a billing batch
// for a schedule (either via automated background run or admin manual run).
func (r *Repository) ExecuteScheduleRun(
	ctx context.Context,
	societyID, scheduleID uuid.UUID,
	triggerSource string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*ScheduleExecution, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	triggerSource = strings.ToUpper(strings.TrimSpace(triggerSource))
	if triggerSource != "SCHEDULED" && triggerSource != "MANUAL" {
		triggerSource = "SCHEDULED"
	}

	// 1. Lock schedule for execution
	var s BillingSchedule
	var filterJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, society_id, name, recurrence_type, template_id, selection_mode,
		       filter_criteria, day_of_month, due_days, timezone, auto_finalize,
		       next_run_at, last_run_at, last_batch_id, status, locked_by, locked_until,
		       created_by, created_at, updated_at
		FROM maintenance_billing_schedules
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, scheduleID, societyID).Scan(
		&s.ID, &s.SocietyID, &s.Name, &s.RecurrenceType, &s.TemplateID,
		&s.SelectionMode, &filterJSON, &s.DayOfMonth, &s.DueDays,
		&s.Timezone, &s.AutoFinalize, &s.NextRunAt, &s.LastRunAt,
		&s.LastBatchID, &s.Status, &s.LockedBy, &s.LockedUntil,
		&s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("lock schedule: %w", err)
	}

	if s.Status == "DISABLED" {
		return nil, ErrScheduleDisabled
	}

	// If locked by another worker in the last 5 minutes, reject concurrent execution
	if triggerSource == "MANUAL" && s.LockedUntil != nil && s.LockedUntil.After(time.Now()) {
		return nil, ErrScheduleAlreadyRunning
	}

	if len(filterJSON) > 0 {
		_ = json.Unmarshal(filterJSON, &s.FilterCriteria)
	}

	loc, _ := time.LoadLocation(s.Timezone)
	if loc == nil {
		loc = time.Local
	}

	// 2. Calculate intended billing period dates
	runTime := s.NextRunAt
	if triggerSource == "MANUAL" || runTime.IsZero() {
		runTime = time.Now().In(loc)
	}
	periodStart, periodEnd, dueDate, periodName := CalculateBillingPeriodDates(runTime, s.RecurrenceType, s.DueDays, loc)

	// Deterministic idempotency key for this schedule and period
	idempotencyKey := fmt.Sprintf("sched-%s-%s-%s", s.ID.String()[:8], s.RecurrenceType, periodStart.Format("20060102"))

	// 3. Insert CLAIMED execution record
	var exec ScheduleExecution
	exec.ScheduleID = s.ID
	exec.SocietyID = societyID
	exec.IdempotencyKey = idempotencyKey
	exec.TriggerSource = triggerSource
	exec.StartedAt = time.Now()

	err = r.pool.QueryRow(ctx, `
		INSERT INTO maintenance_billing_schedule_executions (
			schedule_id, society_id, idempotency_key, trigger_source,
			status, attempt_count, started_at
		) VALUES ($1, $2, $3, $4, 'CLAIMED', 1, $5)
		ON CONFLICT (society_id, idempotency_key) DO UPDATE
		SET attempt_count = maintenance_billing_schedule_executions.attempt_count + 1,
		    status = 'CLAIMED', started_at = EXCLUDED.started_at
		RETURNING id, created_at
	`, s.ID, societyID, idempotencyKey, triggerSource, exec.StartedAt).Scan(&exec.ID, &exec.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record schedule execution start: %w", err)
	}

	// 4. Resolve or create maintenance billing period
	var periodID uuid.UUID
	err = r.pool.QueryRow(ctx, `
		SELECT id FROM maintenance_billing_periods
		WHERE society_id = $1 AND period_start = $2 AND period_end = $3
	`, societyID, periodStart, periodEnd).Scan(&periodID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Create billing period
			err = r.pool.QueryRow(ctx, `
				INSERT INTO maintenance_billing_periods (
					society_id, period_name, period_start, period_end, due_date, status, created_by
				) VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6)
				RETURNING id
			`, societyID, periodName, periodStart, periodEnd, dueDate, actorID).Scan(&periodID)
			if err != nil {
				errMsg := err.Error()
				errCode := "PERIOD_CREATION_FAILED"
				_ = r.failScheduleExecution(ctx, exec.ID, errCode, errMsg)
				return nil, fmt.Errorf("create billing period: %w", err)
			}
		} else {
			errMsg := err.Error()
			errCode := "QUERY_PERIOD_FAILED"
			_ = r.failScheduleExecution(ctx, exec.ID, errCode, errMsg)
			return nil, fmt.Errorf("query billing period: %w", err)
		}
	}
	exec.BillingPeriodID = &periodID
	exec.PeriodName = periodName

	// 5. Create Billing Batch via Phase 7.1 mechanism with deterministic idempotency key
	batchParams := CreateBillingBatchParams{
		BillingPeriodID: periodID,
		TemplateID:      s.TemplateID,
		SelectionMode:   s.SelectionMode,
		FilterCriteria:  s.FilterCriteria,
		IdempotencyKey:  &idempotencyKey,
	}

	batch, err := r.CreateBillingBatch(ctx, societyID, batchParams, actorID, clientIP, userAgent)
	if err != nil {
		errMsg := err.Error()
		errCode := "BATCH_CREATION_FAILED"
		_ = r.failScheduleExecution(ctx, exec.ID, errCode, errMsg)
		return nil, fmt.Errorf("create billing batch: %w", err)
	}
	exec.BatchID = &batch.ID
	exec.BatchNumber = batch.BatchNumber

	// 6. Generate Batch Drafts via Phase 7.1
	if batch.Status == "DRAFT_PREVIEW" || batch.Status == "QUEUED" {
		batch, err = r.GenerateBillingBatchDrafts(ctx, societyID, batch.ID, actorID, clientIP, userAgent)
		if err != nil {
			errMsg := err.Error()
			errCode := "BATCH_GENERATION_FAILED"
			_ = r.failScheduleExecution(ctx, exec.ID, errCode, errMsg)
			return nil, fmt.Errorf("generate billing batch: %w", err)
		}
	}

	// 7. Auto-finalize if requested and all units generated successfully
	if s.AutoFinalize && batch.Status == "GENERATED" {
		batch, err = r.FinalizeBillingBatch(ctx, societyID, batch.ID, actorID, clientIP, userAgent)
		if err != nil {
			errMsg := err.Error()
			errCode := "BATCH_FINALIZATION_FAILED"
			_ = r.failScheduleExecution(ctx, exec.ID, errCode, errMsg)
			return nil, fmt.Errorf("finalize billing batch: %w", err)
		}
	}

	// 8. Advance next_run_at and update schedule metadata
	nextRun := CalculateNextRunAt(runTime, s.RecurrenceType, s.DayOfMonth, loc)
	now := time.Now()

	_, err = r.pool.Exec(ctx, `
		UPDATE maintenance_billing_schedules
		SET next_run_at = $1,
		    last_run_at = $2,
		    last_batch_id = $3,
		    locked_by = NULL,
		    locked_until = NULL,
		    updated_at = $2
		WHERE id = $4 AND society_id = $5
	`, nextRun, now, batch.ID, s.ID, societyID)
	if err != nil {
		slog.Error("failed to advance schedule next_run_at", "schedule_id", s.ID, "error", err)
	}

	// 9. Complete execution audit record
	completedAt := time.Now()
	_, _ = r.pool.Exec(ctx, `
		UPDATE maintenance_billing_schedule_executions
		SET status = 'SUCCESS',
		    billing_period_id = $1,
		    batch_id = $2,
		    completed_at = $3
		WHERE id = $4
	`, periodID, batch.ID, completedAt, exec.ID)

	exec.Status = "SUCCESS"
	exec.CompletedAt = &completedAt

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"schedule_id":    s.ID,
		"execution_id":   exec.ID,
		"batch_id":       batch.ID,
		"batch_number":   batch.BatchNumber,
		"period_name":    periodName,
		"trigger_source": triggerSource,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_SCHEDULE_EXECUTED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return &exec, nil
}

func (r *Repository) failScheduleExecution(ctx context.Context, executionID uuid.UUID, code, msg string) error {
	completedAt := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE maintenance_billing_schedule_executions
		SET status = 'FAILED',
		    error_code = $1,
		    error_message = $2,
		    completed_at = $3
		WHERE id = $4
	`, code, msg, completedAt, executionID)
	return err
}

// ListScheduleExecutions returns paginated history of executions for a schedule.
func (r *Repository) ListScheduleExecutions(
	ctx context.Context,
	societyID, scheduleID uuid.UUID,
	limit, offset int,
) ([]ScheduleExecution, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, `
		SELECT e.id, e.schedule_id, e.society_id, e.billing_period_id,
		       COALESCE(p.period_name, '') as period_name,
		       e.batch_id, COALESCE(b.batch_number, '') as batch_number,
		       e.idempotency_key, e.trigger_source, e.status, e.attempt_count,
		       e.error_code, e.error_message, e.started_at, e.completed_at, e.created_at
		FROM maintenance_billing_schedule_executions e
		LEFT JOIN maintenance_billing_periods p ON e.billing_period_id = p.id AND e.society_id = p.society_id
		LEFT JOIN maintenance_billing_batches b ON e.batch_id = b.id AND e.society_id = b.society_id
		WHERE e.society_id = $1 AND e.schedule_id = $2
		ORDER BY e.created_at DESC
		LIMIT $3 OFFSET $4
	`, societyID, scheduleID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list schedule executions: %w", err)
	}
	defer rows.Close()

	var list []ScheduleExecution
	for rows.Next() {
		var e ScheduleExecution
		if err := rows.Scan(
			&e.ID, &e.ScheduleID, &e.SocietyID, &e.BillingPeriodID,
			&e.PeriodName, &e.BatchID, &e.BatchNumber,
			&e.IdempotencyKey, &e.TriggerSource, &e.Status, &e.AttemptCount,
			&e.ErrorCode, &e.ErrorMessage, &e.StartedAt, &e.CompletedAt, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan schedule execution: %w", err)
		}
		list = append(list, e)
	}
	return list, nil
}
