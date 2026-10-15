package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeviceNotFound       = errors.New("device not found")
	ErrNotificationNotFound = errors.New("notification not found")
)

// Repository manages persistence for notifications, user devices, deliveries, and outbox events.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new notification repository backed by pgxpool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// RegisterDevice registers or updates an active FCM device token for an authenticated user.
func (r *Repository) RegisterDevice(ctx context.Context, userID uuid.UUID, params RegisterDeviceParams, userAgent, clientIP string) (*UserDevice, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	deviceID := strings.TrimSpace(params.DeviceID)
	fcmToken := strings.TrimSpace(params.FCMToken)
	platform := strings.ToUpper(strings.TrimSpace(params.Platform))

	if deviceID == "" || fcmToken == "" {
		return nil, errors.New("device_id and fcm_token are required")
	}
	if platform != PlatformWeb && platform != PlatformAndroid && platform != PlatformIOS {
		platform = PlatformWeb
	}

	query := `
		INSERT INTO user_devices (
			user_id, device_id, fcm_token, platform, user_agent, client_ip, status, last_seen_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'ACTIVE', NOW(), NOW(), NOW()
		)
		ON CONFLICT (user_id, device_id) DO UPDATE SET
			fcm_token = EXCLUDED.fcm_token,
			platform = EXCLUDED.platform,
			user_agent = COALESCE(NULLIF(EXCLUDED.user_agent, ''), user_devices.user_agent),
			client_ip = COALESCE(NULLIF(EXCLUDED.client_ip, ''), user_devices.client_ip),
			status = 'ACTIVE',
			last_seen_at = NOW(),
			updated_at = NOW()
		RETURNING id, user_id, device_id, fcm_token, platform, COALESCE(user_agent, ''), COALESCE(client_ip, ''), status, last_seen_at, created_at, updated_at
	`

	var dev UserDevice
	err := r.pool.QueryRow(ctx, query, userID, deviceID, fcmToken, platform, userAgent, clientIP).Scan(
		&dev.ID,
		&dev.UserID,
		&dev.DeviceID,
		&dev.FCMToken,
		&dev.Platform,
		&dev.UserAgent,
		&dev.ClientIP,
		&dev.Status,
		&dev.LastSeenAt,
		&dev.CreatedAt,
		&dev.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return &dev, nil
}

// DeactivateDevice marks a device inactive for a specific user.
func (r *Repository) DeactivateDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	query := `
		UPDATE user_devices
		SET status = 'INACTIVE', updated_at = NOW()
		WHERE user_id = $1 AND device_id = $2 AND status = 'ACTIVE'
	`
	tag, err := r.pool.Exec(ctx, query, userID, strings.TrimSpace(deviceID))
	if err != nil {
		return fmt.Errorf("failed to deactivate device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// DeactivateToken marks all devices with a given FCM token inactive (e.g. invalid/unregistered token feedback).
func (r *Repository) DeactivateToken(ctx context.Context, fcmToken string) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	query := `
		UPDATE user_devices
		SET status = 'INACTIVE', updated_at = NOW()
		WHERE fcm_token = $1 AND status = 'ACTIVE'
	`
	_, err := r.pool.Exec(ctx, query, strings.TrimSpace(fcmToken))
	if err != nil {
		return fmt.Errorf("failed to deactivate invalid token: %w", err)
	}
	return nil
}

// ListUserDevices returns all active registered devices for an authenticated user.
func (r *Repository) ListUserDevices(ctx context.Context, userID uuid.UUID) ([]UserDevice, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, user_id, device_id, platform, status, last_seen_at, created_at, updated_at
		FROM user_devices
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY last_seen_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user devices: %w", err)
	}
	defer rows.Close()

	var devices []UserDevice
	for rows.Next() {
		var d UserDevice
		if err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.DeviceID,
			&d.Platform,
			&d.Status,
			&d.LastSeenAt,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, d)
	}

	if devices == nil {
		devices = []UserDevice{}
	}
	return devices, nil
}

// ListNotifications returns a paginated list of notifications for the authenticated user.
func (r *Repository) ListNotifications(ctx context.Context, userID uuid.UUID, filters NotificationListFilters) ([]Notification, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, fmt.Sprintf("recipient_user_id = $%d", argIdx))
	args = append(args, userID)
	argIdx++

	if filters.UnreadOnly {
		whereClauses = append(whereClauses, "read_at IS NULL")
	}

	if filters.SocietyID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(society_id = $%d OR society_id IS NULL)", argIdx))
		args = append(args, *filters.SocietyID)
		argIdx++
	}

	whereSQL := "WHERE " + strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM notifications %s`, whereSQL)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, recipient_user_id, society_id, type, title, body,
		       COALESCE(action_url, ''), metadata, COALESCE(reference_type, ''), reference_id,
		       read_at, created_at
		FROM notifications
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	queryArgs := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var n Notification
		var rawMeta []byte
		var refType, actionURL string

		if err := rows.Scan(
			&n.ID,
			&n.RecipientUserID,
			&n.SocietyID,
			&n.Type,
			&n.Title,
			&n.Body,
			&actionURL,
			&rawMeta,
			&refType,
			&n.ReferenceID,
			&n.ReadAt,
			&n.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		n.ActionURL = actionURL
		n.ReferenceType = refType

		if len(rawMeta) > 0 {
			_ = json.Unmarshal(rawMeta, &n.Metadata)
		}
		if n.Metadata == nil {
			n.Metadata = make(map[string]interface{})
		}

		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = []Notification{}
	}
	return notifications, total, nil
}

// GetUnreadCount returns the count of unread notifications for a user.
func (r *Repository) GetUnreadCount(ctx context.Context, userID uuid.UUID, societyID *uuid.UUID) (int, error) {
	if r.pool == nil {
		return 0, errors.New("database pool unavailable")
	}

	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_user_id = $1 AND read_at IS NULL
	`
	args := []interface{}{userID}

	if societyID != nil {
		query += " AND (society_id = $2 OR society_id IS NULL)"
		args = append(args, *societyID)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
}

// MarkAsRead marks a single notification as read if it belongs to the authenticated user.
func (r *Repository) MarkAsRead(ctx context.Context, userID uuid.UUID, notificationID uuid.UUID) (*time.Time, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		UPDATE notifications
		SET read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND recipient_user_id = $2
		RETURNING read_at
	`
	var readAt time.Time
	err := r.pool.QueryRow(ctx, query, notificationID, userID).Scan(&readAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to mark notification read: %w", err)
	}
	return &readAt, nil
}

// MarkAllAsRead marks all unread notifications read for the user, optionally scoped to a society.
func (r *Repository) MarkAllAsRead(ctx context.Context, userID uuid.UUID, societyID *uuid.UUID) (int, error) {
	if r.pool == nil {
		return 0, errors.New("database pool unavailable")
	}

	query := `
		UPDATE notifications
		SET read_at = NOW()
		WHERE recipient_user_id = $1 AND read_at IS NULL
	`
	args := []interface{}{userID}

	if societyID != nil {
		query += " AND (society_id = $2 OR society_id IS NULL)"
		args = append(args, *societyID)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all notifications read: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// EnqueueOutbox inserts an outbox record inside an existing transaction or standalone.
func (r *Repository) EnqueueOutbox(ctx context.Context, tx pgx.Tx, eventType, idempotencyKey string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize outbox payload: %w", err)
	}

	query := `
		INSERT INTO notification_outbox (event_type, idempotency_key, payload, status, attempt_count, next_attempt_at)
		VALUES ($1, $2, $3::jsonb, 'PENDING', 0, NOW())
		ON CONFLICT (idempotency_key) DO NOTHING
	`

	if tx != nil {
		_, err = tx.Exec(ctx, query, eventType, idempotencyKey, string(payloadJSON))
	} else if r.pool != nil {
		_, err = r.pool.Exec(ctx, query, eventType, idempotencyKey, string(payloadJSON))
	} else {
		return errors.New("database pool unavailable")
	}

	if err != nil {
		return fmt.Errorf("failed to enqueue outbox event: %w", err)
	}
	return nil
}

// ClaimOutboxBatch acquires pending or expired-lease outbox events for processing.
func (r *Repository) ClaimOutboxBatch(ctx context.Context, workerID string, limit int) ([]OutboxEvent, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		WITH claimable AS (
			SELECT id
			FROM notification_outbox
			WHERE (status = 'PENDING' OR (status = 'PROCESSING' AND locked_until < NOW()))
			  AND next_attempt_at <= NOW()
			  AND attempt_count < max_attempts
			ORDER BY next_attempt_at ASC, id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE notification_outbox o
		SET status = 'PROCESSING',
		    locked_by = $2,
		    locked_until = NOW() + INTERVAL '2 minutes',
		    attempt_count = attempt_count + 1
		FROM claimable c
		WHERE o.id = c.id
		RETURNING o.id, o.event_type, o.idempotency_key, o.payload, o.status, o.attempt_count, o.max_attempts, o.next_attempt_at, o.locked_until, o.locked_by, o.last_error, o.created_at, o.processed_at
	`

	rows, err := r.pool.Query(ctx, query, limit, workerID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim outbox batch: %w", err)
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		var rawPayload []byte
		if err := rows.Scan(
			&e.ID,
			&e.EventType,
			&e.IdempotencyKey,
			&rawPayload,
			&e.Status,
			&e.AttemptCount,
			&e.MaxAttempts,
			&e.NextAttemptAt,
			&e.LockedUntil,
			&e.LockedBy,
			&e.LastError,
			&e.CreatedAt,
			&e.ProcessedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}
		if len(rawPayload) > 0 {
			_ = json.Unmarshal(rawPayload, &e.Payload)
		}
		events = append(events, e)
	}

	return events, nil
}

// CompleteOutboxEvent marks an outbox event as completed.
func (r *Repository) CompleteOutboxEvent(ctx context.Context, eventID uuid.UUID) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	query := `
		UPDATE notification_outbox
		SET status = 'COMPLETED', locked_until = NULL, locked_by = NULL, processed_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}

// FailOutboxEvent updates an outbox event with error and schedules retry.
func (r *Repository) FailOutboxEvent(ctx context.Context, eventID uuid.UUID, lastErr string, retryAfter time.Duration) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	status := "PENDING"
	query := `
		UPDATE notification_outbox
		SET status = CASE WHEN attempt_count >= max_attempts THEN 'FAILED' ELSE $2 END,
		    locked_until = NULL,
		    locked_by = NULL,
		    last_error = $3,
		    next_attempt_at = NOW() + $4::interval
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, eventID, status, lastErr, fmt.Sprintf("%f seconds", retryAfter.Seconds()))
	return err
}

// CreateNotificationAndDeliveries persists a notification and its delivery channels in a single transaction.
func (r *Repository) CreateNotificationAndDeliveries(
	ctx context.Context,
	n Notification,
	deliveries []NotificationDelivery,
) (*Notification, []NotificationDelivery, error) {
	if r.pool == nil {
		return nil, nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	metaJSON, err := json.Marshal(n.Metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	// Check idempotency: if notification already exists for this outbox event or reference, return existing records
	var createdN Notification
	var rawMeta []byte
	var refType, actionURL string
	var existingFound bool

	outboxEventID, _ := n.Metadata["outbox_event_id"].(string)
	if outboxEventID != "" {
		checkQuery := `
			SELECT id, recipient_user_id, society_id, type, title, body, COALESCE(action_url, ''), metadata, COALESCE(reference_type, ''), reference_id, read_at, created_at
			FROM notifications
			WHERE recipient_user_id = $1 AND metadata->>'outbox_event_id' = $2
			LIMIT 1
		`
		err = tx.QueryRow(ctx, checkQuery, n.RecipientUserID, outboxEventID).Scan(
			&createdN.ID,
			&createdN.RecipientUserID,
			&createdN.SocietyID,
			&createdN.Type,
			&createdN.Title,
			&createdN.Body,
			&actionURL,
			&rawMeta,
			&refType,
			&createdN.ReferenceID,
			&createdN.ReadAt,
			&createdN.CreatedAt,
		)
		if err == nil {
			existingFound = true
		}
	} else if n.ReferenceType != "" && n.ReferenceID != nil {
		checkQuery := `
			SELECT id, recipient_user_id, society_id, type, title, body, COALESCE(action_url, ''), metadata, COALESCE(reference_type, ''), reference_id, read_at, created_at
			FROM notifications
			WHERE recipient_user_id = $1 AND type = $2 AND reference_type = $3 AND reference_id = $4
			LIMIT 1
		`
		err = tx.QueryRow(ctx, checkQuery, n.RecipientUserID, n.Type, n.ReferenceType, n.ReferenceID).Scan(
			&createdN.ID,
			&createdN.RecipientUserID,
			&createdN.SocietyID,
			&createdN.Type,
			&createdN.Title,
			&createdN.Body,
			&actionURL,
			&rawMeta,
			&refType,
			&createdN.ReferenceID,
			&createdN.ReadAt,
			&createdN.CreatedAt,
		)
		if err == nil {
			existingFound = true
		}
	}

	if existingFound {
		createdN.ActionURL = actionURL
		createdN.ReferenceType = refType
		if len(rawMeta) > 0 {
			_ = json.Unmarshal(rawMeta, &createdN.Metadata)
		}

		deliveryQuery := `
			SELECT id, notification_id, channel, target, device_id, status, attempt_count, max_attempts, next_retry_at, provider_message_id, error_code, error_detail, delivered_at, created_at, updated_at
			FROM notification_deliveries
			WHERE notification_id = $1
		`
		dRows, err := tx.Query(ctx, deliveryQuery, createdN.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query existing deliveries: %w", err)
		}
		defer dRows.Close()

		var existingDeliveries []NotificationDelivery
		for dRows.Next() {
			var d NotificationDelivery
			if err := dRows.Scan(
				&d.ID,
				&d.NotificationID,
				&d.Channel,
				&d.Target,
				&d.DeviceID,
				&d.Status,
				&d.AttemptCount,
				&d.MaxAttempts,
				&d.NextRetryAt,
				&d.ProviderMessageID,
				&d.ErrorCode,
				&d.ErrorDetail,
				&d.DeliveredAt,
				&d.CreatedAt,
				&d.UpdatedAt,
			); err != nil {
				return nil, nil, fmt.Errorf("failed to scan existing delivery: %w", err)
			}
			existingDeliveries = append(existingDeliveries, d)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, nil, fmt.Errorf("failed to commit: %w", err)
		}
		return &createdN, existingDeliveries, nil
	}

	insertNotificationQuery := `
		INSERT INTO notifications (
			recipient_user_id, society_id, type, title, body, action_url, metadata, reference_type, reference_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, NOW()
		)
		RETURNING id, recipient_user_id, society_id, type, title, body, COALESCE(action_url, ''), metadata, COALESCE(reference_type, ''), reference_id, read_at, created_at
	`

	err = tx.QueryRow(ctx, insertNotificationQuery,
		n.RecipientUserID,
		n.SocietyID,
		n.Type,
		n.Title,
		n.Body,
		n.ActionURL,
		string(metaJSON),
		n.ReferenceType,
		n.ReferenceID,
	).Scan(
		&createdN.ID,
		&createdN.RecipientUserID,
		&createdN.SocietyID,
		&createdN.Type,
		&createdN.Title,
		&createdN.Body,
		&createdN.ActionURL,
		&rawMeta,
		&createdN.ReferenceType,
		&createdN.ReferenceID,
		&createdN.ReadAt,
		&createdN.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to insert notification: %w", err)
	}
	if len(rawMeta) > 0 {
		_ = json.Unmarshal(rawMeta, &createdN.Metadata)
	}

	var createdDeliveries []NotificationDelivery
	insertDeliveryQuery := `
		INSERT INTO notification_deliveries (
			notification_id, channel, target, device_id, status, attempt_count, max_attempts, next_retry_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 0, 3, NOW(), NOW(), NOW()
		)
		RETURNING id, notification_id, channel, target, device_id, status, attempt_count, max_attempts, next_retry_at, provider_message_id, error_code, error_detail, delivered_at, created_at, updated_at
	`

	for _, d := range deliveries {
		var createdD NotificationDelivery
		err := tx.QueryRow(ctx, insertDeliveryQuery,
			createdN.ID,
			d.Channel,
			d.Target,
			d.DeviceID,
			d.Status,
		).Scan(
			&createdD.ID,
			&createdD.NotificationID,
			&createdD.Channel,
			&createdD.Target,
			&createdD.DeviceID,
			&createdD.Status,
			&createdD.AttemptCount,
			&createdD.MaxAttempts,
			&createdD.NextRetryAt,
			&createdD.ProviderMessageID,
			&createdD.ErrorCode,
			&createdD.ErrorDetail,
			&createdD.DeliveredAt,
			&createdD.CreatedAt,
			&createdD.UpdatedAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to insert notification delivery: %w", err)
		}
		createdDeliveries = append(createdDeliveries, createdD)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to commit notification: %w", err)
	}

	return &createdN, createdDeliveries, nil
}

// UpdateDeliveryOutcome updates the delivery status, provider message ID, and sanitized error.
func (r *Repository) UpdateDeliveryOutcome(
	ctx context.Context,
	deliveryID uuid.UUID,
	status string,
	providerMsgID *string,
	errorCode *string,
	errorDetail *string,
) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	query := `
		UPDATE notification_deliveries
		SET status = $2,
		    provider_message_id = COALESCE($3, provider_message_id),
		    error_code = $4,
		    error_detail = $5,
		    delivered_at = CASE WHEN $2 = 'DELIVERED' THEN NOW() ELSE delivered_at END,
		    attempt_count = attempt_count + 1,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, deliveryID, status, providerMsgID, errorCode, errorDetail)
	return err
}
