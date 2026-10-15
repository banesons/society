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
	"strings"
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

func TestAuditAndAdministrationSuite(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	tm, err := auth.NewTokenManager(cfg.JWTPrivateKey)
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()
	authService := auth.NewService(db.Pool(), tm, smsProvider, rateLimiter, auth.ServiceConfig{}, logger)
	platformRepo := platform.NewRepository(db.Pool())
	societyRepo := society.NewRepository(db.Pool())

	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, cfg.SuperAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	generateToken := func(userID uuid.UUID) string {
		sessionID := uuid.New()
		_, err := db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
			VALUES ($1, $2, $3, 'test-agent', '127.0.0.1', $4)
		`, sessionID, userID, "dummy-hash", time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}
		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return tok
	}

	superToken := generateToken(superAdminRole.UserID)

	// Create test platform ADMIN (not super admin)
	adminPhone := fmt.Sprintf("+91%d", 9820000000+time.Now().UnixNano()%10000000)
	adminRole, err := platformRepo.CreateAdmin(ctx, superAdminRole.UserID, adminPhone, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create test platform admin: %v", err)
	}
	adminToken := generateToken(adminRole.UserID)

	// Create two isolated test societies: Society A and Society B
	createSociety := func(namePrefix string) (uuid.UUID, uuid.UUID, string) {
		socName := fmt.Sprintf("%s %d", namePrefix, time.Now().UnixNano())
		regNo := fmt.Sprintf("REG-AUD-%d", time.Now().UnixNano()%1000000)
		socParams := society.CreateSocietyParams{
			Name:               socName,
			RegistrationNumber: &regNo,
			AddressLine1:       "100 Audit Lane",
			Locality:           "Kalyani Nagar",
			City:               "Pune",
			State:              "Maharashtra",
			Pincode:            "411006",
		}
		socJSON, _ := json.Marshal(socParams)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
		req.Header.Set("Authorization", "Bearer "+superToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("failed to create society: %d %s", w.Code, w.Body.String())
		}
		var socResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &socResp)
		socID := uuid.MustParse(socResp.Data.ID)

		// Activate society
		actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
		actReq.Header.Set("Authorization", "Bearer "+superToken)
		wAct := httptest.NewRecorder()
		router.ServeHTTP(wAct, actReq)

		// Create dedicated SOC_ADMIN user
		saPhone := fmt.Sprintf("+91%d", 9830000000+time.Now().UnixNano()%10000000)
		var saUserID uuid.UUID
		err = db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id
		`, saPhone).Scan(&saUserID)
		if err != nil {
			t.Fatalf("failed to create soc admin user: %v", err)
		}

		// Assign SOC_ADMIN role
		assignBody, _ := json.Marshal(map[string]interface{}{"user_id": saUserID.String()})
		assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader(assignBody))
		assignReq.Header.Set("Authorization", "Bearer "+superToken)
		assignReq.Header.Set("Content-Type", "application/json")
		wAssign := httptest.NewRecorder()
		router.ServeHTTP(wAssign, assignReq)
		if wAssign.Code != http.StatusCreated {
			t.Fatalf("failed to assign soc admin: %d %s", wAssign.Code, wAssign.Body.String())
		}

		return socID, saUserID, generateToken(saUserID)
	}

	socAID, saUserAID, socAToken := createSociety("Audit Soc A")
	socBID, _, _ := createSociety("Audit Soc B")

	t.Run("AC-S14-03 & 04: Platform Audit List, Pagination, Search and Filters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs?limit=5&page=1", nil)
		req.Header.Set("Authorization", "Bearer "+superToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data struct {
				Items      []map[string]interface{} `json:"items"`
				Total      int                      `json:"total"`
				Pagination struct {
					Page       int `json:"page"`
					Limit      int `json:"limit"`
					TotalItems int `json:"total_items"`
					TotalPages int `json:"total_pages"`
				} `json:"pagination"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Data.Items) != 5 {
			t.Errorf("expected 5 items, got %d", len(resp.Data.Items))
		}
		if resp.Data.Pagination.Page != 1 || resp.Data.Pagination.Limit != 5 {
			t.Errorf("expected page 1 limit 5, got %+v", resp.Data.Pagination)
		}
		if resp.Data.Total < 1 {
			t.Errorf("expected total >= 1, got %d", resp.Data.Total)
		}

		// Test action filter
		reqAction := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs?action=SOCIETY_CREATED&limit=3", nil)
		reqAction.Header.Set("Authorization", "Bearer "+superToken)
		wAction := httptest.NewRecorder()
		router.ServeHTTP(wAction, reqAction)
		if wAction.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on action filter, got %d", wAction.Code)
		}

		// Test society filter
		reqSoc := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/platform/audit-logs?society_id=%s", socAID), nil)
		reqSoc.Header.Set("Authorization", "Bearer "+superToken)
		wSoc := httptest.NewRecorder()
		router.ServeHTTP(wSoc, reqSoc)
		if wSoc.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on society filter, got %d", wSoc.Code)
		}
	})

	t.Run("AC-S14-05: Platform Audit Event Detail Endpoint", func(t *testing.T) {
		// First get a log ID from list
		reqList := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs?limit=1", nil)
		reqList.Header.Set("Authorization", "Bearer "+superToken)
		wList := httptest.NewRecorder()
		router.ServeHTTP(wList, reqList)

		var listResp struct {
			Data struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
		if len(listResp.Data.Items) == 0 {
			t.Fatalf("no items returned")
		}
		eventID := listResp.Data.Items[0].ID

		// Get detail
		reqDetail := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs/"+eventID, nil)
		reqDetail.Header.Set("Authorization", "Bearer "+superToken)
		wDetail := httptest.NewRecorder()
		router.ServeHTTP(wDetail, reqDetail)

		if wDetail.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", wDetail.Code, wDetail.Body.String())
		}

		var detailResp struct {
			Data struct {
				Item struct {
					ID          string `json:"id"`
					Action      string `json:"action"`
					ActionLabel string `json:"action_label"`
					ActorName   string `json:"actor_name"`
				} `json:"item"`
			} `json:"data"`
		}
		if err := json.Unmarshal(wDetail.Body.Bytes(), &detailResp); err != nil {
			t.Fatalf("failed to decode detail: %v", err)
		}
		if detailResp.Data.Item.ID != eventID {
			t.Errorf("expected id %s, got %s", eventID, detailResp.Data.Item.ID)
		}
		if detailResp.Data.Item.ActionLabel == "" {
			t.Errorf("expected non-empty action label")
		}
	})

	t.Run("AC-S14-06 & 07: Society Audit History & Strict Tenant Isolation", func(t *testing.T) {
		// Society A admin accessing Society A audit logs -> 200 OK
		reqSocA := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs?limit=10", socAID), nil)
		reqSocA.Header.Set("Authorization", "Bearer "+socAToken)
		wSocA := httptest.NewRecorder()
		router.ServeHTTP(wSocA, reqSocA)

		if wSocA.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for assigned SOC_ADMIN, got %d: %s", wSocA.Code, wSocA.Body.String())
		}

		var socAResp struct {
			Data struct {
				Items []struct {
					ID        string  `json:"id"`
					SocietyID *string `json:"society_id"`
				} `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wSocA.Body.Bytes(), &socAResp)
		for _, it := range socAResp.Data.Items {
			if it.SocietyID != nil && *it.SocietyID != socAID.String() {
				t.Fatalf("leak detected! Event belongs to %s, expected %s", *it.SocietyID, socAID.String())
			}
		}

		// Society A admin accessing Society B audit logs -> 403 Forbidden
		reqSocB := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs", socBID), nil)
		reqSocB.Header.Set("Authorization", "Bearer "+socAToken)
		wSocB := httptest.NewRecorder()
		router.ServeHTTP(wSocB, reqSocB)

		if wSocB.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cross-society audit query, got %d: %s", wSocB.Code, wSocB.Body.String())
		}

		// If Society A has an event, check that Society A admin cannot view it under Society B's path
		if len(socAResp.Data.Items) > 0 {
			eventAID := socAResp.Data.Items[0].ID
			reqCrossDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs/%s", socBID, eventAID), nil)
			reqCrossDetail.Header.Set("Authorization", "Bearer "+superToken) // super admin has access to socB, but event belongs to socA
			wCrossDetail := httptest.NewRecorder()
			router.ServeHTTP(wCrossDetail, reqCrossDetail)

			if wCrossDetail.Code != http.StatusNotFound {
				t.Fatalf("expected 404 Not Found when requesting Society A's event under Society B's path, got %d: %s", wCrossDetail.Code, wCrossDetail.Body.String())
			}
		}
	})

	t.Run("AC-S14-06b: Society Authorization Server-Side Enforcement (Revoked, Unauthorized Roles, Suspended Society)", func(t *testing.T) {
		// 1. Unauthorized user (unauthorized role / no assignment) denied 403
		reqUnauth := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs", socAID), nil)
		reqUnauth.Header.Set("Authorization", "Bearer "+adminToken) // adminToken is platform admin, NOT soc admin
		wUnauth := httptest.NewRecorder()
		router.ServeHTTP(wUnauth, reqUnauth)
		if wUnauth.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for user without SOC_ADMIN role, got %d: %s", wUnauth.Code, wUnauth.Body.String())
		}

		// 2. Client society context mismatch via X-Society-ID header denied 403
		reqMismatch := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs", socAID), nil)
		reqMismatch.Header.Set("Authorization", "Bearer "+socAToken)
		reqMismatch.Header.Set("X-Society-ID", socBID.String()) // Mismatched header
		wMismatch := httptest.NewRecorder()
		router.ServeHTTP(wMismatch, reqMismatch)
		if wMismatch.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for mismatched X-Society-ID header, got %d: %s", wMismatch.Code, wMismatch.Body.String())
		}

		// 3. Revoked assignment denied 403
		// Temporarily revoke socAID admin
		_, err := db.Pool().Exec(ctx, `
			UPDATE society_user_roles
			SET status = 'REVOKED', revoked_by = $3, revoked_at = NOW()
			WHERE society_id = $1 AND user_id = $2
		`, socAID, saUserAID, superAdminRole.UserID)
		if err != nil {
			t.Fatalf("failed to revoke test soc admin: %v", err)
		}

		reqRevoked := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs", socAID), nil)
		reqRevoked.Header.Set("Authorization", "Bearer "+socAToken)
		wRevoked := httptest.NewRecorder()
		router.ServeHTTP(wRevoked, reqRevoked)
		if wRevoked.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for revoked SOC_ADMIN, got %d: %s", wRevoked.Code, wRevoked.Body.String())
		}

		// Restore assignment for subsequent tests
		_, err = db.Pool().Exec(ctx, `
			UPDATE society_user_roles
			SET status = 'ACTIVE', revoked_by = NULL, revoked_at = NULL
			WHERE society_id = $1 AND user_id = $2
		`, socAID, saUserAID)
		if err != nil {
			t.Fatalf("failed to restore test soc admin: %v", err)
		}

		// 4. Suspended society denied 403
		// Temporarily suspend socAID
		_, err = db.Pool().Exec(ctx, `UPDATE societies SET status = 'SUSPENDED' WHERE id = $1`, socAID)
		if err != nil {
			t.Fatalf("failed to suspend test society: %v", err)
		}

		reqSuspended := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs", socAID), nil)
		reqSuspended.Header.Set("Authorization", "Bearer "+socAToken)
		wSuspended := httptest.NewRecorder()
		router.ServeHTTP(wSuspended, reqSuspended)
		if wSuspended.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for suspended society, got %d: %s", wSuspended.Code, wSuspended.Body.String())
		}

		// Restore society status to ACTIVE
		_, err = db.Pool().Exec(ctx, `UPDATE societies SET status = 'ACTIVE' WHERE id = $1`, socAID)
		if err != nil {
			t.Fatalf("failed to restore test society: %v", err)
		}
	})

	t.Run("AC-S14-08: Platform Role Separation", func(t *testing.T) {
		// Platform ADMIN (not Super Admin) must receive 403 on platform audit
		reqAdmin := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs", nil)
		reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
		wAdmin := httptest.NewRecorder()
		router.ServeHTTP(wAdmin, reqAdmin)

		if wAdmin.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for platform ADMIN on audit-logs, got %d: %s", wAdmin.Code, wAdmin.Body.String())
		}

		// Unauthenticated must receive 401
		reqAnon := httptest.NewRequest(http.MethodGet, "/api/v1/platform/audit-logs", nil)
		wAnon := httptest.NewRecorder()
		router.ServeHTTP(wAnon, reqAnon)

		if wAnon.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated caller, got %d", wAnon.Code)
		}
	})

	t.Run("AC-S14-09: Sensitive Metadata Redaction in HTTP Response", func(t *testing.T) {
		// Insert an isolated test audit row with sensitive details
		testEventID := uuid.New()
		sensitiveDetails := map[string]interface{}{
			"society_id":   socAID.String(),
			"title":        "Budget Approval",
			"password":     "supersecret123",
			"access_token": "token_abc_xyz",
			"otp":          "987654",
			"phone":        "+919820000101",
		}
		rawJSON, _ := json.Marshal(sensitiveDetails)

		_, err := db.Pool().Exec(ctx, `
			INSERT INTO platform_audit_logs (id, actor_id, action, society_id, details)
			VALUES ($1, $2, 'BUDGET_APPROVED', $3, $4)
		`, testEventID, superAdminRole.UserID, socAID, string(rawJSON))
		if err != nil {
			t.Fatalf("failed to insert test audit log: %v", err)
		}

		reqDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/audit-logs/%s", socAID, testEventID), nil)
		reqDetail.Header.Set("Authorization", "Bearer "+socAToken)
		wDetail := httptest.NewRecorder()
		router.ServeHTTP(wDetail, reqDetail)

		if wDetail.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", wDetail.Code, wDetail.Body.String())
		}

		bodyStr := wDetail.Body.String()
		if strings.Contains(bodyStr, "supersecret123") {
			t.Errorf("leak detected! Password was not redacted in response")
		}
		if strings.Contains(bodyStr, "token_abc_xyz") {
			t.Errorf("leak detected! Token was not redacted in response")
		}
		if strings.Contains(bodyStr, "987654") {
			t.Errorf("leak detected! OTP was not redacted in response")
		}

		var resp struct {
			Data struct {
				Item struct {
					Details map[string]interface{} `json:"details"`
				} `json:"item"`
			} `json:"data"`
		}
		_ = json.Unmarshal(wDetail.Body.Bytes(), &resp)
		if resp.Data.Item.Details["password"] != "[REDACTED]" {
			t.Errorf("expected password to be [REDACTED], got %v", resp.Data.Item.Details["password"])
		}
		if resp.Data.Item.Details["access_token"] != "[REDACTED]" {
			t.Errorf("expected access_token to be [REDACTED], got %v", resp.Data.Item.Details["access_token"])
		}
		if resp.Data.Item.Details["otp"] != "[REDACTED]" {
			t.Errorf("expected otp to be [REDACTED], got %v", resp.Data.Item.Details["otp"])
		}
		if resp.Data.Item.Details["phone"] != "+91 98*** ***01" {
			t.Errorf("expected phone to be masked as +91 98*** ***01, got %v", resp.Data.Item.Details["phone"])
		}
	})

	t.Run("AC-S14-11: Audit Immutability Trigger Protection", func(t *testing.T) {
		// Pick any row
		var auditID uuid.UUID
		err := db.Pool().QueryRow(ctx, "SELECT id FROM platform_audit_logs LIMIT 1").Scan(&auditID)
		if err != nil {
			t.Fatalf("failed to pick audit log: %v", err)
		}

		// UPDATE attempt must fail with trigger exception
		_, err = db.Pool().Exec(ctx, "UPDATE platform_audit_logs SET action = 'ALTERED' WHERE id = $1", auditID)
		if err == nil {
			t.Errorf("expected error updating platform_audit_logs, got success")
		} else if !strings.Contains(err.Error(), "platform_audit_logs is append-only") {
			t.Errorf("expected append-only trigger message, got %v", err)
		}

		// DELETE attempt must fail with trigger exception
		_, err = db.Pool().Exec(ctx, "DELETE FROM platform_audit_logs WHERE id = $1", auditID)
		if err == nil {
			t.Errorf("expected error deleting from platform_audit_logs, got success")
		} else if !strings.Contains(err.Error(), "platform_audit_logs is append-only") {
			t.Errorf("expected append-only trigger message, got %v", err)
		}
	})
}
