package server

import (
	"bytes"
	"context"
	"encoding/json"
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
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func TestSocietyAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// Ensure Super Admin exists
	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, cfg.SuperAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	// Helper to generate a valid JWT with an active session in user_sessions
	generateToken := func(userID uuid.UUID) string {
		sessionID := uuid.New()
		_, err := db.Pool().Exec(context.Background(), `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at, created_at)
			VALUES ($1, $2, 'dummy-hash', NOW() + INTERVAL '1 hour', NOW())
		`, sessionID, userID)
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}

		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		return tok
	}

	superAdminToken := generateToken(superAdminRole.UserID)

	// Create ordinary resident user (no platform role)
	var ordinaryUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO users (phone, status)
		VALUES ('+919999000001', 'ACTIVE')
		ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
		RETURNING id
	`).Scan(&ordinaryUserID)
	if err != nil {
		t.Fatalf("failed to create ordinary user: %v", err)
	}
	ordinaryUserToken := generateToken(ordinaryUserID)

	// Create platform ADMIN (not super admin)
	var adminUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO users (phone, status)
		VALUES ('+919999000002', 'ACTIVE')
		ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
		RETURNING id
	`).Scan(&adminUserID)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status)
		VALUES ($1, 'ADMIN', 'ACTIVE')
		ON CONFLICT (user_id) DO UPDATE SET role = 'ADMIN', status = 'ACTIVE'
	`, adminUserID)
	adminToken := generateToken(adminUserID)

	t.Run("Unauthenticated request to GET /api/v1/platform/societies returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/societies", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Ordinary resident request to GET /api/v1/platform/societies returns 403 FORBIDDEN", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/societies", nil)
		req.Header.Set("Authorization", "Bearer "+ordinaryUserToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("Platform ADMIN request to POST /api/v1/platform/societies returns 403 FORBIDDEN", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":          "Unauthorized Society",
			"address_line1": "123 Test St",
			"locality":      "Test",
			"city":          "Mumbai",
			"state":         "Maharashtra",
			"pincode":       "400001",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	var createdSocietyID string
	var createdSocietyCode string

	t.Run("SUPER_ADMIN creates society successfully via POST /api/v1/platform/societies", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":          "Shree Ram Cooperative Society",
			"address_line1": "Sector 14, Vashi",
			"locality":      "Vashi",
			"city":          "Mumbai",
			"state":         "Maharashtra",
			"pincode":       "400703",
			"contact_email": "shree.ram@society.in",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				ID          string `json:"id"`
				SocietyCode string `json:"society_code"`
				Status      string `json:"status"`
				Name        string `json:"name"`
			} `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Data.Status != "PENDING" {
			t.Errorf("expected initial status PENDING, got %s", resp.Data.Status)
		}
		if resp.Data.Name != "Shree Ram Cooperative Society" {
			t.Errorf("unexpected name: %s", resp.Data.Name)
		}
		if len(resp.Data.SocietyCode) == 0 {
			t.Errorf("expected non-empty society code")
		}

		createdSocietyID = resp.Data.ID
		createdSocietyCode = resp.Data.SocietyCode
	})

	t.Run("SUPER_ADMIN retrieves society details via GET /api/v1/platform/societies/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/societies/"+createdSocietyID, nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				ID          string `json:"id"`
				SocietyCode string `json:"society_code"`
				Status      string `json:"status"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.SocietyCode != createdSocietyCode {
			t.Errorf("expected society code %s, got %s", createdSocietyCode, resp.Data.SocietyCode)
		}
	})

	t.Run("SUPER_ADMIN lists societies with pagination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/societies?limit=10&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Items []interface{} `json:"items"`
				Total int           `json:"total"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.Total == 0 || len(resp.Data.Items) == 0 {
			t.Errorf("expected at least 1 society in list")
		}
	})

	t.Run("SUPER_ADMIN updates society profile via PATCH", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"name":          "Shree Ram Cooperative Housing Society Ltd",
			"address_line1": "Sector 14, Plot 5, Vashi",
			"locality":      "Vashi",
			"city":          "Mumbai",
			"state":         "Maharashtra",
			"pincode":       "400703",
		})
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/platform/societies/"+createdSocietyID, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("SUPER_ADMIN activates society via POST /api/v1/platform/societies/{id}/activate", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason": "Verification completed",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/activate", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.Status != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %s", resp.Data.Status)
		}
	})

	t.Run("GET /api/v1/platform/society-suspension-reasons returns 401 unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/society-suspension-reasons", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v1/platform/society-suspension-reasons returns 403 for non-SUPER_ADMIN", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/society-suspension-reasons", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("SUPER_ADMIN retrieves catalogue via GET /api/v1/platform/society-suspension-reasons", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/society-suspension-reasons", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool                       `json:"success"`
			Data    []society.SuspensionReason `json:"data"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode catalogue response: %v", err)
		}

		if len(resp.Data) != 8 {
			t.Errorf("expected 8 suspension reasons, got %d", len(resp.Data))
		}
		foundPaymentOverdue := false
		for _, r := range resp.Data {
			if r.ID <= 0 {
				t.Errorf("expected positive ID, got %d", r.ID)
			}
			if r.Code == "PAYMENT_OVERDUE" {
				foundPaymentOverdue = true
				if r.Category != "PAYMENT" {
					t.Errorf("expected category PAYMENT, got %s", r.Category)
				}
				if r.ID != 1 {
					t.Errorf("expected ID 1 for PAYMENT_OVERDUE, got %d", r.ID)
				}
			}
		}
		if !foundPaymentOverdue {
			t.Errorf("expected PAYMENT_OVERDUE in catalogue")
		}
	})

	t.Run("SUPER_ADMIN suspending society without reason_code or reason_id returns 400 SUSPENDED_REASON_REQUIRED", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason_details": "No reason provided",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/suspend", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("SUPER_ADMIN suspending society with unknown reason_id returns 400 INVALID_SUSPENSION_REASON", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason_id": 9999,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/suspend", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("SUPER_ADMIN suspending society with unknown reason_code returns 400 INVALID_SUSPENSION_REASON", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason_code": "UNKNOWN_REASON_CODE",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/suspend", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("SUPER_ADMIN suspending with OTHER_REVIEW_REQUIRED without details returns 400 SUSPENDED_DETAILS_REQUIRED", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason_id":      8,
			"reason_details": "   ",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/suspend", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("SUPER_ADMIN suspends society with valid reason_id via POST /api/v1/platform/societies/{id}/suspend", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason_id":      1,
			"reason_details": "Q3 2026 dues unpaid",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/suspend", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.Status != "SUSPENDED" {
			t.Errorf("expected status SUSPENDED, got %s", resp.Data.Status)
		}

		// Verify audit log captured canonical reason_code, numeric reason_id, snapshot label, origin MANUAL, and details
		var auditCount int
		err := db.Pool().QueryRow(context.Background(), `
			SELECT COUNT(*) FROM platform_audit_logs
			WHERE details->>'society_id' = $1
			  AND action = 'SOCIETY_SUSPENDED'
			  AND details->>'reason_id' = '1'
			  AND details->>'reason_code' = 'PAYMENT_OVERDUE'
			  AND details->>'suspension_origin' = 'MANUAL'
			  AND details->>'reason_details' = 'Q3 2026 dues unpaid'
		`, createdSocietyID).Scan(&auditCount)
		if err != nil || auditCount != 1 {
			t.Errorf("expected 1 audit log entry matching details, got count %d, err %v", auditCount, err)
		}
	})

	t.Run("SUPER_ADMIN reactivates society via POST /api/v1/platform/societies/{id}/reactivate", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"reason": "Audit cleared successfully",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+createdSocietyID+"/reactivate", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.Status != "ACTIVE" {
			t.Errorf("expected status ACTIVE, got %s", resp.Data.Status)
		}
	})
}
