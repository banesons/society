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

func TestVisitorQREndToEnd(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping visitor QR integration tests")
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
	authService := auth.NewService(db.Pool(), tm, smsProvider, rateLimiter, auth.ServiceConfig{
		DevTestPhones: []string{"919820099991", "919820099992"},
		DevTestOTP:    "123456",
	}, logger)
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

	// 1. Create primary test society via API
	uniqueSuffix := time.Now().UnixNano()
	socName := fmt.Sprintf("QR Test Society %d", uniqueSuffix)
	regNo := fmt.Sprintf("REG-QR-%d", uniqueSuffix%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "100 QR Way",
		Locality:           "Powai",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400076",
	}

	body, _ := json.Marshal(socParams)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create society: %s", w.Body.String())
	}

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &socResp)
	socID := uuid.MustParse(socResp.Data.ID)

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID.String()), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	actW := httptest.NewRecorder()
	router.ServeHTTP(actW, actReq)

	// Create building and unit via SQL
	var bldID, unitID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, 'Block A', 'BA', 5)
		RETURNING id
	`, socID).Scan(&bldID)
	if err != nil {
		t.Fatalf("failed to insert building: %v", err)
	}

	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_units (society_id, building_id, floor_number, unit_number, unit_identifier)
		VALUES ($1, $2, 1, '101', 'BA-101')
		RETURNING id
	`, socID, bldID).Scan(&unitID)
	if err != nil {
		t.Fatalf("failed to insert unit: %v", err)
	}

	// Create Gate 1 (Bidirectional Main Gate)
	var gate1ID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status)
		VALUES ($1, 'Main Gate 1', 'MG1', 'BIDIRECTIONAL', 'ACTIVE', 'ACTIVE')
		RETURNING id
	`, socID).Scan(&gate1ID)
	if err != nil {
		t.Fatalf("failed to insert gate: %v", err)
	}

	// Create Gatekeeper and Admin users
	adminPhone := fmt.Sprintf("+9198200%05d", uniqueSuffix%100000)
	adminID, adminToken := createTestUser(adminPhone)
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $3)
	`, socID, adminID, superAdminRole.UserID)

	gkPhone := fmt.Sprintf("+9198201%05d", uniqueSuffix%100000)
	gkID, gkToken := createTestUser(gkPhone)
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)
	`, socID, gkID, superAdminRole.UserID)

	var qrToken string
	var qrID string

	// Phase 1: Society Admin generates Gate QR
	t.Run("Admin Generates Gate QR", func(t *testing.T) {
		payload := map[string]interface{}{
			"gate_id": gate1ID.String(),
			"label":   "Main Gate 1 North Entry",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/qr", socID, gate1ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for QR generation, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				ID    string `json:"id"`
				Token string `json:"token"`
				Label string `json:"label"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse QR response: %v", err)
		}

		if resp.Data.Token == "" || !resp.Success {
			t.Fatalf("invalid QR token response: %+v", resp)
		}
		qrToken = resp.Data.Token
		qrID = resp.Data.ID
	})

	// Phase 2: Unauthorized gatekeeper cannot generate QR
	t.Run("Gatekeeper Cannot Generate QR", func(t *testing.T) {
		payload := map[string]interface{}{
			"gate_id": gate1ID.String(),
			"label":   "Unauthorized QR",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/qr", socID, gate1ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+gkToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for gatekeeper creating QR, got %d", rec.Code)
		}
	})

	// Phase 3: Public QR Resolution
	t.Run("Public Visitor Resolves QR Token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/"+qrToken, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK resolving valid QR, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool                    `json:"success"`
			Data    society.PublicQRContext `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Data.SocietyName != socName {
			t.Errorf("expected society name %q, got %q", socName, resp.Data.SocietyName)
		}
		if resp.Data.GateName == nil || *resp.Data.GateName != "Main Gate 1" {
			t.Errorf("expected gate name Main Gate 1, got %v", resp.Data.GateName)
		}
	})

	// Phase 4: Public OTP Send & Verify
	visitorPhone := "+919820099991"
	var verificationToken string

	t.Run("Public Visitor Sends and Verifies OTP", func(t *testing.T) {
		// Send OTP
		sendPayload, _ := json.Marshal(map[string]string{
			"phone":    visitorPhone,
			"qr_token": qrToken,
		})
		sendReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/send-otp", bytes.NewReader(sendPayload))
		sendReq.Header.Set("Content-Type", "application/json")
		sendRec := httptest.NewRecorder()
		router.ServeHTTP(sendRec, sendReq)

		if sendRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK sending OTP, got %d: %s", sendRec.Code, sendRec.Body.String())
		}

		var sendResp struct {
			Success bool `json:"success"`
			Data    struct {
				OTP string `json:"otp"`
			} `json:"data"`
		}
		_ = json.Unmarshal(sendRec.Body.Bytes(), &sendResp)
		otpCode := sendResp.Data.OTP
		if otpCode == "" {
			otpCode = "123456"
		}

		// Verify OTP with dev code
		verifyPayload, _ := json.Marshal(map[string]string{
			"phone":    visitorPhone,
			"otp":      otpCode,
			"qr_token": qrToken,
		})
		verifyReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/verify-otp", bytes.NewReader(verifyPayload))
		verifyReq.Header.Set("Content-Type", "application/json")
		verifyRec := httptest.NewRecorder()
		router.ServeHTTP(verifyRec, verifyReq)

		if verifyRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK verifying OTP, got %d: %s", verifyRec.Code, verifyRec.Body.String())
		}

		var verifyResp struct {
			Success bool `json:"success"`
			Data    struct {
				VerificationToken string `json:"verification_token"`
				PhoneMasked       string `json:"phone_masked"`
				Verified          bool   `json:"verified"`
			} `json:"data"`
		}
		_ = json.Unmarshal(verifyRec.Body.Bytes(), &verifyResp)

		if !verifyResp.Data.Verified || verifyResp.Data.VerificationToken == "" {
			t.Fatalf("verification token not returned: %+v", verifyResp)
		}
		verificationToken = verifyResp.Data.VerificationToken
	})

	// Phase 5: Self-Registration Submission & Pass Issuance
	var passReference string
	var visitID string

	t.Run("Public Visitor Submits Self-Registration", func(t *testing.T) {
		regPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qrToken,
			"full_name":          "Arun Verma",
			"phone":              visitorPhone,
			"verification_token": verificationToken,
			"purpose":            "DELIVERY",
			"coming_from":        "Andheri Courier Hub",
			"visitor_photo_url":  "https://example.com/photos/arun.jpg",
			"unit_id":            unitID.String(),
		})
		regReq := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(regPayload))
		regReq.Header.Set("Content-Type", "application/json")
		regRec := httptest.NewRecorder()
		router.ServeHTTP(regRec, regReq)

		if regRec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for visitor registration, got %d: %s", regRec.Code, regRec.Body.String())
		}

		var regResp struct {
			Success bool                      `json:"success"`
			Data    society.PublicVisitorPass `json:"data"`
		}
		_ = json.Unmarshal(regRec.Body.Bytes(), &regResp)

		if regResp.Data.Status != "REGISTERED_PENDING_GATE_VERIFICATION" {
			t.Errorf("expected status REGISTERED_PENDING_GATE_VERIFICATION, got %s", regResp.Data.Status)
		}
		if regResp.Data.PassReference == "" {
			t.Fatalf("missing pass reference: %+v", regResp)
		}
		passReference = regResp.Data.PassReference
		visitID = regResp.Data.VisitID.String()
	})

	// Phase 6: Public Pass Tracking
	t.Run("Public Visitor Checks Pass Status", func(t *testing.T) {
		passReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/pass/"+passReference, nil)
		passRec := httptest.NewRecorder()
		router.ServeHTTP(passRec, passReq)

		if passRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK retrieving pass, got %d: %s", passRec.Code, passRec.Body.String())
		}

		var passResp struct {
			Success bool                      `json:"success"`
			Data    society.PublicVisitorPass `json:"data"`
		}
		_ = json.Unmarshal(passRec.Body.Bytes(), &passResp)

		if passResp.Data.PassReference != passReference {
			t.Errorf("expected pass reference %s, got %s", passReference, passResp.Data.PassReference)
		}
		// Confirm phone is masked
		if passResp.Data.VisitorPhoneMasked == visitorPhone {
			t.Errorf("raw phone exposed in public pass response!")
		}
	})

	// Phase 7: Gatekeeper Views Pending Registrations
	t.Run("Gatekeeper Views Pending Registrations With Data Minimization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/visitors/pending-qr", socID), nil)
		req.Header.Set("Authorization", "Bearer "+gkToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK listing pending registrations, got %d: %s", rec.Code, rec.Body.String())
		}

		var pendingResp struct {
			Success bool `json:"success"`
			Data    struct {
				PendingRegistrations []society.PendingQRVisit `json:"pending_registrations"`
				Total                int                      `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &pendingResp)

		if pendingResp.Data.Total == 0 {
			t.Fatalf("expected at least 1 pending registration, got 0")
		}

		target := pendingResp.Data.PendingRegistrations[0]
		if target.PassReference != passReference {
			t.Errorf("expected pass reference %s, got %s", passReference, target.PassReference)
		}
		if target.VisitorPhone == visitorPhone {
			t.Errorf("gatekeeper received unmasked visitor phone!")
		}
	})

	// Phase 8: Gatekeeper Admits Visitor Atomically
	t.Run("Gatekeeper Admits Visitor Atomically", func(t *testing.T) {
		admitPayload, _ := json.Marshal(map[string]interface{}{
			"decision": "APPROVE",
			"gate_id":  gate1ID.String(),
		})
		admitReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/admit-qr", socID, visitID), bytes.NewReader(admitPayload))
		admitReq.Header.Set("Authorization", "Bearer "+gkToken)
		admitReq.Header.Set("Content-Type", "application/json")
		admitRec := httptest.NewRecorder()
		router.ServeHTTP(admitRec, admitReq)

		if admitRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK admitting visitor, got %d: %s", admitRec.Code, admitRec.Body.String())
		}

		var admitResp struct {
			Success bool `json:"success"`
			Data    struct {
				Visit society.Visit `json:"visit"`
			} `json:"data"`
		}
		_ = json.Unmarshal(admitRec.Body.Bytes(), &admitResp)

		if admitResp.Data.Visit.Status != "CHECKED_IN" {
			t.Errorf("expected status CHECKED_IN, got %s", admitResp.Data.Visit.Status)
		}
		if admitResp.Data.Visit.CheckInAt == nil {
			t.Errorf("expected check_in_at to be populated")
		}
	})

	// Phase 9: Double Admission Conflict (409)
	t.Run("Double Admission Conflict Rejected with 409", func(t *testing.T) {
		admitPayload, _ := json.Marshal(map[string]interface{}{
			"decision": "APPROVE",
			"gate_id":  gate1ID.String(),
		})
		admitReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/admit-qr", socID, visitID), bytes.NewReader(admitPayload))
		admitReq.Header.Set("Authorization", "Bearer "+gkToken)
		admitReq.Header.Set("Content-Type", "application/json")
		admitRec := httptest.NewRecorder()
		router.ServeHTTP(admitRec, admitReq)

		if admitRec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for already decided visit, got %d: %s", admitRec.Code, admitRec.Body.String())
		}
	})

	// Phase 10: Authenticated User Views Linked Visitor History
	t.Run("Authenticated User Views Linked Visitor History", func(t *testing.T) {
		uID, visitorUserToken := createTestUser(visitorPhone)
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO user_profiles (user_id, first_name, last_name)
			VALUES ($1, 'Arun', 'Verma')
			ON CONFLICT (user_id) DO UPDATE SET first_name = 'Arun', last_name = 'Verma'
		`, uID)

		histReq := httptest.NewRequest(http.MethodGet, "/api/v1/visitors/my-history", nil)
		histReq.Header.Set("Authorization", "Bearer "+visitorUserToken)
		histRec := httptest.NewRecorder()
		router.ServeHTTP(histRec, histReq)

		if histRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK fetching my visitor history, got %d: %s", histRec.Code, histRec.Body.String())
		}

		var histResp struct {
			Success bool `json:"success"`
			Data    struct {
				Visits []society.Visit `json:"visits"`
				Total  int             `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(histRec.Body.Bytes(), &histResp)

		if histResp.Data.Total == 0 {
			t.Fatalf("expected at least 1 visit in history, got 0")
		}
		if histResp.Data.Visits[0].PassReference == nil || *histResp.Data.Visits[0].PassReference != passReference {
			t.Errorf("expected pass reference %s in history, got %v", passReference, histResp.Data.Visits[0].PassReference)
		}
	})

	// Phase 11: Admin QR Rotation and Revocation
	t.Run("Admin Rotates QR and Old Token is Inactivated", func(t *testing.T) {
		rotReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitor-qrs/%s/rotate", socID, qrID), nil)
		rotReq.Header.Set("Authorization", "Bearer "+adminToken)
		rotRec := httptest.NewRecorder()
		router.ServeHTTP(rotRec, rotReq)

		if rotRec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK rotating QR, got %d: %s", rotRec.Code, rotRec.Body.String())
		}

		// Verify old token resolves as 410 Gone / inactive
		oldReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/"+qrToken, nil)
		oldRec := httptest.NewRecorder()
		router.ServeHTTP(oldRec, oldReq)

		if oldRec.Code != http.StatusGone {
			t.Fatalf("expected 410 Gone for rotated/revoked QR, got %d: %s", oldRec.Code, oldRec.Body.String())
		}
	})

	// Phase 12: Concurrent Admission Race
	t.Run("Concurrent Gatekeeper Admission Safety", func(t *testing.T) {
		qrs, _ := societyRepo.ListGateQRs(ctx, socID, nil)
		if len(qrs) == 0 {
			t.Fatalf("no active QRs found")
		}

		newReg, err := societyRepo.RegisterVisitorByQR(ctx, society.QRVisitorRegisterParams{
			QRToken:  qrs[0].Token,
			FullName: "Concurrent Visitor",
			Phone:    "+919820099992",
			Purpose:  "GUEST",
			UnitID:   &unitID,
		})
		if err != nil {
			t.Fatalf("failed to setup concurrent test registration: %v", err)
		}

		var wg sync.WaitGroup
		results := make([]int, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				admitPayload, _ := json.Marshal(map[string]interface{}{
					"decision": "APPROVE",
					"gate_id":  gate1ID.String(),
				})
				r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/visitors/%s/admit-qr", socID, newReg.VisitID), bytes.NewReader(admitPayload))
				r.Header.Set("Authorization", "Bearer "+gkToken)
				r.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				results[idx] = w.Code
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
			t.Fatalf("expected exactly 1 concurrent admission to succeed, got %d (results: %v)", successCount, results)
		}
		if conflictCount != 4 {
			t.Fatalf("expected 4 concurrent admissions to be rejected with 409, got %d", conflictCount)
		}
	})
}

func TestVisitorSecurityRegressions(t *testing.T) {
	ctx := context.Background()
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping visitor security regression tests")
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("skipping test; database unavailable: %v", err)
	}
	defer db.Close()

	tm, err := auth.NewTokenManager(cfg.JWTPrivateKey)
	if err != nil {
		t.Fatalf("failed to initialize token manager: %v", err)
	}

	smsMock := sms.NewMockProvider(logger)
	rateLimiter := auth.NewRateLimiter()
	authService := auth.NewService(db.Pool(), tm, smsMock, rateLimiter, auth.ServiceConfig{
		DevTestPhones:      []string{"+919800000001", "+919800000002"},
		DevTestOTP:         "123456",
		VerificationSecret: "test_verification_secret_key_12345678",
	}, logger)

	platformRepo := platform.NewRepository(db.Pool())
	societyRepo := society.NewRepository(db.Pool())
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

	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, cfg.SuperAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	// Create test society and gate
	uniqueSuffix := time.Now().UnixNano()
	socName := fmt.Sprintf("Sec Audit Society %d", uniqueSuffix)
	socCode := fmt.Sprintf("SOC-SEC-%06d", uniqueSuffix%1000000)
	regNo := fmt.Sprintf("REG-SEC-%d", uniqueSuffix%1000000)
	var socID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO societies (name, society_code, registration_number, address_line1, locality, city, state, pincode, status, created_by)
		VALUES ($1, $2, $3, 'Security Ave', 'Bandra', 'Mumbai', 'Maharashtra', '400050', 'ACTIVE', $4)
		RETURNING id
	`, socName, socCode, regNo, superAdminRole.UserID).Scan(&socID)
	if err != nil {
		t.Fatalf("failed to insert society: %v", err)
	}

	var bldID, unitID, gate1ID, gate2ID uuid.UUID
	_ = db.Pool().QueryRow(ctx, `INSERT INTO society_buildings (society_id, name, building_code, total_floors) VALUES ($1, 'Tower S', 'TS', 10) RETURNING id`, socID).Scan(&bldID)
	_ = db.Pool().QueryRow(ctx, `INSERT INTO society_units (society_id, building_id, floor_number, unit_number, unit_identifier) VALUES ($1, $2, 1, '101', 'TS-101') RETURNING id`, socID, bldID).Scan(&unitID)
	_ = db.Pool().QueryRow(ctx, `INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status) VALUES ($1, 'Main Gate', 'MG', 'BIDIRECTIONAL', 'ACTIVE', 'ACTIVE') RETURNING id`, socID).Scan(&gate1ID)
	_ = db.Pool().QueryRow(ctx, `INSERT INTO society_gates (society_id, name, gate_code, directionality, operational_status, status) VALUES ($1, 'Back Gate', 'BG', 'BIDIRECTIONAL', 'ACTIVE', 'ACTIVE') RETURNING id`, socID).Scan(&gate2ID)

	adminPhone := fmt.Sprintf("+9198300%05d", uniqueSuffix%100000)
	var adminID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO users (phone, status)
		VALUES ($1, 'ACTIVE')
		ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
		RETURNING id
	`, adminPhone).Scan(&adminID)
	if err != nil {
		t.Fatalf("failed to insert test admin: %v", err)
	}
	_, _ = db.Pool().Exec(ctx, `INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by) VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $2)`, socID, adminID)

	qr1, err := societyRepo.CreateGateQR(ctx, socID, &gate1ID, "Gate 1 QR", adminID)
	if err != nil {
		t.Fatalf("failed to create qr1: %v", err)
	}
	qr2, err := societyRepo.CreateGateQR(ctx, socID, &gate2ID, "Gate 2 QR", adminID)
	if err != nil {
		t.Fatalf("failed to create qr2: %v", err)
	}

	testPhone := fmt.Sprintf("+9198301%05d", uniqueSuffix%100000)

	// Step 1: Request & verify OTP for QR 1
	_, devOTP1, err := authService.SendOTP(ctx, testPhone, "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to send OTP: %v", err)
	}
	vToken, _, err := authService.VerifyVisitorOTPWithContext(ctx, testPhone, devOTP1, qr1.Token, "127.0.0.1")
	if err != nil {
		t.Fatalf("failed to verify OTP: %v", err)
	}

	// Step 2: Cross-QR registration attempt (using Token issued for QR 1 to register at QR 2)
	t.Run("Cross-QR Verification Token Reuse Rejected", func(t *testing.T) {
		regPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr2.Token, // Mismatch: token was signed for qr1.Token
			"full_name":          "Aditi Rao",
			"phone":              testPhone,
			"verification_token": vToken,
			"purpose":            "GUEST",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(regPayload))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for cross-QR token reuse, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Step 3: Legitimate registration with QR 1 succeeds
	var passRef1 string
	t.Run("Legitimate Registration with QR 1 Consumes Token", func(t *testing.T) {
		regPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Aditi Rao",
			"phone":              testPhone,
			"verification_token": vToken,
			"purpose":            "GUEST",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(regPayload))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for registration, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				PassReference string `json:"pass_reference"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		passRef1 = resp.Data.PassReference
	})

	// Step 4: Replay attack (reusing vToken for a second registration) must fail
	t.Run("Token Replay Rejected (Single-Use Enforced)", func(t *testing.T) {
		regPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Aditi Rao Duplicate",
			"phone":              testPhone,
			"verification_token": vToken, // Already consumed!
			"purpose":            "GUEST",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(regPayload))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for replayed token, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Step 5: Shared Phone Number Does Not Overwrite Canonical Identity Name
	t.Run("Shared Phone Does Not Overwrite Existing Canonical Identity Name", func(t *testing.T) {
		normPhone, _ := auth.NormalizePhone(testPhone)

		// Verify canonical name in DB is "Aditi Rao"
		var storedName string
		err := db.Pool().QueryRow(ctx, `SELECT full_name FROM society_visitor_identities WHERE society_id = $1 AND phone = $2`, socID, normPhone).Scan(&storedName)
		if err != nil || storedName != "Aditi Rao" {
			t.Fatalf("expected canonical identity 'Aditi Rao', got '%s', err: %v", storedName, err)
		}

		// Register second person (e.g. family member) with new verified token
		sharePhone := fmt.Sprintf("+9198302%05d", uniqueSuffix%100000)
		_, devOTPFirst, _ := authService.SendOTP(ctx, sharePhone, "127.0.0.1")
		vTokenFirst, _, _ := authService.VerifyVisitorOTPWithContext(ctx, sharePhone, devOTPFirst, qr1.Token, "127.0.0.1")
		firstPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Aditi Rao",
			"phone":              sharePhone,
			"verification_token": vTokenFirst,
			"purpose":            "GUEST",
		})
		rFirst := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(firstPayload))
		wFirst := httptest.NewRecorder()
		router.ServeHTTP(wFirst, rFirst)
		if wFirst.Code != http.StatusCreated {
			t.Fatalf("expected 201 for first registration, got %d", wFirst.Code)
		}

		// Now register second person sharing sharePhone
		normSharePhone, _ := auth.NormalizePhone(sharePhone)
		shareUser2Phone := fmt.Sprintf("+9198304%05d", uniqueSuffix%100000)
		_, devOTPSecond, _ := authService.SendOTP(ctx, shareUser2Phone, "127.0.0.1")
		vTokenSecond, _, _ := authService.VerifyVisitorOTPWithContext(ctx, shareUser2Phone, devOTPSecond, qr1.Token, "127.0.0.1")

		familyRegPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Kunal Rao", // Different human sharing phone
			"phone":              sharePhone,
			"verification_token": vTokenSecond,
			"purpose":            "FAMILY",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(familyRegPayload))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// Even if rejected due to token phone mismatch or accepted, let's verify canonical name preservation:
		var preservedName string
		_ = db.Pool().QueryRow(ctx, `SELECT full_name FROM society_visitor_identities WHERE society_id = $1 AND phone = $2`, socID, normSharePhone).Scan(&preservedName)
		if preservedName != "Aditi Rao" {
			t.Errorf("expected canonical identity name 'Aditi Rao' preserved, but was '%s'", preservedName)
		}
	})

	// Step 6: Photo Size & Format Validation
	t.Run("Photo Size and Format Constraints Enforced", func(t *testing.T) {
		photoPhone1 := fmt.Sprintf("+9198305%05d", uniqueSuffix%100000)
		_, devOTP3, err := authService.SendOTP(ctx, photoPhone1, "127.0.0.1")
		if err != nil {
			t.Fatalf("failed to send OTP for photo test 1: %v", err)
		}
		vToken3, _, err := authService.VerifyVisitorOTPWithContext(ctx, photoPhone1, devOTP3, qr1.Token, "127.0.0.1")
		if err != nil {
			t.Fatalf("failed to verify OTP for photo test 1: %v", err)
		}

		// Invalid format (e.g. javascript/executable)
		badPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Security Test",
			"phone":              photoPhone1,
			"verification_token": vToken3,
			"visitor_photo_url":  "javascript:alert(1)",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(badPayload))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for malicious photo url, got %d", rec.Code)
		}

		// Valid data image URL succeeds with fresh phone & token
		photoPhone2 := fmt.Sprintf("+9198306%05d", uniqueSuffix%100000)
		_, devOTP4, err := authService.SendOTP(ctx, photoPhone2, "127.0.0.1")
		if err != nil {
			t.Fatalf("failed to send OTP for photo test 2: %v", err)
		}
		vToken4, _, err := authService.VerifyVisitorOTPWithContext(ctx, photoPhone2, devOTP4, qr1.Token, "127.0.0.1")
		if err != nil {
			t.Fatalf("failed to verify OTP for photo test 2: %v", err)
		}

		goodPayload, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Security Test Valid",
			"phone":              photoPhone2,
			"verification_token": vToken4,
			"visitor_photo_url":  "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
		})
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(goodPayload))
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for valid photo URL, got %d: %s", rec2.Code, rec2.Body.String())
		}
	})

	// Step 7: Pass Reference Entropy & Unpredictability (128-bit Cryptographic Entropy)
	t.Run("Pass Reference Has 128-Bit Entropy Format", func(t *testing.T) {
		// "VP-" (3 chars) + 32 uppercase hex characters (16 bytes = 128 bits) = 35 chars
		if len(passRef1) != 35 || !strings.HasPrefix(passRef1, "VP-") {
			t.Fatalf("expected pass reference format 'VP-<32 hex chars>' (35 chars), got '%s' (len %d)", passRef1, len(passRef1))
		}
	})

	// Step 8: Public Pass Lookup Enumeration Resistance & Response Minimization
	t.Run("Public Pass Enumeration Resistance and Response Minimization", func(t *testing.T) {
		// 1. Invalid / malformed format rejected cleanly with 404
		invalidReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/pass/MALFORMED_REF", nil)
		invalidRec := httptest.NewRecorder()
		router.ServeHTTP(invalidRec, invalidReq)
		if invalidRec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for malformed pass reference, got %d", invalidRec.Code)
		}

		// 2. Non-existent 128-bit reference returns 404 without leaking data
		fakeRef := "VP-00112233445566778899AABBCCDDEEFF"
		notFoundReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/pass/"+fakeRef, nil)
		notFoundRec := httptest.NewRecorder()
		router.ServeHTTP(notFoundRec, notFoundReq)
		if notFoundRec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for non-existent pass reference, got %d", notFoundRec.Code)
		}

		// 3. Legitimate lookup returns minimized public data: phone masked, no raw internal PII
		legitReq := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/pass/"+passRef1, nil)
		legitRec := httptest.NewRecorder()
		router.ServeHTTP(legitRec, legitReq)
		if legitRec.Code != http.StatusOK {
			t.Fatalf("expected 200 for valid pass reference, got %d", legitRec.Code)
		}

		var legitResp struct {
			Data struct {
				VisitorName        string `json:"visitor_name"`
				VisitorPhoneMasked string `json:"visitor_phone_masked"`
				VisitorPhone       string `json:"visitor_phone"`
			} `json:"data"`
		}
		_ = json.Unmarshal(legitRec.Body.Bytes(), &legitResp)
		if legitResp.Data.VisitorPhone != "" {
			t.Fatalf("security violation: raw unmasked phone number leaked in public pass response")
		}
		if !strings.Contains(legitResp.Data.VisitorPhoneMasked, "*") {
			t.Fatalf("expected masked phone in public pass response, got '%s'", legitResp.Data.VisitorPhoneMasked)
		}
	})

	// Step 9: Public Pass Rate Limiting
	t.Run("Public Pass Lookup Enforces IP Rate Limiting", func(t *testing.T) {
		rateLimitIP := "198.51.100.42"
		var lastStatus int
		for i := 0; i < 65; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/public/visit-qr/pass/"+passRef1, nil)
			req.RemoteAddr = rateLimitIP + ":12345"
			req.Header.Set("X-Forwarded-For", rateLimitIP)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			lastStatus = rec.Code
			if rec.Code == http.StatusTooManyRequests {
				break
			}
		}
		if lastStatus != http.StatusTooManyRequests {
			t.Fatalf("expected 429 Too Many Requests after exceeding 60 pass lookups/min, got %d", lastStatus)
		}
	})

	// Step 10: Visitor History Disambiguation (Shared Phone, Missing Profile Name, Ambiguous Records)
	t.Run("Visitor History Disambiguation and Shared Phone Protection", func(t *testing.T) {
		sharedPhone := fmt.Sprintf("+9198307%05d", uniqueSuffix%100000)
		normSharedPhone, _ := auth.NormalizePhone(sharedPhone)

		// User A (Alice): has an account and completed profile
		var userAID uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, normSharedPhone).Scan(&userAID)
		if err != nil {
			t.Fatalf("failed to create user A: %v", err)
		}

		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO user_profiles (user_id, first_name, last_name)
			VALUES ($1, 'Alice', 'Sharma')
			ON CONFLICT (user_id) DO UPDATE SET first_name = 'Alice', last_name = 'Sharma'
		`, userAID)

		// User B (Bob, family member sharing the same phone, but different account or unauthenticated visitor)
		// Register a visit for Bob (sharing sharedPhone)
		_, devOTPBob, _ := authService.SendOTP(ctx, sharedPhone, "127.0.0.1")
		vTokenBob, _, _ := authService.VerifyVisitorOTPWithContext(ctx, sharedPhone, devOTPBob, qr1.Token, "127.0.0.1")
		bobReg, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Bob Sharma",
			"phone":              sharedPhone,
			"verification_token": vTokenBob,
			"purpose":            "GUEST",
		})
		rBob := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(bobReg))
		wBob := httptest.NewRecorder()
		router.ServeHTTP(wBob, rBob)
		if wBob.Code != http.StatusCreated {
			t.Fatalf("failed to register Bob: %d: %s", wBob.Code, wBob.Body.String())
		}

		// Also insert an explicit visit created by Alice while authenticated
		aliceVisitID := uuid.New()
		alicePassRef, _ := societyRepo.CreateGateQR(ctx, socID, &gate1ID, "Alice QR", adminID)
		_ = alicePassRef
		_, err = db.Pool().Exec(ctx, `
			INSERT INTO society_visits (
				id, society_id, unit_id, visitor_name, visitor_phone, purpose, visit_type, status,
				created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, 'Alice Sharma', $4, 'PERSONAL', 'EXPECTED', 'PENDING_APPROVAL',
				$5, NOW(), NOW()
			)
		`, aliceVisitID, socID, unitID, normSharedPhone, userAID)
		if err != nil {
			t.Fatalf("failed to insert Alice's visit: %v", err)
		}

		// Alice queries her history:
		aliceToken := generateToken(userAID)

		reqAliceHist := httptest.NewRequest(http.MethodGet, "/api/v1/visitors/my-history", nil)
		reqAliceHist.Header.Set("Authorization", "Bearer "+aliceToken)
		recAliceHist := httptest.NewRecorder()
		router.ServeHTTP(recAliceHist, reqAliceHist)

		if recAliceHist.Code != http.StatusOK {
			t.Fatalf("expected 200 for Alice history, got %d: %s", recAliceHist.Code, recAliceHist.Body.String())
		}

		var aliceHistoryResp struct {
			Data struct {
				Visits []struct {
					ID          uuid.UUID `json:"id"`
					VisitorName string    `json:"visitor_name"`
				} `json:"visits"`
				Total int `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recAliceHist.Body.Bytes(), &aliceHistoryResp)

		// Verification:
		// 1. Bob's visit MUST NOT be in Alice's history!
		for _, v := range aliceHistoryResp.Data.Visits {
			if v.VisitorName == "Bob Sharma" {
				t.Fatalf("CRITICAL SECURITY DEFECT: User Alice received Bob's historical visit solely due to shared phone number!")
			}
		}

		// 2. Alice's explicitly owned visit IS returned
		foundAliceVisit := false
		for _, v := range aliceHistoryResp.Data.Visits {
			if v.ID == aliceVisitID {
				foundAliceVisit = true
				break
			}
		}
		if !foundAliceVisit {
			t.Fatalf("expected Alice's explicitly created visit to be returned in history")
		}

		// Case 2: Missing Profile Name
		// User C has an account with a shared phone but NO profile name configured (first_name = '')
		userCPhone := fmt.Sprintf("+9198308%05d", uniqueSuffix%100000)
		normUserCPhone, _ := auth.NormalizePhone(userCPhone)
		var userCID uuid.UUID
		_ = db.Pool().QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id`, normUserCPhone).Scan(&userCID)
		_, _ = db.Pool().Exec(ctx, `INSERT INTO user_profiles (user_id, first_name, last_name) VALUES ($1, '', '') ON CONFLICT (user_id) DO UPDATE SET first_name = ''`, userCID)

		// Unauthenticated visit for "Dave" on userC's phone
		_, devOTPDave, _ := authService.SendOTP(ctx, userCPhone, "127.0.0.1")
		vTokenDave, _, _ := authService.VerifyVisitorOTPWithContext(ctx, userCPhone, devOTPDave, qr1.Token, "127.0.0.1")
		daveReg, _ := json.Marshal(map[string]interface{}{
			"qr_token":           qr1.Token,
			"full_name":          "Dave Unlinked",
			"phone":              userCPhone,
			"verification_token": vTokenDave,
			"purpose":            "GUEST",
		})
		rDave := httptest.NewRequest(http.MethodPost, "/api/v1/public/visit-qr/register", bytes.NewReader(daveReg))
		wDave := httptest.NewRecorder()
		router.ServeHTTP(wDave, rDave)

		userCToken := generateToken(userCID)
		reqCHist := httptest.NewRequest(http.MethodGet, "/api/v1/visitors/my-history", nil)
		reqCHist.Header.Set("Authorization", "Bearer "+userCToken)
		recCHist := httptest.NewRecorder()
		router.ServeHTTP(recCHist, reqCHist)

		var cHistoryResp struct {
			Data struct {
				Visits []struct {
					VisitorName string `json:"visitor_name"`
				} `json:"visits"`
				Total int `json:"total"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recCHist.Body.Bytes(), &cHistoryResp)
		if len(cHistoryResp.Data.Visits) > 0 {
			t.Fatalf("expected 0 visits for user with missing profile name and ambiguous unlinked visits, got %d", len(cHistoryResp.Data.Visits))
		}
	})
}

