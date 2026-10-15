package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/notification"
	"chs/backend/internal/platform"
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func TestNotificationAPIEndpoints(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	tm, err := auth.NewTokenManager(cfg.JWTPrivateKey)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()
	authService := auth.NewService(db.Pool(), tm, smsProvider, rateLimiter, auth.ServiceConfig{}, logger)
	platformRepo := platform.NewRepository(db.Pool())
	societyRepo := society.NewRepository(db.Pool())
	notifRepo := notification.NewRepository(db.Pool())

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	// Helper to create test user and JWT token
	createUserAndToken := func() (uuid.UUID, string) {
		var uID uuid.UUID
		phone := fmt.Sprintf("919%08d", time.Now().UnixNano()%100000000)
		err := db.Pool().QueryRow(ctx, "INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id", phone).Scan(&uID)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		sessionID := uuid.New()
		_, err = db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, client_ip, user_agent, expires_at)
			VALUES ($1, $2, 'dummy_hash', '127.0.0.1', 'Go-Test', NOW() + INTERVAL '1 hour')
		`, sessionID, uID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		token, err := tm.GenerateAccessToken(uID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return uID, token
	}

	userA, tokenA := createUserAndToken()
	userB, tokenB := createUserAndToken()
	_ = userB

	// 1. Unauthorized access (401 without token)
	req401 := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
	w401 := httptest.NewRecorder()
	router.ServeHTTP(w401, req401)
	if w401.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized without token, got %d", w401.Code)
	}

	// 2. Register FCM Device for User A
	registerPayload := map[string]string{
		"device_id": fmt.Sprintf("dev_chrome_%s", uuid.New().String()[:8]),
		"fcm_token": fmt.Sprintf("fcm_token_%s", uuid.New().String()[:8]),
		"platform":  "WEB",
	}
	body, _ := json.Marshal(registerPayload)
	reqDev := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/devices", bytes.NewReader(body))
	reqDev.Header.Set("Authorization", "Bearer "+tokenA)
	reqDev.Header.Set("Content-Type", "application/json")
	wDev := httptest.NewRecorder()
	router.ServeHTTP(wDev, reqDev)
	if wDev.Code != http.StatusOK {
		t.Fatalf("expected 200 OK registering device, got %d: %s", wDev.Code, wDev.Body.String())
	}

	// 3. List Registered Devices for User A
	reqListDev := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/devices", nil)
	reqListDev.Header.Set("Authorization", "Bearer "+tokenA)
	wListDev := httptest.NewRecorder()
	router.ServeHTTP(wListDev, reqListDev)
	if wListDev.Code != http.StatusOK {
		t.Fatalf("expected 200 listing devices, got %d", wListDev.Code)
	}

	// 4. Create in-app notification directly via repository for User A
	var socID *uuid.UUID
	var sID uuid.UUID
	if err := db.Pool().QueryRow(ctx, "SELECT id FROM societies LIMIT 1").Scan(&sID); err == nil {
		socID = &sID
	}
	createdN, _, err := notifRepo.CreateNotificationAndDeliveries(ctx, notification.Notification{
		RecipientUserID: userA,
		SocietyID:       socID,
		Type:            "TEST_ANNOUNCEMENT",
		Title:           "API Test Notification",
		Body:            "Testing HTTP inbox handlers",
		ActionURL:       "/society/feed",
		Metadata:        map[string]interface{}{"key": "val"},
	}, nil)
	if err != nil {
		t.Fatalf("failed to create test notification: %v", err)
	}

	// 5. Get Unread Count for User A
	reqUnreadA := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	reqUnreadA.Header.Set("Authorization", "Bearer "+tokenA)
	wUnreadA := httptest.NewRecorder()
	router.ServeHTTP(wUnreadA, reqUnreadA)
	if wUnreadA.Code != http.StatusOK {
		t.Fatalf("expected 200 for unread count, got %d: %s", wUnreadA.Code, wUnreadA.Body.String())
	}
	var unreadResp struct {
		Data struct {
			UnreadCount int `json:"unread_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUnreadA.Body.Bytes(), &unreadResp)
	if unreadResp.Data.UnreadCount < 1 {
		t.Errorf("expected unread count >= 1, got %d", unreadResp.Data.UnreadCount)
	}

	// 6. User B Unread Count should be 0 (isolation)
	reqUnreadB := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
	reqUnreadB.Header.Set("Authorization", "Bearer "+tokenB)
	wUnreadB := httptest.NewRecorder()
	router.ServeHTTP(wUnreadB, reqUnreadB)
	if wUnreadB.Code != http.StatusOK {
		t.Fatalf("expected 200 for User B unread count, got %d", wUnreadB.Code)
	}
	var unreadRespB struct {
		Data struct {
			UnreadCount int `json:"unread_count"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUnreadB.Body.Bytes(), &unreadRespB)
	if unreadRespB.Data.UnreadCount != 0 {
		t.Errorf("expected User B unread count 0, got %d", unreadRespB.Data.UnreadCount)
	}

	// 7. List Notifications for User A
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=10&page=1", nil)
	reqList.Header.Set("Authorization", "Bearer "+tokenA)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 listing notifications, got %d: %s", wList.Code, wList.Body.String())
	}

	// 8. IDOR protection: User B tries to mark User A's notification read
	reqIDOR := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", createdN.ID), nil)
	reqIDOR.Header.Set("Authorization", "Bearer "+tokenB)
	wIDOR := httptest.NewRecorder()
	router.ServeHTTP(wIDOR, reqIDOR)
	if wIDOR.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for IDOR attempt, got %d: %s", wIDOR.Code, wIDOR.Body.String())
	}

	// 9. User A marks their own notification read
	reqReadA := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", createdN.ID), nil)
	reqReadA.Header.Set("Authorization", "Bearer "+tokenA)
	wReadA := httptest.NewRecorder()
	router.ServeHTTP(wReadA, reqReadA)
	if wReadA.Code != http.StatusOK {
		t.Fatalf("expected 200 marking notification read, got %d: %s", wReadA.Code, wReadA.Body.String())
	}

	// 10. Mark all read
	reqMarkAll := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/mark-all-read", nil)
	reqMarkAll.Header.Set("Authorization", "Bearer "+tokenA)
	wMarkAll := httptest.NewRecorder()
	router.ServeHTTP(wMarkAll, reqMarkAll)
	if wMarkAll.Code != http.StatusOK {
		t.Fatalf("expected 200 marking all read, got %d: %s", wMarkAll.Code, wMarkAll.Body.String())
	}

	// 11. Deactivate registered device
	reqDeact := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/notifications/devices/%s", registerPayload["device_id"]), nil)
	reqDeact.Header.Set("Authorization", "Bearer "+tokenA)
	wDeact := httptest.NewRecorder()
	router.ServeHTTP(wDeact, reqDeact)
	if wDeact.Code != http.StatusOK {
		t.Fatalf("expected 200 deactivating device, got %d: %s", wDeact.Code, wDeact.Body.String())
	}
}
