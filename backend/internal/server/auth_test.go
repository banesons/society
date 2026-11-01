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
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func TestAuthEndpoints(t *testing.T) {
	cfg := &config.Config{
		Port:           "8080",
		Env:            "test",
		AllowedOrigins: []string{"http://localhost:5173"},
		DevTestPhones:  []string{"919967089101", "918422862001"},
		DevTestOTP:     "123456",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tm, err := auth.NewTokenManager("")
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()

	authService := auth.NewService(
		nil, // no database pool for pure handler validation
		tm,
		smsProvider,
		rateLimiter,
		auth.ServiceConfig{
			DevTestPhones: cfg.DevTestPhones,
			DevTestOTP:    cfg.DevTestOTP,
		},
		logger,
	)

	router := SetupRouter(cfg, logger, nil, authService, tm, nil, nil)

	t.Run("POST /auth/otp/send with valid phone returns otp in dev", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"phone": "9967089101"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/send", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var env response.SuccessEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !env.Success {
			t.Errorf("expected success=true")
		}

		dataMap, ok := env.Data.(map[string]interface{})
		if !ok || dataMap["otp"] != "123456" {
			t.Errorf("expected otp=123456 in dev response, got %v", dataMap["otp"])
		}

		// Verify MockProvider received the test OTP
		lastOTP, exists := smsProvider.GetLastOTP("919967089101")
		if !exists || lastOTP != "123456" {
			t.Errorf("expected MockProvider to have OTP 123456, got %s (exists=%v)", lastOTP, exists)
		}
	})

	t.Run("POST /auth/otp/send triggers cooldown on immediate resend", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"phone": "9967089101"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/send", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status 429, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "OTP_COOLDOWN" {
			t.Errorf("expected error code OTP_COOLDOWN, got %s", env.Error.Code)
		}
	})

	t.Run("POST /auth/otp/send with invalid phone returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"phone": "12345"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/otp/send", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "INVALID_PHONE" {
			t.Errorf("expected error code INVALID_PHONE, got %s", env.Error.Code)
		}
	})

	t.Run("POST /auth/refresh with missing token returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"refresh_token": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("GET /auth/me without authorization returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})

	t.Run("GET /auth/profile without authorization returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})

	t.Run("PATCH /auth/profile without authorization returns 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"first_name": "Test"})
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})

	t.Run("GET /auth/sessions without authorization returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})

	t.Run("DELETE /auth/sessions/{id} without authorization returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/550e8400-e29b-41d4-a716-446655440000", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})
}

func TestActiveSessionsAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live DB test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil || !db.IsConnected() {
		t.Skip("database not connected; skipping")
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

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	// User A
	userAID := uuid.New()
	_, err = db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, '919830000001', 'ACTIVE') ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE' RETURNING id`, userAID)
	if err != nil {
		_ = db.Pool().QueryRow(ctx, `SELECT id FROM users WHERE phone = '919830000001'`).Scan(&userAID)
	}

	// User B
	userBID := uuid.New()
	_, err = db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, '919830000002', 'ACTIVE') ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE' RETURNING id`, userBID)
	if err != nil {
		_ = db.Pool().QueryRow(ctx, `SELECT id FROM users WHERE phone = '919830000002'`).Scan(&userBID)
	}

	// Insert Session A1 (macOS Chrome)
	sessionA1ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at, created_at, last_active_at)
		VALUES ($1, $2, 'hash_a1', 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0 Safari/537.36', '127.0.0.1', NOW() + INTERVAL '1 day', NOW() - INTERVAL '1 hour', NOW())
	`, sessionA1ID, userAID)
	if err != nil {
		t.Fatalf("failed to insert session A1: %v", err)
	}
	defer func() {
		_, _ = db.Pool().Exec(context.Background(), "DELETE FROM user_sessions WHERE user_id IN ($1, $2)", userAID, userBID)
	}()

	// Insert Session A2 (iPhone Safari)
	sessionA2ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at, created_at, last_active_at)
		VALUES ($1, $2, 'hash_a2', 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1 Mobile/15E148 Safari/604.1', '127.0.0.1', NOW() + INTERVAL '1 day', NOW(), NOW())
	`, sessionA2ID, userAID)
	if err != nil {
		t.Fatalf("failed to insert session A2: %v", err)
	}

	// Insert Session B1 for User B
	sessionB1ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at, created_at, last_active_at)
		VALUES ($1, $2, 'hash_b1', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) Edg/120.0.0.0', '127.0.0.1', NOW() + INTERVAL '1 day', NOW(), NOW())
	`, sessionB1ID, userBID)
	if err != nil {
		t.Fatalf("failed to insert session B1: %v", err)
	}

	tokenA2, err := tm.GenerateAccessToken(userAID, sessionA2ID)
	if err != nil {
		t.Fatalf("failed to generate token A2: %v", err)
	}

	// 1. GET /api/v1/auth/sessions returns active sessions only, maximum 2
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	reqList.Header.Set("Authorization", "Bearer "+tokenA2)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", recList.Code, recList.Body.String())
	}

	var resp response.SuccessEnvelope
	if err := json.NewDecoder(recList.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode sessions response: %v", err)
	}

	dataMap, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map in response")
	}

	sessionsRaw, ok := dataMap["sessions"].([]interface{})
	if !ok || len(sessionsRaw) != 2 {
		t.Fatalf("expected exactly 2 active sessions for User A, got %v", dataMap["sessions"])
	}

	// Verify no raw client IP is leaked
	for _, sRaw := range sessionsRaw {
		sMap := sRaw.(map[string]interface{})
		if _, hasIP := sMap["client_ip"]; hasIP {
			t.Errorf("SECURITY DEFECT: raw client_ip leaked in active sessions API response!")
		}
		if _, hasIP := sMap["ip"]; hasIP {
			t.Errorf("SECURITY DEFECT: raw ip leaked in active sessions API response!")
		}
	}

	// 2. User A attempts to revoke User B's session (Cross-user isolation)
	reqCrossRevoke := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/auth/sessions/%s", sessionB1ID), nil)
	reqCrossRevoke.Header.Set("Authorization", "Bearer "+tokenA2)
	recCrossRevoke := httptest.NewRecorder()
	router.ServeHTTP(recCrossRevoke, reqCrossRevoke)

	if recCrossRevoke.Code != http.StatusNotFound {
		t.Errorf("expected 404 NOT_FOUND when revoking another user's session, got %d", recCrossRevoke.Code)
	}

	// 3. User A revokes their own Session A1
	reqRevokeA1 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/auth/sessions/%s", sessionA1ID), nil)
	reqRevokeA1.Header.Set("Authorization", "Bearer "+tokenA2)
	recRevokeA1 := httptest.NewRecorder()
	router.ServeHTTP(recRevokeA1, reqRevokeA1)

	if recRevokeA1.Code != http.StatusOK {
		t.Fatalf("expected 200 OK revoking own session, got %d (body: %s)", recRevokeA1.Code, recRevokeA1.Body.String())
	}

	// 4. GET /api/v1/auth/sessions now returns only 1 active session (Session A2)
	reqListAfter := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	reqListAfter.Header.Set("Authorization", "Bearer "+tokenA2)
	recListAfter := httptest.NewRecorder()
	router.ServeHTTP(recListAfter, reqListAfter)

	var respAfter response.SuccessEnvelope
	_ = json.NewDecoder(recListAfter.Body).Decode(&respAfter)
	dataMapAfter := respAfter.Data.(map[string]interface{})
	sessionsAfter := dataMapAfter["sessions"].([]interface{})
	if len(sessionsAfter) != 1 {
		t.Fatalf("expected 1 active session remaining, got %d", len(sessionsAfter))
	}
	remainingSession := sessionsAfter[0].(map[string]interface{})
	if remainingSession["id"] != sessionA2ID.String() {
		t.Errorf("expected remaining session to be %s, got %v", sessionA2ID, remainingSession["id"])
	}
	if remainingSession["is_current"] != true {
		t.Errorf("expected is_current=true for current session, got %v", remainingSession["is_current"])
	}
}
