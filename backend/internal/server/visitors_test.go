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
	"sync"
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

func TestVisitorOperationsAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping visitor operations API integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, cfg.SuperAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	generateToken := func(userID uuid.UUID) string {
		sessionID := uuid.New()
		_, err := db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
			VALUES ($1, $2, 'test_hash', 'test_agent', '127.0.0.1', $3)
		`, sessionID, userID, time.Now().Add(24*time.Hour))
		if err != nil {
			t.Fatalf("failed to insert session: %v", err)
		}
		token, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		return token
	}

	superToken := generateToken(superAdminRole.UserID)

	// Helper to create an isolated test user
	createTestUser := func(phone string) (uuid.UUID, string) {
		var uID uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, phone).Scan(&uID)
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}
		token := generateToken(uID)
		return uID, token
	}

	// 1. Create primary test society A
	uniqueSuffix := time.Now().UnixNano()
	socAName := fmt.Sprintf("Visitor Test Society %d", uniqueSuffix)
	regNoA := fmt.Sprintf("REG-VIS-%d", uniqueSuffix%1000000)
	socAParams := society.CreateSocietyParams{
		Name:               socAName,
		RegistrationNumber: &regNoA,
		AddressLine1:       "100 Perimeter Way",
		Locality:           "Powai",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400076",
	}
	bodyA, _ := json.Marshal(socAParams)
	reqA := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bodyA))
	reqA.Header.Set("Authorization", "Bearer "+superToken)
	wA := httptest.NewRecorder()
	router.ServeHTTP(wA, reqA)
	if wA.Code != http.StatusCreated {
		t.Fatalf("failed to create society A: %s", wA.Body.String())
	}
	socAData := parseData(wA.Body.Bytes())
	socAIDStr := socAData["id"].(string)
	socAID := uuid.MustParse(socAIDStr)

	// Activate society A
	actReqA := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socAIDStr), nil)
	actReqA.Header.Set("Authorization", "Bearer "+superToken)
	actWA := httptest.NewRecorder()
	router.ServeHTTP(actWA, actReqA)
	if actWA.Code != http.StatusOK {
		t.Fatalf("failed to activate society A: %s", actWA.Body.String())
	}

	// 2. Create secondary test society B (for tenant isolation tests)
	socBName := fmt.Sprintf("Visitor Other Society %d", uniqueSuffix)
	regNoB := fmt.Sprintf("REG-OTH-%d", uniqueSuffix%1000000)
	socBParams := society.CreateSocietyParams{
		Name:               socBName,
		RegistrationNumber: &regNoB,
		AddressLine1:       "200 Foreign Road",
		Locality:           "Kalyani Nagar",
		City:               "Pune",
		State:              "Maharashtra",
		Pincode:            "411006",
	}
	bodyB, _ := json.Marshal(socBParams)
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bodyB))
	reqB.Header.Set("Authorization", "Bearer "+superToken)
	wB := httptest.NewRecorder()
	router.ServeHTTP(wB, reqB)
	socBData := parseData(wB.Body.Bytes())
	socBIDStr := socBData["id"].(string)
	socBID := uuid.MustParse(socBIDStr)

	actReqB := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socBIDStr), nil)
	actReqB.Header.Set("Authorization", "Bearer "+superToken)
	actWB := httptest.NewRecorder()
	router.ServeHTTP(actWB, actReqB)

	// Create building and unit in Society A
	var bldAID, unitAID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, 'Tower 1', 'T1', 5)
		RETURNING id
	`, socAID).Scan(&bldAID)
	if err != nil {
		t.Fatalf("failed to insert building: %v", err)
	}

	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_units (society_id, building_id, floor_number, unit_number, unit_identifier)
		VALUES ($1, $2, 1, '101', 'T1-101')
		RETURNING id
	`, socAID, bldAID).Scan(&unitAID)
	if err != nil {
		t.Fatalf("failed to insert unit: %v", err)
	}

	// Create unit in Society B
	var bldBID, unitBID uuid.UUID
	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, 'Block A', 'BA', 3)
		RETURNING id
	`, socBID).Scan(&bldBID)
	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_units (society_id, building_id, floor_number, unit_number, unit_identifier)
		VALUES ($1, $2, 1, 'B1', 'BA-B1')
		RETURNING id
	`, socBID, bldBID).Scan(&unitBID)

	// Create physical gates in Society A
	var mainGateAID, egressGateAID, inactiveGateAID uuid.UUID
	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status)
		VALUES ($1, 'Main Gate', 'MG1', 'BIDIRECTIONAL', 'ACTIVE', 'ACTIVE')
		RETURNING id
	`, socAID).Scan(&mainGateAID)

	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status)
		VALUES ($1, 'Exit Only Gate', 'EG1', 'EGRESS_ONLY', 'ACTIVE', 'ACTIVE')
		RETURNING id
	`, socAID).Scan(&egressGateAID)

	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status)
		VALUES ($1, 'Maintenance Gate', 'MX1', 'BIDIRECTIONAL', 'MAINTENANCE', 'ACTIVE')
		RETURNING id
	`, socAID).Scan(&inactiveGateAID)

	// Create users:
	// Admin A: SOC_ADMIN of Society A
	adminAUserPhone := fmt.Sprintf("+9198%08d", uniqueSuffix%100000000)
	adminAUserID, adminAToken := createTestUser(adminAUserPhone)
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $3)
	`, socAID, adminAUserID, superAdminRole.UserID)
	if err != nil {
		t.Fatalf("failed to insert admin user role: %v", err)
	}

	// Gatekeeper A: GATEKEEPER of Society A
	guardAUserPhone := fmt.Sprintf("+9197%08d", uniqueSuffix%100000000)
	guardAUserID, guardAToken := createTestUser(guardAUserPhone)
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)
	`, socAID, guardAUserID, superAdminRole.UserID)
	if err != nil {
		t.Fatalf("failed to insert gatekeeper user role: %v", err)
	}

	// Resident A: linked member occupying unit A
	residentAUserPhone := fmt.Sprintf("+9196%08d", uniqueSuffix%100000000)
	residentAUserID, residentAToken := createTestUser(residentAUserPhone)
	var memberAID uuid.UUID
	_ = db.Pool().QueryRow(ctx, `
		INSERT INTO society_members (society_id, user_id, first_name, last_name, phone, status)
		VALUES ($1, $2, 'Raj', 'Malhotra', $3, 'ACTIVE')
		RETURNING id
	`, socAID, residentAUserID, residentAUserPhone).Scan(&memberAID)

	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO unit_relationships (society_id, unit_id, member_id, relationship_type, is_occupant, status)
		VALUES ($1, $2, $3, 'PRIMARY_OWNER', true, 'ACTIVE')
	`, socAID, unitAID, memberAID)

	// Gatekeeper A starts active duty session at Main Gate A
	var sessionAID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_gate_sessions (society_id, gate_id, user_id, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		RETURNING id
	`, socAID, mainGateAID, guardAUserID).Scan(&sessionAID)
	if err != nil {
		t.Fatalf("failed to insert gate duty session: %v", err)
	}

	// Platform ADMIN (unassigned to society)
	platAdminPhone := fmt.Sprintf("+9195%08d", uniqueSuffix%100000000)
	platAdminID, platAdminToken := createTestUser(platAdminPhone)
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status)
		VALUES ($1, 'ADMIN', 'ACTIVE')
	`, platAdminID)

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		childTables := []string{
			"society_gate_events",
			"society_visitor_passes",
			"society_visit_approvals",
			"society_visits",
			"society_gate_sessions",
			"society_gate_shifts",
			"society_staffing_requirements",
			"society_gates",
			"society_committee_appointments",
			"society_designations",
			"unit_relationships",
			"society_units",
			"society_wings",
			"society_buildings",
			"society_members",
			"society_user_roles",
		}
		for _, tbl := range childTables {
			_, _ = db.Pool().Exec(cleanupCtx, fmt.Sprintf("DELETE FROM %s WHERE society_id IN ($1, $2)", tbl), socAID, socBID)
		}
		_, _ = db.Pool().Exec(cleanupCtx, "DELETE FROM societies WHERE id IN ($1, $2)", socAID, socBID)
		_, _ = db.Pool().Exec(cleanupCtx, "DELETE FROM platform_roles WHERE user_id = $1", platAdminID)
		_, _ = db.Pool().Exec(cleanupCtx, "DELETE FROM users WHERE id IN ($1, $2, $3, $4)", adminAUserID, guardAUserID, residentAUserID, platAdminID)
	})

	// TEST SUITE:

	t.Run("A. Authorization & Tenant Isolation Checks", func(t *testing.T) {
		// 1. Unauthenticated request -> 401
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/visitors", socAIDStr), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", w.Code)
		}

		// 2. Platform ADMIN receives 403 Forbidden
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/visitors", socAIDStr), nil)
		req.Header.Set("Authorization", "Bearer "+platAdminToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for platform admin, got %d", w.Code)
		}

		// 3. Gatekeeper A accessing Society B -> 403
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/visitors", socBIDStr), nil)
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for cross-society gatekeeper, got %d", w.Code)
		}

		// 4. Authorized Gatekeeper A accessing Society A -> 200 OK
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/visitors", socAIDStr), nil)
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for assigned gatekeeper, got %d", w.Code)
		}
	})

	t.Run("B. Visit Registration (Expected & Walk-in)", func(t *testing.T) {
		// Initial user count check: Ensure visitors do NOT create records in `users`
		var userCountBefore int
		_ = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCountBefore)

		// 1. Valid Expected Visit created by resident
		expBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":           unitAID.String(),
			"visitor_name":      "Sunil Gavaskar",
			"visitor_phone":     "+919820112233",
			"purpose":           "GUEST",
			"requires_approval": false,
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/expected", socAIDStr), bytes.NewReader(expBody))
		req.Header.Set("Authorization", "Bearer "+residentAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for expected visit, got %d: %s", w.Code, w.Body.String())
		}
		data := parseData(w.Body.Bytes())
		visitData := data["visit"].(map[string]interface{})
		if visitData["status"] != "APPROVED" {
			t.Errorf("expected status APPROVED, got %v", visitData["status"])
		}
		if data["pass"] == nil {
			t.Errorf("expected visitor pass to be issued for pre-approved visit")
		}

		// Check user count: must NOT have increased!
		var userCountAfter int
		_ = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCountAfter)
		if userCountAfter != userCountBefore {
			t.Errorf("visitor registration polluted global users table! before=%d, after=%d", userCountBefore, userCountAfter)
		}

		// 2. Unit from other society rejected (cross-society unit reference)
		crossBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":       unitBID.String(), // belongs to Society B!
			"visitor_name":  "Sneha Patel",
			"visitor_phone": "+919820998877",
			"purpose":       "DELIVERY",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/expected", socAIDStr), bytes.NewReader(crossBody))
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
			t.Errorf("expected 404 or 403 for cross-society unit, got %d: %s", w.Code, w.Body.String())
		}

		// 3. Walk-in visit with default unit policy (approval OFF by default -> APPROVED)
		walkBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":       unitAID.String(),
			"visitor_name":  "Swiggy Delivery Partner",
			"visitor_phone": "+919819445566",
			"purpose":       "DELIVERY",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/walk-in", socAIDStr), bytes.NewReader(walkBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for walk-in visit, got %d: %s", w.Code, w.Body.String())
		}
		walkData := parseData(w.Body.Bytes())["visit"].(map[string]interface{})
		if walkData["status"] != "APPROVED" {
			t.Errorf("expected status APPROVED (default approval OFF), got %v", walkData["status"])
		}
	})

	t.Run("C. Resident Approval and Rejection Flow", func(t *testing.T) {
		// Enable approval on unitAID for testing approval workflow
		_, err := db.Pool().Exec(ctx, `UPDATE society_units SET requires_visitor_approval = true WHERE id = $1`, unitAID)
		if err != nil {
			t.Fatalf("failed to enable approval on unit: %v", err)
		}

		// Register a walk-in requiring approval
		walkBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":       unitAID.String(),
			"visitor_name":  "Plumber Ramesh",
			"visitor_phone": "+919876543210",
			"purpose":       "MAINTENANCE",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/walk-in", socAIDStr), bytes.NewReader(walkBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		visitID := parseData(w.Body.Bytes())["visit"].(map[string]interface{})["id"].(string)

		// 1. Guard attempts to check in BEFORE approval -> 409 Conflict
		checkInBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": mainGateAID.String(),
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(checkInBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for check-in before approval, got %d", w.Code)
		}

		// 2. Resident approves the visit
		appBody, _ := json.Marshal(map[string]interface{}{
			"notes": "Approved by resident for tap repair",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/approve", socAIDStr, visitID), bytes.NewReader(appBody))
		req.Header.Set("Authorization", "Bearer "+residentAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for resident approval, got %d: %s", w.Code, w.Body.String())
		}
		appData := parseData(w.Body.Bytes())["visit"].(map[string]interface{})
		if appData["status"] != "APPROVED" {
			t.Errorf("expected status APPROVED, got %v", appData["status"])
		}

		// 3. Second approval attempt -> 409 Conflict (deterministic idempotency)
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/approve", socAIDStr, visitID), bytes.NewReader(appBody))
		req.Header.Set("Authorization", "Bearer "+residentAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for re-approval, got %d", w.Code)
		}

		// 4. Test Rejection flow on a separate visit
		walkBody2, _ := json.Marshal(map[string]interface{}{
			"unit_id":           unitAID.String(),
			"visitor_name":      "Unsolicited Salesperson",
			"visitor_phone":     "+919811223344",
			"purpose":           "OTHER",
			"requires_approval": true,
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/walk-in", socAIDStr), bytes.NewReader(walkBody2))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		visitID2 := parseData(w.Body.Bytes())["visit"].(map[string]interface{})["id"].(string)

		rejBody, _ := json.Marshal(map[string]interface{}{
			"reason": "Not expecting any salesperson",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/reject", socAIDStr, visitID2), bytes.NewReader(rejBody))
		req.Header.Set("Authorization", "Bearer "+residentAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for rejection, got %d: %s", w.Code, w.Body.String())
		}

		// Check-in on rejected visit must fail
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID2), bytes.NewReader(checkInBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when checking in rejected visit, got %d", w.Code)
		}
	})

	t.Run("D. Pass Verification, Single-Use, and Expiry", func(t *testing.T) {
		// Register and pre-approve visit
		expBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":           unitAID.String(),
			"visitor_name":      "Deepak Verma",
			"visitor_phone":     "+919833445566",
			"purpose":           "GUEST",
			"requires_approval": false,
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/expected", socAIDStr), bytes.NewReader(expBody))
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		data := parseData(w.Body.Bytes())
		visitID := data["visit"].(map[string]interface{})["id"].(string)
		passData := data["pass"].(map[string]interface{})
		passCode := passData["pass_code"].(string)
		bearerToken := passData["bearer_token"].(string)

		// 1. Verify with pass code
		verifyBody, _ := json.Marshal(map[string]interface{}{
			"token_or_code": passCode,
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/passes/verify", socAIDStr), bytes.NewReader(verifyBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for pass verification, got %d: %s", w.Code, w.Body.String())
		}

		// 2. Verify with bearer token
		verifyTokenBody, _ := json.Marshal(map[string]interface{}{
			"token_or_code": bearerToken,
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/passes/verify", socAIDStr), bytes.NewReader(verifyTokenBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for bearer token verification, got %d: %s", w.Code, w.Body.String())
		}

		// 3. Cross-society verification must fail
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/passes/verify", socBIDStr), bytes.NewReader(verifyBody))
		req.Header.Set("Authorization", "Bearer "+superToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-society pass, got %d", w.Code)
		}

		// 4. Complete check-in consumes the pass
		ciBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": mainGateAID.String(),
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(ciBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for check-in, got %d: %s", w.Code, w.Body.String())
		}

		// 5. Subsequent verification of consumed pass -> 409 Conflict
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/passes/verify", socAIDStr), bytes.NewReader(verifyBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for consumed pass replay, got %d", w.Code)
		}
	})

	t.Run("E. Entry, Exit, Gate Constraints, and Concurrency", func(t *testing.T) {
		// Register a pre-approved visit
		expBody, _ := json.Marshal(map[string]interface{}{
			"unit_id":           unitAID.String(),
			"visitor_name":      "Kavita Roy",
			"visitor_phone":     "+919822778899",
			"purpose":           "GUEST",
			"requires_approval": false,
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/expected", socAIDStr), bytes.NewReader(expBody))
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		visitID := parseData(w.Body.Bytes())["visit"].(map[string]interface{})["id"].(string)

		// 1. Check-in at EGRESS_ONLY gate -> 400 Bad Request
		egressBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": egressGateAID.String(),
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(egressBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for check-in at EGRESS_ONLY gate, got %d", w.Code)
		}

		// 2. Check-in at MAINTENANCE gate -> 400 Bad Request
		maintBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": inactiveGateAID.String(),
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(maintBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for check-in at MAINTENANCE gate, got %d", w.Code)
		}

		// 3. CONCURRENCY TEST: Simultaneous check-in attempts on same visit
		validCIBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": mainGateAID.String(),
		})
		concurrency := 6
		results := make([]int, concurrency)
		var wg sync.WaitGroup
		wg.Add(concurrency)

		for i := 0; i < concurrency; i++ {
			go func(idx int) {
				defer wg.Done()
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(validCIBody))
				r.Header.Set("Authorization", "Bearer "+guardAToken)
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, r)
				results[idx] = rec.Code
			}(i)
		}
		wg.Wait()

		successCount := 0
		conflictCount := 0
		for _, code := range results {
			if code == http.StatusOK {
				successCount++
			} else if code == http.StatusConflict {
				conflictCount++
			}
		}

		if successCount != 1 {
			t.Errorf("expected exactly 1 successful check-in under concurrency, got %d (results: %v)", successCount, results)
		}
		if conflictCount != concurrency-1 {
			t.Errorf("expected %d conflicts under concurrency, got %d", concurrency-1, conflictCount)
		}

		// 4. Duplicate check-in on already checked-in visit -> 409
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visitID), bytes.NewReader(validCIBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate check-in, got %d", w.Code)
		}

		// 5. Check-out at main gate -> 200 OK
		coBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": mainGateAID.String(),
			"notes":   "Departed normally",
		})
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-out", socAIDStr, visitID), bytes.NewReader(coBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for checkout, got %d: %s", w.Code, w.Body.String())
		}

		// 6. Duplicate check-out -> 409 Conflict
		req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-out", socAIDStr, visitID), bytes.NewReader(coBody))
		req.Header.Set("Authorization", "Bearer "+guardAToken)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate checkout, got %d", w.Code)
		}
	})

	t.Run("F. Audit Trail Fail-Closed Verification", func(t *testing.T) {
		// Verify audit logs were written for operations in society A
		var auditCount int
		err := db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM platform_audit_logs 
			WHERE details->>'society_id' = $1
		`, socAIDStr).Scan(&auditCount)
		if err != nil {
			t.Fatalf("failed to query audit logs: %v", err)
		}
		if auditCount == 0 {
			t.Errorf("expected audit logs for society A mutations, got 0")
		}

		// Verify audit fail-closed behavior:
		// Attempting mutation with invalid actor ID (violating foreign key constraint on platform_audit_logs.actor_id)
		// must roll back the visitor insert.
		fakeActorID := uuid.New() // does not exist in users table
		params := society.RegisterWalkInVisitParams{
			UnitID:           unitAID,
			VisitorName:      "Test Audit Rollback",
			VisitorPhone:     "+919800000000",
			Purpose:          "DELIVERY",
			RequiresApproval: false,
		}
		_, err = societyRepo.RegisterWalkInVisit(ctx, socAID, params, fakeActorID, "127.0.0.1", "test")
		if err == nil {
			t.Errorf("expected error due to audit FK violation, got nil")
		}

		// Confirm visit was NOT persisted
		var visitPersisted bool
		_ = db.Pool().QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM society_visits 
				WHERE visitor_name = 'Test Audit Rollback'
			)
		`).Scan(&visitPersisted)
		if visitPersisted {
			t.Errorf("transaction failed to roll back on audit log failure! visit was persisted")
		}
	})

	t.Run("G. Resident Unit Policy, Member Access, Cross-Unit Authorization, and Duty Session Enforcement", func(t *testing.T) {
		// Reset unitAID approval setting to default false
		_, _ = db.Pool().Exec(ctx, "UPDATE society_units SET requires_visitor_approval = false WHERE id = $1", unitAID)

		// 1. Create second unit in Society A (Flat 102)
		var unitA2ID uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO society_units (society_id, building_id, floor_number, unit_number, unit_identifier)
			VALUES ($1, $2, 1, '102', 'T1-102')
			RETURNING id
		`, socAID, bldAID).Scan(&unitA2ID)
		if err != nil {
			t.Fatalf("failed to insert second unit: %v", err)
		}

		// 2. Pre-register Member 1 for Flat 101 and Member 2 for Flat 102
		mem1Phone := fmt.Sprintf("+9198200%05d", uniqueSuffix%100000)
		mem2Phone := fmt.Sprintf("+9198201%05d", uniqueSuffix%100000)

		var mem1ID, mem2ID uuid.UUID
		err = db.Pool().QueryRow(ctx, `
			INSERT INTO society_members (society_id, first_name, last_name, phone, status)
			VALUES ($1, 'ResidentOne', 'Sharma', $2, 'ACTIVE')
			RETURNING id
		`, socAID, mem1Phone).Scan(&mem1ID)
		if err != nil {
			t.Fatalf("failed to insert member 1: %v", err)
		}

		err = db.Pool().QueryRow(ctx, `
			INSERT INTO society_members (society_id, first_name, last_name, phone, status)
			VALUES ($1, 'ResidentTwo', 'Verma', $2, 'ACTIVE')
			RETURNING id
		`, socAID, mem2Phone).Scan(&mem2ID)
		if err != nil {
			t.Fatalf("failed to insert member 2: %v", err)
		}

		// Link unit relationships: Member 1 -> Flat 101, Member 2 -> Flat 102
		_, err = db.Pool().Exec(ctx, `
			INSERT INTO unit_relationships (society_id, unit_id, member_id, relationship_type, is_occupant, status, start_date)
			VALUES ($1, $2, $3, 'PRIMARY_OWNER', true, 'ACTIVE', '2026-01-01'),
			       ($1, $4, $5, 'PRIMARY_TENANT', true, 'ACTIVE', '2026-01-01')
		`, socAID, unitAID, mem1ID, unitA2ID, mem2ID)
		if err != nil {
			t.Fatalf("failed to link unit relationships: %v", err)
		}

		// 3. User 1 logs in and links membership
		user1ID, user1Token := createTestUser(mem1Phone)
		err = societyRepo.SafeLinkUserMembers(ctx, user1ID, mem1Phone)
		if err != nil {
			t.Fatalf("failed to link user 1 to member 1: %v", err)
		}

		user2ID, user2Token := createTestUser(mem2Phone)
		err = societyRepo.SafeLinkUserMembers(ctx, user2ID, mem2Phone)
		if err != nil {
			t.Fatalf("failed to link user 2 to member 2: %v", err)
		}

		// 4. Verify /auth/me returns society_assignments with role = 'RESIDENT'
		reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		reqMe.Header.Set("Authorization", "Bearer "+user1Token)
		wMe := httptest.NewRecorder()
		router.ServeHTTP(wMe, reqMe)
		if wMe.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /auth/me, got %d: %s", wMe.Code, wMe.Body.String())
		}
		meData := parseData(wMe.Body.Bytes())
		socAssignments := meData["society_assignments"].([]interface{})
		if len(socAssignments) == 0 {
			t.Fatalf("expected at least 1 society assignment for linked resident, got 0")
		}
		firstAssign := socAssignments[0].(map[string]interface{})
		if firstAssign["role"] != "RESIDENT" {
			t.Errorf("expected assignment role RESIDENT, got %v", firstAssign["role"])
		}

		// 5. User 1 calls GET /units/my -> Flat 101 defaults to requires_visitor_approval = false
		reqMyUnits := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/units/my", socAIDStr), nil)
		reqMyUnits.Header.Set("Authorization", "Bearer "+user1Token)
		wMyUnits := httptest.NewRecorder()
		router.ServeHTTP(wMyUnits, reqMyUnits)
		if wMyUnits.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for /units/my, got %d: %s", wMyUnits.Code, wMyUnits.Body.String())
		}
		myUnitsData := parseData(wMyUnits.Body.Bytes())
		unitsList := myUnitsData["units"].([]interface{})
		if len(unitsList) != 1 {
			t.Fatalf("expected exactly 1 unit for Resident 1, got %d", len(unitsList))
		}
		unit1Data := unitsList[0].(map[string]interface{})
		if unit1Data["requires_visitor_approval"] != false {
			t.Errorf("expected requires_visitor_approval default FALSE, got %v", unit1Data["requires_visitor_approval"])
		}

		// 6. User 1 updates policy for Flat 101 to TRUE
		policyBody, _ := json.Marshal(map[string]interface{}{
			"requires_visitor_approval": true,
		})
		reqPolicy := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/units/%s/visitor-policy", socAIDStr, unitAID), bytes.NewReader(policyBody))
		reqPolicy.Header.Set("Authorization", "Bearer "+user1Token)
		wPolicy := httptest.NewRecorder()
		router.ServeHTTP(wPolicy, reqPolicy)
		if wPolicy.Code != http.StatusOK {
			t.Fatalf("expected 200 OK updating visitor policy, got %d: %s", wPolicy.Code, wPolicy.Body.String())
		}
		updatedUnitData := parseData(wPolicy.Body.Bytes())["unit"].(map[string]interface{})
		if updatedUnitData["requires_visitor_approval"] != true {
			t.Errorf("expected requires_visitor_approval to be true, got %v", updatedUnitData["requires_visitor_approval"])
		}

		// 7. Gatekeeper attempts to modify Flat 101 policy -> 403 Forbidden
		reqGatekeeperPolicy := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/units/%s/visitor-policy", socAIDStr, unitAID), bytes.NewReader(policyBody))
		reqGatekeeperPolicy.Header.Set("Authorization", "Bearer "+guardAToken)
		wGatekeeperPolicy := httptest.NewRecorder()
		router.ServeHTTP(wGatekeeperPolicy, reqGatekeeperPolicy)
		if wGatekeeperPolicy.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when gatekeeper attempts to modify unit policy, got %d", wGatekeeperPolicy.Code)
		}

		// 8. User 2 (occupant of Flat 102) attempts to modify Flat 101 policy -> 403 Forbidden
		reqUser2Policy := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/units/%s/visitor-policy", socAIDStr, unitAID), bytes.NewReader(policyBody))
		reqUser2Policy.Header.Set("Authorization", "Bearer "+user2Token)
		wUser2Policy := httptest.NewRecorder()
		router.ServeHTTP(wUser2Policy, reqUser2Policy)
		if wUser2Policy.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when unauthorized resident attempts to modify another flat's policy, got %d", wUser2Policy.Code)
		}

		// 9. Gatekeeper registers walk-in for Flat 101 -> backend resolves policy=true -> PENDING_APPROVAL
		walkInPayload, _ := json.Marshal(map[string]interface{}{
			"unit_id":       unitAID.String(),
			"visitor_name":  "Courier Person",
			"visitor_phone": "+919876543210",
			"purpose":       "DELIVERY",
		})
		reqWalkIn := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/walk-in", socAIDStr), bytes.NewReader(walkInPayload))
		reqWalkIn.Header.Set("Authorization", "Bearer "+guardAToken)
		wWalkIn := httptest.NewRecorder()
		router.ServeHTTP(wWalkIn, reqWalkIn)
		if wWalkIn.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for walk-in registration, got %d: %s", wWalkIn.Code, wWalkIn.Body.String())
		}
		visit1Data := parseData(wWalkIn.Body.Bytes())["visit"].(map[string]interface{})
		visit1ID := visit1Data["id"].(string)
		if visit1Data["status"] != "PENDING_APPROVAL" {
			t.Errorf("expected status PENDING_APPROVAL because unit policy is true, got %v", visit1Data["status"])
		}

		// 10. Gatekeeper attempts to approve visit1 -> 403 Forbidden
		approvePayload, _ := json.Marshal(map[string]interface{}{"notes": "Gatekeeper trying to approve"})
		reqGatekeeperApprove := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/approve", socAIDStr, visit1ID), bytes.NewReader(approvePayload))
		reqGatekeeperApprove.Header.Set("Authorization", "Bearer "+guardAToken)
		wGatekeeperApprove := httptest.NewRecorder()
		router.ServeHTTP(wGatekeeperApprove, reqGatekeeperApprove)
		if wGatekeeperApprove.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when gatekeeper attempts to approve visit, got %d", wGatekeeperApprove.Code)
		}

		// 11. User 2 (occupant of Flat 102) attempts to approve visit1 (for Flat 101) -> 403 Forbidden
		reqUser2Approve := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/approve", socAIDStr, visit1ID), bytes.NewReader(approvePayload))
		reqUser2Approve.Header.Set("Authorization", "Bearer "+user2Token)
		wUser2Approve := httptest.NewRecorder()
		router.ServeHTTP(wUser2Approve, reqUser2Approve)
		if wUser2Approve.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when unauthorized resident attempts to approve cross-unit visit, got %d", wUser2Approve.Code)
		}

		// 12. User 1 (authorized resident of Flat 101) approves visit1 -> 200 OK -> APPROVED
		reqUser1Approve := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/approve", socAIDStr, visit1ID), bytes.NewReader(approvePayload))
		reqUser1Approve.Header.Set("Authorization", "Bearer "+user1Token)
		wUser1Approve := httptest.NewRecorder()
		router.ServeHTTP(wUser1Approve, reqUser1Approve)
		if wUser1Approve.Code != http.StatusOK {
			t.Fatalf("expected 200 OK when authorized resident approves visit, got %d: %s", wUser1Approve.Code, wUser1Approve.Body.String())
		}
		approvedVisitData := parseData(wUser1Approve.Body.Bytes())["visit"].(map[string]interface{})
		if approvedVisitData["status"] != "APPROVED" {
			t.Errorf("expected status APPROVED, got %v", approvedVisitData["status"])
		}

		// 13. Create Guard 2 (has GATEKEEPER assignment, but NO active duty session)
		guard2Phone := fmt.Sprintf("+9198300%05d", uniqueSuffix%100000)
		guard2ID, guard2Token := createTestUser(guard2Phone)
		_, err = db.Pool().Exec(ctx, `
			INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
			VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)
		`, socAID, guard2ID, superAdminRole.UserID)
		if err != nil {
			t.Fatalf("failed to assign guard 2 role: %v", err)
		}

		// Guard 2 attempts check-in -> 403 DUTY_SESSION_REQUIRED
		checkInBody, _ := json.Marshal(map[string]interface{}{
			"gate_id": mainGateAID.String(),
		})
		reqGuard2CheckIn := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visit1ID), bytes.NewReader(checkInBody))
		reqGuard2CheckIn.Header.Set("Authorization", "Bearer "+guard2Token)
		wGuard2CheckIn := httptest.NewRecorder()
		router.ServeHTTP(wGuard2CheckIn, reqGuard2CheckIn)
		if wGuard2CheckIn.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for guard without active duty session, got %d: %s", wGuard2CheckIn.Code, wGuard2CheckIn.Body.String())
		}

		// 14. Guard 1 (with active duty session) checks in visit1 -> 200 OK -> CHECKED_IN
		reqGuard1CheckIn := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visit1ID), bytes.NewReader(checkInBody))
		reqGuard1CheckIn.Header.Set("Authorization", "Bearer "+guardAToken)
		wGuard1CheckIn := httptest.NewRecorder()
		router.ServeHTTP(wGuard1CheckIn, reqGuard1CheckIn)
		if wGuard1CheckIn.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for guard with active duty session, got %d: %s", wGuard1CheckIn.Code, wGuard1CheckIn.Body.String())
		}
		checkedInVisitData := parseData(wGuard1CheckIn.Body.Bytes())["visit"].(map[string]interface{})
		if checkedInVisitData["status"] != "CHECKED_IN" {
			t.Errorf("expected status CHECKED_IN, got %v", checkedInVisitData["status"])
		}

		// 15. Rejection flow: register second visit for Flat 101 -> PENDING_APPROVAL
		walkInPayload2, _ := json.Marshal(map[string]interface{}{
			"unit_id":       unitAID.String(),
			"visitor_name":  "Unwanted Salesperson",
			"visitor_phone": "+919876543211",
			"purpose":       "OTHER",
		})
		reqWalkIn2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/walk-in", socAIDStr), bytes.NewReader(walkInPayload2))
		reqWalkIn2.Header.Set("Authorization", "Bearer "+guardAToken)
		wWalkIn2 := httptest.NewRecorder()
		router.ServeHTTP(wWalkIn2, reqWalkIn2)
		visit2ID := parseData(wWalkIn2.Body.Bytes())["visit"].(map[string]interface{})["id"].(string)

		// User 1 rejects visit2 -> 200 OK -> REJECTED
		rejectPayload, _ := json.Marshal(map[string]interface{}{"reason": "Solicitation not allowed"})
		reqReject := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/reject", socAIDStr, visit2ID), bytes.NewReader(rejectPayload))
		reqReject.Header.Set("Authorization", "Bearer "+user1Token)
		wReject := httptest.NewRecorder()
		router.ServeHTTP(wReject, reqReject)
		if wReject.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for rejection, got %d: %s", wReject.Code, wReject.Body.String())
		}

		// Guard 1 attempts check-in on rejected visit -> 409 Conflict
		reqCheckInRejected := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/check-in", socAIDStr, visit2ID), bytes.NewReader(checkInBody))
		reqCheckInRejected.Header.Set("Authorization", "Bearer "+guardAToken)
		wCheckInRejected := httptest.NewRecorder()
		router.ServeHTTP(wCheckInRejected, reqCheckInRejected)
		if wCheckInRejected.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict attempting to check in rejected visit, got %d", wCheckInRejected.Code)
		}
	})
}
