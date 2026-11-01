package billingpdf

import (
	"time"

	"github.com/google/uuid"
)

// BillDocument represents the complete, pre-calculated data model required
// to render a print-ready, professional A4 maintenance bill PDF.
type BillDocument struct {
	// Society Information
	SocietyID          uuid.UUID `json:"society_id"`
	SocietyName        string    `json:"society_name"`
	RegistrationNumber string    `json:"registration_number"`
	AddressLine1       string    `json:"address_line1"`
	Locality           string    `json:"locality"`
	City               string    `json:"city"`
	State              string    `json:"state"`
	Pincode            string    `json:"pincode"`
	SocietyGSTIN       string    `json:"society_gstin,omitempty"`
	SocietyPAN         string    `json:"society_pan,omitempty"`
	SocietyEmail       string    `json:"society_email,omitempty"`
	SocietyPhone       string    `json:"society_phone,omitempty"`

	// Recipient / Unit Information
	UnitID       uuid.UUID `json:"unit_id"`
	UnitNumber   string    `json:"unit_number"`
	BuildingName string    `json:"building_name,omitempty"`
	WingName     string    `json:"wing_name,omitempty"`
	FloorNumber  int       `json:"floor_number,omitempty"`
	UnitType     string    `json:"unit_type"`
	AreaSqFt     float64   `json:"area_sqft"`
	MemberName   string    `json:"member_name"`
	MemberPhone  string    `json:"member_phone,omitempty"`
	MemberEmail  string    `json:"member_email,omitempty"`

	// Invoice Metadata
	BillID          uuid.UUID `json:"bill_id"`
	BillNumber      string    `json:"bill_number"`
	BillType        string    `json:"bill_type"` // REGULAR, SUPPLEMENTARY, REVISED
	BillTitle       string    `json:"bill_title"`
	BillDate        time.Time `json:"bill_date"`
	DueDate         time.Time `json:"due_date"`
	BillingPeriodID uuid.UUID `json:"billing_period_id"`
	PeriodName      string    `json:"period_name"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	Currency        string    `json:"currency"`
	TemplateVersion int       `json:"template_version"`

	// Charge Line Items (sorted by DisplayOrder)
	LineItems []DocumentLineItem `json:"line_items"`

	// Financial Totals (from immutable bill snapshot)
	ChargeSubtotal     float64 `json:"charge_subtotal"`
	TaxableAmount      float64 `json:"taxable_amount"`
	CGSTAmount         float64 `json:"cgst_amount"`
	SGSTAmount         float64 `json:"sgst_amount"`
	IGSTAmount         float64 `json:"igst_amount"`
	TotalTaxAmount     float64 `json:"total_tax_amount"`
	ArrearsAmount      float64 `json:"arrears_amount"`
	InterestAmount     float64 `json:"interest_amount"`
	AdvanceDeduction   float64 `json:"advance_deduction"`
	RoundingAdjustment float64 `json:"rounding_adjustment"`
	TotalPayable       float64 `json:"total_payable"`
	AmountInWords      string  `json:"amount_in_words"`

	// Payment & Banking Instructions
	BankDetails         map[string]interface{} `json:"bank_details"`
	PaymentInstructions string                 `json:"payment_instructions,omitempty"`
	HeaderNotes         string                 `json:"header_notes,omitempty"`
	FooterText          string                 `json:"footer_text,omitempty"`

	// Watermark Configuration
	Watermark WatermarkConfig `json:"watermark"`

	// Configured Template Fields (controls field visibility and custom labels)
	FieldOverrides map[string]TemplateFieldConfig `json:"field_overrides,omitempty"`
}

// DocumentLineItem represents an individual line item on the bill.
type DocumentLineItem struct {
	Index             int     `json:"index"`
	ChargeHeadName    string  `json:"charge_head_name"`
	Description       string  `json:"description,omitempty"`
	SACCode           string  `json:"sac_code,omitempty"`
	TaxClassification string  `json:"tax_classification"`
	TaxRate           float64 `json:"tax_rate"`
	Rate              float64 `json:"rate"`
	Quantity          float64 `json:"quantity"`
	Amount            float64 `json:"amount"`
	TaxAmount         float64 `json:"tax_amount"`
	FormulaExpression string  `json:"formula_expression,omitempty"`
	DisplayOrder      int     `json:"display_order"`
}

// WatermarkConfig controls the background logo watermark rendering.
type WatermarkConfig struct {
	Enabled  bool    `json:"enabled"`
	Opacity  float64 `json:"opacity"` // e.g. 0.08 (subtle)
	Scale    float64 `json:"scale"`   // e.g. 0.40 of page width
	Position string  `json:"position"` // CENTER, TOP_RIGHT
}

// TemplateFieldConfig controls visibility and custom labels for configurable bill fields.
type TemplateFieldConfig struct {
	IsVisible    bool   `json:"is_visible"`
	DisplayLabel string `json:"display_label"`
}
