package society

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Lifecycle states
const (
	StatusPending   = "PENDING"
	StatusActive    = "ACTIVE"
	StatusSuspended = "SUSPENDED"
)

// Society roles
const (
	RoleSocietyAdmin = "SOC_ADMIN"
	RoleGatekeeper   = "GATEKEEPER"
)

// Assignment statuses
const (
	AssignmentStatusActive  = "ACTIVE"
	AssignmentStatusRevoked = "REVOKED"
)

// Audit actions
const (
	ActionSocietyCreated          = "SOCIETY_CREATED"
	ActionSocietyProfileUpdated   = "SOCIETY_PROFILE_UPDATED"
	ActionSocietyActivated        = "SOCIETY_ACTIVATED"
	ActionSocietySuspended        = "SOCIETY_SUSPENDED"
	ActionSocietyReactivated      = "SOCIETY_REACTIVATED"
	ActionSocietyAdminAssigned    = "SOCIETY_ADMIN_ASSIGNED"
	ActionSocietyAdminRevoked     = "SOCIETY_ADMIN_REVOKED"
	ActionGatekeeperAssigned      = "GATEKEEPER_ASSIGNED"
	ActionGatekeeperRevoked       = "GATEKEEPER_REVOKED"
	ActionGateDutyStarted         = "GATE_DUTY_STARTED"
	ActionGateDutyEnded           = "GATE_DUTY_ENDED"
	ActionShiftCreated            = "SHIFT_CREATED"
	ActionAnnouncementCreated     = "ANNOUNCEMENT_CREATED"
	ActionAnnouncementUpdated     = "ANNOUNCEMENT_UPDATED"
	ActionAnnouncementPublished   = "ANNOUNCEMENT_PUBLISHED"
	ActionAnnouncementArchived    = "ANNOUNCEMENT_ARCHIVED"
	ActionChargeCategoryCreated   = "CHARGE_CATEGORY_CREATED"
	ActionChargeCategoryUpdated   = "CHARGE_CATEGORY_UPDATED"
	ActionChargeConfigCreated     = "CHARGE_CONFIG_CREATED"
	ActionChargeConfigDeactivated = "CHARGE_CONFIG_DEACTIVATED"
	ActionBillingPeriodCreated    = "BILLING_PERIOD_CREATED"
	ActionBillsGenerated          = "BILLS_GENERATED"
	ActionBillVoided              = "BILL_VOIDED"
	ActionPaymentRecorded         = "MAINTENANCE_PAYMENT_RECORDED"
	ActionPaymentReversed         = "MAINTENANCE_PAYMENT_REVERSED"
	ActionReceiptIssued           = "MAINTENANCE_RECEIPT_ISSUED"
)

// Maintenance charge types
const (
	ChargeTypeFlatRecurring     = "FLAT_RECURRING"
	ChargeTypeUnitTypeRecurring = "UNIT_TYPE_RECURRING"
	ChargeTypeOneOff            = "ONE_OFF"
)

// Maintenance billing frequencies
const (
	FrequencyMonthly   = "MONTHLY"
	FrequencyQuarterly = "QUARTERLY"
	FrequencyAnnually  = "ANNUALLY"
	FrequencyOneOff    = "ONE_OFF"
)

// Maintenance bill statuses
const (
	BillStatusDraft  = "DRAFT"
	BillStatusIssued = "ISSUED"
	BillStatusVoid   = "VOID"
)

// Derived bill display statuses
const (
	DerivedStatusIssued  = "ISSUED"
	DerivedStatusOverdue = "OVERDUE"
	DerivedStatusVoid    = "VOID"
)

// Announcement statuses
const (
	AnnouncementStatusDraft     = "DRAFT"
	AnnouncementStatusPublished = "PUBLISHED"
	AnnouncementStatusArchived  = "ARCHIVED"
)

// Announcement audience types
const (
	AudienceTypeSociety  = "SOCIETY"
	AudienceTypeBuilding = "BUILDING"
	AudienceTypeWing     = "WING"
	AudienceTypeUnit     = "UNIT"
)

// Announcement categories
const (
	CategoryGeneral        = "GENERAL"
	CategoryMaintenance    = "MAINTENANCE"
	CategorySecurity       = "SECURITY"
	CategoryEvent          = "EVENT"
	CategoryEmergency      = "EMERGENCY"
	CategoryAdministrative = "ADMINISTRATIVE"
)

// Announcement priorities
const (
	PriorityNormal   = "NORMAL"
	PriorityUrgent   = "URGENT"
	PriorityCritical = "CRITICAL"
)

// Suspension origins
const (
	OriginManual    = "MANUAL"
	OriginAutomated = "AUTOMATED"
)

// Domain errors
var (
	ErrSocietyNotFound                = errors.New("society not found")
	ErrInvalidStatusTransition        = errors.New("invalid status transition")
	ErrAlreadyInStatus                = errors.New("society already in requested status")
	ErrReasonRequired                 = errors.New("reason is required for this action")
	ErrSocietyCodeExists              = errors.New("society code already exists")
	ErrRegNumberExists                = errors.New("registration number already exists in this state")
	ErrMaxRetriesExceeded             = errors.New("exceeded maximum code collision retries")
	ErrInvalidPincode                 = errors.New("pincode must be exactly 6 Indian postal digits")
	ErrInvalidName                    = errors.New("society name must be between 3 and 255 characters")
	ErrRequiredFieldMissing           = errors.New("required address fields cannot be blank")
	ErrInvalidSuspensionReason        = errors.New("invalid suspension reason code")
	ErrSuspensionReasonRequired       = errors.New("suspension reason code is required")
	ErrSuspensionDetailsRequired      = errors.New("additional details are required for this suspension reason")
	ErrSuspensionReasonInactive       = errors.New("selected suspension reason is inactive")
	ErrManualSuspensionNotAllowed     = errors.New("selected suspension reason is not permitted for manual administrative use")
	ErrDetailsTooLong                 = errors.New("additional details must not exceed 1000 characters")
	ErrUserNotFound                   = errors.New("user not found")
	ErrSocietyAdminAlreadyAssigned    = errors.New("society administrator already assigned")
	ErrAssignmentNotFound             = errors.New("society administrator assignment not found")
	ErrAssignmentAlreadyRevoked       = errors.New("society administrator assignment is already revoked")
	ErrSocietyPending                 = errors.New("society is pending activation")
	ErrSocietySuspended               = errors.New("society is currently suspended")
	ErrSocietyAccessDenied            = errors.New("society access denied")
	ErrInvalidRole                    = errors.New("invalid society role")
	ErrGatekeeperNotFound             = errors.New("gatekeeper assignment not found")
	ErrGatekeeperAlreadyAssigned      = errors.New("user already assigned as gatekeeper in this society")
	ErrGateInactive                   = errors.New("gate is inactive")
	ErrSessionNotFound                = errors.New("active duty session not found")
	ErrSessionAlreadyActive           = errors.New("user already checked into an active duty session")
	ErrShiftNameExists                = errors.New("shift with this name already exists")
	ErrInvalidTimeRange               = errors.New("invalid shift time range")
	ErrAnnouncementNotFound           = errors.New("announcement not found")
	ErrAnnouncementInvalidLifecycle   = errors.New("invalid announcement lifecycle transition")
	ErrAnnouncementTargetMismatch     = errors.New("announcement target does not match audience type")
	ErrAnnouncementCrossSocietyTarget = errors.New("target resource does not belong to the society")
	ErrAnnouncementTitleRequired      = errors.New("announcement title is required")
	ErrAnnouncementBodyRequired       = errors.New("announcement body is required")
	ErrAnnouncementAccessDenied       = errors.New("announcement access denied")
	ErrChargeCategoryNotFound         = errors.New("charge category not found")
	ErrChargeCategoryNameExists       = errors.New("charge category with this name already exists in society")
	ErrChargeConfigNotFound           = errors.New("charge configuration not found")
	ErrBillingPeriodNotFound          = errors.New("billing period not found")
	ErrBillingPeriodOverlap           = errors.New("billing period with overlapping dates already exists in society")
	ErrInvalidBillingDates            = errors.New("billing period end date must be on or after start date, and due date must be on or after start date")
	ErrBillNotFound                   = errors.New("maintenance bill not found")
	ErrBillAlreadyVoid                = errors.New("maintenance bill is already void")
	ErrBillAlreadyExists              = errors.New("maintenance bill already exists for this unit and billing period")
	ErrNoEligibleCharges              = errors.New("no active charge configurations match the billing period")
	ErrInvalidAmount                  = errors.New("amount must be greater than zero")
	ErrPaymentNotFound                = errors.New("maintenance payment not found")
	ErrReceiptNotFound                = errors.New("maintenance receipt not found")
	ErrInvalidPaymentMethod           = errors.New("invalid payment method, must be CASH, CHEQUE, BANK_TRANSFER, UPI_RECORDED_OFFLINE, or OTHER")
	ErrAllocationExceedsBalance       = errors.New("allocated amount exceeds outstanding bill balance")
	ErrBillCannotBePaid               = errors.New("cannot allocate payment to a voided bill")
	ErrPaymentAlreadyReversed         = errors.New("maintenance payment has already been reversed")
	ErrReversalReasonRequired         = errors.New("reversal reason is required")
	ErrIdempotencyConflict            = errors.New("idempotency key reused with conflicting payload")
	ErrBillHasAllocations             = errors.New("cannot void bill with active payment allocations; reverse allocations first")
)

// SuspensionReason represents a catalogue record for society suspensions.
type SuspensionReason struct {
	ID              int16     `json:"id"`
	Code            string    `json:"code"`
	Label           string    `json:"label"`
	Description     *string   `json:"description,omitempty"`
	Category        string    `json:"category"`
	AllowsManual    bool      `json:"allows_manual"`
	AllowsAutomated bool      `json:"allows_automated"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SuspendSocietyParams contains input parameters for suspending a society.
type SuspendSocietyParams struct {
	ReasonID      *int16  `json:"reason_id,omitempty"`
	ReasonCode    *string `json:"reason_code,omitempty"`
	ReasonDetails *string `json:"reason_details,omitempty"`
}

// Society represents a registered housing society tenant entity.
type Society struct {
	ID                 uuid.UUID  `json:"id"`
	SocietyCode        string     `json:"society_code"`
	Name               string     `json:"name"`
	RegistrationNumber *string    `json:"registration_number,omitempty"`
	AddressLine1       string     `json:"address_line1"`
	AddressLine2       *string    `json:"address_line2,omitempty"`
	Locality           string     `json:"locality"`
	City               string     `json:"city"`
	State              string     `json:"state"`
	Pincode            string     `json:"pincode"`
	ContactEmail       *string    `json:"contact_email,omitempty"`
	ContactPhone       *string    `json:"contact_phone,omitempty"`
	Status             string     `json:"status"`
	CreatedBy          uuid.UUID  `json:"created_by"`
	UpdatedBy          *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CreateSocietyParams holds input values for registering a new society.
type CreateSocietyParams struct {
	Name               string  `json:"name"`
	RegistrationNumber *string `json:"registration_number,omitempty"`
	AddressLine1       string  `json:"address_line1"`
	AddressLine2       *string `json:"address_line2,omitempty"`
	Locality           string  `json:"locality"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Pincode            string  `json:"pincode"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
}

// UpdateSocietyParams holds input values for modifying an existing society's profile.
type UpdateSocietyParams struct {
	Name               string  `json:"name"`
	RegistrationNumber *string `json:"registration_number,omitempty"`
	AddressLine1       string  `json:"address_line1"`
	AddressLine2       *string `json:"address_line2,omitempty"`
	Locality           string  `json:"locality"`
	City               string  `json:"city"`
	State              string  `json:"state"`
	Pincode            string  `json:"pincode"`
	ContactEmail       *string `json:"contact_email,omitempty"`
	ContactPhone       *string `json:"contact_phone,omitempty"`
}

// ListSocietiesFilter holds filtering and pagination options.
type ListSocietiesFilter struct {
	Status string `json:"status,omitempty"`
	Search string `json:"search,omitempty"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// SocietyAdminAssignment represents an administrator role mapping in society_user_roles.
type SocietyAdminAssignment struct {
	ID              uuid.UUID  `json:"id"`
	SocietyID       uuid.UUID  `json:"society_id"`
	UserID          uuid.UUID  `json:"user_id"`
	UserPhoneMasked string     `json:"user_phone_masked,omitempty"`
	Role            string     `json:"role"`
	Status          string     `json:"status"`
	AssignedBy      uuid.UUID  `json:"assigned_by"`
	RevokedBy       *uuid.UUID `json:"revoked_by,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AssignSocietyAdminParams contains inputs to map an administrator to a society.
type AssignSocietyAdminParams struct {
	SocietyID uuid.UUID  `json:"society_id"`
	Phone     string     `json:"phone,omitempty"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Role      string     `json:"role"`
}

// ListSocietyAdminsFilter specifies query filters for listing a society's administrators.
type ListSocietyAdminsFilter struct {
	SocietyID uuid.UUID `json:"society_id"`
	Status    string    `json:"status,omitempty"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// SocietyAdminListResult wraps the paginated administrators result.
type SocietyAdminListResult struct {
	Items  []SocietyAdminAssignment `json:"items"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// UserSocietyAssignment represents an active society assignment record discovered for an authenticated user.
type UserSocietyAssignment struct {
	SocietyID   uuid.UUID `json:"society_id"`
	SocietyCode string    `json:"society_code"`
	SocietyName string    `json:"society_name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	City        string    `json:"city,omitempty"`
	State       string    `json:"state,omitempty"`
}

// UpdateGateParams contains input for updating an existing gate.
type UpdateGateParams struct {
	Name                  *string `json:"name,omitempty"`
	Directionality        *string `json:"directionality,omitempty"`
	OperationalStatus     *string `json:"operational_status,omitempty"`
	RequiredStaffPerShift *int    `json:"required_staff_per_shift,omitempty"`
	Description           *string `json:"description,omitempty"`
}

// GateShift represents a recurring shift schedule.
type GateShift struct {
	ID        uuid.UUID `json:"id"`
	SocietyID uuid.UUID `json:"society_id"`
	Name      string    `json:"name"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateGateShiftParams input for defining a new shift.
type CreateGateShiftParams struct {
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// GateDutySession represents a guard's active or concluded duty session at a gate.
type GateDutySession struct {
	ID            uuid.UUID  `json:"id"`
	SocietyID     uuid.UUID  `json:"society_id"`
	GateID        uuid.UUID  `json:"gate_id"`
	GateName      string     `json:"gate_name"`
	GateCode      string     `json:"gate_code"`
	UserID        uuid.UUID  `json:"user_id"`
	UserPhone     string     `json:"user_phone"`
	ShiftID       *uuid.UUID `json:"shift_id,omitempty"`
	ShiftName     *string    `json:"shift_name,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	Status        string     `json:"status"`
	HandoverNotes *string    `json:"handover_notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// CheckInDutyParams input for checking into gate duty.
type CheckInDutyParams struct {
	ShiftID *uuid.UUID `json:"shift_id,omitempty"`
}

// CheckOutDutyParams input for concluding gate duty.
type CheckOutDutyParams struct {
	HandoverNotes *string `json:"handover_notes,omitempty"`
}

// MaintenanceChargeCategory represents a classification of society charges.
type MaintenanceChargeCategory struct {
	ID          uuid.UUID `json:"id"`
	SocietyID   uuid.UUID `json:"society_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateChargeCategoryParams input for creating a category.
type CreateChargeCategoryParams struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdateChargeCategoryParams input for updating a category.
type UpdateChargeCategoryParams struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
}

// MaintenanceChargeConfig represents a charging rule.
type MaintenanceChargeConfig struct {
	ID                 uuid.UUID  `json:"id"`
	SocietyID          uuid.UUID  `json:"society_id"`
	CategoryID         uuid.UUID  `json:"category_id"`
	CategoryName       string     `json:"category_name,omitempty"`
	Name               string     `json:"name"`
	ChargeType         string     `json:"charge_type"`
	ApplicableUnitType *string    `json:"applicable_unit_type,omitempty"`
	Amount             float64    `json:"amount"`
	BillingFrequency   string     `json:"billing_frequency"`
	EffectiveFrom      time.Time  `json:"effective_from"`
	EffectiveUntil     *time.Time `json:"effective_until,omitempty"`
	Status             string     `json:"status"`
	CreatedBy          uuid.UUID  `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CreateChargeConfigParams input for creating a configuration.
type CreateChargeConfigParams struct {
	CategoryID         uuid.UUID  `json:"category_id"`
	Name               string     `json:"name"`
	ChargeType         string     `json:"charge_type"`
	ApplicableUnitType *string    `json:"applicable_unit_type,omitempty"`
	Amount             float64    `json:"amount"`
	BillingFrequency   string     `json:"billing_frequency"`
	EffectiveFrom      time.Time  `json:"effective_from"`
	EffectiveUntil     *time.Time `json:"effective_until,omitempty"`
}

// MaintenanceBillingPeriod represents a billing cycle window.
type MaintenanceBillingPeriod struct {
	ID          uuid.UUID `json:"id"`
	SocietyID   uuid.UUID `json:"society_id"`
	PeriodName  string    `json:"period_name"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateBillingPeriodParams input for creating a period.
type CreateBillingPeriodParams struct {
	PeriodName  string    `json:"period_name"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	DueDate     time.Time `json:"due_date"`
}

// MaintenanceBillItem represents an immutable line item snapshot.
type MaintenanceBillItem struct {
	ID               uuid.UUID  `json:"id"`
	BillID           uuid.UUID  `json:"bill_id"`
	SocietyID        uuid.UUID  `json:"society_id"`
	ChargeCategoryID *uuid.UUID `json:"charge_category_id,omitempty"`
	CategoryName     string     `json:"category_name"`
	ChargeConfigID   *uuid.UUID `json:"charge_config_id,omitempty"`
	Description      string     `json:"description"`
	Rate             float64    `json:"rate"`
	Quantity         float64    `json:"quantity"`
	Amount           float64    `json:"amount"`
	CreatedAt        time.Time  `json:"created_at"`
}

// MaintenanceBill represents an issued bill for a unit and period.
type MaintenanceBill struct {
	ID              uuid.UUID             `json:"id"`
	SocietyID       uuid.UUID             `json:"society_id"`
	UnitID          uuid.UUID             `json:"unit_id"`
	UnitIdentifier  string                `json:"unit_identifier,omitempty"`
	BuildingName    string                `json:"building_name,omitempty"`
	FloorNumber     int                   `json:"floor_number,omitempty"`
	BillingPeriodID uuid.UUID             `json:"billing_period_id"`
	PeriodName      string                `json:"period_name,omitempty"`
	BillNumber      string                `json:"bill_number"`
	IssueDate       time.Time             `json:"issue_date"`
	DueDate         time.Time             `json:"due_date"`
	Currency        string                `json:"currency"`
	Subtotal        float64               `json:"subtotal"`
	TotalAmount     float64               `json:"total_amount"`
	Status          string                `json:"status"`
	DerivedStatus   string                `json:"derived_status"`
	CreatedBy       uuid.UUID             `json:"created_by"`
	VoidedBy        *uuid.UUID            `json:"voided_by,omitempty"`
	VoidedAt        *time.Time            `json:"voided_at,omitempty"`
	VoidReason      *string               `json:"void_reason,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	LineItems       []MaintenanceBillItem `json:"line_items,omitempty"`
	Items           []MaintenanceBillItem `json:"items,omitempty"`
}

// BillGenerationPreview provides projected billing information.
type BillGenerationPreview struct {
	PeriodID             uuid.UUID                 `json:"period_id"`
	PeriodName           string                    `json:"period_name"`
	TotalUnits           int                       `json:"total_units"`
	AlreadyBilledUnits   int                       `json:"already_billed_units"`
	EligibleUnits        int                       `json:"eligible_units"`
	ProjectedTotalAmount float64                   `json:"projected_total_amount"`
	ApplicableConfigs    []MaintenanceChargeConfig `json:"applicable_configs"`
}

// BillGenerationResult summary after generation run.
type BillGenerationResult struct {
	PeriodID             uuid.UUID `json:"period_id"`
	BillsCreated         int       `json:"bills_created"`
	UnitsSkippedExisting int       `json:"units_skipped_existing"`
	TotalBilledAmount    float64   `json:"total_billed_amount"`
}

// ListBillsFilter provides query filtering options for bills.
type ListBillsFilter struct {
	PeriodID *uuid.UUID `json:"period_id,omitempty"`
	UnitID   *uuid.UUID `json:"unit_id,omitempty"`
	Status   string     `json:"status,omitempty"`
	Search   string     `json:"search,omitempty"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

// ListBillsResult provides paginated bills.
type ListBillsResult struct {
	Items  []MaintenanceBill `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// Payment methods
const (
	PaymentMethodCash         = "CASH"
	PaymentMethodCheque       = "CHEQUE"
	PaymentMethodBankTransfer = "BANK_TRANSFER"
	PaymentMethodUPIOffline   = "UPI_RECORDED_OFFLINE"
	PaymentMethodOther        = "OTHER"
)

// Payment and allocation statuses
const (
	PaymentStatusRecorded = "RECORDED"
	PaymentStatusReversed = "REVERSED"

	AllocationStatusActive   = "ACTIVE"
	AllocationStatusReversed = "REVERSED"

	ReceiptStatusIssued   = "ISSUED"
	ReceiptStatusReversed = "REVERSED"
)

// MaintenancePayment represents money recorded as received by the society.
type MaintenancePayment struct {
	ID                uuid.UUID  `json:"id"`
	SocietyID         uuid.UUID  `json:"society_id"`
	PaymentReference  string     `json:"payment_reference"`
	PaymentMethod     string     `json:"payment_method"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	PaymentDate       time.Time  `json:"payment_date"`
	ExternalReference *string    `json:"external_reference,omitempty"`
	Status            string     `json:"status"`
	RecordedBy        uuid.UUID  `json:"recorded_by"`
	PayerName         *string    `json:"payer_name,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	ReversedAt        *time.Time `json:"reversed_at,omitempty"`
	ReversedBy        *uuid.UUID `json:"reversed_by,omitempty"`
	ReversalReason    *string    `json:"reversal_reason,omitempty"`
	IdempotencyKey    *string    `json:"idempotency_key,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// MaintenancePaymentAllocation links recorded funds to a maintenance bill.
type MaintenancePaymentAllocation struct {
	ID              uuid.UUID  `json:"id"`
	SocietyID       uuid.UUID  `json:"society_id"`
	PaymentID       uuid.UUID  `json:"payment_id"`
	BillID          uuid.UUID  `json:"bill_id"`
	AllocatedAmount float64    `json:"allocated_amount"`
	Status          string     `json:"status"`
	AllocatedBy     uuid.UUID  `json:"allocated_by"`
	ReversedAt      *time.Time `json:"reversed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// MaintenanceReceipt represents an issued receipt document.
type MaintenanceReceipt struct {
	ID                uuid.UUID  `json:"id"`
	SocietyID         uuid.UUID  `json:"society_id"`
	PaymentID         uuid.UUID  `json:"payment_id"`
	ReceiptNumber     string     `json:"receipt_number"`
	ReceiptDate       time.Time  `json:"receipt_date"`
	TotalAmount       float64    `json:"total_amount"`
	Currency          string     `json:"currency"`
	IssuedBy          uuid.UUID  `json:"issued_by"`
	Status            string     `json:"status"`
	ReversedAt        *time.Time `json:"reversed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	SocietyName       string     `json:"society_name,omitempty"`
	SocietyCode       string     `json:"society_code,omitempty"`
	PaymentReference  string     `json:"payment_reference,omitempty"`
	PaymentMethod     string     `json:"payment_method,omitempty"`
	ExternalReference *string    `json:"external_reference,omitempty"`
	PayerName         *string    `json:"payer_name,omitempty"`
	BillNumber        string     `json:"bill_number,omitempty"`
	UnitIdentifier    string     `json:"unit_identifier,omitempty"`
}

// RecordPaymentParams captures input for recording and allocating a payment.
type RecordPaymentParams struct {
	BillID            uuid.UUID
	Amount            float64
	PaymentMethod     string
	PaymentDate       time.Time
	ExternalReference *string
	PayerName         *string
	Notes             *string
	IdempotencyKey    *string
}

// RecordPaymentResult encapsulates the atomic outcome of recording a payment.
type RecordPaymentResult struct {
	Payment          MaintenancePayment           `json:"payment"`
	Receipt          MaintenanceReceipt           `json:"receipt"`
	Allocation       MaintenancePaymentAllocation `json:"allocation"`
	RemainingBalance float64                      `json:"remaining_balance"`
}

// CollectionRegisterItem represents a row in the admin collection register.
type CollectionRegisterItem struct {
	BillID             uuid.UUID  `json:"bill_id"`
	BillNumber         string     `json:"bill_number"`
	UnitID             uuid.UUID  `json:"unit_id"`
	UnitIdentifier     string     `json:"unit_identifier"`
	PeriodID           uuid.UUID  `json:"period_id"`
	PeriodName         string     `json:"period_name"`
	DueDate            time.Time  `json:"due_date"`
	TotalAmount        float64    `json:"total_amount"`
	TotalPaid          float64    `json:"total_paid"`
	OutstandingBalance float64    `json:"outstanding_balance"`
	BillStatus         string     `json:"bill_status"`
	DerivedStatus      string     `json:"derived_status"`
	CollectionStatus   string     `json:"collection_status"`
	LastPaymentDate    *time.Time `json:"last_payment_date,omitempty"`
}

// ListCollectionRegisterFilter provides filtering options for collections register.
type ListCollectionRegisterFilter struct {
	PeriodID *uuid.UUID `json:"period_id,omitempty"`
	Status   string     `json:"status,omitempty"` // UNPAID, PARTIALLY_PAID, PAID, VOIDED
	Search   string     `json:"search,omitempty"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

// ListCollectionRegisterResult returns paginated collections.
type ListCollectionRegisterResult struct {
	Items  []CollectionRegisterItem `json:"items"`
	Total  int                      `json:"total"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}

// CollectionSummary aggregates society collection metrics.
type CollectionSummary struct {
	TotalBilled             float64 `json:"total_billed"`
	TotalCollected          float64 `json:"total_collected"`
	TotalOutstanding        float64 `json:"total_outstanding"`
	UnpaidBillsCount        int     `json:"unpaid_bills_count"`
	PartiallyPaidBillsCount int     `json:"partially_paid_bills_count"`
	PaidBillsCount          int     `json:"paid_bills_count"`
	ReversedPaymentsCount   int     `json:"reversed_payments_count"`
	ReversedAmount          float64 `json:"reversed_amount"`
}

// ResidentPaymentItem provides resident-visible payment and receipt summary.
type ResidentPaymentItem struct {
	PaymentID        uuid.UUID `json:"payment_id"`
	PaymentReference string    `json:"payment_reference"`
	ReceiptID        uuid.UUID `json:"receipt_id"`
	ReceiptNumber    string    `json:"receipt_number"`
	BillID           uuid.UUID `json:"bill_id"`
	BillNumber       string    `json:"bill_number"`
	UnitIdentifier   string    `json:"unit_identifier"`
	Amount           float64   `json:"amount"`
	PaymentMethod    string    `json:"payment_method"`
	PaymentDate      time.Time `json:"payment_date"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// S13 Documents & Records Constants
const (
	ClassificationSocietyResidents = "SOCIETY_RESIDENTS"
	ClassificationCommittee        = "COMMITTEE"
	ClassificationSocAdminOnly     = "SOC_ADMIN_ONLY"

	DocumentStatusActive   = "ACTIVE"
	DocumentStatusArchived = "ARCHIVED"

	ActionDocumentCreated         = "DOCUMENT_CREATED"
	ActionDocumentUpdated         = "DOCUMENT_UPDATED"
	ActionDocumentVersionUploaded = "DOCUMENT_VERSION_UPLOADED"
	ActionDocumentArchived        = "DOCUMENT_ARCHIVED"
	ActionDocumentRestored        = "DOCUMENT_RESTORED"
	ActionDocumentDownloaded      = "DOCUMENT_DOWNLOADED"
	ActionDocumentCategoryCreated = "DOCUMENT_CATEGORY_CREATED"
)

// DocumentCategory represents a classification bucket for society records.
type DocumentCategory struct {
	ID            uuid.UUID `json:"id"`
	SocietyID     uuid.UUID `json:"society_id"`
	Name          string    `json:"name"`
	Description   *string   `json:"description,omitempty"`
	IsSystem      bool      `json:"is_system"`
	IsActive      bool      `json:"is_active"`
	DisplayOrder  int       `json:"display_order"`
	DocumentCount int       `json:"document_count,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DocumentVersion represents an append-only revision of a society document.
type DocumentVersion struct {
	ID             uuid.UUID `json:"id"`
	SocietyID      uuid.UUID `json:"society_id"`
	DocumentID     uuid.UUID `json:"document_id"`
	VersionNumber  int       `json:"version_number"`
	StoragePath    string    `json:"storage_path"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	MimeType       string    `json:"mime_type"`
	FileHash       string    `json:"file_hash"`
	ChangeSummary  *string   `json:"change_summary,omitempty"`
	UploadedBy     uuid.UUID `json:"uploaded_by"`
	UploadedByName string    `json:"uploaded_by_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// SocietyDocument represents the primary metadata record of an official society document.
type SocietyDocument struct {
	ID               uuid.UUID        `json:"id"`
	SocietyID        uuid.UUID        `json:"society_id"`
	CategoryID       uuid.UUID        `json:"category_id"`
	CategoryName     string           `json:"category_name,omitempty"`
	DocumentNumber   string           `json:"document_number"`
	Title            string           `json:"title"`
	Description      *string          `json:"description,omitempty"`
	Classification   string           `json:"classification"`
	Status           string           `json:"status"`
	CurrentVersionID *uuid.UUID       `json:"current_version_id,omitempty"`
	CurrentVersion   *DocumentVersion `json:"current_version,omitempty"`
	BuildingID       *uuid.UUID       `json:"building_id,omitempty"`
	BuildingName     *string          `json:"building_name,omitempty"`
	WingID           *uuid.UUID       `json:"wing_id,omitempty"`
	WingName         *string          `json:"wing_name,omitempty"`
	UnitID           *uuid.UUID       `json:"unit_id,omitempty"`
	UnitNumber       *string          `json:"unit_number,omitempty"`
	MemberID         *uuid.UUID       `json:"member_id,omitempty"`
	MemberName       *string          `json:"member_name,omitempty"`
	CreatedBy        uuid.UUID        `json:"created_by"`
	CreatedByName    string           `json:"created_by_name,omitempty"`
	UpdatedBy        uuid.UUID        `json:"updated_by"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	ArchivedAt       *time.Time       `json:"archived_at,omitempty"`
	ArchivedBy       *uuid.UUID       `json:"archived_by,omitempty"`
	ArchivedReason   *string          `json:"archived_reason,omitempty"`
	VersionCount     int              `json:"version_count,omitempty"`
}

// ListDocumentsFilter provides query parameters for filtering society documents.
type ListDocumentsFilter struct {
	CategoryID     *uuid.UUID `json:"category_id,omitempty"`
	Classification string     `json:"classification,omitempty"`
	Status         string     `json:"status,omitempty"`
	Search         string     `json:"search,omitempty"`
	BuildingID     *uuid.UUID `json:"building_id,omitempty"`
	WingID         *uuid.UUID `json:"wing_id,omitempty"`
	UnitID         *uuid.UUID `json:"unit_id,omitempty"`
	MemberID       *uuid.UUID `json:"member_id,omitempty"`
	Limit          int        `json:"limit"`
	Offset         int        `json:"offset"`
}

// ListDocumentsResult encapsulates a paginated set of society documents.
type ListDocumentsResult struct {
	Items  []SocietyDocument `json:"items"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// CreateDocumentParams holds inputs for uploading an initial document and version.
type CreateDocumentParams struct {
	CategoryID     uuid.UUID
	Title          string
	Description    *string
	Classification string
	BuildingID     *uuid.UUID
	WingID         *uuid.UUID
	UnitID         *uuid.UUID
	MemberID       *uuid.UUID
	FileName       string
	FileSize       int64
	MimeType       string
	FileContent    []byte
	ChangeSummary  *string
	ActorID        uuid.UUID
	ClientIP       string
	UserAgent      string
}

// UploadVersionParams holds inputs for creating a new version of an existing document.
type UploadVersionParams struct {
	DocumentID    uuid.UUID
	FileName      string
	FileSize      int64
	MimeType      string
	FileContent   []byte
	ChangeSummary *string
	ActorID       uuid.UUID
	ClientIP      string
	UserAgent     string
}

// UpdateDocumentParams holds inputs for updating mutable document metadata.
type UpdateDocumentParams struct {
	DocumentID     uuid.UUID
	Title          string
	Description    *string
	CategoryID     uuid.UUID
	Classification string
	BuildingID     *uuid.UUID
	WingID         *uuid.UUID
	UnitID         *uuid.UUID
	MemberID       *uuid.UUID
	ActorID        uuid.UUID
	ClientIP       string
	UserAgent      string
}
