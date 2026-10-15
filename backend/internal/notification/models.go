package notification

import (
	"time"

	"github.com/google/uuid"
)

// Platform types for registered user devices.
const (
	PlatformWeb     = "WEB"
	PlatformAndroid = "ANDROID"
	PlatformIOS     = "IOS"
)

// Device status types.
const (
	DeviceStatusActive   = "ACTIVE"
	DeviceStatusInactive = "INACTIVE"
	DeviceStatusRevoked  = "REVOKED"
)

// Notification channels.
const (
	ChannelFCM   = "FCM"
	ChannelEmail = "EMAIL"
)

// Delivery statuses.
const (
	DeliveryStatusPending           = "PENDING"
	DeliveryStatusProcessing        = "PROCESSING"
	DeliveryStatusDelivered         = "DELIVERED"
	DeliveryStatusFailed            = "FAILED"
	DeliveryStatusPermanentlyFailed = "PERMANENTLY_FAILED"
	DeliveryStatusSkipped           = "SKIPPED"
)

// Outbox statuses.
const (
	OutboxStatusPending    = "PENDING"
	OutboxStatusProcessing = "PROCESSING"
	OutboxStatusCompleted  = "COMPLETED"
	OutboxStatusFailed     = "FAILED"
)

// Supported domain notification event types.
const (
	EventVisitorApprovalRequested = "VISITOR_APPROVAL_REQUESTED"
	EventVisitApproved            = "VISIT_APPROVED"
	EventVisitRejected            = "VISIT_REJECTED"
	EventAnnouncementPublished    = "ANNOUNCEMENT_PUBLISHED"
	EventBillIssued               = "BILL_ISSUED"
	EventPaymentRecorded          = "PAYMENT_RECORDED"
	EventPaymentReversed          = "PAYMENT_REVERSED"
)

// UserDevice models a registered device holding an FCM push token.
type UserDevice struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	DeviceID   string    `json:"device_id"`
	FCMToken   string    `json:"fcm_token,omitempty"`
	Platform   string    `json:"platform"`
	UserAgent  string    `json:"user_agent,omitempty"`
	ClientIP   string    `json:"client_ip,omitempty"`
	Status     string    `json:"status"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Notification models a user-facing in-app notification ledger item.
type Notification struct {
	ID              uuid.UUID              `json:"id"`
	RecipientUserID uuid.UUID              `json:"recipient_user_id"`
	SocietyID       *uuid.UUID             `json:"society_id,omitempty"`
	Type            string                 `json:"type"`
	Title           string                 `json:"title"`
	Body            string                 `json:"body"`
	ActionURL       string                 `json:"action_url,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
	ReferenceType   string                 `json:"reference_type,omitempty"`
	ReferenceID     *uuid.UUID             `json:"reference_id,omitempty"`
	ReadAt          *time.Time             `json:"read_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

// NotificationDelivery models an outbound channel delivery attempt.
type NotificationDelivery struct {
	ID                uuid.UUID  `json:"id"`
	NotificationID    uuid.UUID  `json:"notification_id"`
	Channel           string     `json:"channel"`
	Target            string     `json:"target"`
	DeviceID          *uuid.UUID `json:"device_id,omitempty"`
	Status            string     `json:"status"`
	AttemptCount      int        `json:"attempt_count"`
	MaxAttempts       int        `json:"max_attempts"`
	NextRetryAt       time.Time  `json:"next_retry_at"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	ErrorCode         *string    `json:"error_code,omitempty"`
	ErrorDetail       *string    `json:"error_detail,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// OutboxEvent models a durable pending event emitted from domain operations.
type OutboxEvent struct {
	ID             uuid.UUID              `json:"id"`
	EventType      string                 `json:"event_type"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Payload        map[string]interface{} `json:"payload"`
	Status         string                 `json:"status"`
	AttemptCount   int                    `json:"attempt_count"`
	MaxAttempts    int                    `json:"max_attempts"`
	NextAttemptAt  time.Time              `json:"next_attempt_at"`
	LockedUntil    *time.Time             `json:"locked_until,omitempty"`
	LockedBy       *string                `json:"locked_by,omitempty"`
	LastError      *string                `json:"last_error,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	ProcessedAt    *time.Time             `json:"processed_at,omitempty"`
}

// RegisterDeviceParams input contract for registering/refreshing an FCM device token.
type RegisterDeviceParams struct {
	DeviceID string `json:"device_id"`
	FCMToken string `json:"fcm_token"`
	Platform string `json:"platform"`
}

// NotificationListFilters query filter options for listing user notifications.
type NotificationListFilters struct {
	Page       int
	Limit      int
	UnreadOnly bool
	SocietyID  *uuid.UUID
}
