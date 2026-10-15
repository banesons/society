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

func TestFormatINR(t *testing.T) {
	tests := []struct {
		paise    int64
		expected string
	}{
		{0, "₹0.00"},
		{50, "₹0.50"},
		{100, "₹1.00"},
		{250000, "₹2,500.00"},
		{48500000, "₹4,85,000.00"},
		{100000000, "₹10,00,000.00"},
		{1000000000, "₹1,00,00,000.00"},
		{-50000, "-₹500.00"},
	}

	for _, tc := range tests {
		actual := society.FormatINR(tc.paise)
		if actual != tc.expected {
			t.Errorf("FormatINR(%d): expected %s, got %s", tc.paise, tc.expected, actual)
		}
	}
}

func TestDashboardReportingSuite(t *testing.T) {
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

	superAdminToken := generateToken(superAdminRole.UserID)

	// Create random user for unprivileged tests
	var randUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id
	`, fmt.Sprintf("+919%09d", time.Now().UnixNano()%1000000000)).Scan(&randUserID)
	if err != nil {
		t.Fatalf("failed to create rand user: %v", err)
	}
	randUserToken := generateToken(randUserID)

	// 1. Platform Dashboard Tests
	t.Run("Platform Dashboard - Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/summary", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Platform Dashboard - Non Admin Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/summary", nil)
		req.Header.Set("Authorization", "Bearer "+randUserToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for non-admin, got %d", rec.Code)
		}
	})

	t.Run("Platform Dashboard - SuperAdmin Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/summary", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool                             `json:"success"`
			Data    society.PlatformDashboardSummary `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if !resp.Success {
			t.Fatalf("expected success true")
		}
		if resp.Data.Societies.Total < 0 {
			t.Errorf("invalid societies count: %d", resp.Data.Societies.Total)
		}
		if resp.Data.FinancialVelocity30d.AmountPaise < 0 {
			t.Errorf("invalid financial velocity: %d", resp.Data.FinancialVelocity30d.AmountPaise)
		}
	})

	// 2. Setup Society for Scoped Dashboard Tests via API
	socName := fmt.Sprintf("Dashboard Test Society %d", time.Now().UnixNano()%1000000)
	regNo := fmt.Sprintf("REG-DASH-%d", time.Now().UnixNano()%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "123 Metric Way",
		Locality:           "Analytics Enclave",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400001",
	}
	socJSON, _ := json.Marshal(socParams)
	socReq := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	socReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	socReq.Header.Set("Content-Type", "application/json")
	socRec := httptest.NewRecorder()
	router.ServeHTTP(socRec, socReq)
	if socRec.Code != http.StatusCreated {
		t.Fatalf("failed to create test society: %d %s", socRec.Code, socRec.Body.String())
	}

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(socRec.Body.Bytes(), &socResp)
	testSocID := uuid.MustParse(socResp.Data.ID)

	// Activate Society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", testSocID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	actRec := httptest.NewRecorder()
	router.ServeHTTP(actRec, actReq)
	if actRec.Code != http.StatusOK {
		t.Fatalf("failed to activate test society: %d", actRec.Code)
	}

	// Create dedicated SOC_ADMIN user
	var saUserID uuid.UUID
	saPhone := fmt.Sprintf("+918%09d", time.Now().UnixNano()%1000000000)
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id
	`, saPhone).Scan(&saUserID)
	if err != nil {
		t.Fatalf("failed to create soc admin user: %v", err)
	}

	assignBody, _ := json.Marshal(map[string]interface{}{"user_id": saUserID.String()})
	assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", testSocID), bytes.NewReader(assignBody))
	assignReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	assignReq.Header.Set("Content-Type", "application/json")
	assignRec := httptest.NewRecorder()
	router.ServeHTTP(assignRec, assignReq)
	if assignRec.Code != http.StatusCreated {
		t.Fatalf("failed to assign soc admin: %d %s", assignRec.Code, assignRec.Body.String())
	}
	socAdminToken := generateToken(saUserID)

	// Create dedicated GATEKEEPER via AssignGatekeeper
	gkPhone := fmt.Sprintf("987%07d", time.Now().UnixNano()%10000000)
	gkAssignBody, _ := json.Marshal(map[string]interface{}{"phone": gkPhone})
	gkReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", testSocID), bytes.NewReader(gkAssignBody))
	gkReq.Header.Set("Authorization", "Bearer "+socAdminToken)
	gkReq.Header.Set("Content-Type", "application/json")
	gkRec := httptest.NewRecorder()
	router.ServeHTTP(gkRec, gkReq)
	if gkRec.Code != http.StatusCreated {
		t.Fatalf("failed to assign gatekeeper: %d %s", gkRec.Code, gkRec.Body.String())
	}
	var gkResp struct {
		Data struct {
			Gatekeeper struct {
				UserID string `json:"user_id"`
			} `json:"gatekeeper"`
		} `json:"data"`
	}
	_ = json.Unmarshal(gkRec.Body.Bytes(), &gkResp)
	gatekeeperToken := generateToken(uuid.MustParse(gkResp.Data.Gatekeeper.UserID))

	// 3. Society Dashboard - Authorization & Scoping
	t.Run("Society Dashboard - Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", testSocID), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("Society Dashboard - Cross-Society / Unprivileged Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", testSocID), nil)
		req.Header.Set("Authorization", "Bearer "+randUserToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", rec.Code)
		}
	})

	t.Run("Society Dashboard - Empty State Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", testSocID), nil)
		req.Header.Set("Authorization", "Bearer "+socAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool                            `json:"success"`
			Data    society.SocietyDashboardSummary `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if !resp.Success {
			t.Fatalf("expected success true")
		}
		if resp.Data.Occupancy.TotalUnits != 0 {
			t.Errorf("expected 0 total units, got %d", resp.Data.Occupancy.TotalUnits)
		}
		if resp.Data.Occupancy.OccupancyRatePercent != 0.0 {
			t.Errorf("expected 0.0 occupancy rate, got %f", resp.Data.Occupancy.OccupancyRatePercent)
		}
		if resp.Data.Finance.BilledPaise != 0 {
			t.Errorf("expected 0 billed paise, got %d", resp.Data.Finance.BilledPaise)
		}
		if resp.Data.Finance.CollectedPaise != 0 {
			t.Errorf("expected 0 collected paise, got %d", resp.Data.Finance.CollectedPaise)
		}
		if resp.Data.Finance.CollectionEfficiencyPercent != 0.0 {
			t.Errorf("expected 0.0 collection efficiency, got %f", resp.Data.Finance.CollectionEfficiencyPercent)
		}
	})

	// 4. Populate Society Units, Members, and Occupancy
	bldgReq := society.CreateBuildingParams{
		Name:         "Block A",
		BuildingCode: "BA",
		TotalFloors:  5,
	}
	bldgJSON, _ := json.Marshal(bldgReq)
	bldgRec := httptest.NewRecorder()
	bldgHttpReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", testSocID), bytes.NewReader(bldgJSON))
	bldgHttpReq.Header.Set("Authorization", "Bearer "+socAdminToken)
	bldgHttpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(bldgRec, bldgHttpReq)
	if bldgRec.Code != http.StatusCreated {
		t.Fatalf("failed to create building: %d %s", bldgRec.Code, bldgRec.Body.String())
	}
	var bldgResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bldgRec.Body.Bytes(), &bldgResp)
	bldgID := uuid.MustParse(bldgResp.Data.ID)

	genReq := society.GenerateUnitsParams{
		BuildingID:         bldgID,
		StartingFloor:      1,
		TotalFloors:        2,
		UnitsPerFloor:      2,
		StartingUnitNumber: 101,
		Prefix:             "BA",
	}
	genJSON, _ := json.Marshal(genReq)
	genRec := httptest.NewRecorder()
	genHttpReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/generate", testSocID), bytes.NewReader(genJSON))
	genHttpReq.Header.Set("Authorization", "Bearer "+socAdminToken)
	genHttpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(genRec, genHttpReq)
	if genRec.Code != http.StatusCreated {
		t.Fatalf("failed to generate units: %d %s", genRec.Code, genRec.Body.String())
	}

	// List generated units to get IDs
	uRec := httptest.NewRecorder()
	uHttpReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/units", testSocID), nil)
	uHttpReq.Header.Set("Authorization", "Bearer "+socAdminToken)
	router.ServeHTTP(uRec, uHttpReq)
	if uRec.Code != http.StatusOK {
		t.Fatalf("failed to list units: %d %s", uRec.Code, uRec.Body.String())
	}
	var uResp struct {
		Data struct {
			Units []struct {
				ID string `json:"id"`
			} `json:"units"`
		} `json:"data"`
	}
	_ = json.Unmarshal(uRec.Body.Bytes(), &uResp)
	if len(uResp.Data.Units) != 4 {
		t.Fatalf("expected 4 units, got %d", len(uResp.Data.Units))
	}
	unit0ID := uuid.MustParse(uResp.Data.Units[0].ID)
	unit1ID := uuid.MustParse(uResp.Data.Units[1].ID)

	// Register 2 members
	regMem := func(first, last, phone string) uuid.UUID {
		memBody, _ := json.Marshal(map[string]interface{}{
			"first_name": first,
			"last_name":  last,
			"phone":      phone,
		})
		memReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/members", testSocID), bytes.NewReader(memBody))
		memReq.Header.Set("Authorization", "Bearer "+socAdminToken)
		memReq.Header.Set("Content-Type", "application/json")
		memRec := httptest.NewRecorder()
		router.ServeHTTP(memRec, memReq)
		if memRec.Code != http.StatusCreated {
			t.Fatalf("failed to register member: %d %s", memRec.Code, memRec.Body.String())
		}
		var mResp struct {
			Data struct {
				Member struct {
					ID string `json:"id"`
				} `json:"member"`
			} `json:"data"`
		}
		_ = json.Unmarshal(memRec.Body.Bytes(), &mResp)
		return uuid.MustParse(mResp.Data.Member.ID)
	}

	mem1ID := regMem("Ramesh", "Patel", fmt.Sprintf("+919%09d", time.Now().UnixNano()%1000000000))
	mem2ID := regMem("Suresh", "Rao", fmt.Sprintf("+918%09d", time.Now().UnixNano()%1000000000))

	// Link Unit 0: Owner-Occupant (mem1ID)
	rel0Body, _ := json.Marshal(map[string]interface{}{
		"member_id":         mem1ID.String(),
		"relationship_type": "PRIMARY_OWNER",
		"is_occupant":       true,
		"start_date":        time.Now().Format("2006-01-02"),
	})
	rel0Req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", testSocID, unit0ID), bytes.NewReader(rel0Body))
	rel0Req.Header.Set("Authorization", "Bearer "+socAdminToken)
	rel0Req.Header.Set("Content-Type", "application/json")
	rel0Rec := httptest.NewRecorder()
	router.ServeHTTP(rel0Rec, rel0Req)
	if rel0Rec.Code != http.StatusCreated {
		t.Fatalf("failed to link unit 0: %d %s", rel0Rec.Code, rel0Rec.Body.String())
	}

	// Link Unit 1: Tenant-Occupant (mem2ID)
	rel1Body, _ := json.Marshal(map[string]interface{}{
		"member_id":         mem2ID.String(),
		"relationship_type": "PRIMARY_TENANT",
		"is_occupant":       true,
		"start_date":        time.Now().Format("2006-01-02"),
	})
	rel1Req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", testSocID, unit1ID), bytes.NewReader(rel1Body))
	rel1Req.Header.Set("Authorization", "Bearer "+socAdminToken)
	rel1Req.Header.Set("Content-Type", "application/json")
	rel1Rec := httptest.NewRecorder()
	router.ServeHTTP(rel1Rec, rel1Req)
	if rel1Rec.Code != http.StatusCreated {
		t.Fatalf("failed to link unit 1: %d %s", rel1Rec.Code, rel1Rec.Body.String())
	}

	// Units 2 and 3 remain vacant

	t.Run("Society Dashboard - Occupancy Calculations", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", testSocID), nil)
		req.Header.Set("Authorization", "Bearer "+socAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data society.SocietyDashboardSummary `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Data.Occupancy.TotalUnits != 4 {
			t.Errorf("expected 4 total units, got %d", resp.Data.Occupancy.TotalUnits)
		}
		if resp.Data.Occupancy.OccupiedUnits != 2 {
			t.Errorf("expected 2 occupied units, got %d", resp.Data.Occupancy.OccupiedUnits)
		}
		if resp.Data.Occupancy.VacantUnits != 2 {
			t.Errorf("expected 2 vacant units, got %d", resp.Data.Occupancy.VacantUnits)
		}
		if resp.Data.Occupancy.OccupancyRatePercent != 50.0 {
			t.Errorf("expected 50.0%% occupancy rate, got %f", resp.Data.Occupancy.OccupancyRatePercent)
		}
		if resp.Data.Occupancy.OwnerOccupiedUnits != 1 {
			t.Errorf("expected 1 owner occupied unit, got %d", resp.Data.Occupancy.OwnerOccupiedUnits)
		}
		if resp.Data.Occupancy.TenantOccupiedUnits != 1 {
			t.Errorf("expected 1 tenant occupied unit, got %d", resp.Data.Occupancy.TenantOccupiedUnits)
		}
		if resp.Data.Members.TotalActiveMembers != 2 {
			t.Errorf("expected 2 active members, got %d", resp.Data.Members.TotalActiveMembers)
		}
	})

	// 5. Gate Dashboard Tests
	t.Run("Gate Dashboard - Gatekeeper and SocAdmin Access", func(t *testing.T) {
		// Gatekeeper request
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", testSocID), nil)
		req.Header.Set("Authorization", "Bearer "+gatekeeperToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for gatekeeper, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data society.GateDashboardSummary `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp.Data.SocietyID != testSocID {
			t.Errorf("expected society ID %s, got %s", testSocID, resp.Data.SocietyID)
		}

		// SocAdmin request
		req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", testSocID), nil)
		req2.Header.Set("Authorization", "Bearer "+socAdminToken)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for soc_admin, got %d", rec2.Code)
		}
	})

	t.Run("Gate Dashboard - Unprivileged Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", testSocID), nil)
		req.Header.Set("Authorization", "Bearer "+randUserToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d", rec.Code)
		}
	})

	// 6. Non-Mutating Execution Check (AC-S16-07)
	t.Run("Non-Mutating Execution", func(t *testing.T) {
		var auditCountBefore, auditCountAfter int
		_ = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM platform_audit_logs WHERE society_id = $1", testSocID).Scan(&auditCountBefore)

		// Execute dashboard queries multiple times
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", testSocID), nil)
			req.Header.Set("Authorization", "Bearer "+socAdminToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("iteration %d failed: %d", i, rec.Code)
			}
		}

		_ = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM platform_audit_logs WHERE society_id = $1", testSocID).Scan(&auditCountAfter)
		if auditCountBefore != auditCountAfter {
			t.Errorf("read-only dashboard query mutated database: before %d, after %d", auditCountBefore, auditCountAfter)
		}
	})
}

func TestCrossSocietyAuthorization(t *testing.T) {
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
			VALUES ($1, $2, 'dummy-hash', 'test-agent', '127.0.0.1', $3)
		`, sessionID, userID, time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}
		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return tok
	}

	superAdminToken := generateToken(superAdminRole.UserID)

	// Helper to create & activate a society
	createSociety := func(namePrefix string) uuid.UUID {
		unique := time.Now().UnixNano()
		regNo := fmt.Sprintf("REG-ISO-%d", unique%10000000)
		socParams := society.CreateSocietyParams{
			Name:               fmt.Sprintf("%s %d", namePrefix, unique),
			RegistrationNumber: &regNo,
			AddressLine1:       "100 Isolation Blvd",
			Locality:           "Security Sector",
			City:               "Mumbai",
			State:              "Maharashtra",
			Pincode:            "400001",
		}
		socJSON, _ := json.Marshal(socParams)
		socReq := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
		socReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		socReq.Header.Set("Content-Type", "application/json")
		socRec := httptest.NewRecorder()
		router.ServeHTTP(socRec, socReq)
		if socRec.Code != http.StatusCreated {
			t.Fatalf("failed to create society %s: %d %s", namePrefix, socRec.Code, socRec.Body.String())
		}
		var socResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(socRec.Body.Bytes(), &socResp)
		socID := uuid.MustParse(socResp.Data.ID)

		actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
		actReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		actRec := httptest.NewRecorder()
		router.ServeHTTP(actRec, actReq)
		if actRec.Code != http.StatusOK {
			t.Fatalf("failed to activate society %s: %d", namePrefix, actRec.Code)
		}
		return socID
	}

	// Helper to assign a SOC_ADMIN
	assignSocAdmin := func(socID uuid.UUID) (uuid.UUID, string) {
		var uID uuid.UUID
		phone := fmt.Sprintf("+918%09d", time.Now().UnixNano()%1000000000)
		_ = db.Pool().QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id`, phone).Scan(&uID)

		assignBody, _ := json.Marshal(map[string]interface{}{"user_id": uID.String()})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader(assignBody))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("failed to assign soc admin: %d %s", rec.Code, rec.Body.String())
		}
		return uID, generateToken(uID)
	}

	// Helper to assign a GATEKEEPER
	assignGatekeeper := func(socID uuid.UUID, adminToken string) (uuid.UUID, string) {
		gkPhone := fmt.Sprintf("987%07d", time.Now().UnixNano()%10000000)
		gkAssignBody, _ := json.Marshal(map[string]interface{}{"phone": gkPhone})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), bytes.NewReader(gkAssignBody))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("failed to assign gatekeeper: %d %s", rec.Code, rec.Body.String())
		}
		var gkResp struct {
			Data struct {
				Gatekeeper struct {
					UserID string `json:"user_id"`
				} `json:"gatekeeper"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &gkResp)
		gkUID := uuid.MustParse(gkResp.Data.Gatekeeper.UserID)
		return gkUID, generateToken(gkUID)
	}

	// Setup Society A and Society B
	socAID := createSociety("Tenant Isolation Society A")
	socBID := createSociety("Tenant Isolation Society B")

	_, adminAToken := assignSocAdmin(socAID)
	_, adminBToken := assignSocAdmin(socBID)

	_, gkBToken := assignGatekeeper(socBID, adminBToken)
	_, gkAToken := assignGatekeeper(socAID, adminAToken)

	// Matrix of Cross-Society Authorization Invariants
	t.Run("Assigned SOC_ADMIN can read own society dashboard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Assigned SOC_ADMIN cannot read another society's dashboard (Cross-Tenant 403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socBID), nil)
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for cross-society access, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Assigned SOC_ADMIN cannot read platform dashboard (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/summary", nil)
		req.Header.Set("Authorization", "Bearer "+adminAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for soc_admin on platform dashboard, got %d", rec.Code)
		}
	})

	t.Run("Assigned GATEKEEPER can access permitted gate metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+gkAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for gatekeeper, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Unassigned GATEKEEPER cannot access another society's gate metrics (Cross-Tenant 403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", socBID), nil)
		req.Header.Set("Authorization", "Bearer "+gkAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for unassigned gatekeeper, got %d", rec.Code)
		}
	})

	t.Run("Assigned GATEKEEPER B can access permitted gate metrics on Society B", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", socBID), nil)
		req.Header.Set("Authorization", "Bearer "+gkBToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for gatekeeper B, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Assigned GATEKEEPER B cannot access Society A gate metrics (Cross-Tenant 403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+gkBToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for gatekeeper B on society A, got %d", rec.Code)
		}
	})

	t.Run("Assigned SOC_ADMIN B cannot read Society A dashboard (Cross-Tenant 403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+adminBToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for soc_admin B on society A, got %d", rec.Code)
		}
	})

	t.Run("GATEKEEPER cannot access society operations dashboard (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+gkAToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for gatekeeper on society dashboard, got %d", rec.Code)
		}
	})

	t.Run("SuperAdmin platform role follows documented authorization rules", func(t *testing.T) {
		// Platform dashboard
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/platform/dashboard/summary", nil)
		req1.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)
		if rec1.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for superadmin on platform dashboard, got %d", rec1.Code)
		}

		// Society A dashboard (governance role bypass)
		req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socAID), nil)
		req2.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for superadmin on society A dashboard, got %d", rec2.Code)
		}

		// Society B dashboard (governance role bypass)
		req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socBID), nil)
		req3.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec3 := httptest.NewRecorder()
		router.ServeHTTP(rec3, req3)
		if rec3.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for superadmin on society B dashboard, got %d", rec3.Code)
		}
	})

	t.Run("Missing authentication produces 401 Unauthorized", func(t *testing.T) {
		endpoints := []string{
			"/api/v1/platform/dashboard/summary",
			fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", socAID),
			fmt.Sprintf("/api/v1/societies/%s/gate/dashboard", socAID),
		}
		for _, ep := range endpoints {
			req := httptest.NewRequest(http.MethodGet, ep, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized for endpoint %s, got %d", ep, rec.Code)
			}
		}
	})
}

func TestFinancialCorrectnessAndModuleComparison(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	societyRepo := society.NewRepository(db.Pool())

	// 1. Create dedicated isolated test society directly in DB
	var creatorID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&creatorID)

	socID := uuid.New()
	socCode := fmt.Sprintf("SOC-FIN-%06d", time.Now().UnixNano()%1000000)
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO societies (id, name, society_code, address_line1, locality, city, state, pincode, status, created_by)
		VALUES ($1, 'Financial Correctness Test Society', $2, '101 Ledger Lane', 'Audit Zone', 'Mumbai', 'Maharashtra', '400001', 'ACTIVE', $3)
	`, socID, socCode, creatorID)
	if err != nil {
		t.Fatalf("failed to insert test society: %v", err)
	}

	// 2. Insert Building & 3 Units
	bldgID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_buildings (id, society_id, name, building_code, total_floors)
		VALUES ($1, $2, 'Tower Fin', 'TF', 5)
	`, bldgID, socID)
	if err != nil {
		t.Fatalf("failed to insert building: %v", err)
	}

	unit1ID := uuid.New()
	unit2ID := uuid.New()
	unit3ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_units (id, society_id, building_id, unit_number, unit_identifier, floor_number, status)
		VALUES 
			($1, $4, $5, '101', 'TF-101', 1, 'ACTIVE'),
			($2, $4, $5, '102', 'TF-102', 1, 'ACTIVE'),
			($3, $4, $5, '103', 'TF-103', 1, 'ACTIVE')
	`, unit1ID, unit2ID, unit3ID, socID, bldgID)
	if err != nil {
		t.Fatalf("failed to insert units: %v", err)
	}

	// 3. Insert Billing Period: current calendar month
	periodID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_billing_periods (id, society_id, period_name, period_start, period_end, due_date, status, created_by)
		VALUES (
			$1, $2, 'Oct 2026 Audit Period',
			DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date,
			(DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date) + INTERVAL '1 month - 1 day')::date,
			(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date - INTERVAL '2 days',
			'ACTIVE', $3
		)
	`, periodID, socID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert billing period: %v", err)
	}

	// 4. Insert Bills:
	// - Bill 1: ISSUED, ₹1,000.00 (100,000 paise), due yesterday (overdue)
	// - Bill 2: ISSUED, ₹2,000.00 (200,000 paise), due yesterday (overdue)
	// - Bill 3: VOID, ₹500.00 (50,000 paise), due yesterday
	bill1ID := uuid.New()
	bill2ID := uuid.New()
	bill3ID := uuid.New()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_bills (id, society_id, unit_id, billing_period_id, bill_number, subtotal, total_amount, due_date, status, created_by)
		VALUES 
			($1, $4, $5, $6, 'BILL-FIN-01', 1000.00, 1000.00, $7, 'ISSUED', $8),
			($2, $4, $9, $6, 'BILL-FIN-02', 2000.00, 2000.00, $7, 'ISSUED', $8),
			($3, $4, $10, $6, 'BILL-FIN-03', 500.00, 500.00, $7, 'VOID', $8)
	`, bill1ID, bill2ID, bill3ID, socID, unit1ID, periodID, yesterday, creatorID, unit2ID, unit3ID)
	if err != nil {
		t.Fatalf("failed to insert bills: %v", err)
	}

	// 5. Insert Payments & Allocations:
	// - Payment 1: ₹1,000.00, RECORDED. Allocation to Bill 1: ₹1,000.00 (ACTIVE)
	pay1ID := uuid.New()
	alloc1ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payments (id, society_id, payment_reference, payment_method, amount, status, payment_date, recorded_by)
		VALUES ($1, $2, 'REF-FIN-01', 'UPI_RECORDED_OFFLINE', 1000.00, 'RECORDED', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date, $3)
	`, pay1ID, socID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert pay1: %v", err)
	}
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payment_allocations (id, society_id, payment_id, bill_id, allocated_amount, status, allocated_by)
		VALUES ($1, $2, $3, $4, 1000.00, 'ACTIVE', $5)
	`, alloc1ID, socID, pay1ID, bill1ID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert alloc1: %v", err)
	}

	// - Payment 2: ₹500.00, RECORDED. Allocation to Bill 2: ₹500.00 (ACTIVE) -> Partial allocation!
	pay2ID := uuid.New()
	alloc2ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payments (id, society_id, payment_reference, payment_method, amount, status, payment_date, recorded_by)
		VALUES ($1, $2, 'REF-FIN-02', 'BANK_TRANSFER', 500.00, 'RECORDED', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date, $3)
	`, pay2ID, socID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert pay2: %v", err)
	}
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payment_allocations (id, society_id, payment_id, bill_id, allocated_amount, status, allocated_by)
		VALUES ($1, $2, $3, $4, 500.00, 'ACTIVE', $5)
	`, alloc2ID, socID, pay2ID, bill2ID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert alloc2: %v", err)
	}

	// - Payment 3: ₹700.00, REVERSED. Allocation to Bill 2: ₹700.00 (REVERSED) -> Must be excluded!
	pay3ID := uuid.New()
	alloc3ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payments (id, society_id, payment_reference, payment_method, amount, status, payment_date, recorded_by, reversed_at, reversal_reason)
		VALUES ($1, $2, 'REF-FIN-03', 'UPI_RECORDED_OFFLINE', 700.00, 'REVERSED', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date, $3, NOW(), 'Customer dispute')
	`, pay3ID, socID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert pay3: %v", err)
	}
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payment_allocations (id, society_id, payment_id, bill_id, allocated_amount, status, allocated_by, reversed_at)
		VALUES ($1, $2, $3, $4, 700.00, 'REVERSED', $5, NOW())
	`, alloc3ID, socID, pay3ID, bill2ID, creatorID)
	if err != nil {
		t.Fatalf("failed to insert alloc3: %v", err)
	}

	// 6. Test Society Dashboard Summary calculations
	dashSummary, err := societyRepo.GetSocietyDashboardSummary(ctx, socID)
	if err != nil {
		t.Fatalf("failed to get dashboard summary: %v", err)
	}

	// Invariant A: Void bills excluded from BilledPaise
	// Expected: Bill 1 (1000) + Bill 2 (2000) = ₹3000 = 300,000 paise (Bill 3 of ₹500 is VOID and excluded)
	if dashSummary.Finance.BilledPaise != 300000 {
		t.Errorf("BilledPaise: expected 300000 (excluding VOID), got %d", dashSummary.Finance.BilledPaise)
	}

	// Invariant B: Reversed allocations excluded from CollectedPaise
	// Expected: Alloc 1 (1000) + Alloc 2 (500) = ₹1500 = 150,000 paise (Alloc 3 of ₹700 is REVERSED and excluded)
	if dashSummary.Finance.CollectedPaise != 150000 {
		t.Errorf("CollectedPaise: expected 150000 (excluding REVERSED), got %d", dashSummary.Finance.CollectedPaise)
	}

	// Invariant C: Collection Efficiency correctly computed
	// Expected: 150000 * 100.0 / 300000 = 50.0%
	if dashSummary.Finance.CollectionEfficiencyPercent != 50.0 {
		t.Errorf("CollectionEfficiencyPercent: expected 50.0, got %f", dashSummary.Finance.CollectionEfficiencyPercent)
	}

	// Invariant D: Overdue arrears correctly calculate partial balance
	// Bill 1: 1000 - 1000 = 0.
	// Bill 2: 2000 - 500 = 1500 unpaid.
	// Expected overdue arrears: 150,000 paise (₹1,500.00)
	if dashSummary.Finance.OverdueArrearsPaise != 150000 {
		t.Errorf("OverdueArrearsPaise: expected 150000, got %d", dashSummary.Finance.OverdueArrearsPaise)
	}

	// Invariant E: Payments collected MTD excludes reversed payments
	// Payment 1 (1000) + Payment 2 (500) = 150,000 paise (Payment 3 of ₹700 is REVERSED)
	if dashSummary.Finance.PaymentsCollectedMTDPaise != 150000 {
		t.Errorf("PaymentsCollectedMTDPaise: expected 150000, got %d", dashSummary.Finance.PaymentsCollectedMTDPaise)
	}

	// Invariant F: Paise and formatted INR values agree exactly
	if society.FormatINR(dashSummary.Finance.BilledPaise) != "₹3,000.00" {
		t.Errorf("FormatINR Billed: expected ₹3,000.00, got %s", society.FormatINR(dashSummary.Finance.BilledPaise))
	}
	if society.FormatINR(dashSummary.Finance.CollectedPaise) != "₹1,500.00" {
		t.Errorf("FormatINR Collected: expected ₹1,500.00, got %s", society.FormatINR(dashSummary.Finance.CollectedPaise))
	}
	if society.FormatINR(dashSummary.Finance.OverdueArrearsPaise) != "₹1,500.00" {
		t.Errorf("FormatINR Overdue: expected ₹1,500.00, got %s", society.FormatINR(dashSummary.Finance.OverdueArrearsPaise))
	}

	// Invariant G: Comparison with GetCollectionSummary from maintenance module
	colSummary, err := societyRepo.GetCollectionSummary(ctx, socID, &periodID)
	if err != nil {
		t.Fatalf("failed to get collection summary: %v", err)
	}

	if colSummary.TotalBilled != float64(dashSummary.Finance.BilledPaise)/100.0 {
		t.Errorf("Module Mismatch TotalBilled: maintenance module %f vs dashboard paise %d",
			colSummary.TotalBilled, dashSummary.Finance.BilledPaise)
	}
	if colSummary.TotalCollected != float64(dashSummary.Finance.CollectedPaise)/100.0 {
		t.Errorf("Module Mismatch TotalCollected: maintenance module %f vs dashboard paise %d",
			colSummary.TotalCollected, dashSummary.Finance.CollectedPaise)
	}
	if colSummary.TotalOutstanding != float64(dashSummary.Finance.OverdueArrearsPaise)/100.0 {
		t.Errorf("Module Mismatch TotalOutstanding: maintenance module %f vs dashboard overdue paise %d",
			colSummary.TotalOutstanding, dashSummary.Finance.OverdueArrearsPaise)
	}
	if colSummary.ReversedAmount != 700.00 {
		t.Errorf("ReversedAmount: expected 700.00, got %f", colSummary.ReversedAmount)
	}
}

func TestTimezoneBoundaries(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	societyRepo := society.NewRepository(db.Pool())

	// 1. Create dedicated isolated test society for timezone verification
	var creatorID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "SELECT id FROM users LIMIT 1").Scan(&creatorID)

	socID := uuid.New()
	socCode := fmt.Sprintf("SOC-TZN-%06d", time.Now().UnixNano()%1000000)
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO societies (id, name, society_code, address_line1, locality, city, state, pincode, status, created_by)
		VALUES ($1, 'Timezone Boundary Society', $2, '101 Meridian Way', 'IST Zone', 'Mumbai', 'Maharashtra', '400001', 'ACTIVE', $3)
	`, socID, socCode, creatorID)
	if err != nil {
		t.Fatalf("failed to insert test society: %v", err)
	}

	// 2. Building, Units & Gate
	bldgID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_buildings (id, society_id, name, building_code, total_floors)
		VALUES ($1, $2, 'TZ Tower', 'TZT', 3)
	`, bldgID, socID)
	if err != nil {
		t.Fatalf("failed to insert building: %v", err)
	}

	unitAID := uuid.New()
	unitBID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_units (id, society_id, building_id, unit_number, unit_identifier, floor_number, status)
		VALUES 
			($1, $3, $4, '101', 'TZT-101', 1, 'ACTIVE'),
			($2, $3, $4, '102', 'TZT-102', 1, 'ACTIVE')
	`, unitAID, unitBID, socID, bldgID)
	if err != nil {
		t.Fatalf("failed to insert units: %v", err)
	}

	gateID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_gates (id, society_id, name, gate_code, gate_type, status)
		VALUES ($1, $2, 'TZ Main Gate', 'TZG-1', 'VEHICLE_AND_PEDESTRIAN', 'ACTIVE')
	`, gateID, socID)
	if err != nil {
		t.Fatalf("failed to insert gate: %v", err)
	}

	// 3. Gate Entries IST Midnight Boundary:
	// Insert Visitor 1: check_in_at = IST midnight today + 1 second -> MUST be included in today_entries
	// Insert Visitor 2: check_in_at = IST midnight today - 1 second -> MUST be excluded from today_entries
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_visits (id, society_id, unit_id, visitor_name, visitor_phone, purpose, visit_type, status, check_in_at, check_out_at, entry_gate_id, exit_gate_id, created_by)
		VALUES 
			(gen_random_uuid(), $1, $2, 'Today Visitor', '919800000001', 'Delivery', 'WALK_IN', 'CHECKED_IN', ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata') + INTERVAL '1 second', NULL, $4, NULL, $3),
			(gen_random_uuid(), $1, $2, 'Yesterday Visitor', '919800000002', 'Delivery', 'WALK_IN', 'CHECKED_OUT', ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata') - INTERVAL '1 second', ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata') - INTERVAL '1 second' + INTERVAL '1 minute', $4, $4, $3)
	`, socID, unitAID, creatorID, gateID)
	if err != nil {
		t.Fatalf("failed to insert test visits: %v", err)
	}

	gateSummary, err := societyRepo.GetGateDashboardSummary(ctx, socID)
	if err != nil {
		t.Fatalf("failed to get gate dashboard summary: %v", err)
	}

	if gateSummary.TodayEntries != 1 {
		t.Errorf("TodayEntries boundary: expected exactly 1 (IST today), got %d", gateSummary.TodayEntries)
	}
	if gateSummary.VisitorsOnPremises != 1 {
		t.Errorf("VisitorsOnPremises: expected exactly 1 (CHECKED_IN), got %d", gateSummary.VisitorsOnPremises)
	}

	// 4. Maintenance Due Date Overdue Boundary:
	// Bill A: due_date = IST today -> NOT overdue (due_date < today is false)
	// Bill B: due_date = IST today - 1 day -> IS overdue (due_date < today is true)
	periodID := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_billing_periods (id, society_id, period_name, period_start, period_end, due_date, status, created_by)
		VALUES ($1, $2, 'TZ Test Period', 
			DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date,
			(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date,
			(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date,
			'ACTIVE', $3)
	`, periodID, socID, creatorID)

	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_bills (id, society_id, unit_id, billing_period_id, bill_number, subtotal, total_amount, due_date, status, created_by)
		VALUES 
			(gen_random_uuid(), $1, $2, $3, 'BILL-TZ-DUE-TODAY', 1000.00, 1000.00, (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date, 'ISSUED', $4),
			(gen_random_uuid(), $1, $5, $3, 'BILL-TZ-DUE-YESTERDAY', 2000.00, 2000.00, (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date - INTERVAL '1 day', 'ISSUED', $4)
	`, socID, unitAID, periodID, creatorID, unitBID)

	socSummary, err := societyRepo.GetSocietyDashboardSummary(ctx, socID)
	if err != nil {
		t.Fatalf("failed to get society dashboard summary: %v", err)
	}

	// Bill A (1000.00 due today) is NOT overdue. Bill B (2000.00 due yesterday) IS overdue.
	// Total overdue arrears should be exactly 200,000 paise (₹2,000.00).
	if socSummary.Finance.OverdueArrearsPaise != 200000 {
		t.Errorf("OverdueArrearsPaise boundary: expected 200000 (only bill due yesterday), got %d", socSummary.Finance.OverdueArrearsPaise)
	}

	// 5. MTD Month Boundary:
	// Payment on 1st of current month -> included in MTD
	// Payment on last day of previous month -> excluded from MTD
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO maintenance_payments (id, society_id, payment_reference, payment_method, amount, status, payment_date, recorded_by)
		VALUES 
			(gen_random_uuid(), $1, 'PAY-TZ-MTD-START', 'UPI_RECORDED_OFFLINE', 500.00, 'RECORDED', DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date, $2),
			(gen_random_uuid(), $1, 'PAY-TZ-MTD-PREV-MONTH', 'UPI_RECORDED_OFFLINE', 800.00, 'RECORDED', DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date - INTERVAL '1 day', $2)
	`, socID, creatorID)

	socSummaryMTD, err := societyRepo.GetSocietyDashboardSummary(ctx, socID)
	if err != nil {
		t.Fatalf("failed to get society dashboard summary: %v", err)
	}

	// PaymentsCollectedMTDPaise should include only PAY-TZ-MTD-START (₹500.00 = 50,000 paise)
	if socSummaryMTD.Finance.PaymentsCollectedMTDPaise != 50000 {
		t.Errorf("PaymentsCollectedMTDPaise month boundary: expected 50000, got %d", socSummaryMTD.Finance.PaymentsCollectedMTDPaise)
	}
}
