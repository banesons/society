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
)

// Assignment statuses
const (
	AssignmentStatusActive  = "ACTIVE"
	AssignmentStatusRevoked = "REVOKED"
)

// Audit actions
const (
	ActionSocietyCreated        = "SOCIETY_CREATED"
	ActionSocietyProfileUpdated = "SOCIETY_PROFILE_UPDATED"
	ActionSocietyActivated      = "SOCIETY_ACTIVATED"
	ActionSocietySuspended      = "SOCIETY_SUSPENDED"
	ActionSocietyReactivated    = "SOCIETY_REACTIVATED"
	ActionSocietyAdminAssigned  = "SOCIETY_ADMIN_ASSIGNED"
	ActionSocietyAdminRevoked   = "SOCIETY_ADMIN_REVOKED"
)

// Suspension origins
const (
	OriginManual    = "MANUAL"
	OriginAutomated = "AUTOMATED"
)

// Domain errors
var (
	ErrSocietyNotFound             = errors.New("society not found")
	ErrInvalidStatusTransition     = errors.New("invalid status transition")
	ErrAlreadyInStatus             = errors.New("society already in requested status")
	ErrReasonRequired              = errors.New("reason is required for this action")
	ErrSocietyCodeExists           = errors.New("society code already exists")
	ErrRegNumberExists             = errors.New("registration number already exists in this state")
	ErrMaxRetriesExceeded          = errors.New("exceeded maximum code collision retries")
	ErrInvalidPincode              = errors.New("pincode must be exactly 6 Indian postal digits")
	ErrInvalidName                 = errors.New("society name must be between 3 and 255 characters")
	ErrRequiredFieldMissing        = errors.New("required address fields cannot be blank")
	ErrInvalidSuspensionReason     = errors.New("invalid suspension reason code")
	ErrSuspensionReasonRequired    = errors.New("suspension reason code is required")
	ErrSuspensionDetailsRequired   = errors.New("additional details are required for this suspension reason")
	ErrSuspensionReasonInactive    = errors.New("selected suspension reason is inactive")
	ErrManualSuspensionNotAllowed  = errors.New("selected suspension reason is not permitted for manual administrative use")
	ErrDetailsTooLong              = errors.New("additional details must not exceed 1000 characters")
	ErrUserNotFound                = errors.New("user not found")
	ErrSocietyAdminAlreadyAssigned = errors.New("society administrator already assigned")
	ErrAssignmentNotFound          = errors.New("society administrator assignment not found")
	ErrAssignmentAlreadyRevoked    = errors.New("society administrator assignment is already revoked")
	ErrSocietyPending              = errors.New("society is pending activation")
	ErrSocietySuspended            = errors.New("society is currently suspended")
	ErrSocietyAccessDenied         = errors.New("society access denied")
	ErrInvalidRole                 = errors.New("invalid society role")
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
}
