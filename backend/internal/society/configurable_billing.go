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

	"chs/backend/internal/formula"
)

// Domain Errors for Configurable Billing
var (
	ErrTemplateNotFound          = errors.New("billing template not found")
	ErrTemplateNameExists        = errors.New("a billing template with this name already exists in this society")
	ErrTemplateImmutable         = errors.New("published billing templates cannot be modified directly; create a draft revision")
	ErrChargeHeadNotFound        = errors.New("charge head not found")
	ErrChargeHeadCodeExists       = errors.New("a charge head with this code already exists in this society")
	ErrUnitOverrideNotFound      = errors.New("unit billing override not found")
	ErrBillingBatchNotFound      = errors.New("billing batch not found")
	ErrBatchInvalidState         = errors.New("invalid batch state transition")
	ErrDuplicateBillingDetected  = errors.New("active or issued bill already exists for one or more selected units in this period")
	ErrBatchAlreadyFinalized     = errors.New("billing batch is already finalized and cannot be modified")
	ErrBatchEmptyUnits           = errors.New("no eligible units found matching selection criteria")
	ErrBatchAlreadyProcessing    = errors.New("billing batch is currently being processed by another worker")
	ErrInvalidBatchTransition    = errors.New("invalid billing batch lifecycle transition")
	ErrBatchCancelled            = errors.New("billing batch is cancelled")
	ErrNoFailedUnitsToRetry      = errors.New("no failed units found in this batch to retry")
)

// IsValidBatchTransition validates permitted batch lifecycle transitions
func IsValidBatchTransition(from, to string) bool {
	switch from {
	case "DRAFT_PREVIEW":
		return to == "QUEUED" || to == "PROCESSING" || to == "CANCELLED"
	case "QUEUED":
		return to == "PROCESSING" || to == "CANCELLED"
	case "PROCESSING":
		return to == "GENERATED" || to == "COMPLETED_WITH_ERRORS" || to == "FAILED"
	case "GENERATED":
		return to == "FINALIZED" || to == "CANCELLED"
	case "COMPLETED_WITH_ERRORS":
		return to == "PROCESSING" || to == "FINALIZED" || to == "CANCELLED"
	case "FAILED":
		return to == "PROCESSING" || to == "CANCELLED"
	case "FINALIZED", "CANCELLED":
		return false
	default:
		return false
	}
}

// Models

type MaintenanceBillTemplate struct {
	ID                   uuid.UUID                `json:"id"`
	SocietyID            uuid.UUID                `json:"society_id"`
	Name                 string                   `json:"name"`
	Description          *string                  `json:"description,omitempty"`
	BillType             string                   `json:"bill_type"`
	BillingFrequency     string                   `json:"billing_frequency"`
	ApplicableUnitTypes  []string                 `json:"applicable_unit_types"`
	Currency             string                   `json:"currency"`
	BillTitle            string                   `json:"bill_title"`
	HeaderNotes          *string                  `json:"header_notes,omitempty"`
	FooterText           *string                  `json:"footer_text,omitempty"`
	PaymentInstructions  *string                  `json:"payment_instructions,omitempty"`
	BankDetails          map[string]interface{}   `json:"bank_details"`
	WatermarkEnabled     bool                     `json:"watermark_enabled"`
	WatermarkOpacity     float64                  `json:"watermark_opacity"`
	Status               string                   `json:"status"` // DRAFT, ACTIVE, ARCHIVED
	CurrentVersion       int                      `json:"current_version"`
	CreatedBy            uuid.UUID                `json:"created_by"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
	Fields               []MaintenanceTemplateField `json:"fields,omitempty"`
}

type MaintenanceTemplateField struct {
	ID            uuid.UUID              `json:"id"`
	SocietyID     uuid.UUID              `json:"society_id"`
	TemplateID    uuid.UUID              `json:"template_id"`
	FieldKey      string                 `json:"field_key"`
	FieldLabel    string                 `json:"field_label"`
	FieldCategory string                 `json:"field_category"` // SOCIETY_INFO, BILL_INFO, MEMBER_INFO, FINANCIAL, CUSTOM
	IsVisible     bool                   `json:"is_visible"`
	IsRequired    bool                   `json:"is_required"`
	DisplayOrder  int                    `json:"display_order"`
	DefaultValue  *string                `json:"default_value,omitempty"`
	Formatting    map[string]interface{} `json:"formatting"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type MaintenanceBillTemplateVersion struct {
	ID                 uuid.UUID              `json:"id"`
	SocietyID          uuid.UUID              `json:"society_id"`
	TemplateID         uuid.UUID              `json:"template_id"`
	Version            int                    `json:"version"`
	DefinitionSnapshot map[string]interface{} `json:"definition_snapshot"`
	PublishedBy        uuid.UUID              `json:"published_by"`
	PublishedAt        time.Time              `json:"published_at"`
}

type MaintenanceChargeHead struct {
	ID                uuid.UUID  `json:"id"`
	SocietyID         uuid.UUID  `json:"society_id"`
	CategoryID        *uuid.UUID `json:"category_id,omitempty"`
	CategoryName      *string    `json:"category_name,omitempty"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	SACCode           *string    `json:"sac_code,omitempty"`
	TaxClassification string     `json:"tax_classification"` // TAXABLE, EXEMPT, NIL_RATED, NON_GST
	DefaultTaxRate    float64    `json:"default_tax_rate"`
	CalculationMethod string     `json:"calculation_method"` // FIXED, AREA_MULTIPLIER, METERED, PERCENTAGE, FORMULA
	DefaultFormulaID  *uuid.UUID `json:"default_formula_id,omitempty"`
	FormulaExpression *string    `json:"formula_expression,omitempty"`
	IsActive          bool       `json:"is_active"`
	DisplayOrder      int        `json:"display_order"`
	CreatedBy         uuid.UUID  `json:"created_by"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type MaintenanceUnitBillingOverride struct {
	ID                        uuid.UUID  `json:"id"`
	SocietyID                 uuid.UUID  `json:"society_id"`
	UnitID                    uuid.UUID  `json:"unit_id"`
	UnitNumber                string     `json:"unit_number,omitempty"`
	ChargeHeadID              uuid.UUID  `json:"charge_head_id"`
	ChargeHeadName            string     `json:"charge_head_name,omitempty"`
	OverrideType              string     `json:"override_type"` // FIXED_AMOUNT, RATE_OVERRIDE, FORMULA_OVERRIDE, EXEMPTION, PERCENTAGE_DISCOUNT
	OverrideValue             *float64   `json:"override_value,omitempty"`
	FormulaOverrideExpression *string    `json:"formula_override_expression,omitempty"`
	EffectiveFrom             time.Time  `json:"effective_from"`
	EffectiveUntil            *time.Time `json:"effective_until,omitempty"`
	Reason                    string     `json:"reason"`
	Status                    string     `json:"status"` // ACTIVE, INACTIVE
	CreatedBy                 uuid.UUID  `json:"created_by"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

type MaintenanceBillingBatch struct {
	ID                  uuid.UUID              `json:"id"`
	SocietyID           uuid.UUID              `json:"society_id"`
	BillingPeriodID     uuid.UUID              `json:"billing_period_id"`
	PeriodName          string                 `json:"period_name,omitempty"`
	TemplateID          uuid.UUID              `json:"template_id"`
	TemplateName        string                 `json:"template_name,omitempty"`
	BatchNumber         string                 `json:"batch_number"`
	SelectionMode       string                 `json:"selection_mode"` // ALL_UNITS, SELECTED_UNITS, SINGLE_UNIT, UNIT_GROUP
	FilterCriteria      map[string]interface{} `json:"filter_criteria"`
	TargetUnitCount     int                    `json:"target_unit_count"`
	ProcessedUnitCount  int                    `json:"processed_unit_count"`
	SuccessfulUnitCount int                    `json:"successful_unit_count"`
	FailedUnitCount     int                    `json:"failed_unit_count"`
	TotalAmount         float64                `json:"total_amount"`
	Status              string                 `json:"status"` // DRAFT_PREVIEW, QUEUED, PROCESSING, GENERATED, COMPLETED_WITH_ERRORS, FINALIZED, FAILED, CANCELLED
	IdempotencyKey      *string                `json:"idempotency_key,omitempty"`
	ErrorSummary        map[string]interface{} `json:"error_summary,omitempty"`
	CreatedBy           uuid.UUID              `json:"created_by"`
	FinalizedBy         *uuid.UUID             `json:"finalized_by,omitempty"`
	FinalizedAt         *time.Time             `json:"finalized_at,omitempty"`
	StartedAt           *time.Time             `json:"started_at,omitempty"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
	CancelledAt         *time.Time             `json:"cancelled_at,omitempty"`
	CancelledBy         *uuid.UUID             `json:"cancelled_by,omitempty"`
	CancellationReason  *string                `json:"cancellation_reason,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type BatchUnit struct {
	ID                  uuid.UUID              `json:"id"`
	BatchID             uuid.UUID              `json:"batch_id"`
	SocietyID           uuid.UUID              `json:"society_id"`
	UnitID              uuid.UUID              `json:"unit_id"`
	UnitNumber          string                 `json:"unit_number,omitempty"`
	BillID              *uuid.UUID             `json:"bill_id,omitempty"`
	Status              string                 `json:"status"` // PENDING, CALCULATED, GENERATED, EXCLUDED, FAILED
	CalculationSnapshot map[string]interface{} `json:"calculation_snapshot"`
	ErrorMessage        *string                `json:"error_message,omitempty"`
	ErrorCode           *string                `json:"error_code,omitempty"`
	RetryCount          int                    `json:"retry_count"`
	LastAttemptedAt     *time.Time             `json:"last_attempted_at,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type BatchProgressInfo struct {
	BatchID             uuid.UUID  `json:"batch_id"`
	Status              string     `json:"status"`
	TargetUnitCount     int        `json:"target_unit_count"`
	ProcessedUnitCount  int        `json:"processed_unit_count"`
	SuccessfulUnitCount int        `json:"successful_unit_count"`
	FailedUnitCount     int        `json:"failed_unit_count"`
	PendingUnitCount    int        `json:"pending_unit_count"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	FinalizedAt         *time.Time `json:"finalized_at,omitempty"`
}

// Calculation Pipeline Data Transfer Objects

type CalculatedLineItem struct {
	ChargeHeadID      uuid.UUID `json:"charge_head_id"`
	Name              string    `json:"name"`
	SACCode           *string   `json:"sac_code,omitempty"`
	TaxClassification string    `json:"tax_classification"`
	TaxRate           float64   `json:"tax_rate"`
	Rate              float64   `json:"rate"`
	Quantity          float64   `json:"quantity"`
	Amount            float64   `json:"amount"`
	TaxAmount         float64   `json:"tax_amount"`
	FormulaExpression *string   `json:"formula_expression,omitempty"`
	DisplayOrder      int       `json:"display_order"`
}

type UnitCalculationResult struct {
	UnitID           uuid.UUID            `json:"unit_id"`
	UnitNumber       string               `json:"unit_number"`
	MemberName       string               `json:"member_name"`
	UnitType         string               `json:"unit_type"`
	AreaSqFt         float64              `json:"area_sqft"`
	Status           string               `json:"status"` // ELIGIBLE, EXCLUDED, FAILED
	LineItems        []CalculatedLineItem `json:"line_items"`
	Subtotal         float64              `json:"subtotal"`
	TaxableAmount    float64              `json:"taxable_amount"`
	CGSTAmount       float64              `json:"cgst_amount"`
	SGSTAmount       float64              `json:"sgst_amount"`
	IGSTAmount       float64              `json:"igst_amount"`
	TotalTaxAmount   float64              `json:"total_tax_amount"`
	ArrearsAmount    float64              `json:"arrears_amount"`
	InterestAmount   float64              `json:"interest_amount"`
	AdvanceDeduction float64              `json:"advance_deduction"`
	PayableAmount    float64              `json:"payable_amount"`
	Warnings         []string             `json:"warnings"`
	ErrorMessage     *string              `json:"error_message,omitempty"`
}

type BatchCalculationSummary struct {
	EligibleUnitCount int                     `json:"eligible_unit_count"`
	ExcludedUnitCount int                     `json:"excluded_unit_count"`
	TotalSubtotal     float64                 `json:"total_subtotal"`
	TotalTaxable      float64                 `json:"total_taxable"`
	TotalCGST         float64                 `json:"total_cgst"`
	TotalSGST         float64                 `json:"total_sgst"`
	TotalIGST         float64                 `json:"total_igst"`
	TotalTax          float64                 `json:"total_tax"`
	TotalArrears      float64                 `json:"total_arrears"`
	TotalInterest     float64                 `json:"total_interest"`
	TotalAdvances     float64                 `json:"total_advances"`
	TotalPayable      float64                 `json:"total_payable"`
	Units             []UnitCalculationResult `json:"units"`
}

// Params Structs

type CreateBillTemplateParams struct {
	Name                 string                   `json:"name"`
	Description          *string                  `json:"description"`
	BillType             string                   `json:"bill_type"`
	BillingFrequency     string                   `json:"billing_frequency"`
	ApplicableUnitTypes  []string                 `json:"applicable_unit_types"`
	Currency             string                   `json:"currency"`
	BillTitle            string                   `json:"bill_title"`
	HeaderNotes          *string                  `json:"header_notes"`
	FooterText           *string                  `json:"footer_text"`
	PaymentInstructions  *string                  `json:"payment_instructions"`
	BankDetails          map[string]interface{}   `json:"bank_details"`
	WatermarkEnabled     bool                     `json:"watermark_enabled"`
	WatermarkOpacity     float64                  `json:"watermark_opacity"`
	Fields               []CreateTemplateFieldReq `json:"fields"`
}

type CreateTemplateFieldReq struct {
	FieldKey      string                 `json:"field_key"`
	FieldLabel    string                 `json:"field_label"`
	FieldCategory string                 `json:"field_category"`
	IsVisible     bool                   `json:"is_visible"`
	IsRequired    bool                   `json:"is_required"`
	DisplayOrder  int                    `json:"display_order"`
	DefaultValue  *string                `json:"default_value"`
	Formatting    map[string]interface{} `json:"formatting"`
}

type UpdateBillTemplateParams struct {
	Name                *string                  `json:"name"`
	Description         *string                  `json:"description"`
	BillTitle           *string                  `json:"bill_title"`
	HeaderNotes         *string                  `json:"header_notes"`
	FooterText          *string                  `json:"footer_text"`
	PaymentInstructions *string                  `json:"payment_instructions"`
	BankDetails         map[string]interface{}   `json:"bank_details"`
	WatermarkEnabled    *bool                    `json:"watermark_enabled"`
	WatermarkOpacity    *float64                 `json:"watermark_opacity"`
	Fields              []CreateTemplateFieldReq `json:"fields"`
}

type CreateChargeHeadParams struct {
	CategoryID        *uuid.UUID `json:"category_id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Description       *string    `json:"description"`
	SACCode           *string    `json:"sac_code"`
	TaxClassification string     `json:"tax_classification"`
	DefaultTaxRate    float64    `json:"default_tax_rate"`
	CalculationMethod string     `json:"calculation_method"`
	DefaultFormulaID  *uuid.UUID `json:"default_formula_id"`
	IsActive          bool       `json:"is_active"`
	DisplayOrder      int        `json:"display_order"`
}

type UpdateChargeHeadParams struct {
	Name              *string    `json:"name"`
	Description       *string    `json:"description"`
	SACCode           *string    `json:"sac_code"`
	TaxClassification *string    `json:"tax_classification"`
	DefaultTaxRate    *float64   `json:"default_tax_rate"`
	CalculationMethod *string    `json:"calculation_method"`
	DefaultFormulaID  *uuid.UUID `json:"default_formula_id"`
	IsActive          *bool      `json:"is_active"`
	DisplayOrder      *int       `json:"display_order"`
}

type CreateUnitBillingOverrideParams struct {
	ChargeHeadID              uuid.UUID  `json:"charge_head_id"`
	OverrideType              string     `json:"override_type"`
	OverrideValue             *float64   `json:"override_value"`
	FormulaOverrideExpression *string    `json:"formula_override_expression"`
	EffectiveFrom             time.Time  `json:"effective_from"`
	EffectiveUntil            *time.Time `json:"effective_until"`
	Reason                    string     `json:"reason"`
}

type CreateBillingBatchParams struct {
	BillingPeriodID uuid.UUID              `json:"billing_period_id"`
	TemplateID      uuid.UUID              `json:"template_id"`
	SelectionMode   string                 `json:"selection_mode"` // ALL_UNITS, SELECTED_UNITS, SINGLE_UNIT, UNIT_GROUP
	FilterCriteria  map[string]interface{} `json:"filter_criteria"`
	IdempotencyKey  *string                `json:"idempotency_key"`
}

// ------------------------------------------------------------------------------
// 1. Template Management Repository Implementation
// ------------------------------------------------------------------------------

func (r *Repository) ListBillTemplates(ctx context.Context, societyID uuid.UUID) ([]MaintenanceBillTemplate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, society_id, name, description, bill_type, billing_frequency, applicable_unit_types,
		       currency, bill_title, header_notes, footer_text, payment_instructions, bank_details,
		       watermark_enabled, watermark_opacity, status, current_version, created_by, created_at, updated_at
		FROM maintenance_bill_templates
		WHERE society_id = $1
		ORDER BY created_at DESC
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []MaintenanceBillTemplate
	for rows.Next() {
		var t MaintenanceBillTemplate
		var unitTypesBytes, bankDetailsBytes []byte
		if err := rows.Scan(
			&t.ID, &t.SocietyID, &t.Name, &t.Description, &t.BillType, &t.BillingFrequency,
			&unitTypesBytes, &t.Currency, &t.BillTitle, &t.HeaderNotes, &t.FooterText,
			&t.PaymentInstructions, &bankDetailsBytes, &t.WatermarkEnabled, &t.WatermarkOpacity,
			&t.Status, &t.CurrentVersion, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		_ = json.Unmarshal(unitTypesBytes, &t.ApplicableUnitTypes)
		_ = json.Unmarshal(bankDetailsBytes, &t.BankDetails)
		templates = append(templates, t)
	}

	if templates == nil {
		templates = []MaintenanceBillTemplate{}
	}
	return templates, nil
}

func (r *Repository) GetBillTemplate(ctx context.Context, societyID, templateID uuid.UUID) (*MaintenanceBillTemplate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var t MaintenanceBillTemplate
	var unitTypesBytes, bankDetailsBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, society_id, name, description, bill_type, billing_frequency, applicable_unit_types,
		       currency, bill_title, header_notes, footer_text, payment_instructions, bank_details,
		       watermark_enabled, watermark_opacity, status, current_version, created_by, created_at, updated_at
		FROM maintenance_bill_templates
		WHERE id = $1 AND society_id = $2
	`, templateID, societyID).Scan(
		&t.ID, &t.SocietyID, &t.Name, &t.Description, &t.BillType, &t.BillingFrequency,
		&unitTypesBytes, &t.Currency, &t.BillTitle, &t.HeaderNotes, &t.FooterText,
		&t.PaymentInstructions, &bankDetailsBytes, &t.WatermarkEnabled, &t.WatermarkOpacity,
		&t.Status, &t.CurrentVersion, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get template: %w", err)
	}

	_ = json.Unmarshal(unitTypesBytes, &t.ApplicableUnitTypes)
	_ = json.Unmarshal(bankDetailsBytes, &t.BankDetails)

	// Fetch template fields
	fRows, err := r.pool.Query(ctx, `
		SELECT id, society_id, template_id, field_key, field_label, field_category,
		       is_visible, is_required, display_order, default_value, formatting, created_at, updated_at
		FROM maintenance_template_fields
		WHERE template_id = $1 AND society_id = $2
		ORDER BY display_order ASC, field_key ASC
	`, templateID, societyID)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var f MaintenanceTemplateField
			var fmtBytes []byte
			if err := fRows.Scan(
				&f.ID, &f.SocietyID, &f.TemplateID, &f.FieldKey, &f.FieldLabel, &f.FieldCategory,
				&f.IsVisible, &f.IsRequired, &f.DisplayOrder, &f.DefaultValue, &fmtBytes, &f.CreatedAt, &f.UpdatedAt,
			); err == nil {
				_ = json.Unmarshal(fmtBytes, &f.Formatting)
				t.Fields = append(t.Fields, f)
			}
		}
	}

	return &t, nil
}

func (r *Repository) CreateBillTemplate(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateBillTemplateParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillTemplate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("template name is required")
	}
	if params.BillType == "" {
		params.BillType = "MAINTENANCE"
	}
	if params.BillingFrequency == "" {
		params.BillingFrequency = "MONTHLY"
	}
	if params.Currency == "" {
		params.Currency = "INR"
	}
	if params.BillTitle == "" {
		params.BillTitle = "Tax Invoice / Maintenance Bill"
	}
	if params.ApplicableUnitTypes == nil || len(params.ApplicableUnitTypes) == 0 {
		params.ApplicableUnitTypes = []string{"RESIDENTIAL", "COMMERCIAL", "PENTHOUSE", "PARKING", "OTHER"}
	}
	if params.WatermarkOpacity <= 0 {
		params.WatermarkOpacity = 0.08
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_bill_templates
			WHERE society_id = $1 AND LOWER(TRIM(name)) = LOWER($2)
		)
	`, societyID, name).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check template exists: %w", err)
	}
	if exists {
		return nil, ErrTemplateNameExists
	}

	if params.BankDetails == nil {
		params.BankDetails = make(map[string]interface{})
	}
	unitTypesBytes, _ := json.Marshal(params.ApplicableUnitTypes)
	bankDetailsBytes, _ := json.Marshal(params.BankDetails)

	tpl := MaintenanceBillTemplate{
		SocietyID:           societyID,
		Name:                name,
		Description:         params.Description,
		BillType:            params.BillType,
		BillingFrequency:    params.BillingFrequency,
		ApplicableUnitTypes: params.ApplicableUnitTypes,
		Currency:            params.Currency,
		BillTitle:           params.BillTitle,
		HeaderNotes:         params.HeaderNotes,
		FooterText:          params.FooterText,
		PaymentInstructions: params.PaymentInstructions,
		BankDetails:         params.BankDetails,
		WatermarkEnabled:    params.WatermarkEnabled,
		WatermarkOpacity:    params.WatermarkOpacity,
		Status:              "DRAFT",
		CurrentVersion:      1,
		CreatedBy:           actorID,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_bill_templates (
			society_id, name, description, bill_type, billing_frequency, applicable_unit_types,
			currency, bill_title, header_notes, footer_text, payment_instructions, bank_details,
			watermark_enabled, watermark_opacity, status, current_version, created_by
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12::jsonb, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at
	`, societyID, tpl.Name, tpl.Description, tpl.BillType, tpl.BillingFrequency, string(unitTypesBytes),
		tpl.Currency, tpl.BillTitle, tpl.HeaderNotes, tpl.FooterText, tpl.PaymentInstructions,
		string(bankDetailsBytes), tpl.WatermarkEnabled, tpl.WatermarkOpacity, tpl.Status, tpl.CurrentVersion, tpl.CreatedBy,
	).Scan(&tpl.ID, &tpl.CreatedAt, &tpl.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert template: %w", err)
	}

	// Insert Fields if provided
	for i, f := range params.Fields {
		fKey := strings.TrimSpace(f.FieldKey)
		if fKey == "" {
			continue
		}
		if f.FieldLabel == "" {
			f.FieldLabel = fKey
		}
		if f.Formatting == nil {
			f.Formatting = make(map[string]interface{})
		}
		fmtBytes, _ := json.Marshal(f.Formatting)
		var fieldID uuid.UUID
		var fCreated, fUpdated time.Time
		err = tx.QueryRow(ctx, `
			INSERT INTO maintenance_template_fields (
				society_id, template_id, field_key, field_label, field_category,
				is_visible, is_required, display_order, default_value, formatting
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
			RETURNING id, created_at, updated_at
		`, societyID, tpl.ID, fKey, f.FieldLabel, f.FieldCategory, f.IsVisible, f.IsRequired, i, f.DefaultValue, string(fmtBytes)).
			Scan(&fieldID, &fCreated, &fUpdated)
		if err == nil {
			tpl.Fields = append(tpl.Fields, MaintenanceTemplateField{
				ID:            fieldID,
				SocietyID:     societyID,
				TemplateID:    tpl.ID,
				FieldKey:      fKey,
				FieldLabel:    f.FieldLabel,
				FieldCategory: f.FieldCategory,
				IsVisible:     f.IsVisible,
				IsRequired:    f.IsRequired,
				DisplayOrder:  i,
				DefaultValue:  f.DefaultValue,
				Formatting:    f.Formatting,
				CreatedAt:     fCreated,
				UpdatedAt:     fUpdated,
			})
		}
	}

	// Audit Log
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"template_id": tpl.ID,
		"name":        tpl.Name,
		"bill_type":   tpl.BillType,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILL_TEMPLATE_CREATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &tpl, nil
}

func (r *Repository) UpdateBillTemplate(
	ctx context.Context,
	societyID, templateID uuid.UUID,
	params UpdateBillTemplateParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillTemplate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tpl, err := r.GetBillTemplate(ctx, societyID, templateID)
	if err != nil {
		return nil, err
	}

	// Check if active/published: published templates cannot have arbitrary in-place field updates
	// unless in draft or transitioning.
	if tpl.Status == "ACTIVE" {
		return nil, ErrTemplateImmutable
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if params.Name != nil && strings.TrimSpace(*params.Name) != "" {
		tpl.Name = strings.TrimSpace(*params.Name)
	}
	if params.Description != nil {
		tpl.Description = params.Description
	}
	if params.BillTitle != nil && strings.TrimSpace(*params.BillTitle) != "" {
		tpl.BillTitle = strings.TrimSpace(*params.BillTitle)
	}
	if params.HeaderNotes != nil {
		tpl.HeaderNotes = params.HeaderNotes
	}
	if params.FooterText != nil {
		tpl.FooterText = params.FooterText
	}
	if params.PaymentInstructions != nil {
		tpl.PaymentInstructions = params.PaymentInstructions
	}
	if params.BankDetails != nil {
		tpl.BankDetails = params.BankDetails
	}
	if params.WatermarkEnabled != nil {
		tpl.WatermarkEnabled = *params.WatermarkEnabled
	}
	if params.WatermarkOpacity != nil && *params.WatermarkOpacity >= 0 {
		tpl.WatermarkOpacity = *params.WatermarkOpacity
	}

	if tpl.BankDetails == nil {
		tpl.BankDetails = make(map[string]interface{})
	}
	bankDetailsBytes, _ := json.Marshal(tpl.BankDetails)

	err = tx.QueryRow(ctx, `
		UPDATE maintenance_bill_templates
		SET name = $1, description = $2, bill_title = $3, header_notes = $4, footer_text = $5,
		    payment_instructions = $6, bank_details = $7::jsonb, watermark_enabled = $8,
		    watermark_opacity = $9, updated_at = NOW()
		WHERE id = $10 AND society_id = $11
		RETURNING updated_at
	`, tpl.Name, tpl.Description, tpl.BillTitle, tpl.HeaderNotes, tpl.FooterText,
		tpl.PaymentInstructions, string(bankDetailsBytes), tpl.WatermarkEnabled, tpl.WatermarkOpacity,
		templateID, societyID).Scan(&tpl.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}

	// If fields are passed, replace fields
	if len(params.Fields) > 0 {
		_, _ = tx.Exec(ctx, `DELETE FROM maintenance_template_fields WHERE template_id = $1 AND society_id = $2`, templateID, societyID)
		tpl.Fields = nil
		for i, f := range params.Fields {
			fKey := strings.TrimSpace(f.FieldKey)
			if fKey == "" {
				continue
			}
			if f.Formatting == nil {
				f.Formatting = make(map[string]interface{})
			}
			fmtBytes, _ := json.Marshal(f.Formatting)
			var fieldID uuid.UUID
			var fCreated, fUpdated time.Time
			_ = tx.QueryRow(ctx, `
				INSERT INTO maintenance_template_fields (
					society_id, template_id, field_key, field_label, field_category,
					is_visible, is_required, display_order, default_value, formatting
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
				RETURNING id, created_at, updated_at
			`, societyID, templateID, fKey, f.FieldLabel, f.FieldCategory, f.IsVisible, f.IsRequired, i, f.DefaultValue, string(fmtBytes)).
				Scan(&fieldID, &fCreated, &fUpdated)

			tpl.Fields = append(tpl.Fields, MaintenanceTemplateField{
				ID:            fieldID,
				SocietyID:     societyID,
				TemplateID:    templateID,
				FieldKey:      fKey,
				FieldLabel:    f.FieldLabel,
				FieldCategory: f.FieldCategory,
				IsVisible:     f.IsVisible,
				IsRequired:    f.IsRequired,
				DisplayOrder:  i,
				DefaultValue:  f.DefaultValue,
				Formatting:    f.Formatting,
				CreatedAt:     fCreated,
				UpdatedAt:     fUpdated,
			})
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return tpl, nil
}

func (r *Repository) DuplicateBillTemplate(
	ctx context.Context,
	societyID, templateID uuid.UUID,
	newName string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillTemplate, error) {
	orig, err := r.GetBillTemplate(ctx, societyID, templateID)
	if err != nil {
		return nil, err
	}

	dupName := strings.TrimSpace(newName)
	if dupName == "" {
		dupName = fmt.Sprintf("%s (Copy)", orig.Name)
	}

	var fieldsReq []CreateTemplateFieldReq
	for _, f := range orig.Fields {
		fieldsReq = append(fieldsReq, CreateTemplateFieldReq{
			FieldKey:      f.FieldKey,
			FieldLabel:    f.FieldLabel,
			FieldCategory: f.FieldCategory,
			IsVisible:     f.IsVisible,
			IsRequired:    f.IsRequired,
			DisplayOrder:  f.DisplayOrder,
			DefaultValue:  f.DefaultValue,
			Formatting:    f.Formatting,
		})
	}

	return r.CreateBillTemplate(ctx, societyID, CreateBillTemplateParams{
		Name:                 dupName,
		Description:          orig.Description,
		BillType:             orig.BillType,
		BillingFrequency:     orig.BillingFrequency,
		ApplicableUnitTypes:  orig.ApplicableUnitTypes,
		Currency:             orig.Currency,
		BillTitle:            orig.BillTitle,
		HeaderNotes:          orig.HeaderNotes,
		FooterText:           orig.FooterText,
		PaymentInstructions:  orig.PaymentInstructions,
		BankDetails:          orig.BankDetails,
		WatermarkEnabled:     orig.WatermarkEnabled,
		WatermarkOpacity:     orig.WatermarkOpacity,
		Fields:               fieldsReq,
	}, actorID, clientIP, userAgent)
}

func (r *Repository) PublishBillTemplate(
	ctx context.Context,
	societyID, templateID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillTemplateVersion, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tpl, err := r.GetBillTemplate(ctx, societyID, templateID)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Fetch highest version
	var maxVer int
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM maintenance_bill_template_versions
		WHERE template_id = $1 AND society_id = $2
	`, templateID, societyID).Scan(&maxVer)

	newVersion := maxVer + 1

	// Create snapshot
	snapshotJSON, _ := json.Marshal(tpl)
	var snapshotMap map[string]interface{}
	_ = json.Unmarshal(snapshotJSON, &snapshotMap)

	var ver MaintenanceBillTemplateVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_bill_template_versions (
			society_id, template_id, version, definition_snapshot, published_by
		) VALUES ($1, $2, $3, $4::jsonb, $5)
		RETURNING id, published_at
	`, societyID, templateID, newVersion, string(snapshotJSON), actorID).Scan(&ver.ID, &ver.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("insert template version: %w", err)
	}

	ver.SocietyID = societyID
	ver.TemplateID = templateID
	ver.Version = newVersion
	ver.DefinitionSnapshot = snapshotMap
	ver.PublishedBy = actorID

	// Update template status to ACTIVE and increment current_version
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_bill_templates
		SET status = 'ACTIVE', current_version = $1, updated_at = NOW()
		WHERE id = $2 AND society_id = $3
	`, newVersion, templateID, societyID)
	if err != nil {
		return nil, fmt.Errorf("update template status: %w", err)
	}

	// Audit Log
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"template_id": templateID,
		"version":     newVersion,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILL_TEMPLATE_PUBLISHED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &ver, nil
}

func (r *Repository) ArchiveBillTemplate(
	ctx context.Context,
	societyID, templateID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	res, err := r.pool.Exec(ctx, `
		UPDATE maintenance_bill_templates
		SET status = 'ARCHIVED', updated_at = NOW()
		WHERE id = $1 AND society_id = $2
	`, templateID, societyID)
	if err != nil {
		return fmt.Errorf("archive template: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}

	// Audit log
	auditDetails, _ := json.Marshal(map[string]interface{}{"template_id": templateID})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILL_TEMPLATE_ARCHIVED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return nil
}

func (r *Repository) ListBillTemplateVersions(ctx context.Context, societyID, templateID uuid.UUID) ([]MaintenanceBillTemplateVersion, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, society_id, template_id, version, definition_snapshot, published_by, published_at
		FROM maintenance_bill_template_versions
		WHERE template_id = $1 AND society_id = $2
		ORDER BY version DESC
	`, templateID, societyID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []MaintenanceBillTemplateVersion
	for rows.Next() {
		var v MaintenanceBillTemplateVersion
		var snapBytes []byte
		if err := rows.Scan(&v.ID, &v.SocietyID, &v.TemplateID, &v.Version, &snapBytes, &v.PublishedBy, &v.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		_ = json.Unmarshal(snapBytes, &v.DefinitionSnapshot)
		versions = append(versions, v)
	}

	if versions == nil {
		versions = []MaintenanceBillTemplateVersion{}
	}
	return versions, nil
}

// ------------------------------------------------------------------------------
// 2. Charge Head Management Repository Implementation
// ------------------------------------------------------------------------------

func (r *Repository) ListChargeHeads(ctx context.Context, societyID uuid.UUID, activeOnly bool) ([]MaintenanceChargeHead, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT h.id, h.society_id, h.category_id, c.name as category_name, h.code, h.name,
		       h.description, h.sac_code, h.tax_classification, h.default_tax_rate,
		       h.calculation_method, h.default_formula_id, f.expression as formula_expression,
		       h.is_active, h.display_order, h.created_by, h.created_at, h.updated_at
		FROM maintenance_charge_heads h
		LEFT JOIN maintenance_charge_categories c ON h.category_id = c.id
		LEFT JOIN maintenance_formula_definitions f ON h.default_formula_id = f.id
		WHERE h.society_id = $1
	`
	if activeOnly {
		query += " AND h.is_active = TRUE"
	}
	query += " ORDER BY h.display_order ASC, h.code ASC"

	rows, err := r.pool.Query(ctx, query, societyID)
	if err != nil {
		return nil, fmt.Errorf("list charge heads: %w", err)
	}
	defer rows.Close()

	var heads []MaintenanceChargeHead
	for rows.Next() {
		var h MaintenanceChargeHead
		if err := rows.Scan(
			&h.ID, &h.SocietyID, &h.CategoryID, &h.CategoryName, &h.Code, &h.Name,
			&h.Description, &h.SACCode, &h.TaxClassification, &h.DefaultTaxRate,
			&h.CalculationMethod, &h.DefaultFormulaID, &h.FormulaExpression,
			&h.IsActive, &h.DisplayOrder, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan charge head: %w", err)
		}
		heads = append(heads, h)
	}

	if heads == nil {
		heads = []MaintenanceChargeHead{}
	}
	return heads, nil
}

func (r *Repository) GetChargeHead(ctx context.Context, societyID, headID uuid.UUID) (*MaintenanceChargeHead, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var h MaintenanceChargeHead
	err := r.pool.QueryRow(ctx, `
		SELECT h.id, h.society_id, h.category_id, c.name as category_name, h.code, h.name,
		       h.description, h.sac_code, h.tax_classification, h.default_tax_rate,
		       h.calculation_method, h.default_formula_id, f.expression as formula_expression,
		       h.is_active, h.display_order, h.created_by, h.created_at, h.updated_at
		FROM maintenance_charge_heads h
		LEFT JOIN maintenance_charge_categories c ON h.category_id = c.id
		LEFT JOIN maintenance_formula_definitions f ON h.default_formula_id = f.id
		WHERE h.id = $1 AND h.society_id = $2
	`, headID, societyID).Scan(
		&h.ID, &h.SocietyID, &h.CategoryID, &h.CategoryName, &h.Code, &h.Name,
		&h.Description, &h.SACCode, &h.TaxClassification, &h.DefaultTaxRate,
		&h.CalculationMethod, &h.DefaultFormulaID, &h.FormulaExpression,
		&h.IsActive, &h.DisplayOrder, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChargeHeadNotFound
		}
		return nil, fmt.Errorf("get charge head: %w", err)
	}

	return &h, nil
}

func (r *Repository) CreateChargeHead(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateChargeHeadParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceChargeHead, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	code := strings.ToUpper(strings.TrimSpace(params.Code))
	if code == "" {
		return nil, errors.New("charge head code is required")
	}
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, errors.New("charge head name is required")
	}
	if params.TaxClassification == "" {
		params.TaxClassification = "EXEMPT"
	}
	calcMethod := strings.ToUpper(strings.TrimSpace(params.CalculationMethod))
	if calcMethod == "AREA" {
		calcMethod = "AREA_MULTIPLIER"
	} else if calcMethod == "" {
		calcMethod = "FIXED"
	}
	params.CalculationMethod = calcMethod

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM maintenance_charge_heads
			WHERE society_id = $1 AND code = $2
		)
	`, societyID, code).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check charge head code exists: %w", err)
	}
	if exists {
		return nil, ErrChargeHeadCodeExists
	}

	head := MaintenanceChargeHead{
		SocietyID:         societyID,
		CategoryID:        params.CategoryID,
		Code:              code,
		Name:              name,
		Description:       params.Description,
		SACCode:           params.SACCode,
		TaxClassification: params.TaxClassification,
		DefaultTaxRate:    params.DefaultTaxRate,
		CalculationMethod: params.CalculationMethod,
		DefaultFormulaID:  params.DefaultFormulaID,
		IsActive:          params.IsActive,
		DisplayOrder:      params.DisplayOrder,
		CreatedBy:         actorID,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO maintenance_charge_heads (
			society_id, category_id, code, name, description, sac_code,
			tax_classification, default_tax_rate, calculation_method,
			default_formula_id, is_active, display_order, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`, societyID, head.CategoryID, head.Code, head.Name, head.Description, head.SACCode,
		head.TaxClassification, head.DefaultTaxRate, head.CalculationMethod,
		head.DefaultFormulaID, head.IsActive, head.DisplayOrder, head.CreatedBy,
	).Scan(&head.ID, &head.CreatedAt, &head.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert charge head: %w", err)
	}

	// Audit Log
	auditDetails, _ := json.Marshal(map[string]interface{}{
		"head_id":  head.ID,
		"code":     head.Code,
		"name":     head.Name,
		"sac_code": head.SACCode,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'CHARGE_HEAD_CREATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &head, nil
}

func (r *Repository) UpdateChargeHead(
	ctx context.Context,
	societyID, headID uuid.UUID,
	params UpdateChargeHeadParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceChargeHead, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	head, err := r.GetChargeHead(ctx, societyID, headID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil && strings.TrimSpace(*params.Name) != "" {
		head.Name = strings.TrimSpace(*params.Name)
	}
	if params.Description != nil {
		head.Description = params.Description
	}
	if params.SACCode != nil {
		head.SACCode = params.SACCode
	}
	if params.TaxClassification != nil {
		head.TaxClassification = *params.TaxClassification
	}
	if params.DefaultTaxRate != nil && *params.DefaultTaxRate >= 0 {
		head.DefaultTaxRate = *params.DefaultTaxRate
	}
	if params.CalculationMethod != nil {
		cm := strings.ToUpper(strings.TrimSpace(*params.CalculationMethod))
		if cm == "AREA" {
			cm = "AREA_MULTIPLIER"
		}
		head.CalculationMethod = cm
	}
	if params.DefaultFormulaID != nil {
		head.DefaultFormulaID = params.DefaultFormulaID
	}
	if params.IsActive != nil {
		head.IsActive = *params.IsActive
	}
	if params.DisplayOrder != nil {
		head.DisplayOrder = *params.DisplayOrder
	}

	err = r.pool.QueryRow(ctx, `
		UPDATE maintenance_charge_heads
		SET name = $1, description = $2, sac_code = $3, tax_classification = $4,
		    default_tax_rate = $5, calculation_method = $6, default_formula_id = $7,
		    is_active = $8, display_order = $9, updated_at = NOW()
		WHERE id = $10 AND society_id = $11
		RETURNING updated_at
	`, head.Name, head.Description, head.SACCode, head.TaxClassification,
		head.DefaultTaxRate, head.CalculationMethod, head.DefaultFormulaID,
		head.IsActive, head.DisplayOrder, headID, societyID).Scan(&head.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update charge head: %w", err)
	}

	// Audit Log
	auditDetails, _ := json.Marshal(map[string]interface{}{"head_id": headID})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'CHARGE_HEAD_UPDATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return head, nil
}

func (r *Repository) ArchiveChargeHead(
	ctx context.Context,
	societyID, headID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	res, err := r.pool.Exec(ctx, `
		UPDATE maintenance_charge_heads
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1 AND society_id = $2
	`, headID, societyID)
	if err != nil {
		return fmt.Errorf("archive charge head: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrChargeHeadNotFound
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{"head_id": headID})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'CHARGE_HEAD_ARCHIVED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return nil
}

// ------------------------------------------------------------------------------
// 3. Unit-Specific Billing Overrides Implementation
// ------------------------------------------------------------------------------

func (r *Repository) ListUnitBillingOverrides(ctx context.Context, societyID, unitID uuid.UUID) ([]MaintenanceUnitBillingOverride, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.society_id, o.unit_id, u.unit_number, o.charge_head_id, h.name as charge_head_name,
		       o.override_type, o.override_value, o.formula_override_expression, o.effective_from,
		       o.effective_until, o.reason, o.status, o.created_by, o.created_at, o.updated_at
		FROM maintenance_unit_billing_overrides o
		JOIN society_units u ON o.unit_id = u.id AND o.society_id = u.society_id
		JOIN maintenance_charge_heads h ON o.charge_head_id = h.id AND o.society_id = h.society_id
		WHERE o.unit_id = $1 AND o.society_id = $2
		ORDER BY o.effective_from DESC, o.created_at DESC
	`, unitID, societyID)
	if err != nil {
		return nil, fmt.Errorf("list unit overrides: %w", err)
	}
	defer rows.Close()

	var overrides []MaintenanceUnitBillingOverride
	for rows.Next() {
		var o MaintenanceUnitBillingOverride
		if err := rows.Scan(
			&o.ID, &o.SocietyID, &o.UnitID, &o.UnitNumber, &o.ChargeHeadID, &o.ChargeHeadName,
			&o.OverrideType, &o.OverrideValue, &o.FormulaOverrideExpression, &o.EffectiveFrom,
			&o.EffectiveUntil, &o.Reason, &o.Status, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan unit override: %w", err)
		}
		overrides = append(overrides, o)
	}

	if overrides == nil {
		overrides = []MaintenanceUnitBillingOverride{}
	}
	return overrides, nil
}

func (r *Repository) CreateUnitBillingOverride(
	ctx context.Context,
	societyID, unitID uuid.UUID,
	params CreateUnitBillingOverrideParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceUnitBillingOverride, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	reason := strings.TrimSpace(params.Reason)
	if reason == "" {
		return nil, errors.New("reason is required for an explicit billing override")
	}
	if params.OverrideType == "" {
		params.OverrideType = "FIXED_AMOUNT"
	}
	if params.EffectiveFrom.IsZero() {
		params.EffectiveFrom = time.Now().Truncate(24 * time.Hour)
	}

	var ov MaintenanceUnitBillingOverride
	err := r.pool.QueryRow(ctx, `
		INSERT INTO maintenance_unit_billing_overrides (
			society_id, unit_id, charge_head_id, override_type, override_value,
			formula_override_expression, effective_from, effective_until, reason, status, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'ACTIVE', $10)
		RETURNING id, created_at, updated_at
	`, societyID, unitID, params.ChargeHeadID, params.OverrideType, params.OverrideValue,
		params.FormulaOverrideExpression, params.EffectiveFrom, params.EffectiveUntil, reason, actorID,
	).Scan(&ov.ID, &ov.CreatedAt, &ov.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert unit override: %w", err)
	}

	ov.SocietyID = societyID
	ov.UnitID = unitID
	ov.ChargeHeadID = params.ChargeHeadID
	ov.OverrideType = params.OverrideType
	ov.OverrideValue = params.OverrideValue
	ov.FormulaOverrideExpression = params.FormulaOverrideExpression
	ov.EffectiveFrom = params.EffectiveFrom
	ov.EffectiveUntil = params.EffectiveUntil
	ov.Reason = reason
	ov.Status = "ACTIVE"
	ov.CreatedBy = actorID

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"override_id": ov.ID,
		"unit_id":     unitID,
		"head_id":     params.ChargeHeadID,
		"type":        params.OverrideType,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'UNIT_OVERRIDE_CREATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return &ov, nil
}

func (r *Repository) UpdateUnitBillingOverride(
	ctx context.Context,
	societyID, overrideID uuid.UUID,
	overrideValue *float64,
	status *string,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	res, err := r.pool.Exec(ctx, `
		UPDATE maintenance_unit_billing_overrides
		SET override_value = COALESCE($1, override_value),
		    status = COALESCE($2, status),
		    updated_at = NOW()
		WHERE id = $3 AND society_id = $4
	`, overrideValue, status, overrideID, societyID)
	if err != nil {
		return fmt.Errorf("update override: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrUnitOverrideNotFound
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{"override_id": overrideID})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'UNIT_OVERRIDE_UPDATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return nil
}

func (r *Repository) DeleteUnitBillingOverride(
	ctx context.Context,
	societyID, overrideID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	res, err := r.pool.Exec(ctx, `
		DELETE FROM maintenance_unit_billing_overrides
		WHERE id = $1 AND society_id = $2
	`, overrideID, societyID)
	if err != nil {
		return fmt.Errorf("delete override: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrUnitOverrideNotFound
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{"override_id": overrideID})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'UNIT_OVERRIDE_DELETED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return nil
}

// ------------------------------------------------------------------------------
// 4. Billing Batches & Calculation Engine Implementation
// ------------------------------------------------------------------------------

func (r *Repository) CreateBillingBatch(
	ctx context.Context,
	societyID uuid.UUID,
	params CreateBillingBatchParams,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Handle idempotency check
	if params.IdempotencyKey != nil && strings.TrimSpace(*params.IdempotencyKey) != "" {
		existing, err := r.GetBillingBatchByIdempotencyKey(ctx, societyID, strings.TrimSpace(*params.IdempotencyKey))
		if err == nil && existing != nil {
			return existing, nil
		}
	}

	if params.SelectionMode == "" {
		params.SelectionMode = "ALL_UNITS"
	}
	if params.FilterCriteria == nil {
		params.FilterCriteria = make(map[string]interface{})
	}

	// Verify billing period exists and is not closed
	period, err := r.GetBillingPeriod(ctx, societyID, params.BillingPeriodID)
	if err != nil {
		return nil, err
	}
	if period.Status == "CLOSED" {
		return nil, errors.New("cannot create billing batch for closed billing period")
	}

	// Verify template exists
	tpl, err := r.GetBillTemplate(ctx, societyID, params.TemplateID)
	if err != nil {
		return nil, err
	}

	// Next sequential batch number: BATCH-YYYYMM-XXXX
	var nextSeq int64
	err = r.pool.QueryRow(ctx, `SELECT nextval('maintenance_batch_seq')`).Scan(&nextSeq)
	if err != nil {
		nextSeq = time.Now().Unix() % 100000
	}
	batchNumber := fmt.Sprintf("BATCH-%s-%04d", period.PeriodStart.Format("200601"), nextSeq)

	filterJSON, _ := json.Marshal(params.FilterCriteria)

	var b MaintenanceBillingBatch
	err = r.pool.QueryRow(ctx, `
		INSERT INTO maintenance_billing_batches (
			society_id, billing_period_id, template_id, batch_number, selection_mode,
			filter_criteria, target_unit_count, processed_unit_count, status, idempotency_key, created_by
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, 0, 0, 'DRAFT_PREVIEW', $7, $8)
		RETURNING id, created_at, updated_at
	`, societyID, params.BillingPeriodID, params.TemplateID, batchNumber, params.SelectionMode,
		string(filterJSON), params.IdempotencyKey, actorID,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_maint_batches_idempotency") && params.IdempotencyKey != nil {
			return r.GetBillingBatchByIdempotencyKey(ctx, societyID, strings.TrimSpace(*params.IdempotencyKey))
		}
		return nil, fmt.Errorf("insert billing batch: %w", err)
	}

	b.SocietyID = societyID
	b.BillingPeriodID = params.BillingPeriodID
	b.PeriodName = period.PeriodName
	b.TemplateID = params.TemplateID
	b.TemplateName = tpl.Name
	b.BatchNumber = batchNumber
	b.SelectionMode = params.SelectionMode
	b.FilterCriteria = params.FilterCriteria
	b.Status = "DRAFT_PREVIEW"
	b.IdempotencyKey = params.IdempotencyKey
	b.CreatedBy = actorID

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"batch_id":     b.ID,
		"batch_number": b.BatchNumber,
	})
	_, _ = r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_BATCH_CREATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return &b, nil
}

func (r *Repository) GetBillingBatch(ctx context.Context, societyID, batchID uuid.UUID) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var b MaintenanceBillingBatch
	var filterBytes, errorBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT b.id, b.society_id, b.billing_period_id, p.period_name, b.template_id, t.name as template_name,
		       b.batch_number, b.selection_mode, b.filter_criteria, b.target_unit_count, b.processed_unit_count,
		       b.successful_unit_count, b.failed_unit_count, b.total_amount, b.status, b.idempotency_key, b.error_summary,
		       b.created_by, b.finalized_by, b.finalized_at, b.started_at, b.completed_at, b.cancelled_at,
		       b.cancelled_by, b.cancellation_reason, b.created_at, b.updated_at
		FROM maintenance_billing_batches b
		JOIN maintenance_billing_periods p ON b.billing_period_id = p.id AND b.society_id = p.society_id
		JOIN maintenance_bill_templates t ON b.template_id = t.id AND b.society_id = t.society_id
		WHERE b.id = $1 AND b.society_id = $2
	`, batchID, societyID).Scan(
		&b.ID, &b.SocietyID, &b.BillingPeriodID, &b.PeriodName, &b.TemplateID, &b.TemplateName,
		&b.BatchNumber, &b.SelectionMode, &filterBytes, &b.TargetUnitCount, &b.ProcessedUnitCount,
		&b.SuccessfulUnitCount, &b.FailedUnitCount, &b.TotalAmount, &b.Status, &b.IdempotencyKey, &errorBytes,
		&b.CreatedBy, &b.FinalizedBy, &b.FinalizedAt, &b.StartedAt, &b.CompletedAt, &b.CancelledAt,
		&b.CancelledBy, &b.CancellationReason, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("get billing batch: %w", err)
	}

	_ = json.Unmarshal(filterBytes, &b.FilterCriteria)
	if len(errorBytes) > 0 {
		_ = json.Unmarshal(errorBytes, &b.ErrorSummary)
	}

	return &b, nil
}

func (r *Repository) GetBillingBatchByIdempotencyKey(ctx context.Context, societyID uuid.UUID, key string) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var b MaintenanceBillingBatch
	var filterBytes, errorBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT b.id, b.society_id, b.billing_period_id, p.period_name, b.template_id, t.name as template_name,
		       b.batch_number, b.selection_mode, b.filter_criteria, b.target_unit_count, b.processed_unit_count,
		       b.successful_unit_count, b.failed_unit_count, b.total_amount, b.status, b.idempotency_key, b.error_summary,
		       b.created_by, b.finalized_by, b.finalized_at, b.started_at, b.completed_at, b.cancelled_at,
		       b.cancelled_by, b.cancellation_reason, b.created_at, b.updated_at
		FROM maintenance_billing_batches b
		JOIN maintenance_billing_periods p ON b.billing_period_id = p.id AND b.society_id = p.society_id
		JOIN maintenance_bill_templates t ON b.template_id = t.id AND b.society_id = t.society_id
		WHERE b.society_id = $1 AND b.idempotency_key = $2
	`, societyID, key).Scan(
		&b.ID, &b.SocietyID, &b.BillingPeriodID, &b.PeriodName, &b.TemplateID, &b.TemplateName,
		&b.BatchNumber, &b.SelectionMode, &filterBytes, &b.TargetUnitCount, &b.ProcessedUnitCount,
		&b.SuccessfulUnitCount, &b.FailedUnitCount, &b.TotalAmount, &b.Status, &b.IdempotencyKey, &errorBytes,
		&b.CreatedBy, &b.FinalizedBy, &b.FinalizedAt, &b.StartedAt, &b.CompletedAt, &b.CancelledAt,
		&b.CancelledBy, &b.CancellationReason, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("get billing batch by idempotency key: %w", err)
	}

	_ = json.Unmarshal(filterBytes, &b.FilterCriteria)
	if len(errorBytes) > 0 {
		_ = json.Unmarshal(errorBytes, &b.ErrorSummary)
	}

	return &b, nil
}

func (r *Repository) ListBillingBatches(ctx context.Context, societyID uuid.UUID, limit, offset int) ([]MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx, `
		SELECT b.id, b.society_id, b.billing_period_id, p.period_name, b.template_id, t.name as template_name,
		       b.batch_number, b.selection_mode, b.filter_criteria, b.target_unit_count, b.processed_unit_count,
		       b.successful_unit_count, b.failed_unit_count, b.total_amount, b.status, b.idempotency_key, b.error_summary,
		       b.created_by, b.finalized_by, b.finalized_at, b.started_at, b.completed_at, b.cancelled_at,
		       b.cancelled_by, b.cancellation_reason, b.created_at, b.updated_at
		FROM maintenance_billing_batches b
		JOIN maintenance_billing_periods p ON b.billing_period_id = p.id AND b.society_id = p.society_id
		JOIN maintenance_bill_templates t ON b.template_id = t.id AND b.society_id = t.society_id
		WHERE b.society_id = $1
		ORDER BY b.created_at DESC
		LIMIT $2 OFFSET $3
	`, societyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()

	var batches []MaintenanceBillingBatch
	for rows.Next() {
		var b MaintenanceBillingBatch
		var filterBytes, errorBytes []byte
		if err := rows.Scan(
			&b.ID, &b.SocietyID, &b.BillingPeriodID, &b.PeriodName, &b.TemplateID, &b.TemplateName,
			&b.BatchNumber, &b.SelectionMode, &filterBytes, &b.TargetUnitCount, &b.ProcessedUnitCount,
			&b.SuccessfulUnitCount, &b.FailedUnitCount, &b.TotalAmount, &b.Status, &b.IdempotencyKey, &errorBytes,
			&b.CreatedBy, &b.FinalizedBy, &b.FinalizedAt, &b.StartedAt, &b.CompletedAt, &b.CancelledAt,
			&b.CancelledBy, &b.CancellationReason, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan batch: %w", err)
		}
		_ = json.Unmarshal(filterBytes, &b.FilterCriteria)
		if len(errorBytes) > 0 {
			_ = json.Unmarshal(errorBytes, &b.ErrorSummary)
		}
		batches = append(batches, b)
	}
	if batches == nil {
		batches = []MaintenanceBillingBatch{}
	}
	return batches, nil
}

// Internal Unit struct for calculation
type unitRecord struct {
	ID         uuid.UUID
	UnitNumber string
	UnitType   string
	MemberName string
}

func (r *Repository) resolveUnitsForBatch(
	ctx context.Context,
	societyID uuid.UUID,
	selectionMode string,
	filterCriteria map[string]interface{},
) ([]unitRecord, error) {
	unitsQuery := `
		SELECT u.id, u.unit_number, u.unit_type,
		       COALESCE(NULLIF(TRIM(m.first_name || ' ' || COALESCE(m.last_name, '')), ''), 'Resident') as member_name
		FROM society_units u
		LEFT JOIN unit_relationships ur ON u.id = ur.unit_id AND u.society_id = ur.society_id AND ur.status = 'ACTIVE' AND ur.is_occupant = TRUE
		LEFT JOIN society_members m ON ur.member_id = m.id AND ur.society_id = m.society_id
		WHERE u.society_id = $1 AND u.status = 'ACTIVE'
	`
	var queryArgs []interface{}
	queryArgs = append(queryArgs, societyID)

	if selectionMode == "SINGLE_UNIT" {
		if rawUnitID, ok := filterCriteria["unit_id"].(string); ok {
			if uID, parseErr := uuid.Parse(rawUnitID); parseErr == nil {
				unitsQuery += " AND u.id = $2"
				queryArgs = append(queryArgs, uID)
			}
		}
	} else if selectionMode == "SELECTED_UNITS" {
		if rawList, ok := filterCriteria["unit_ids"].([]interface{}); ok && len(rawList) > 0 {
			var idStrs []string
			for _, item := range rawList {
				if s, ok := item.(string); ok && s != "" {
					if _, err := uuid.Parse(s); err == nil {
						idStrs = append(idStrs, s)
					}
				}
			}
			if len(idStrs) > 0 {
				unitsQuery += " AND u.id::text = ANY($2)"
				queryArgs = append(queryArgs, idStrs)
			}
		}
	}

	unitsQuery += " ORDER BY u.unit_number ASC"

	rows, err := r.pool.Query(ctx, unitsQuery, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query units: %w", err)
	}
	defer rows.Close()

	var units []unitRecord
	for rows.Next() {
		var u unitRecord
		if scanErr := rows.Scan(&u.ID, &u.UnitNumber, &u.UnitType, &u.MemberName); scanErr != nil {
			return nil, fmt.Errorf("scan unit: %w", scanErr)
		}
		units = append(units, u)
	}
	return units, nil
}

func (r *Repository) calculateSingleUnit(
	ctx context.Context,
	societyID uuid.UUID,
	unit unitRecord,
	chargeHeads []MaintenanceChargeHead,
	unitOverrides map[uuid.UUID]MaintenanceUnitBillingOverride,
) (*UnitCalculationResult, error) {
	area := 1000.0 // default reference area (sqft)

	unitRes := &UnitCalculationResult{
		UnitID:     unit.ID,
		UnitNumber: unit.UnitNumber,
		MemberName: unit.MemberName,
		UnitType:   unit.UnitType,
		AreaSqFt:   area,
		Status:     "ELIGIBLE",
		LineItems:  make([]CalculatedLineItem, 0, len(chargeHeads)),
		Warnings:   make([]string, 0),
	}

	arrears := r.getUnitOutstandingBalance(ctx, societyID, unit.ID)
	advance := r.getUnitAdvanceBalance(ctx, societyID, unit.ID)
	unitRes.ArrearsAmount = arrears

	interest := 0.0
	if arrears > 0 {
		interest = math.Round((arrears*0.18*30.0/365.0)*100) / 100
	}
	unitRes.InterestAmount = interest

	baseVars := map[string]formula.Decimal{
		"area":              formula.NewDecimalFromFloat(area),
		"principal_arrears": formula.NewDecimalFromFloat(arrears),
		"interest_rate":     formula.NewDecimalFromFloat(0.18),
		"overdue_days":      formula.NewDecimalFromInt(30),
		"advance_balance":   formula.NewDecimalFromFloat(advance),
	}

	var lineItemsTotal formula.Decimal
	var taxableTotal formula.Decimal
	var cgstTotal formula.Decimal
	var sgstTotal formula.Decimal
	var igstTotal formula.Decimal

	for _, head := range chargeHeads {
		ov, hasOverride := unitOverrides[head.ID]

		if hasOverride && ov.OverrideType == "EXEMPTION" {
			unitRes.LineItems = append(unitRes.LineItems, CalculatedLineItem{
				ChargeHeadID:      head.ID,
				Name:              head.Name,
				SACCode:           head.SACCode,
				TaxClassification: "EXEMPT",
				TaxRate:           0,
				Rate:              0,
				Quantity:          1,
				Amount:            0,
				TaxAmount:         0,
				DisplayOrder:      head.DisplayOrder,
			})
			continue
		}

		var lineAmount formula.Decimal
		var lineRate float64 = 0

		switch head.CalculationMethod {
		case "FIXED":
			if hasOverride && ov.OverrideType == "FIXED_AMOUNT" && ov.OverrideValue != nil {
				lineAmount = formula.NewDecimalFromFloat(*ov.OverrideValue)
				lineRate = *ov.OverrideValue
			} else {
				lineAmount = formula.NewDecimalFromFloat(500.0)
				lineRate = 500.0
			}

		case "AREA_MULTIPLIER", "AREA":
			rate := 3.50
			if hasOverride && ov.OverrideType == "RATE_OVERRIDE" && ov.OverrideValue != nil {
				rate = *ov.OverrideValue
			}
			lineRate = rate
			lineAmount = formula.NewDecimalFromFloat(area).Mul(formula.NewDecimalFromFloat(rate)).Round(2)

		case "FORMULA":
			expr := ""
			if hasOverride && ov.OverrideType == "FORMULA_OVERRIDE" && ov.FormulaOverrideExpression != nil {
				expr = *ov.FormulaOverrideExpression
			} else if head.FormulaExpression != nil {
				expr = *head.FormulaExpression
			}

			if expr != "" {
				evalCtx := formula.EvaluationContext{Variables: baseVars}
				res, evalErr := formula.Evaluate(expr, evalCtx)
				if evalErr != nil {
					return nil, fmt.Errorf("formula evaluation failed for %s: %w", head.Name, evalErr)
				}
				lineAmount = res.Round(2)
			}

		default:
			lineAmount = formula.NewDecimalFromFloat(500.0)
			lineRate = 500.0
		}

		// Tax calculation
		var taxAmt formula.Decimal
		taxClass := head.TaxClassification
		taxRate := head.DefaultTaxRate

		if taxClass == "TAXABLE" && taxRate > 0 {
			taxableTotal = taxableTotal.Add(lineAmount)
			taxDecimal := formula.NewDecimalFromFloat(taxRate / 100.0)
			taxAmt = lineAmount.Mul(taxDecimal).Round(2)

			halfVal, _ := taxAmt.Div(formula.NewDecimalFromInt(2))
			halfRounded := halfVal.Round(2)

			cgstTotal = cgstTotal.Add(halfRounded)
			sgstTotal = sgstTotal.Add(halfRounded)
		}

		lineItemsTotal = lineItemsTotal.Add(lineAmount)

		unitRes.LineItems = append(unitRes.LineItems, CalculatedLineItem{
			ChargeHeadID:      head.ID,
			Name:              head.Name,
			SACCode:           head.SACCode,
			TaxClassification: taxClass,
			TaxRate:           taxRate,
			Rate:              lineRate,
			Quantity:          1,
			Amount:            lineAmount.Float64(),
			TaxAmount:         taxAmt.Float64(),
			FormulaExpression: head.FormulaExpression,
			DisplayOrder:      head.DisplayOrder,
		})
	}

	unitRes.Subtotal = lineItemsTotal.Float64()
	unitRes.TaxableAmount = taxableTotal.Float64()
	unitRes.CGSTAmount = cgstTotal.Float64()
	unitRes.SGSTAmount = sgstTotal.Float64()
	unitRes.IGSTAmount = igstTotal.Float64()
	unitRes.TotalTaxAmount = cgstTotal.Add(sgstTotal).Add(igstTotal).Float64()

	currentObligation := unitRes.Subtotal + unitRes.TotalTaxAmount + unitRes.ArrearsAmount + unitRes.InterestAmount
	advanceDeduction := math.Min(advance, currentObligation)
	advanceDeduction = math.Round(advanceDeduction*100) / 100
	unitRes.AdvanceDeduction = advanceDeduction
	unitRes.PayableAmount = math.Round((currentObligation-advanceDeduction)*100) / 100

	return unitRes, nil
}

// CalculateBatchUnits executes the deterministic calculation pipeline for a batch preview.
func (r *Repository) CalculateBatchUnits(ctx context.Context, societyID, batchID uuid.UUID) (*BatchCalculationSummary, error) {
	batch, err := r.GetBillingBatch(ctx, societyID, batchID)
	if err != nil {
		return nil, err
	}

	tpl, err := r.GetBillTemplate(ctx, societyID, batch.TemplateID)
	if err != nil {
		return nil, err
	}

	chargeHeads, err := r.ListChargeHeads(ctx, societyID, true)
	if err != nil {
		return nil, err
	}

	units, err := r.resolveUnitsForBatch(ctx, societyID, batch.SelectionMode, batch.FilterCriteria)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, ErrBatchEmptyUnits
	}

	allOverrides, err := r.listAllActiveUnitOverrides(ctx, societyID)
	if err != nil {
		return nil, err
	}

	summary := BatchCalculationSummary{
		Units: make([]UnitCalculationResult, 0, len(units)),
	}

	for _, unit := range units {
		unitRes, calcErr := r.calculateSingleUnit(ctx, societyID, unit, chargeHeads, allOverrides[unit.ID])
		if calcErr != nil {
			unitRes = &UnitCalculationResult{
				UnitID:     unit.ID,
				UnitNumber: unit.UnitNumber,
				MemberName: unit.MemberName,
				UnitType:   unit.UnitType,
				Status:     "ERROR",
				Warnings:   []string{calcErr.Error()},
			}
		}

		summary.Units = append(summary.Units, *unitRes)
		summary.EligibleUnitCount++
		summary.TotalSubtotal += unitRes.Subtotal
		summary.TotalTaxable += unitRes.TaxableAmount
		summary.TotalCGST += unitRes.CGSTAmount
		summary.TotalSGST += unitRes.SGSTAmount
		summary.TotalIGST += unitRes.IGSTAmount
		summary.TotalTax += unitRes.TotalTaxAmount
		summary.TotalArrears += unitRes.ArrearsAmount
		summary.TotalInterest += unitRes.InterestAmount
		summary.TotalAdvances += unitRes.AdvanceDeduction
		summary.TotalPayable += unitRes.PayableAmount
	}

	summary.TotalSubtotal = math.Round(summary.TotalSubtotal*100) / 100
	summary.TotalTaxable = math.Round(summary.TotalTaxable*100) / 100
	summary.TotalCGST = math.Round(summary.TotalCGST*100) / 100
	summary.TotalSGST = math.Round(summary.TotalSGST*100) / 100
	summary.TotalIGST = math.Round(summary.TotalIGST*100) / 100
	summary.TotalTax = math.Round(summary.TotalTax*100) / 100
	summary.TotalArrears = math.Round(summary.TotalArrears*100) / 100
	summary.TotalInterest = math.Round(summary.TotalInterest*100) / 100
	summary.TotalAdvances = math.Round(summary.TotalAdvances*100) / 100
	summary.TotalPayable = math.Round(summary.TotalPayable*100) / 100

	summaryJSON, _ := json.Marshal(summary)
	_, _ = r.pool.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET target_unit_count = $1, total_amount = $2, error_summary = $3, updated_at = NOW()
		WHERE id = $4 AND society_id = $5
	`, summary.EligibleUnitCount, summary.TotalPayable, summaryJSON, batchID, societyID)

	_ = tpl
	return &summary, nil
}

func (r *Repository) PreviewBillingBatch(ctx context.Context, societyID, batchID uuid.UUID) (*BatchCalculationSummary, error) {
	return r.CalculateBatchUnits(ctx, societyID, batchID)
}

func (r *Repository) GenerateBillingBatchDrafts(
	ctx context.Context,
	societyID, batchID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Pessimistic locking: SELECT FOR UPDATE
	var currentStatus string
	var periodID, templateID uuid.UUID
	var selectionMode string
	var filterBytes []byte

	err = tx.QueryRow(ctx, `
		SELECT status, billing_period_id, template_id, selection_mode, filter_criteria
		FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, batchID, societyID).Scan(&currentStatus, &periodID, &templateID, &selectionMode, &filterBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("lock billing batch: %w", err)
	}

	if currentStatus == "PROCESSING" {
		return nil, ErrBatchAlreadyProcessing
	}
	if currentStatus == "FINALIZED" {
		return nil, ErrBatchAlreadyFinalized
	}
	if currentStatus == "CANCELLED" {
		return nil, ErrBatchCancelled
	}
	if !IsValidBatchTransition(currentStatus, "PROCESSING") {
		return nil, fmt.Errorf("%w: cannot transition from %s to PROCESSING", ErrInvalidBatchTransition, currentStatus)
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = 'PROCESSING', started_at = COALESCE(started_at, $1), updated_at = $1
		WHERE id = $2 AND society_id = $3
	`, now, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("set batch processing: %w", err)
	}

	period, err := r.GetBillingPeriod(ctx, societyID, periodID)
	if err != nil {
		return nil, err
	}
	if period.Status == "CLOSED" {
		return nil, errors.New("cannot generate bills for closed billing period")
	}

	tpl, err := r.GetBillTemplate(ctx, societyID, templateID)
	if err != nil {
		return nil, err
	}

	chargeHeads, err := r.ListChargeHeads(ctx, societyID, true)
	if err != nil {
		return nil, err
	}
	allOverrides, err := r.listAllActiveUnitOverrides(ctx, societyID)
	if err != nil {
		return nil, err
	}

	var filterCriteria map[string]interface{}
	_ = json.Unmarshal(filterBytes, &filterCriteria)
	units, err := r.resolveUnitsForBatch(ctx, societyID, selectionMode, filterCriteria)
	if err != nil {
		return nil, err
	}
	if len(units) == 0 {
		return nil, ErrBatchEmptyUnits
	}

	var societyCode string
	_ = tx.QueryRow(ctx, `SELECT society_code FROM societies WHERE id = $1`, societyID).Scan(&societyCode)
	if societyCode == "" {
		societyCode = "CHS"
	}

	successfulUnits := 0
	failedUnits := 0
	var totalBatchPayable formula.Decimal

	for _, unit := range units {
		// 1. Check if unit already has an active, non-void bill for this period and bill type
		var existingBillID uuid.UUID
		var existingTotal float64
		err = tx.QueryRow(ctx, `
			SELECT id, total_amount FROM maintenance_bills
			WHERE society_id = $1 AND billing_period_id = $2 AND unit_id = $3 AND bill_type = $4 AND status != 'VOID'
		`, societyID, periodID, unit.ID, tpl.BillType).Scan(&existingBillID, &existingTotal)
		if err == nil {
			// Existing bill preserved without recalculating
			_, _ = tx.Exec(ctx, `
				INSERT INTO maintenance_batch_units (
					society_id, batch_id, unit_id, bill_id, status, error_message, error_code, last_attempted_at, updated_at
				) VALUES ($1, $2, $3, $4, 'GENERATED', NULL, NULL, NOW(), NOW())
				ON CONFLICT (batch_id, unit_id) DO UPDATE
				SET bill_id = EXCLUDED.bill_id, status = 'GENERATED', error_message = NULL, error_code = NULL,
				    last_attempted_at = NOW(), updated_at = NOW()
			`, societyID, batchID, unit.ID, existingBillID)

			successfulUnits++
			totalBatchPayable = totalBatchPayable.Add(formula.NewDecimalFromFloat(existingTotal))
			continue
		}

		// 2. Calculate unit
		unitRes, calcErr := r.calculateSingleUnit(ctx, societyID, unit, chargeHeads, allOverrides[unit.ID])
		if calcErr != nil {
			errMsg := calcErr.Error()
			errCode := "CALCULATION_ERROR"
			_, _ = tx.Exec(ctx, `
				INSERT INTO maintenance_batch_units (
					society_id, batch_id, unit_id, status, error_message, error_code, retry_count, last_attempted_at
				) VALUES ($1, $2, $3, 'FAILED', $4, $5, 1, NOW())
				ON CONFLICT (batch_id, unit_id) DO UPDATE
				SET status = 'FAILED', error_message = EXCLUDED.error_message, error_code = EXCLUDED.error_code,
				    retry_count = maintenance_batch_units.retry_count + 1, last_attempted_at = NOW(), updated_at = NOW()
			`, societyID, batchID, unit.ID, errMsg, errCode)

			failedUnits++
			continue
		}

		// 3. Generate sequential bill number
		var billSeq int64
		_ = tx.QueryRow(ctx, `SELECT nextval('maintenance_bill_seq')`).Scan(&billSeq)
		billNumber := fmt.Sprintf("%s-%s-%s-%04d", societyCode, period.PeriodStart.Format("200601"), unit.UnitNumber, billSeq%10000)

		var billID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO maintenance_bills (
				society_id, unit_id, billing_period_id, bill_number, bill_type, template_id,
				template_version, batch_id, issue_date, due_date, currency, subtotal,
				taxable_amount, cgst_amount, sgst_amount, igst_amount, total_tax_amount,
				arrears_amount, interest_amount, advance_adjusted, total_amount, status, created_by
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'INR', $11,
				$12, $13, $14, $15, $16, $17, $18, $19, $20, 'DRAFT', $21
			) RETURNING id
		`, societyID, unit.ID, periodID, billNumber, tpl.BillType, tpl.ID,
			tpl.CurrentVersion, batchID, period.PeriodStart, period.DueDate,
			unitRes.Subtotal, unitRes.TaxableAmount, unitRes.CGSTAmount, unitRes.SGSTAmount,
			unitRes.IGSTAmount, unitRes.TotalTaxAmount, unitRes.ArrearsAmount, unitRes.InterestAmount,
			unitRes.AdvanceDeduction, unitRes.PayableAmount, actorID,
		).Scan(&billID)
		if err != nil {
			errMsg := fmt.Sprintf("insert draft bill: %v", err)
			errCode := "DB_ERROR"
			_, _ = tx.Exec(ctx, `
				INSERT INTO maintenance_batch_units (
					society_id, batch_id, unit_id, status, error_message, error_code, retry_count, last_attempted_at
				) VALUES ($1, $2, $3, 'FAILED', $4, $5, 1, NOW())
				ON CONFLICT (batch_id, unit_id) DO UPDATE
				SET status = 'FAILED', error_message = EXCLUDED.error_message, error_code = EXCLUDED.error_code,
				    retry_count = maintenance_batch_units.retry_count + 1, last_attempted_at = NOW(), updated_at = NOW()
			`, societyID, batchID, unit.ID, errMsg, errCode)

			failedUnits++
			continue
		}

		// Insert immutable line items
		for _, li := range unitRes.LineItems {
			_, err = tx.Exec(ctx, `
				INSERT INTO maintenance_bill_items (
					bill_id, society_id, charge_head_id, category_name, description,
					sac_code, tax_classification, tax_rate, tax_amount, rate, quantity,
					amount, formula_expression, display_order
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			`, billID, societyID, li.ChargeHeadID, li.Name, li.Name,
				li.SACCode, li.TaxClassification, li.TaxRate, li.TaxAmount, li.Rate, li.Quantity,
				li.Amount, li.FormulaExpression, li.DisplayOrder)
			if err != nil {
				return nil, fmt.Errorf("insert line item for unit %s: %w", unit.UnitNumber, err)
			}
		}

		snapshotBytes, _ := json.Marshal(unitRes)
		_, _ = tx.Exec(ctx, `
			INSERT INTO maintenance_batch_units (
				society_id, batch_id, unit_id, bill_id, status, calculation_snapshot, error_message, error_code, last_attempted_at
			) VALUES ($1, $2, $3, $4, 'GENERATED', $5::jsonb, NULL, NULL, NOW())
			ON CONFLICT (batch_id, unit_id) DO UPDATE
			SET bill_id = EXCLUDED.bill_id, status = 'GENERATED', calculation_snapshot = EXCLUDED.calculation_snapshot,
			    error_message = NULL, error_code = NULL, last_attempted_at = NOW(), updated_at = NOW()
		`, societyID, batchID, unit.ID, billID, string(snapshotBytes))

		successfulUnits++
		totalBatchPayable = totalBatchPayable.Add(formula.NewDecimalFromFloat(unitRes.PayableAmount))
	}

	targetCount := len(units)
	processedCount := successfulUnits + failedUnits

	var finalStatus string
	if failedUnits == 0 {
		finalStatus = "GENERATED"
	} else if successfulUnits > 0 {
		finalStatus = "COMPLETED_WITH_ERRORS"
	} else {
		finalStatus = "FAILED"
	}

	completedAt := time.Now()
	totalAmountVal := totalBatchPayable.Round(2).Float64()

	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = $1,
		    target_unit_count = $2,
		    processed_unit_count = $3,
		    successful_unit_count = $4,
		    failed_unit_count = $5,
		    total_amount = $6,
		    completed_at = $7,
		    updated_at = $7
		WHERE id = $8 AND society_id = $9
	`, finalStatus, targetCount, processedCount, successfulUnits, failedUnits, totalAmountVal, completedAt, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("update batch final status: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"batch_id":         batchID,
		"status":           finalStatus,
		"target_units":     targetCount,
		"successful_units": successfulUnits,
		"failed_units":     failedUnits,
		"total_amount":     totalAmountVal,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_BATCH_GENERATED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return r.GetBillingBatch(ctx, societyID, batchID)
}

func (r *Repository) RetryFailedBatchUnits(
	ctx context.Context,
	societyID, batchID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Pessimistic lock FOR UPDATE
	var currentStatus string
	var periodID, templateID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT status, billing_period_id, template_id
		FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, batchID, societyID).Scan(&currentStatus, &periodID, &templateID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("lock batch for retry: %w", err)
	}

	if currentStatus == "PROCESSING" {
		return nil, ErrBatchAlreadyProcessing
	}
	if currentStatus == "FINALIZED" {
		return nil, ErrBatchAlreadyFinalized
	}
	if currentStatus == "CANCELLED" {
		return nil, ErrBatchCancelled
	}
	if currentStatus != "COMPLETED_WITH_ERRORS" && currentStatus != "FAILED" && currentStatus != "GENERATED" {
		return nil, fmt.Errorf("%w: cannot retry batch in %s status", ErrInvalidBatchTransition, currentStatus)
	}

	// 2. Fetch only failed units
	rows, err := tx.Query(ctx, `
		SELECT bu.unit_id, u.unit_number, u.unit_type,
		       COALESCE(NULLIF(TRIM(m.first_name || ' ' || COALESCE(m.last_name, '')), ''), 'Resident') as member_name
		FROM maintenance_batch_units bu
		JOIN society_units u ON bu.unit_id = u.id AND bu.society_id = u.society_id
		LEFT JOIN unit_relationships ur ON u.id = ur.unit_id AND u.society_id = ur.society_id AND ur.status = 'ACTIVE' AND ur.is_occupant = TRUE
		LEFT JOIN society_members m ON ur.member_id = m.id AND ur.society_id = m.society_id
		WHERE bu.batch_id = $1 AND bu.society_id = $2 AND bu.status = 'FAILED'
		ORDER BY u.unit_number ASC
	`, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("query failed units: %w", err)
	}

	var failedUnitsList []unitRecord
	for rows.Next() {
		var u unitRecord
		if scanErr := rows.Scan(&u.ID, &u.UnitNumber, &u.UnitType, &u.MemberName); scanErr != nil {
			rows.Close()
			return nil, fmt.Errorf("scan failed unit: %w", scanErr)
		}
		failedUnitsList = append(failedUnitsList, u)
	}
	rows.Close()

	if len(failedUnitsList) == 0 {
		return nil, ErrNoFailedUnitsToRetry
	}

	// 3. Mark batch as PROCESSING
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = 'PROCESSING', updated_at = NOW()
		WHERE id = $1 AND society_id = $2
	`, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("set batch processing: %w", err)
	}

	period, err := r.GetBillingPeriod(ctx, societyID, periodID)
	if err != nil {
		return nil, err
	}
	tpl, err := r.GetBillTemplate(ctx, societyID, templateID)
	if err != nil {
		return nil, err
	}
	chargeHeads, err := r.ListChargeHeads(ctx, societyID, true)
	if err != nil {
		return nil, err
	}
	allOverrides, err := r.listAllActiveUnitOverrides(ctx, societyID)
	if err != nil {
		return nil, err
	}

	var societyCode string
	_ = tx.QueryRow(ctx, `SELECT society_code FROM societies WHERE id = $1`, societyID).Scan(&societyCode)
	if societyCode == "" {
		societyCode = "CHS"
	}

	for _, unit := range failedUnitsList {
		// Check if active bill already exists
		var existingBillID uuid.UUID
		err = tx.QueryRow(ctx, `
			SELECT id FROM maintenance_bills
			WHERE society_id = $1 AND billing_period_id = $2 AND unit_id = $3 AND bill_type = $4 AND status != 'VOID'
		`, societyID, periodID, unit.ID, tpl.BillType).Scan(&existingBillID)
		if err == nil {
			_, _ = tx.Exec(ctx, `
				UPDATE maintenance_batch_units
				SET bill_id = $1, status = 'GENERATED', error_message = NULL, error_code = NULL,
				    retry_count = retry_count + 1, last_attempted_at = NOW(), updated_at = NOW()
				WHERE batch_id = $2 AND unit_id = $3 AND society_id = $4
			`, existingBillID, batchID, unit.ID, societyID)
			continue
		}

		unitRes, calcErr := r.calculateSingleUnit(ctx, societyID, unit, chargeHeads, allOverrides[unit.ID])
		if calcErr != nil {
			errMsg := calcErr.Error()
			_, _ = tx.Exec(ctx, `
				UPDATE maintenance_batch_units
				SET status = 'FAILED', error_message = $1, retry_count = retry_count + 1,
				    last_attempted_at = NOW(), updated_at = NOW()
				WHERE batch_id = $2 AND unit_id = $3 AND society_id = $4
			`, errMsg, batchID, unit.ID, societyID)
			continue
		}

		var billSeq int64
		_ = tx.QueryRow(ctx, `SELECT nextval('maintenance_bill_seq')`).Scan(&billSeq)
		billNumber := fmt.Sprintf("%s-%s-%s-%04d", societyCode, period.PeriodStart.Format("200601"), unit.UnitNumber, billSeq%10000)

		var billID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO maintenance_bills (
				society_id, unit_id, billing_period_id, bill_number, bill_type, template_id,
				template_version, batch_id, issue_date, due_date, currency, subtotal,
				taxable_amount, cgst_amount, sgst_amount, igst_amount, total_tax_amount,
				arrears_amount, interest_amount, advance_adjusted, total_amount, status, created_by
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'INR', $11,
				$12, $13, $14, $15, $16, $17, $18, $19, $20, 'DRAFT', $21
			) RETURNING id
		`, societyID, unit.ID, periodID, billNumber, tpl.BillType, tpl.ID,
			tpl.CurrentVersion, batchID, period.PeriodStart, period.DueDate,
			unitRes.Subtotal, unitRes.TaxableAmount, unitRes.CGSTAmount, unitRes.SGSTAmount,
			unitRes.IGSTAmount, unitRes.TotalTaxAmount, unitRes.ArrearsAmount, unitRes.InterestAmount,
			unitRes.AdvanceDeduction, unitRes.PayableAmount, actorID,
		).Scan(&billID)
		if err != nil {
			errMsg := fmt.Sprintf("insert bill on retry: %v", err)
			_, _ = tx.Exec(ctx, `
				UPDATE maintenance_batch_units
				SET status = 'FAILED', error_message = $1, error_code = 'DB_ERROR',
				    retry_count = retry_count + 1, last_attempted_at = NOW(), updated_at = NOW()
				WHERE batch_id = $2 AND unit_id = $3 AND society_id = $4
			`, errMsg, batchID, unit.ID, societyID)
			continue
		}

		for _, li := range unitRes.LineItems {
			_, _ = tx.Exec(ctx, `
				INSERT INTO maintenance_bill_items (
					bill_id, society_id, charge_head_id, category_name, description,
					sac_code, tax_classification, tax_rate, tax_amount, rate, quantity,
					amount, formula_expression, display_order
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			`, billID, societyID, li.ChargeHeadID, li.Name, li.Name,
				li.SACCode, li.TaxClassification, li.TaxRate, li.TaxAmount, li.Rate, li.Quantity,
				li.Amount, li.FormulaExpression, li.DisplayOrder)
		}

		snapshotBytes, _ := json.Marshal(unitRes)
		_, _ = tx.Exec(ctx, `
			UPDATE maintenance_batch_units
			SET bill_id = $1, status = 'GENERATED', calculation_snapshot = $2::jsonb,
			    error_message = NULL, error_code = NULL, retry_count = retry_count + 1,
			    last_attempted_at = NOW(), updated_at = NOW()
			WHERE batch_id = $3 AND unit_id = $4 AND society_id = $5
		`, billID, string(snapshotBytes), batchID, unit.ID, societyID)
	}

	// 4. Re-calculate batch counters
	var succCount, failCount, totalUnits int
	_ = tx.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'GENERATED'),
			COUNT(*) FILTER (WHERE status = 'FAILED'),
			COUNT(*)
		FROM maintenance_batch_units
		WHERE batch_id = $1 AND society_id = $2
	`, batchID, societyID).Scan(&succCount, &failCount, &totalUnits)

	var updatedTotalPayable float64
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0.00)
		FROM maintenance_bills
		WHERE batch_id = $1 AND society_id = $2 AND status != 'VOID'
	`, batchID, societyID).Scan(&updatedTotalPayable)

	var finalStatus string
	if failCount == 0 {
		finalStatus = "GENERATED"
	} else if succCount > 0 {
		finalStatus = "COMPLETED_WITH_ERRORS"
	} else {
		finalStatus = "FAILED"
	}

	completedAt := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = $1,
		    processed_unit_count = $2,
		    successful_unit_count = $3,
		    failed_unit_count = $4,
		    total_amount = $5,
		    completed_at = $6,
		    updated_at = $6
		WHERE id = $7 AND society_id = $8
	`, finalStatus, succCount+failCount, succCount, failCount, updatedTotalPayable, completedAt, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("update batch post-retry: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"batch_id":         batchID,
		"retried_units":    len(failedUnitsList),
		"successful_units": succCount,
		"failed_units":     failCount,
		"status":           finalStatus,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_BATCH_RETRIED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit retry tx: %w", err)
	}

	return r.GetBillingBatch(ctx, societyID, batchID)
}

func (r *Repository) GetBatchProgress(ctx context.Context, societyID, batchID uuid.UUID) (*BatchProgressInfo, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var p BatchProgressInfo
	err := r.pool.QueryRow(ctx, `
		SELECT id, status, target_unit_count, processed_unit_count, successful_unit_count,
		       failed_unit_count, started_at, completed_at, finalized_at
		FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
	`, batchID, societyID).Scan(
		&p.BatchID, &p.Status, &p.TargetUnitCount, &p.ProcessedUnitCount,
		&p.SuccessfulUnitCount, &p.FailedUnitCount,
		&p.StartedAt, &p.CompletedAt, &p.FinalizedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("get batch progress: %w", err)
	}

	pending := p.TargetUnitCount - p.ProcessedUnitCount
	if pending < 0 {
		pending = 0
	}
	p.PendingUnitCount = pending

	return &p, nil
}

func (r *Repository) ListBatchUnits(ctx context.Context, societyID, batchID uuid.UUID, statusFilter string, limit, offset int) ([]BatchUnit, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT bu.id, bu.batch_id, bu.society_id, bu.unit_id, u.unit_number,
		       bu.bill_id, bu.status, bu.calculation_snapshot, bu.error_message,
		       bu.error_code, bu.retry_count, bu.last_attempted_at, bu.created_at, bu.updated_at
		FROM maintenance_batch_units bu
		JOIN society_units u ON bu.unit_id = u.id AND bu.society_id = u.society_id
		WHERE bu.batch_id = $1 AND bu.society_id = $2
	`
	args := []interface{}{batchID, societyID}
	if statusFilter != "" {
		query += " AND bu.status = $3"
		args = append(args, statusFilter)
	}
	query += fmt.Sprintf(" ORDER BY u.unit_number ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list batch units: %w", err)
	}
	defer rows.Close()

	var units []BatchUnit
	for rows.Next() {
		var bu BatchUnit
		var snapshotBytes []byte
		if scanErr := rows.Scan(
			&bu.ID, &bu.BatchID, &bu.SocietyID, &bu.UnitID, &bu.UnitNumber,
			&bu.BillID, &bu.Status, &snapshotBytes, &bu.ErrorMessage,
			&bu.ErrorCode, &bu.RetryCount, &bu.LastAttemptedAt, &bu.CreatedAt, &bu.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan batch unit: %w", scanErr)
		}
		if len(snapshotBytes) > 0 {
			_ = json.Unmarshal(snapshotBytes, &bu.CalculationSnapshot)
		}
		units = append(units, bu)
	}
	if units == nil {
		units = []BatchUnit{}
	}
	return units, nil
}

func (r *Repository) FinalizeBillingBatch(
	ctx context.Context,
	societyID, batchID uuid.UUID,
	actorID uuid.UUID,
	clientIP, userAgent string,
) (*MaintenanceBillingBatch, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Pessimistic lock
	var currentStatus string
	var totalAmount float64
	err = tx.QueryRow(ctx, `
		SELECT status, total_amount
		FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, batchID, societyID).Scan(&currentStatus, &totalAmount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillingBatchNotFound
		}
		return nil, fmt.Errorf("lock batch for finalization: %w", err)
	}

	if currentStatus == "FINALIZED" {
		return nil, ErrBatchAlreadyFinalized
	}
	if currentStatus == "PROCESSING" {
		return nil, ErrBatchAlreadyProcessing
	}
	if currentStatus == "CANCELLED" {
		return nil, ErrBatchCancelled
	}
	if currentStatus != "GENERATED" && currentStatus != "COMPLETED_WITH_ERRORS" {
		return nil, fmt.Errorf("%w: batch must be in GENERATED or COMPLETED_WITH_ERRORS status before finalization; current: %s", ErrInvalidBatchTransition, currentStatus)
	}

	// Transition all bills in this batch from DRAFT to ISSUED
	res, err := tx.Exec(ctx, `
		UPDATE maintenance_bills
		SET status = 'ISSUED', updated_at = NOW()
		WHERE batch_id = $1 AND society_id = $2 AND status = 'DRAFT'
	`, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("finalize bills: %w", err)
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = 'FINALIZED', finalized_by = $1, finalized_at = $2, updated_at = $2
		WHERE id = $3 AND society_id = $4
	`, actorID, now, batchID, societyID)
	if err != nil {
		return nil, fmt.Errorf("update batch finalized status: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"batch_id":     batchID,
		"issued_bills": res.RowsAffected(),
		"total_amount": totalAmount,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_BATCH_FINALIZED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit finalize tx: %w", err)
	}

	return r.GetBillingBatch(ctx, societyID, batchID)
}

func (r *Repository) CancelBillingBatch(
	ctx context.Context,
	societyID, batchID uuid.UUID,
	reason string,
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

	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, batchID, societyID).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBillingBatchNotFound
		}
		return fmt.Errorf("lock batch for cancel: %w", err)
	}

	if currentStatus == "FINALIZED" {
		return errors.New("cannot cancel a finalized billing batch; bills must be voided individually")
	}
	if currentStatus == "CANCELLED" {
		return nil // idempotent cancel
	}
	if currentStatus == "PROCESSING" {
		return ErrBatchAlreadyProcessing
	}

	// Delete unissued draft bills
	_, _ = tx.Exec(ctx, `
		DELETE FROM maintenance_bills
		WHERE batch_id = $1 AND society_id = $2 AND status = 'DRAFT'
	`, batchID, societyID)

	now := time.Now()
	_, err = tx.Exec(ctx, `
		UPDATE maintenance_billing_batches
		SET status = 'CANCELLED', cancelled_at = $1, cancelled_by = $2, cancellation_reason = $3, updated_at = $1
		WHERE id = $4 AND society_id = $5
	`, now, actorID, reason, batchID, societyID)
	if err != nil {
		return fmt.Errorf("cancel batch: %w", err)
	}

	auditDetails, _ := json.Marshal(map[string]interface{}{
		"batch_id": batchID,
		"reason":   reason,
	})
	_, _ = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, 'BILLING_BATCH_CANCELLED', NULL, $2, $3, $4)
	`, actorID, string(auditDetails), clientIP, userAgent)

	return tx.Commit(ctx)
}

// ------------------------------------------------------------------------------
// Internal Helpers
// ------------------------------------------------------------------------------

func (r *Repository) listAllActiveUnitOverrides(ctx context.Context, societyID uuid.UUID) (map[uuid.UUID]map[uuid.UUID]MaintenanceUnitBillingOverride, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, society_id, unit_id, charge_head_id, override_type, override_value,
		       formula_override_expression, effective_from, effective_until, reason, status
		FROM maintenance_unit_billing_overrides
		WHERE society_id = $1 AND status = 'ACTIVE'
	`, societyID)
	if err != nil {
		return nil, fmt.Errorf("query overrides: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]map[uuid.UUID]MaintenanceUnitBillingOverride)
	for rows.Next() {
		var o MaintenanceUnitBillingOverride
		if err := rows.Scan(
			&o.ID, &o.SocietyID, &o.UnitID, &o.ChargeHeadID, &o.OverrideType,
			&o.OverrideValue, &o.FormulaOverrideExpression, &o.EffectiveFrom,
			&o.EffectiveUntil, &o.Reason, &o.Status,
		); err == nil {
			if _, exists := result[o.UnitID]; !exists {
				result[o.UnitID] = make(map[uuid.UUID]MaintenanceUnitBillingOverride)
			}
			result[o.UnitID][o.ChargeHeadID] = o
		}
	}
	return result, nil
}

func (r *Repository) getUnitOutstandingBalance(ctx context.Context, societyID, unitID uuid.UUID) float64 {
	var totalIssued float64
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM maintenance_bills
		WHERE society_id = $1 AND unit_id = $2 AND status = 'ISSUED'
	`, societyID, unitID).Scan(&totalIssued)

	var totalAllocated float64
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(a.allocated_amount), 0)
		FROM maintenance_payment_allocations a
		JOIN maintenance_bills b ON a.bill_id = b.id AND a.society_id = b.society_id
		WHERE a.society_id = $1 AND b.unit_id = $2 AND a.status = 'ACTIVE'
	`, societyID, unitID).Scan(&totalAllocated)

	balance := totalIssued - totalAllocated
	if balance < 0 {
		return 0
	}
	return math.Round(balance*100) / 100
}

func (r *Repository) getUnitAdvanceBalance(ctx context.Context, societyID, unitID uuid.UUID) float64 {
	// Unallocated payments made by unit occupants
	var totalPaid float64
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.amount), 0)
		FROM maintenance_payments p
		WHERE p.society_id = $1 AND p.status = 'RECORDED'
	`, societyID).Scan(&totalPaid)

	var totalAllocated float64
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(a.allocated_amount), 0)
		FROM maintenance_payment_allocations a
		WHERE a.society_id = $1 AND a.status = 'ACTIVE'
	`, societyID).Scan(&totalAllocated)

	unallocated := totalPaid - totalAllocated
	if unallocated < 0 {
		return 0
	}
	return math.Round(unallocated*100) / 100
}
