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

func TestSocietyStructureAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
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

	superAdminRole, err := platformRepo.BootstrapSuperAdmin(ctx, cfg.SuperAdminPhone)
	if err != nil {
		t.Fatalf("super admin bootstrap failed: %v", err)
	}

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	generateToken := func(userID uuid.UUID) string {
		sessionID := uuid.New()
		_, err := db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at, created_at)
			VALUES ($1, $2, 'dummy-hash', NOW() + INTERVAL '1 hour', NOW())
		`, sessionID, userID)
		if err != nil {
			t.Fatalf("failed to insert session: %v", err)
		}

		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return tok
	}

	superAdminToken := generateToken(superAdminRole.UserID)

	// Create an active test society
	socSuffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	createSocReq := CreateSocietyRequest{
		Name:         "Structure Test Society " + socSuffix,
		AddressLine1: "123 High Street",
		Locality:     "Downtown",
		City:         "Mumbai",
		State:        "Maharashtra",
		Pincode:      "400001",
	}
	bodyJSON, _ := json.Marshal(createSocReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+superAdminToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create test society: %d %s", rec.Code, rec.Body.String())
	}
	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &socResp)
	testSocID := socResp.Data.ID

	// Activate the test society so operational changes are permitted
	actReq := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+testSocID+"/activate", nil)
	actReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	actRec := httptest.NewRecorder()
	router.ServeHTTP(actRec, actReq)
	if actRec.Code != http.StatusOK {
		t.Fatalf("failed to activate test society: %d %s", actRec.Code, actRec.Body.String())
	}

	// 1. Unauthenticated request -> 401
	t.Run("Unauthenticated structure request returns 401", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/buildings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	// 2. Unauthorized caller without society or super admin role -> 403
	t.Run("Unauthorized user returns 403", func(t *testing.T) {
		unauthUserID := uuid.New()
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO users (id, phone, status, created_at, updated_at)
			VALUES ($1, $2, 'ACTIVE', NOW(), NOW())
		`, unauthUserID, fmt.Sprintf("919999%06d", time.Now().UnixNano()%1000000))
		unauthToken := generateToken(unauthUserID)

		r := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/buildings", nil)
		r.Header.Set("Authorization", "Bearer "+unauthToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	var testBuildingID string
	var testWingID string

	// 3. Super Admin creates building
	t.Run("Create building A with 9 floors", func(t *testing.T) {
		bReq := society.CreateBuildingParams{
			Name:         "Tower A",
			BuildingCode: "A",
			TotalFloors:  9,
		}
		bJSON, _ := json.Marshal(bReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/buildings", bytes.NewReader(bJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var bResp struct {
			Data struct {
				ID           string `json:"id"`
				Name         string `json:"name"`
				BuildingCode string `json:"building_code"`
				TotalFloors  int    `json:"total_floors"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &bResp)
		if bResp.Data.BuildingCode != "A" || bResp.Data.TotalFloors != 9 {
			t.Errorf("unexpected building data: %+v", bResp.Data)
		}
		testBuildingID = bResp.Data.ID
	})

	// 4. Duplicate building code rejected -> 409
	t.Run("Duplicate building code rejected", func(t *testing.T) {
		bReq := society.CreateBuildingParams{
			Name:         "Tower A Duplicate",
			BuildingCode: "A",
			TotalFloors:  5,
		}
		bJSON, _ := json.Marshal(bReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/buildings", bytes.NewReader(bJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 5. Create Wing A under Building A
	t.Run("Create wing A under Building A", func(t *testing.T) {
		wReq := society.CreateWingParams{
			Name:        "Wing A",
			WingCode:    "A",
			TotalFloors: 9,
		}
		wJSON, _ := json.Marshal(wReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/buildings/"+testBuildingID+"/wings", bytes.NewReader(wJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var wingResp struct {
			Data struct {
				ID       string `json:"id"`
				WingCode string `json:"wing_code"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &wingResp)
		if wingResp.Data.WingCode != "A" {
			t.Errorf("unexpected wing code: %s", wingResp.Data.WingCode)
		}
		testWingID = wingResp.Data.ID
	})

	// 6. Preview Units Generation (9 floors, 3 units per floor -> 27 units, A101 to A903)
	t.Run("Preview units 9 floors x 3 units = 27 units", func(t *testing.T) {
		bUUID := uuid.MustParse(testBuildingID)
		wUUID := uuid.MustParse(testWingID)
		genReq := society.GenerateUnitsParams{
			BuildingID:         bUUID,
			WingID:             &wUUID,
			StartingFloor:      1,
			TotalFloors:        9,
			UnitsPerFloor:      3,
			StartingUnitNumber: 101,
			Prefix:             "A",
		}
		genJSON, _ := json.Marshal(genReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/units/preview", bytes.NewReader(genJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var pResp struct {
			Data society.UnitPreviewResult `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &pResp)
		if pResp.Data.TotalUnits != 27 {
			t.Fatalf("expected 27 units in preview, got %d", pResp.Data.TotalUnits)
		}
		if pResp.Data.Units[0].UnitIdentifier != "A101" {
			t.Errorf("expected first unit A101, got %s", pResp.Data.Units[0].UnitIdentifier)
		}
		if pResp.Data.Units[26].UnitIdentifier != "A903" {
			t.Errorf("expected last unit A903, got %s", pResp.Data.Units[26].UnitIdentifier)
		}
	})

	// 7. Persist Unit Generation (Bulk Generate)
	t.Run("Persist 27 units transactionally", func(t *testing.T) {
		bUUID := uuid.MustParse(testBuildingID)
		wUUID := uuid.MustParse(testWingID)
		genReq := society.GenerateUnitsParams{
			BuildingID:         bUUID,
			WingID:             &wUUID,
			StartingFloor:      1,
			TotalFloors:        9,
			UnitsPerFloor:      3,
			StartingUnitNumber: 101,
			Prefix:             "A",
		}
		genJSON, _ := json.Marshal(genReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/units/generate", bytes.NewReader(genJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var genResp struct {
			Data struct {
				GeneratedCount int `json:"generated_count"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &genResp)
		if genResp.Data.GeneratedCount != 27 {
			t.Errorf("expected 27 generated units, got %d", genResp.Data.GeneratedCount)
		}
	})

	// 8. Duplicate generation fails with Conflict
	t.Run("Duplicate unit generation fails with 409", func(t *testing.T) {
		bUUID := uuid.MustParse(testBuildingID)
		wUUID := uuid.MustParse(testWingID)
		genReq := society.GenerateUnitsParams{
			BuildingID:         bUUID,
			WingID:             &wUUID,
			StartingFloor:      1,
			TotalFloors:        9,
			UnitsPerFloor:      3,
			StartingUnitNumber: 101,
			Prefix:             "A",
		}
		genJSON, _ := json.Marshal(genReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/units/generate", bytes.NewReader(genJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate units, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 9. List Units verifies 27 units
	t.Run("List units returns 27 units", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/units?limit=100", nil)
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var listResp struct {
			Data struct {
				Total int            `json:"total"`
				Units []society.Unit `json:"units"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &listResp)
		if listResp.Data.Total != 27 {
			t.Errorf("expected 27 units in list, got %d", listResp.Data.Total)
		}
	})

	// 10. Gate Configuration
	t.Run("Create Gate and prevent duplicate gate code", func(t *testing.T) {
		gateReq := society.CreateGateParams{
			Name:                  "Main Gate",
			GateCode:              "GATE-01",
			GateType:              society.GateTypeVehicleAndPedestrian,
			Status:                "ACTIVE",
			RequiredStaffPerShift: 2,
		}
		gateJSON, _ := json.Marshal(gateReq)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/gates", bytes.NewReader(gateJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		// Duplicate gate code
		r2 := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/gates", bytes.NewReader(gateJSON))
		r2.Header.Set("Authorization", "Bearer "+superAdminToken)
		r2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate gate code, got %d: %s", w2.Code, w2.Body.String())
		}
	})

	// 11. Staffing Configuration
	t.Run("Set and get staffing requirements", func(t *testing.T) {
		notes := "24/7 security with 3 shifts"
		staffReq := society.UpdateStaffingParams{
			ShiftsPerDay:           3,
			TotalRequiredHeadcount: 10,
			Notes:                  &notes,
		}
		staffJSON, _ := json.Marshal(staffReq)
		r := httptest.NewRequest(http.MethodPut, "/api/v1/societies/"+testSocID+"/staffing", bytes.NewReader(staffJSON))
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Get staffing
		rGet := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/staffing", nil)
		rGet.Header.Set("Authorization", "Bearer "+superAdminToken)
		wGet := httptest.NewRecorder()
		router.ServeHTTP(wGet, rGet)
		if wGet.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wGet.Code, wGet.Body.String())
		}
		var srResp struct {
			Data society.StaffingRequirement `json:"data"`
		}
		_ = json.Unmarshal(wGet.Body.Bytes(), &srResp)
		if srResp.Data.TotalRequiredHeadcount != 10 || srResp.Data.ShiftsPerDay != 3 {
			t.Errorf("unexpected staffing policy: %+v", srResp.Data)
		}
	})

	// 12. Formation Summary
	t.Run("Get formation summary returns complete accurate metrics", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/formation-summary", nil)
		r.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var sumResp struct {
			Data society.FormationSummary `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &sumResp)
		sum := sumResp.Data
		if sum.TotalBuildings != 1 {
			t.Errorf("expected 1 building, got %d", sum.TotalBuildings)
		}
		if sum.TotalWings != 1 {
			t.Errorf("expected 1 wing, got %d", sum.TotalWings)
		}
		if sum.TotalUnits != 27 {
			t.Errorf("expected 27 units, got %d", sum.TotalUnits)
		}
		if sum.TotalGates != 1 {
			t.Errorf("expected 1 gate, got %d", sum.TotalGates)
		}
		if sum.RequiredStaffPerShift != 2 {
			t.Errorf("expected 2 staff per shift, got %d", sum.RequiredStaffPerShift)
		}
		if sum.TotalRequiredHeadcount != 10 {
			t.Errorf("expected 10 required headcount, got %d", sum.TotalRequiredHeadcount)
		}
		// Honest representation: 0 registered, 0 assigned
		if sum.RegisteredGatekeepers != 0 || sum.AssignedGatekeepers != 0 {
			t.Errorf("registered or assigned gatekeepers must be 0, got registered=%d, assigned=%d", sum.RegisteredGatekeepers, sum.AssignedGatekeepers)
		}
	})

	// 13. Cross-society parent combination rejected
	t.Run("Cross-society building and wing attachment rejected", func(t *testing.T) {
		// Create a second active society
		soc2Req := CreateSocietyRequest{
			Name:         "Second Society " + fmt.Sprintf("%d", time.Now().UnixNano()%1000000),
			AddressLine1: "456 Side Street",
			Locality:     "Uptown",
			City:         "Pune",
			State:        "Maharashtra",
			Pincode:      "411001",
		}
		b2JSON, _ := json.Marshal(soc2Req)
		r2 := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(b2JSON))
		r2.Header.Set("Authorization", "Bearer "+superAdminToken)
		r2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, r2)
		if w2.Code != http.StatusCreated {
			t.Fatalf("failed to create second society: %d %s", w2.Code, w2.Body.String())
		}
		var s2Resp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w2.Body.Bytes(), &s2Resp)
		soc2ID := s2Resp.Data.ID

		// Activate society 2
		actR := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+soc2ID+"/activate", nil)
		actR.Header.Set("Authorization", "Bearer "+superAdminToken)
		actW := httptest.NewRecorder()
		router.ServeHTTP(actW, actR)
		if actW.Code != http.StatusOK {
			t.Fatalf("failed to activate second society: %d %s", actW.Code, actW.Body.String())
		}

		// Attempt to create a wing under Society 2 using Society 1's building
		wReq := society.CreateWingParams{
			Name:        "Foreign Wing",
			WingCode:    "FW",
			TotalFloors: 5,
		}
		wJSON, _ := json.Marshal(wReq)
		rCross := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+soc2ID+"/buildings/"+testBuildingID+"/wings", bytes.NewReader(wJSON))
		rCross.Header.Set("Authorization", "Bearer "+superAdminToken)
		rCross.Header.Set("Content-Type", "application/json")
		wCross := httptest.NewRecorder()
		router.ServeHTTP(wCross, rCross)
		if wCross.Code != http.StatusNotFound {
			t.Errorf("expected 404 BUILDING_NOT_FOUND when using cross-society building, got %d: %s", wCross.Code, wCross.Body.String())
		}

		// Attempt to generate units under Society 2 using Society 1's building
		genCrossReq := society.GenerateUnitsParams{
			BuildingID:         uuid.MustParse(testBuildingID),
			StartingFloor:      1,
			TotalFloors:        2,
			UnitsPerFloor:      2,
			StartingUnitNumber: 101,
		}
		genCrossJSON, _ := json.Marshal(genCrossReq)
		rCrossGen := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+soc2ID+"/units/generate", bytes.NewReader(genCrossJSON))
		rCrossGen.Header.Set("Authorization", "Bearer "+superAdminToken)
		rCrossGen.Header.Set("Content-Type", "application/json")
		wCrossGen := httptest.NewRecorder()
		router.ServeHTTP(wCrossGen, rCrossGen)
		if wCrossGen.Code != http.StatusNotFound {
			t.Errorf("expected 404 BUILDING_NOT_FOUND when generating units for cross-society building, got %d: %s", wCrossGen.Code, wCrossGen.Body.String())
		}
	})

	// 14. Suspended society structure operations rejected
	t.Run("Suspended society structure mutations rejected with 403", func(t *testing.T) {
		// Suspend testSocID
		suspPayload := map[string]interface{}{
			"reason_code": "LEGAL_DISPUTE",
		}
		suspJSON, _ := json.Marshal(suspPayload)
		rSusp := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+testSocID+"/suspend", bytes.NewReader(suspJSON))
		rSusp.Header.Set("Authorization", "Bearer "+superAdminToken)
		rSusp.Header.Set("Content-Type", "application/json")
		wSusp := httptest.NewRecorder()
		router.ServeHTTP(wSusp, rSusp)
		if wSusp.Code != http.StatusOK {
			t.Fatalf("failed to suspend society: %d %s", wSusp.Code, wSusp.Body.String())
		}

		// Attempt to create building on suspended society
		bReq := society.CreateBuildingParams{
			Name:         "Tower Blocked",
			BuildingCode: "TB",
			TotalFloors:  5,
		}
		bJSON, _ := json.Marshal(bReq)
		rBlocked := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+testSocID+"/buildings", bytes.NewReader(bJSON))
		rBlocked.Header.Set("Authorization", "Bearer "+superAdminToken)
		rBlocked.Header.Set("Content-Type", "application/json")
		wBlocked := httptest.NewRecorder()
		router.ServeHTTP(wBlocked, rBlocked)
		if wBlocked.Code != http.StatusForbidden {
			t.Errorf("expected 403 SOCIETY_SUSPENDED for suspended society, got %d: %s", wBlocked.Code, wBlocked.Body.String())
		}

		// Reactivate for subsequent checks
		rReact := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+testSocID+"/reactivate", nil)
		rReact.Header.Set("Authorization", "Bearer "+superAdminToken)
		wReact := httptest.NewRecorder()
		router.ServeHTTP(wReact, rReact)
		if wReact.Code != http.StatusOK {
			t.Fatalf("failed to reactivate society: %d %s", wReact.Code, wReact.Body.String())
		}
	})

	// 15. Assigned SOC_ADMIN access verified & Platform ADMIN without assignment rejected
	t.Run("Assigned SOC_ADMIN permitted; unassigned platform ADMIN rejected", func(t *testing.T) {
		// Create a platform ADMIN user (not SUPER_ADMIN)
		platAdminID := uuid.New()
		platPhone := fmt.Sprintf("918%09d", time.Now().UnixNano()%1000000000)
		if err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (id, phone, status, created_at, updated_at)
			VALUES ($1, $2, 'ACTIVE', NOW(), NOW())
			ON CONFLICT (phone) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`, platAdminID, platPhone).Scan(&platAdminID); err != nil {
			t.Fatalf("failed to insert plat admin: %v", err)
		}
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status, created_at, updated_at)
			VALUES ($1, 'ADMIN', 'ACTIVE', NOW(), NOW())
			ON CONFLICT (user_id, role) DO UPDATE SET status = 'ACTIVE'
		`, platAdminID)
		platAdminToken := generateToken(platAdminID)

		// Regular platform ADMIN should be rejected (403 SOCIETY_ACCESS_DENIED)
		rPlat := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/buildings", nil)
		rPlat.Header.Set("Authorization", "Bearer "+platAdminToken)
		wPlat := httptest.NewRecorder()
		router.ServeHTTP(wPlat, rPlat)
		if wPlat.Code != http.StatusForbidden {
			t.Errorf("expected 403 for unassigned platform ADMIN, got %d: %s", wPlat.Code, wPlat.Body.String())
		}

		// Assign this user as SOC_ADMIN for testSocID
		socAdminID := uuid.New()
		socPhone := fmt.Sprintf("917%09d", (time.Now().UnixNano()+777)%1000000000)
		if err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (id, phone, status, created_at, updated_at)
			VALUES ($1, $2, 'ACTIVE', NOW(), NOW())
			ON CONFLICT (phone) DO UPDATE SET updated_at = NOW()
			RETURNING id
		`, socAdminID, socPhone).Scan(&socAdminID); err != nil {
			t.Fatalf("failed to insert soc admin: %v", err)
		}
		if _, err := db.Pool().Exec(ctx, `
			INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by, created_at, updated_at)
			VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $3, NOW(), NOW())
			ON CONFLICT (society_id, user_id, role) WHERE status = 'ACTIVE' DO UPDATE SET updated_at = NOW()
		`, uuid.MustParse(testSocID), socAdminID, superAdminRole.UserID); err != nil {
			t.Fatalf("failed to insert society_user_roles: %v", err)
		}
		socAdminToken := generateToken(socAdminID)

		// SOC_ADMIN should succeed
		rSoc := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+testSocID+"/buildings", nil)
		rSoc.Header.Set("Authorization", "Bearer "+socAdminToken)
		wSoc := httptest.NewRecorder()
		router.ServeHTTP(wSoc, rSoc)
		if wSoc.Code != http.StatusOK {
			t.Errorf("expected 200 for assigned SOC_ADMIN, got %d: %s", wSoc.Code, wSoc.Body.String())
		}
	})

	// 16. Transactional Audit Log Records Verified
	t.Run("Verify platform_audit_logs entries written for all structure mutations", func(t *testing.T) {
		expectedActions := []string{
			society.ActionBuildingCreated,
			society.ActionWingCreated,
			society.ActionUnitsGenerated,
			society.ActionGateCreated,
			society.ActionStaffingUpdated,
		}

		for _, action := range expectedActions {
			var count int
			err := db.Pool().QueryRow(ctx, `
				SELECT COUNT(*)
				FROM platform_audit_logs
				WHERE action = $1 AND details->>'society_id' = $2
			`, action, testSocID).Scan(&count)
			if err != nil {
				t.Fatalf("failed to query audit log for %s: %v", action, err)
			}
			if count < 1 {
				t.Errorf("expected at least 1 audit entry for %s, found %d", action, count)
			}
		}
	})
}
