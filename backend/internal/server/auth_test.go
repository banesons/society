package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/response"
	"chs/backend/internal/sms"
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
}
