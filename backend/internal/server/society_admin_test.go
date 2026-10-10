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
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func TestSocietyAdminAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

	// Helper to create test user
	createTestUser := func(phone string) (uuid.UUID, string) {
		var uid uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, phone).Scan(&uid)
		if err != nil {
			t.Fatalf("failed to create test user %s: %v", phone, err)
		}
		tok := generateToken(uid)
		return uid, tok
	}

	// Create test users
	targetUserPhone1 := "+919820555001"
	_, targetUserToken1 := createTestUser(targetUserPhone1)

	targetUserPhone2 := "+919820555002"
	targetUserID2, targetUserToken2 := createTestUser(targetUserPhone2)

	// Platform ADMIN (not super admin)
	platformAdminPhone := "+919820555003"
	platformAdminID, platformAdminToken := createTestUser(platformAdminPhone)
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status, assigned_by)
		VALUES ($1, 'ADMIN', 'ACTIVE', $2)
		ON CONFLICT (user_id) DO UPDATE SET role = 'ADMIN', status = 'ACTIVE'
	`, platformAdminID, superAdminRole.UserID)

	// Helper to create test society
	createSociety := func(name, city string) *society.Society {
		soc, err := societyRepo.CreateSociety(ctx, society.CreateSocietyParams{
			Name:         name,
			AddressLine1: "123 Main Road",
			Locality:     "Bandra West",
			City:         city,
			State:        "Maharashtra",
			Pincode:      "400050",
		}, superAdminRole.UserID, "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to create test society: %v", err)
		}
		return soc
	}

	t.Run("AC-01: Authorization - only SUPER_ADMIN can assign society admins", func(t *testing.T) {
		soc := createSociety("AC01 Heights Society", "Mumbai")

		body := map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		}
		bodyBytes, _ := json.Marshal(body)

		// 1. Unauthenticated request -> 401
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated request, got %d: %s", w.Code, w.Body.String())
		}

		// 2. Platform ADMIN caller -> 403
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+platformAdminToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for platform ADMIN caller, got %d: %s", w.Code, w.Body.String())
		}

		// 3. Ordinary user caller -> 403
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+targetUserToken1)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for ordinary user caller, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("AC-13: Pre-assignment - SUPER_ADMIN assigns to PENDING society, operational access blocked until ACTIVE", func(t *testing.T) {
		soc := createSociety("AC13 PreAssign Society", "Mumbai")
		// Society starts in PENDING status

		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 for pre-assignment to PENDING society, got %d: %s", w.Code, w.Body.String())
		}

		// Operational profile request to PENDING society -> 403 SOCIETY_PENDING
		reqProfile := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqProfile.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wProfile := httptest.NewRecorder()
		router.ServeHTTP(wProfile, reqProfile)
		if wProfile.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for operational request to PENDING society, got %d: %s", wProfile.Code, wProfile.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(wProfile.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		if errObj["code"] != "SOCIETY_PENDING" {
			t.Fatalf("expected error code SOCIETY_PENDING, got %v", errObj["code"])
		}

		// Activate society
		_, err := societyRepo.ActivateSociety(ctx, soc.ID, "Onboarding verification complete", superAdminRole.UserID, "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to activate society: %v", err)
		}

		// Now operational request succeeds
		reqProfileActive := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqProfileActive.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wProfileActive := httptest.NewRecorder()
		router.ServeHTTP(wProfileActive, reqProfileActive)
		if wProfileActive.Code != http.StatusOK {
			t.Fatalf("expected 200 for operational request after activation, got %d: %s", wProfileActive.Code, wProfileActive.Body.String())
		}
	})

	t.Run("AC-02 & AC-04: Explicit assignment & Least privilege - Unassigned user & Platform ADMIN denied", func(t *testing.T) {
		soc := createSociety("AC02 LeastPrivilege Society", "Mumbai")
		_, err := societyRepo.ActivateSociety(ctx, soc.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to activate society: %v", err)
		}

		// Platform ADMIN has no assignment -> 403 SOCIETY_ACCESS_DENIED
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		req.Header.Set("Authorization", "Bearer "+platformAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for unassigned platform admin, got %d: %s", w.Code, w.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		if errObj["code"] != "SOCIETY_ACCESS_DENIED" {
			t.Fatalf("expected error code SOCIETY_ACCESS_DENIED, got %v", errObj["code"])
		}
	})

	t.Run("AC-03: Role Segregation - Assigning SOC_ADMIN does not modify platform_roles", func(t *testing.T) {
		soc := createSociety("AC03 Segregation Society", "Mumbai")
		_, _ = societyRepo.ActivateSociety(ctx, soc.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")

		// Assign targetUser2 as SOC_ADMIN
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone2,
			"role":  "SOC_ADMIN",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// Verify targetUser2 still has no platform_roles in database
		roles, err := platformRepo.GetUserPlatformRoles(ctx, targetUserID2)
		if err != nil {
			t.Fatalf("failed to query platform roles: %v", err)
		}
		if len(roles) != 0 {
			t.Fatalf("expected target user to have zero platform roles, got %v", roles)
		}

		// Verify targetUser2 cannot access /api/v1/platform/* (AC-09)
		reqPlat := httptest.NewRequest(http.MethodGet, "/api/v1/platform/admins", nil)
		reqPlat.Header.Set("Authorization", "Bearer "+targetUserToken2)
		wPlat := httptest.NewRecorder()
		router.ServeHTTP(wPlat, reqPlat)
		if wPlat.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for SOC_ADMIN accessing platform endpoint, got %d: %s", wPlat.Code, wPlat.Body.String())
		}
	})

	t.Run("Assignment Constraints: Unknown phone, duplicate active assignment conflict, and masking", func(t *testing.T) {
		soc := createSociety("Constraint Society", "Mumbai")

		// 1. Unknown phone -> 404 USER_NOT_FOUND
		bodyUnknown, _ := json.Marshal(map[string]interface{}{
			"phone": "+919999999999",
			"role":  "SOC_ADMIN",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyUnknown))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for unregistered user phone, got %d: %s", w.Code, w.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		if errObj["code"] != "USER_NOT_FOUND" {
			t.Fatalf("expected USER_NOT_FOUND, got %v", errObj["code"])
		}

		// 2. First assignment -> 201 Created with masked phone
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}
		var assignResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &assignResp)
		data, _ := assignResp["data"].(map[string]interface{})
		if data["user_phone_masked"] != auth.MaskPhone(targetUserPhone1) {
			t.Fatalf("expected masked phone %s, got %v", auth.MaskPhone(targetUserPhone1), data["user_phone_masked"])
		}

		// 3. Duplicate active assignment -> 409 SOCIETY_ADMIN_ALREADY_ASSIGNED
		reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		reqDup.Header.Set("Authorization", "Bearer "+superAdminToken)
		wDup := httptest.NewRecorder()
		router.ServeHTTP(wDup, reqDup)
		if wDup.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for duplicate active assignment, got %d: %s", wDup.Code, wDup.Body.String())
		}
		var dupErrResp map[string]interface{}
		_ = json.Unmarshal(wDup.Body.Bytes(), &dupErrResp)
		dupErrObj, _ := dupErrResp["error"].(map[string]interface{})
		if dupErrObj["code"] != "SOCIETY_ADMIN_ALREADY_ASSIGNED" {
			t.Fatalf("expected code SOCIETY_ADMIN_ALREADY_ASSIGNED, got %v", dupErrObj["code"])
		}
	})

	t.Run("AC-07: Immediate Revocation and Historical Reassignment", func(t *testing.T) {
		soc := createSociety("Revocation Society", "Mumbai")
		_, _ = societyRepo.ActivateSociety(ctx, soc.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")

		// 1. Assign targetUser1
		bodyBytes, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var assignResp map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &assignResp)
		data, _ := assignResp["data"].(map[string]interface{})
		assignmentID := data["id"].(string)

		// 2. Can access profile
		reqProfile := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqProfile.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wProfile := httptest.NewRecorder()
		router.ServeHTTP(wProfile, reqProfile)
		if wProfile.Code != http.StatusOK {
			t.Fatalf("expected 200 before revocation, got %d: %s", wProfile.Code, wProfile.Body.String())
		}

		// 3. Super Admin revokes assignment
		reqRevoke := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/platform/societies/%s/admins/%s", soc.ID, assignmentID), nil)
		reqRevoke.Header.Set("Authorization", "Bearer "+superAdminToken)
		wRevoke := httptest.NewRecorder()
		router.ServeHTTP(wRevoke, reqRevoke)
		if wRevoke.Code != http.StatusOK {
			t.Fatalf("expected 200 for revocation, got %d: %s", wRevoke.Code, wRevoke.Body.String())
		}

		// 4. Immediately denied on subsequent call with existing JWT (no token expiry needed)
		reqProfileAfter := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqProfileAfter.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wProfileAfter := httptest.NewRecorder()
		router.ServeHTTP(wProfileAfter, reqProfileAfter)
		if wProfileAfter.Code != http.StatusForbidden {
			t.Fatalf("expected 403 immediately after revocation, got %d: %s", wProfileAfter.Code, wProfileAfter.Body.String())
		}

		// 5. Re-assigning targetUser1 creates a new active assignment record without conflict
		reqReassign := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(bodyBytes))
		reqReassign.Header.Set("Authorization", "Bearer "+superAdminToken)
		wReassign := httptest.NewRecorder()
		router.ServeHTTP(wReassign, reqReassign)
		if wReassign.Code != http.StatusCreated {
			t.Fatalf("expected 201 for re-assignment after revocation, got %d: %s", wReassign.Code, wReassign.Body.String())
		}

		// 6. List admins with history retains both REVOKED and ACTIVE records
		reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), nil)
		reqList.Header.Set("Authorization", "Bearer "+superAdminToken)
		wList := httptest.NewRecorder()
		router.ServeHTTP(wList, reqList)
		if wList.Code != http.StatusOK {
			t.Fatalf("expected 200 for list admins, got %d: %s", wList.Code, wList.Body.String())
		}
		var listResp map[string]interface{}
		_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
		listData, _ := listResp["data"].(map[string]interface{})
		total := int(listData["total"].(float64))
		if total < 2 {
			t.Fatalf("expected at least 2 historical assignments (1 revoked, 1 active), got %d", total)
		}
	})

	t.Run("AC-05: Multi-Society Support and auth/me Discovery", func(t *testing.T) {
		soc1 := createSociety("Multi Society Alpha", "Mumbai")
		soc2 := createSociety("Multi Society Beta", "Pune")
		_, _ = societyRepo.ActivateSociety(ctx, soc1.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")
		_, _ = societyRepo.ActivateSociety(ctx, soc2.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")

		// Assign targetUser2 to both soc1 and soc2
		assignToSoc := func(socID uuid.UUID) {
			b, _ := json.Marshal(map[string]interface{}{
				"phone": targetUserPhone2,
				"role":  "SOC_ADMIN",
			})
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+superAdminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("failed to assign to society %s, got %d: %s", socID, w.Code, w.Body.String())
			}
		}
		assignToSoc(soc1.ID)
		assignToSoc(soc2.ID)

		// Target user 2 can access profile in soc1
		req1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc1.ID), nil)
		req1.Header.Set("Authorization", "Bearer "+targetUserToken2)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("expected 200 for soc1 profile, got %d: %s", w1.Code, w1.Body.String())
		}

		// Target user 2 can access profile in soc2
		req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc2.ID), nil)
		req2.Header.Set("Authorization", "Bearer "+targetUserToken2)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("expected 200 for soc2 profile, got %d: %s", w2.Code, w2.Body.String())
		}

		// Check GET /api/v1/auth/me for targetUser2 includes both societies in society_assignments
		reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqMe.Header.Set("Authorization", "Bearer "+targetUserToken2)
		wMe := httptest.NewRecorder()
		router.ServeHTTP(wMe, reqMe)
		if wMe.Code != http.StatusOK {
			t.Fatalf("expected 200 for /auth/me, got %d: %s", wMe.Code, wMe.Body.String())
		}
		var meResp map[string]interface{}
		_ = json.Unmarshal(wMe.Body.Bytes(), &meResp)
		meData, _ := meResp["data"].(map[string]interface{})
		assignments, ok := meData["society_assignments"].([]interface{})
		if !ok || len(assignments) < 2 {
			t.Fatalf("expected at least 2 society_assignments in /auth/me, got %v", meData["society_assignments"])
		}
	})

	t.Run("AC-08: Client Context Forgery - Header mismatch rejected", func(t *testing.T) {
		soc := createSociety("Context Forgery Society", "Mumbai")
		_, _ = societyRepo.ActivateSociety(ctx, soc.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")

		// Assign targetUser1
		b, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		reqAssign := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(b))
		reqAssign.Header.Set("Authorization", "Bearer "+superAdminToken)
		wAssign := httptest.NewRecorder()
		router.ServeHTTP(wAssign, reqAssign)
		if wAssign.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", wAssign.Code)
		}

		// Provide conflicting X-Society-ID header
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		req.Header.Set("Authorization", "Bearer "+targetUserToken1)
		req.Header.Set("X-Society-ID", uuid.New().String()) // Forged header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for mismatched X-Society-ID header, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("AC-10 & AC-11: Society Suspension & Reactivation Decoupling", func(t *testing.T) {
		soc := createSociety("Suspension Decoupling Society", "Mumbai")
		_, _ = societyRepo.ActivateSociety(ctx, soc.ID, "Activate", superAdminRole.UserID, "127.0.0.1", "test-agent")

		// Assign targetUser1
		b, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		reqAssign := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(b))
		reqAssign.Header.Set("Authorization", "Bearer "+superAdminToken)
		wAssign := httptest.NewRecorder()
		router.ServeHTTP(wAssign, reqAssign)
		if wAssign.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", wAssign.Code)
		}

		// Suspend society
		suspendParams := society.SuspendSocietyParams{
			ReasonCode: new(string),
		}
		*suspendParams.ReasonCode = "PAYMENT_OVERDUE"
		_, err := societyRepo.SuspendSociety(ctx, soc.ID, suspendParams, superAdminRole.UserID, "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to suspend society: %v", err)
		}

		// AC-10: Operational request now rejected with 403 SOCIETY_SUSPENDED
		reqProfile := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqProfile.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wProfile := httptest.NewRecorder()
		router.ServeHTTP(wProfile, reqProfile)
		if wProfile.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for suspended society, got %d: %s", wProfile.Code, wProfile.Body.String())
		}
		var errResp map[string]interface{}
		_ = json.Unmarshal(wProfile.Body.Bytes(), &errResp)
		errObj, _ := errResp["error"].(map[string]interface{})
		if errObj["code"] != "SOCIETY_SUSPENDED" {
			t.Fatalf("expected SOCIETY_SUSPENDED, got %v", errObj["code"])
		}

		// Super Admin reactivates society
		_, err = societyRepo.ReactivateSociety(ctx, soc.ID, "Payment received in full", superAdminRole.UserID, "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("failed to reactivate society: %v", err)
		}

		// AC-11: Access restored without recreating assignments
		reqRestored := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/profile", soc.ID), nil)
		reqRestored.Header.Set("Authorization", "Bearer "+targetUserToken1)
		wRestored := httptest.NewRecorder()
		router.ServeHTTP(wRestored, reqRestored)
		if wRestored.Code != http.StatusOK {
			t.Fatalf("expected 200 after reactivation, got %d: %s", wRestored.Code, wRestored.Body.String())
		}
	})

	t.Run("AC-06: Audit Trail - Assignment creation and revocation emit transactional audit logs", func(t *testing.T) {
		soc := createSociety("Audit Society", "Mumbai")

		// Assign
		b, _ := json.Marshal(map[string]interface{}{
			"phone": targetUserPhone1,
			"role":  "SOC_ADMIN",
		})
		reqAssign := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", soc.ID), bytes.NewReader(b))
		reqAssign.Header.Set("Authorization", "Bearer "+superAdminToken)
		wAssign := httptest.NewRecorder()
		router.ServeHTTP(wAssign, reqAssign)
		if wAssign.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", wAssign.Code)
		}

		var assignResp map[string]interface{}
		_ = json.Unmarshal(wAssign.Body.Bytes(), &assignResp)
		data, _ := assignResp["data"].(map[string]interface{})
		assignmentID := data["id"].(string)

		// Verify audit log for SOCIETY_ADMIN_ASSIGNED
		var assignedCount int
		err := db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM platform_audit_logs
			WHERE action = 'SOCIETY_ADMIN_ASSIGNED'
			  AND (details->>'assignment_id') = $1
		`, assignmentID).Scan(&assignedCount)
		if err != nil {
			t.Fatalf("failed to query audit log for assignment: %v", err)
		}
		if assignedCount != 1 {
			t.Fatalf("expected 1 audit log record for SOCIETY_ADMIN_ASSIGNED, got %d", assignedCount)
		}

		// Revoke
		reqRevoke := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/platform/societies/%s/admins/%s", soc.ID, assignmentID), nil)
		reqRevoke.Header.Set("Authorization", "Bearer "+superAdminToken)
		wRevoke := httptest.NewRecorder()
		router.ServeHTTP(wRevoke, reqRevoke)
		if wRevoke.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", wRevoke.Code)
		}

		// Verify audit log for SOCIETY_ADMIN_REVOKED
		var revokedCount int
		err = db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM platform_audit_logs
			WHERE action = 'SOCIETY_ADMIN_REVOKED'
			  AND (details->>'assignment_id') = $1
		`, assignmentID).Scan(&revokedCount)
		if err != nil {
			t.Fatalf("failed to query audit log for revocation: %v", err)
		}
		if revokedCount != 1 {
			t.Fatalf("expected 1 audit log record for SOCIETY_ADMIN_REVOKED, got %d", revokedCount)
		}
	})
}
