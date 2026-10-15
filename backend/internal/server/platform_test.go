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
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/platform"
	"chs/backend/internal/response"
	"chs/backend/internal/sms"
)

func TestPlatformAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping platform API tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil || !db.IsConnected() {
		t.Skip("database not connected; skipping platform API tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tm, err := auth.NewTokenManager(cfg.JWTPrivateKey)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()
	authService := auth.NewService(
		db.Pool(),
		tm,
		smsProvider,
		rateLimiter,
		auth.ServiceConfig{
			DevTestPhones: cfg.DevTestPhones,
			DevTestOTP:    cfg.DevTestOTP,
		},
		logger,
	)

	platformRepo := platform.NewRepository(db.Pool())

	// 1. Bootstrap Super Admin
	superAdminPhone := "919967089101"
	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, superAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, nil)

	// Helper to issue access token for a given user with an active session in user_sessions
	generateToken := func(userID uuid.UUID) string {
		var sessionID uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at)
			VALUES ($1, $2, $3)
			RETURNING id
		`, userID, auth.HashToken(uuid.NewString()), time.Now().Add(auth.RefreshTokenLifespan)).Scan(&sessionID)
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}
		token, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return token
	}

	superAdminToken := generateToken(superAdminRole.UserID)

	// Helper to make authenticated requests
	doRequest := func(method, path, token string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
		var reqBody io.Reader
		if body != nil {
			jsonBytes, _ := json.Marshal(body)
			reqBody = bytes.NewReader(jsonBytes)
		}
		req := httptest.NewRequest(method, path, reqBody)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var resMap map[string]interface{}
		_ = json.NewDecoder(rec.Body).Decode(&resMap)
		return rec, resMap
	}

	// 1. Unauthenticated requests return 401
	t.Run("Unauthenticated request to GET /api/v1/platform/admins returns 401", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/admins", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %v", errObj["code"])
		}
	})

	// 2. Ordinary authenticated user without platform role returns 403
	var ordinaryUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, "INSERT INTO users (phone, status) VALUES ('919999000001', 'ACTIVE') ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE' RETURNING id").Scan(&ordinaryUserID)
	if err != nil {
		t.Fatalf("failed to create ordinary user: %v", err)
	}
	// Ensure ordinary user has NO platform role
	_, _ = db.Pool().Exec(ctx, "DELETE FROM platform_roles WHERE user_id = $1", ordinaryUserID)
	ordinaryToken := generateToken(ordinaryUserID)

	t.Run("Ordinary user request to GET /api/v1/platform/admins returns 403 FORBIDDEN", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/admins", ordinaryToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "FORBIDDEN" {
			t.Errorf("expected FORBIDDEN, got %v", errObj["code"])
		}
	})

	// 3. Active SUPER_ADMIN can list administrators
	t.Run("Active SUPER_ADMIN lists admins successfully", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/admins", superAdminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		items, _ := data["items"].([]interface{})
		if len(items) == 0 {
			t.Errorf("expected at least 1 admin (super admin)")
		}
	})

	// 4. Create new platform ADMIN and verify duplicate conflict
	adminTestPhone := fmt.Sprintf("917%09d", (time.Now().UnixNano()^int64(os.Getpid()))%1000000000)
	var createdAdminRoleID string
	var createdAdminUserID string

	t.Run("Active SUPER_ADMIN creates platform ADMIN", func(t *testing.T) {
		rec, res := doRequest(http.MethodPost, "/api/v1/platform/admins", superAdminToken, map[string]string{
			"phone": adminTestPhone,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d (body: %v)", rec.Code, res)
		}
		data, _ := res["data"].(map[string]interface{})
		if data["role"] != "ADMIN" {
			t.Errorf("expected role ADMIN, got %v", data["role"])
		}
		if data["status"] != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %v", data["status"])
		}
		createdAdminRoleID, _ = data["id"].(string)
		createdAdminUserID, _ = data["user_id"].(string)
	})

	t.Run("Duplicate platform ADMIN creation returns 409 ADMIN_ALREADY_EXISTS", func(t *testing.T) {
		rec, res := doRequest(http.MethodPost, "/api/v1/platform/admins", superAdminToken, map[string]string{
			"phone": adminTestPhone,
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "ADMIN_ALREADY_EXISTS" {
			t.Errorf("expected ADMIN_ALREADY_EXISTS, got %v", errObj["code"])
		}
	})

	// 5. Active ADMIN attempting privileged operations -> 403 FORBIDDEN
	adminUUID, _ := uuid.Parse(createdAdminUserID)
	adminToken := generateToken(adminUUID)

	t.Run("Active ADMIN attempting GET /api/v1/platform/admins returns 403 FORBIDDEN", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/admins", adminToken, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "FORBIDDEN" {
			t.Errorf("expected FORBIDDEN, got %v", errObj["code"])
		}
	})

	// 6. Self-disable protection: Super Admin cannot disable own account
	t.Run("SUPER_ADMIN attempting to disable self returns 400 CANNOT_DISABLE_SELF", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/platform/admins/%s/status", superAdminRole.ID.String())
		rec, res := doRequest(http.MethodPatch, path, superAdminToken, map[string]string{
			"status": "DISABLED",
			"reason": "Test self-disable attempt",
		})
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "CANNOT_DISABLE_SELF" {
			t.Errorf("expected CANNOT_DISABLE_SELF, got %v", errObj["code"])
		}
	})

	// 7. Deactivation and reactivation of ADMIN
	t.Run("Deactivate platform ADMIN successfully", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/platform/admins/%s/status", createdAdminRoleID)
		rec, res := doRequest(http.MethodPatch, path, superAdminToken, map[string]string{
			"status": "DISABLED",
			"reason": "Security compliance review",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		if data["status"] != "DISABLED" {
			t.Errorf("expected status DISABLED, got %v", data["status"])
		}
	})

	// 8. Disabled ADMIN: deactivated session is immediately rejected with 401 SESSION_REVOKED; fresh sessions rejected with 403 ACCOUNT_DISABLED
	t.Run("Disabled ADMIN attempting request is blocked", func(t *testing.T) {
		// Revoked session blocked at authentication layer
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/admins", adminToken, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		errObj, _ := res["error"].(map[string]interface{})
		if errObj["code"] != "SESSION_REVOKED" {
			t.Errorf("expected SESSION_REVOKED, got %v", errObj["code"])
		}

		// A newly created active session for the disabled admin is blocked by authorization with 403 ACCOUNT_DISABLED
		newAdminToken := generateToken(uuid.MustParse(createdAdminUserID))
		rec2, res2 := doRequest(http.MethodGet, "/api/v1/platform/admins", newAdminToken, nil)
		if rec2.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec2.Code)
		}
		errObj2, _ := res2["error"].(map[string]interface{})
		if errObj2["code"] != "ACCOUNT_DISABLED" {
			t.Errorf("expected ACCOUNT_DISABLED, got %v", errObj2["code"])
		}
	})

	// 9. Reactivate platform ADMIN
	t.Run("Reactivate platform ADMIN successfully", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/platform/admins/%s/status", createdAdminRoleID)
		rec, res := doRequest(http.MethodPatch, path, superAdminToken, map[string]string{
			"status": "ACTIVE",
			"reason": "Reinstated after review",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		if data["status"] != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %v", data["status"])
		}
	})

	// 10. List Audit Logs and verify pagination and authorization
	t.Run("Active SUPER_ADMIN lists audit logs with pagination", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/platform/audit-logs?limit=10&offset=0", superAdminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		items, _ := data["items"].([]interface{})
		if len(items) == 0 {
			t.Errorf("expected at least 1 audit log item")
		}
		total, _ := data["total"].(float64)
		if total <= 0 {
			t.Errorf("expected total > 0, got %v", total)
		}
	})

	// 11. GET /api/v1/auth/me returns platform_roles correctly
	t.Run("GET /auth/me returns SUPER_ADMIN role for super admin", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/auth/me", superAdminToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		roles, ok := data["platform_roles"].([]interface{})
		if !ok || len(roles) == 0 || roles[0] != "SUPER_ADMIN" {
			t.Errorf("expected platform_roles to contain SUPER_ADMIN, got %v", data["platform_roles"])
		}
	})

	t.Run("GET /auth/me returns empty platform_roles for ordinary user", func(t *testing.T) {
		rec, res := doRequest(http.MethodGet, "/api/v1/auth/me", ordinaryToken, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		data, _ := res["data"].(map[string]interface{})
		roles, ok := data["platform_roles"].([]interface{})
		if !ok || len(roles) != 0 {
			t.Errorf("expected empty platform_roles, got %v", data["platform_roles"])
		}
	})

	// 12. S01 Regression: Health endpoint continues to function
	t.Run("Health check continues to report healthy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var env response.SuccessEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		dataMap := env.Data.(map[string]interface{})
		if dataMap["status"] != "healthy" {
			t.Errorf("expected status healthy, got %v", dataMap["status"])
		}
	})
}
