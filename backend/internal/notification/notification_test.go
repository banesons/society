package notification_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/notification"

	"github.com/google/uuid"
)

func TestMockProviders(t *testing.T) {
	ctx := context.Background()

	// 1. Email Mock
	emailMock := notification.NewMockEmailProvider()
	msgID, err := emailMock.Send(ctx, notification.EmailMessage{
		To:          "user@example.com",
		Subject:     "Test Subject",
		HTMLBody:    "<p>Hello</p>",
		ReferenceID: "ref-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID == "" {
		t.Errorf("expected non-empty message ID")
	}
	if len(emailMock.SentMessages) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(emailMock.SentMessages))
	}
	if emailMock.SentMessages[0].To != "user@example.com" {
		t.Errorf("expected recipient user@example.com, got %s", emailMock.SentMessages[0].To)
	}

	// 2. Push Mock Success
	pushMock := notification.NewMockPushProvider()
	res := pushMock.Send(ctx, notification.PushMessage{
		Token: "valid_fcm_token_123",
		Title: "Push Title",
		Body:  "Push Body",
	})
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if res.InvalidToken {
		t.Errorf("token should not be marked invalid")
	}
	if res.ProviderMessageID == "" {
		t.Errorf("expected provider message ID")
	}

	// 3. Push Mock Invalid Token
	resInvalid := pushMock.Send(ctx, notification.PushMessage{
		Token: "invalid_fcm_token",
		Title: "Push Title",
		Body:  "Push Body",
	})
	if !resInvalid.InvalidToken {
		t.Errorf("expected invalid token flag")
	}
	if resInvalid.Err == nil {
		t.Errorf("expected error for invalid token")
	}
}

func TestResendEmailProvider_Validation(t *testing.T) {
	ctx := context.Background()

	// Provider unconfigured
	var nilProvider *notification.ResendEmailProvider
	_, err := nilProvider.Send(ctx, notification.EmailMessage{})
	if err != notification.ErrProviderNotConfigured {
		t.Errorf("expected ErrProviderNotConfigured, got %v", err)
	}

	prov := notification.NewResendEmailProvider("re_test_dummy_key", "no-reply@chsneu.dev")
	if prov == nil {
		t.Fatalf("expected initialized provider")
	}

	// Invalid recipient
	_, err = prov.Send(ctx, notification.EmailMessage{
		To: "not-an-email",
	})
	if err != notification.ErrInvalidRecipient {
		t.Errorf("expected ErrInvalidRecipient, got %v", err)
	}
}

func TestFCMPushProvider_Validation(t *testing.T) {
	ctx := context.Background()

	var nilProvider *notification.FCMPushProvider
	res := nilProvider.Send(ctx, notification.PushMessage{})
	if res.Err != notification.ErrProviderNotConfigured {
		t.Errorf("expected ErrProviderNotConfigured, got %v", res.Err)
	}
}

func setupTestDB(t *testing.T) *database.DB {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("skipping database test: DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil || !db.IsConnected() {
		t.Skipf("skipping database test: failed to connect: %v", err)
	}
	return db
}

func createTestUser(t *testing.T, db *database.DB) uuid.UUID {
	var uid uuid.UUID
	phone := fmt.Sprintf("919%08d", time.Now().UnixNano()%100000000)
	err := db.Pool().QueryRow(context.Background(), "INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id", phone).Scan(&uid)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return uid
}

func getTestSocietyID(db *database.DB) *uuid.UUID {
	var socID uuid.UUID
	err := db.Pool().QueryRow(context.Background(), "SELECT id FROM societies LIMIT 1").Scan(&socID)
	if err == nil {
		return &socID
	}
	return nil
}

func TestNotificationRepository_DeviceLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := notification.NewRepository(db.Pool())

	testUserID := createTestUser(t, db)
	deviceToken1 := fmt.Sprintf("test_token_chrome_%s", uuid.New().String()[:8])
	deviceID1 := fmt.Sprintf("device_chrome_%s", uuid.New().String()[:8])

	// 1. Register device
	dev, err := repo.RegisterDevice(ctx, testUserID, notification.RegisterDeviceParams{
		DeviceID: deviceID1,
		FCMToken: deviceToken1,
		Platform: "WEB",
	}, "Chrome 120", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to register device: %v", err)
	}
	if dev.UserID != testUserID {
		t.Errorf("expected user ID %s, got %s", testUserID, dev.UserID)
	}
	if dev.Status != "ACTIVE" {
		t.Errorf("expected device status ACTIVE, got %s", dev.Status)
	}

	// 2. Register second device for same user (multiple devices allowed)
	deviceToken2 := fmt.Sprintf("test_token_mobile_%s", uuid.New().String()[:8])
	deviceID2 := fmt.Sprintf("device_mobile_%s", uuid.New().String()[:8])
	dev2, err := repo.RegisterDevice(ctx, testUserID, notification.RegisterDeviceParams{
		DeviceID: deviceID2,
		FCMToken: deviceToken2,
		Platform: "ANDROID",
	}, "Pixel 8", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to register second device: %v", err)
	}

	// 3. List active devices
	devices, err := repo.ListUserDevices(ctx, testUserID)
	if err != nil {
		t.Fatalf("failed to list devices: %v", err)
	}
	if len(devices) < 2 {
		t.Errorf("expected at least 2 active devices, got %d", len(devices))
	}

	// 4. Update existing device with refreshed token
	newToken := fmt.Sprintf("refreshed_token_%s", uuid.New().String()[:8])
	devUpdated, err := repo.RegisterDevice(ctx, testUserID, notification.RegisterDeviceParams{
		DeviceID: deviceID1,
		FCMToken: newToken,
		Platform: "WEB",
	}, "Chrome 121", "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to refresh device token: %v", err)
	}
	if devUpdated.FCMToken != newToken {
		t.Errorf("expected updated token %s, got %s", newToken, devUpdated.FCMToken)
	}

	// 5. Deactivate single device by device ID
	err = repo.DeactivateDevice(ctx, testUserID, dev2.DeviceID)
	if err != nil {
		t.Fatalf("failed to deactivate device: %v", err)
	}

	// 6. Token feedback invalidation (DeactivateToken)
	err = repo.DeactivateToken(ctx, newToken)
	if err != nil {
		t.Fatalf("failed to deactivate token: %v", err)
	}

	// Verify both devices inactive
	activeDevices, err := repo.ListUserDevices(ctx, testUserID)
	if err != nil {
		t.Fatalf("failed to list devices: %v", err)
	}
	for _, d := range activeDevices {
		if d.ID == dev.ID || d.ID == dev2.ID {
			t.Errorf("device %s should have been deactivated", d.ID)
		}
	}
}

func TestNotificationRepository_InboxAndReadStates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := notification.NewRepository(db.Pool())

	userA := createTestUser(t, db)
	userB := createTestUser(t, db)
	socID := getTestSocietyID(db)

	refID := uuid.New()
	// Create test notification for userA
	notifA := notification.Notification{
		RecipientUserID: userA,
		SocietyID:       socID,
		Type:            "TEST_EVENT",
		Title:           "Important Society Notice",
		Body:            "Water supply maintenance tomorrow",
		ActionURL:       "/society/notice",
		Metadata:        map[string]interface{}{"priority": "HIGH"},
		ReferenceType:   "NOTICE",
		ReferenceID:     &refID,
	}

	deliveries := []notification.NotificationDelivery{
		{
			Channel: notification.ChannelFCM,
			Target:  "fcm_target_userA",
			Status:  notification.DeliveryStatusDelivered,
		},
	}

	created, createdDeliveries, err := repo.CreateNotificationAndDeliveries(ctx, notifA, deliveries)
	if err != nil {
		t.Fatalf("failed to create notification: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatalf("expected valid UUID for created notification")
	}
	if len(createdDeliveries) != 1 {
		t.Fatalf("expected 1 delivery record, got %d", len(createdDeliveries))
	}

	// Idempotency: Re-invoking CreateNotificationAndDeliveries with same reference should return existing notification
	createdIdempotent, deliveriesIdempotent, err := repo.CreateNotificationAndDeliveries(ctx, notifA, deliveries)
	if err != nil {
		t.Fatalf("failed to test idempotency: %v", err)
	}
	if createdIdempotent.ID != created.ID {
		t.Fatalf("expected idempotent reuse of notification ID %s, got %s", created.ID, createdIdempotent.ID)
	}
	if len(deliveriesIdempotent) != len(createdDeliveries) {
		t.Fatalf("expected %d existing deliveries returned, got %d", len(createdDeliveries), len(deliveriesIdempotent))
	}

	// Unread count for userA should be at least 1
	unreadCount, err := repo.GetUnreadCount(ctx, userA, socID)
	if err != nil {
		t.Fatalf("failed to get unread count: %v", err)
	}
	if unreadCount < 1 {
		t.Errorf("expected unread count >= 1, got %d", unreadCount)
	}

	// UserB should NOT see userA's notification (Tenant/User isolation)
	userBCount, err := repo.GetUnreadCount(ctx, userB, socID)
	if err != nil {
		t.Fatalf("failed to get userB unread count: %v", err)
	}
	if userBCount != 0 {
		t.Errorf("expected userB unread count 0, got %d", userBCount)
	}

	// IDOR defense: UserB cannot mark userA's notification read
	_, err = repo.MarkAsRead(ctx, userB, created.ID)
	if err == nil {
		t.Errorf("expected error for IDOR attempt, got nil")
	}

	// UserA marks read
	readAt, err := repo.MarkAsRead(ctx, userA, created.ID)
	if err != nil {
		t.Fatalf("failed to mark notification read: %v", err)
	}
	if readAt == nil {
		t.Errorf("expected readAt to be returned")
	}

	// List notifications for userA
	items, total, err := repo.ListNotifications(ctx, userA, notification.NotificationListFilters{
		SocietyID: socID,
		Limit:     10,
		Page:      1,
	})
	if err != nil {
		t.Fatalf("failed to list notifications: %v", err)
	}
	if total < 1 || len(items) < 1 {
		t.Errorf("expected at least 1 item in inbox, got %d total %d", len(items), total)
	}
	if items[0].ReadAt == nil {
		t.Errorf("expected ReadAt to be set after marking read")
	}

	// Mark all read
	count, err := repo.MarkAllAsRead(ctx, userA, socID)
	if err != nil {
		t.Fatalf("failed to mark all read: %v", err)
	}
	_ = count
}

func TestWorker_OutboxClaimAndIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	repo := notification.NewRepository(db.Pool())

	recipientUID := createTestUser(t, db)
	socID := getTestSocietyID(db)
	var socIDStr string
	if socID != nil {
		socIDStr = socID.String()
	}
	idemKey := fmt.Sprintf("IDEM_TEST_%s", uuid.New().String())

	// 1. Enqueue outbox event
	payload := map[string]interface{}{
		"recipient_user_id": recipientUID.String(),
		"society_id":        socIDStr,
		"title":             "Idempotent Alert",
		"message":           "Testing outbox processing",
	}

	err := repo.EnqueueOutbox(ctx, nil, "SOCIETY_NOTICE", idemKey, payload)
	if err != nil {
		t.Fatalf("failed to enqueue outbox event: %v", err)
	}

	// 2. Enqueue duplicate with same idempotency key (ON CONFLICT DO NOTHING)
	err = repo.EnqueueOutbox(ctx, nil, "SOCIETY_NOTICE", idemKey, payload)
	if err != nil {
		t.Fatalf("duplicate enqueue should not error: %v", err)
	}

	// 3. Worker claims batch
	emailMock := notification.NewMockEmailProvider()
	pushMock := notification.NewMockPushProvider()
	worker := notification.NewWorker(repo, emailMock, pushMock, nil)

	processed := worker.ProcessBatch(ctx, 10)
	if processed < 1 {
		t.Errorf("expected at least 1 processed outbox event, got %d", processed)
	}

	// 4. Verify notification was created in inbox
	items, total, err := repo.ListNotifications(ctx, recipientUID, notification.NotificationListFilters{
		SocietyID: socID,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("failed to list inbox: %v", err)
	}
	if total < 1 || len(items) == 0 {
		t.Errorf("expected notification created in inbox for recipient, got %d", total)
	}
	if items[0].Title != "Society Notice" {
		t.Errorf("expected title 'Society Notice', got '%s'", items[0].Title)
	}
}

func TestLiveFCM_CredentialsInitialization(t *testing.T) {
	cfg := config.Load()
	credsFile := cfg.FirebaseCredentialsFile
	if credsFile == "" {
		t.Skip("FIREBASE_CREDENTIALS_FILE not set")
	}
	projectID := cfg.FirebaseProjectID
	if projectID == "" {
		projectID = "chsneu-2021"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prov, err := notification.NewFCMPushProvider(ctx, projectID, credsFile)
	if err != nil {
		t.Fatalf("failed to initialize FCM provider with credentials: %v", err)
	}

	// Verify live communication with Google FCM HTTP v1 by sending invalid token (should classify as invalid)
	res := prov.Send(ctx, notification.PushMessage{
		Token: "test_dummy_unregistered_token",
		Title: "CHS Verification",
		Body:  "Testing FCM initialization",
	})
	if res.Err == nil {
		t.Logf("FCM accepted message: %s", res.ProviderMessageID)
	} else {
		t.Logf("FCM live response received: invalidToken=%v, err=%v", res.InvalidToken, res.Err)
		if !res.InvalidToken {
			t.Errorf("expected InvalidToken classification from Google FCM, got err: %v", res.Err)
		}
	}
}

func TestCheckExistingRegisteredDevices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	var count int
	err := db.Pool().QueryRow(ctx, "SELECT count(*) FROM user_devices WHERE status = 'ACTIVE'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query user_devices: %v", err)
	}
	t.Logf("Active user_devices count in DB: %d", count)

	rows, err := db.Pool().Query(ctx, "SELECT id, user_id, device_id, platform, status, last_seen_at FROM user_devices WHERE status = 'ACTIVE'")
	if err != nil {
		t.Fatalf("failed to query rows: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, uid, did, plat, stat string
		var lastSeen time.Time
		if err := rows.Scan(&id, &uid, &did, &plat, &stat, &lastSeen); err == nil {
			var token string
			_ = db.Pool().QueryRow(ctx, "SELECT fcm_token FROM user_devices WHERE id = $1", id).Scan(&token)
			isMock := len(token) > 5 && token[:5] == "mock_"
			t.Logf("Device: id=%s, user=%s, device_id=%s, platform=%s, status=%s, is_mock_token=%v, token_len=%d", id, uid, did, plat, stat, isMock, len(token))
		}
	}
}
