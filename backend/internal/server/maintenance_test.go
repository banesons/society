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

func parseTestData(body []byte) map[string]interface{} {
	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(body, &resp)
	return resp.Data
}

func TestMaintenanceCategoryAndConfigLifecycle(t *testing.T) {
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

	// Create test society via API
	uniqueTime := time.Now().UnixNano()
	regNo := fmt.Sprintf("REG-MAINT-%d", uniqueTime%1000000)
	socParams := society.CreateSocietyParams{
		Name:               fmt.Sprintf("Maint Test Society %d", uniqueTime),
		RegistrationNumber: &regNo,
		AddressLine1:       "100 Financial Way",
		Locality:           "BKC",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400051",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)
	if wSoc.Code != http.StatusCreated {
		t.Fatalf("expected 201 created society, got %d: %s", wSoc.Code, wSoc.Body.String())
	}
	socData := parseTestData(wSoc.Body.Bytes())
	socID := uuid.MustParse(socData["id"].(string))

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)
	if wAct.Code != http.StatusOK {
		t.Fatalf("expected 200 activated, got %d: %s", wAct.Code, wAct.Body.String())
	}

	// 1. Create Charge Category
	catPayload := map[string]interface{}{
		"name":        "General Maintenance",
		"description": "Monthly upkeep fees",
	}
	body, _ := json.Marshal(catPayload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/categories", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for category creation, got %d: %s", rec.Code, rec.Body.String())
	}

	var catResp struct {
		Success bool `json:"success"`
		Data    struct {
			Category society.MaintenanceChargeCategory `json:"category"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &catResp)
	categoryID := catResp.Data.Category.ID

	// 2. Reject Duplicate Category Name
	reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/categories", socID), bytes.NewReader(body))
	reqDup.Header.Set("Authorization", "Bearer "+superToken)
	reqDup.Header.Set("Content-Type", "application/json")
	recDup := httptest.NewRecorder()
	router.ServeHTTP(recDup, reqDup)

	if recDup.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for duplicate category name, got %d", recDup.Code)
	}

	// 3. Update Category
	updatePayload := map[string]interface{}{
		"name":        "Monthly General Maintenance",
		"description": "Updated upkeep fees",
		"status":      "ACTIVE",
	}
	upBody, _ := json.Marshal(updatePayload)
	reqUp := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/societies/%s/maintenance/categories/%s", socID, categoryID), bytes.NewReader(upBody))
	reqUp.Header.Set("Authorization", "Bearer "+superToken)
	reqUp.Header.Set("Content-Type", "application/json")
	recUp := httptest.NewRecorder()
	router.ServeHTTP(recUp, reqUp)

	if recUp.Code != http.StatusOK {
		t.Fatalf("expected 200 for category update, got %d: %s", recUp.Code, recUp.Body.String())
	}

	// 4. Create Charge Config (Negative Amount Validation)
	invalidConfig := map[string]interface{}{
		"category_id":       categoryID,
		"name":              "Invalid Config",
		"charge_type":       "FLAT_RECURRING",
		"amount":            -50.0,
		"billing_frequency": "MONTHLY",
		"effective_from":    "2026-10-01",
	}
	invBody, _ := json.Marshal(invalidConfig)
	reqInv := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/configs", socID), bytes.NewReader(invBody))
	reqInv.Header.Set("Authorization", "Bearer "+superToken)
	reqInv.Header.Set("Content-Type", "application/json")
	recInv := httptest.NewRecorder()
	router.ServeHTTP(recInv, reqInv)

	if recInv.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for negative amount, got %d", recInv.Code)
	}

	// 5. Create Valid Charge Config
	validConfig := map[string]interface{}{
		"category_id":          categoryID,
		"name":                 "Standard Flat Fee",
		"charge_type":          "FLAT_RECURRING",
		"applicable_unit_type": "RESIDENTIAL",
		"amount":               2500.0,
		"billing_frequency":    "MONTHLY",
		"effective_from":       "2026-10-01",
	}
	valBody, _ := json.Marshal(validConfig)
	reqVal := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/configs", socID), bytes.NewReader(valBody))
	reqVal.Header.Set("Authorization", "Bearer "+superToken)
	reqVal.Header.Set("Content-Type", "application/json")
	recVal := httptest.NewRecorder()
	router.ServeHTTP(recVal, reqVal)

	if recVal.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid charge config, got %d: %s", recVal.Code, recVal.Body.String())
	}

	var cfgResp struct {
		Success bool `json:"success"`
		Data    struct {
			Config society.MaintenanceChargeConfig `json:"config"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recVal.Body.Bytes(), &cfgResp)
	configID := cfgResp.Data.Config.ID

	// 6. Deactivate Config
	reqDeact := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/configs/%s/deactivate", socID, configID), nil)
	reqDeact.Header.Set("Authorization", "Bearer "+superToken)
	recDeact := httptest.NewRecorder()
	router.ServeHTTP(recDeact, reqDeact)

	if recDeact.Code != http.StatusOK {
		t.Fatalf("expected 200 for deactivation, got %d: %s", recDeact.Code, recDeact.Body.String())
	}
}

func TestMaintenanceBillingPeriodAndGeneration(t *testing.T) {
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

	// Create test society via API
	uniqueTime := time.Now().UnixNano()
	regNo := fmt.Sprintf("REG-GEN-%d", uniqueTime%1000000)
	socParams := society.CreateSocietyParams{
		Name:               fmt.Sprintf("Gen Test Society %d", uniqueTime),
		RegistrationNumber: &regNo,
		AddressLine1:       "200 Billing Boulevard",
		Locality:           "Kothrud",
		City:               "Pune",
		State:              "Maharashtra",
		Pincode:            "411038",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)
	socData := parseTestData(wSoc.Body.Bytes())
	socID := uuid.MustParse(socData["id"].(string))

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	// Create building via API
	bldReq := map[string]interface{}{
		"name":          "Tower A",
		"building_code": fmt.Sprintf("TA%d", uniqueTime%10000),
		"total_floors":  2,
	}
	bldJSON, _ := json.Marshal(bldReq)
	reqBld := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", socID), bytes.NewReader(bldJSON))
	reqBld.Header.Set("Authorization", "Bearer "+superToken)
	reqBld.Header.Set("Content-Type", "application/json")
	wBld := httptest.NewRecorder()
	router.ServeHTTP(wBld, reqBld)
	bldData := parseTestData(wBld.Body.Bytes())
	bldID := bldData["id"].(string)

	// Generate 2 units
	genReq := society.GenerateUnitsParams{
		BuildingID:         uuid.MustParse(bldID),
		StartingFloor:      1,
		TotalFloors:        2,
		UnitsPerFloor:      1,
		StartingUnitNumber: 101,
		Prefix:             fmt.Sprintf("A%d", uniqueTime%1000),
	}
	genJSON, _ := json.Marshal(genReq)
	reqGenU := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/generate", socID), bytes.NewReader(genJSON))
	reqGenU.Header.Set("Authorization", "Bearer "+superToken)
	reqGenU.Header.Set("Content-Type", "application/json")
	wGenU := httptest.NewRecorder()
	router.ServeHTTP(wGenU, reqGenU)
	if wGenU.Code != http.StatusCreated {
		t.Fatalf("failed to generate units: %d %s", wGenU.Code, wGenU.Body.String())
	}

	// Create category & active config
	cat, err := societyRepo.CreateChargeCategory(ctx, socID, society.CreateChargeCategoryParams{
		Name: "Base Maintenance",
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	effFrom, _ := time.Parse("2006-01-02", "2026-10-01")
	_, err = societyRepo.CreateChargeConfig(ctx, socID, society.CreateChargeConfigParams{
		CategoryID:       cat.ID,
		Name:             "Monthly Standard Flat Fee",
		ChargeType:       "FLAT_RECURRING",
		Amount:           3000.00,
		BillingFrequency: "MONTHLY",
		EffectiveFrom:    effFrom,
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	// 1. Create Billing Period (Validation for invalid dates)
	invalidPeriod := map[string]interface{}{
		"period_name":  "Bad Period",
		"period_start": "2026-10-31",
		"period_end":   "2026-10-01", // end < start
		"due_date":     "2026-11-10",
	}
	invPBody, _ := json.Marshal(invalidPeriod)
	reqInvP := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods", socID), bytes.NewReader(invPBody))
	reqInvP.Header.Set("Authorization", "Bearer "+superToken)
	reqInvP.Header.Set("Content-Type", "application/json")
	recInvP := httptest.NewRecorder()
	router.ServeHTTP(recInvP, reqInvP)

	if recInvP.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for inverted period dates, got %d", recInvP.Code)
	}

	// 2. Create Valid Billing Period
	validPeriod := map[string]interface{}{
		"period_name":  "October 2026",
		"period_start": "2026-10-01",
		"period_end":   "2026-10-31",
		"due_date":     "2026-11-10",
	}
	valPBody, _ := json.Marshal(validPeriod)
	reqValP := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods", socID), bytes.NewReader(valPBody))
	reqValP.Header.Set("Authorization", "Bearer "+superToken)
	reqValP.Header.Set("Content-Type", "application/json")
	recValP := httptest.NewRecorder()
	router.ServeHTTP(recValP, reqValP)

	if recValP.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid period, got %d: %s", recValP.Code, recValP.Body.String())
	}

	var pResp struct {
		Success bool `json:"success"`
		Data    struct {
			Period society.MaintenanceBillingPeriod `json:"period"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recValP.Body.Bytes(), &pResp)
	periodID := pResp.Data.Period.ID

	// 3. Preview Bill Generation
	reqPrev := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods/%s/preview", socID, periodID), nil)
	reqPrev.Header.Set("Authorization", "Bearer "+superToken)
	recPrev := httptest.NewRecorder()
	router.ServeHTTP(recPrev, reqPrev)

	if recPrev.Code != http.StatusOK {
		t.Fatalf("expected 200 for preview, got %d: %s", recPrev.Code, recPrev.Body.String())
	}

	var prevResp struct {
		Success bool `json:"success"`
		Data    struct {
			Preview society.BillGenerationPreview `json:"preview"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recPrev.Body.Bytes(), &prevResp)
	if prevResp.Data.Preview.EligibleUnits != 2 {
		t.Fatalf("expected 2 eligible units, got %d", prevResp.Data.Preview.EligibleUnits)
	}
	if prevResp.Data.Preview.ProjectedTotalAmount != 6000.00 {
		t.Fatalf("expected projected total 6000.00, got %f", prevResp.Data.Preview.ProjectedTotalAmount)
	}

	// 4. Generate Bills
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods/%s/generate", socID, periodID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	recGen := httptest.NewRecorder()
	router.ServeHTTP(recGen, reqGen)

	if recGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for generation, got %d: %s", recGen.Code, recGen.Body.String())
	}

	var genResp struct {
		Success bool `json:"success"`
		Data    struct {
			Result society.BillGenerationResult `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recGen.Body.Bytes(), &genResp)
	if genResp.Data.Result.BillsCreated != 2 {
		t.Fatalf("expected 2 bills created, got %d", genResp.Data.Result.BillsCreated)
	}

	// 5. Test Idempotency: Repeating generation must create 0 duplicates!
	reqRepeat := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods/%s/generate", socID, periodID), nil)
	reqRepeat.Header.Set("Authorization", "Bearer "+superToken)
	recRepeat := httptest.NewRecorder()
	router.ServeHTTP(recRepeat, reqRepeat)

	if recRepeat.Code != http.StatusOK {
		t.Fatalf("expected 200 for repeated generation, got %d: %s", recRepeat.Code, recRepeat.Body.String())
	}

	var repResp struct {
		Success bool `json:"success"`
		Data    struct {
			Result society.BillGenerationResult `json:"result"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recRepeat.Body.Bytes(), &repResp)
	if repResp.Data.Result.BillsCreated != 0 {
		t.Fatalf("idempotency violation: expected 0 bills created on retry, got %d", repResp.Data.Result.BillsCreated)
	}
	if repResp.Data.Result.UnitsSkippedExisting != 2 {
		t.Fatalf("expected 2 units skipped as existing, got %d", repResp.Data.Result.UnitsSkippedExisting)
	}

	// 6. Query Admin Bills Register
	reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/bills", socID), nil)
	reqList.Header.Set("Authorization", "Bearer "+superToken)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 for bills list, got %d: %s", recList.Code, recList.Body.String())
	}

	var listResp struct {
		Success bool `json:"success"`
		Data    struct {
			Bills []society.MaintenanceBill `json:"bills"`
			Total int                       `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recList.Body.Bytes(), &listResp)
	if listResp.Data.Total != 2 || len(listResp.Data.Bills) != 2 {
		t.Fatalf("expected 2 bills in register, got total=%d len=%d", listResp.Data.Total, len(listResp.Data.Bills))
	}

	firstBill := listResp.Data.Bills[0]

	// 7. Get Bill Details with Line Items
	reqDet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/bills/%s", socID, firstBill.ID), nil)
	reqDet.Header.Set("Authorization", "Bearer "+superToken)
	recDet := httptest.NewRecorder()
	router.ServeHTTP(recDet, reqDet)

	if recDet.Code != http.StatusOK {
		t.Fatalf("expected 200 for bill details, got %d: %s", recDet.Code, recDet.Body.String())
	}

	var detResp struct {
		Success bool `json:"success"`
		Data    struct {
			Bill society.MaintenanceBill `json:"bill"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recDet.Body.Bytes(), &detResp)
	if len(detResp.Data.Bill.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(detResp.Data.Bill.LineItems))
	}
	if detResp.Data.Bill.LineItems[0].Amount != 3000.00 {
		t.Fatalf("expected line item amount 3000.00, got %f", detResp.Data.Bill.LineItems[0].Amount)
	}

	// 8. Void Bill
	voidPayload := map[string]interface{}{"reason": "Test administrative void"}
	vBody, _ := json.Marshal(voidPayload)
	reqVoid := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/bills/%s/void", socID, firstBill.ID), bytes.NewReader(vBody))
	reqVoid.Header.Set("Authorization", "Bearer "+superToken)
	reqVoid.Header.Set("Content-Type", "application/json")
	recVoid := httptest.NewRecorder()
	router.ServeHTTP(recVoid, reqVoid)

	if recVoid.Code != http.StatusOK {
		t.Fatalf("expected 200 for void bill, got %d: %s", recVoid.Code, recVoid.Body.String())
	}

	// 9. Prevent Double Void
	recVoid2 := httptest.NewRecorder()
	reqVoid2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/bills/%s/void", socID, firstBill.ID), bytes.NewReader(vBody))
	reqVoid2.Header.Set("Authorization", "Bearer "+superToken)
	reqVoid2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recVoid2, reqVoid2)

	if recVoid2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for double void, got %d", recVoid2.Code)
	}
}

func TestMaintenanceSecurityAndResidentAuthorization(t *testing.T) {
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

	// Create test society via API
	uniqueTime := time.Now().UnixNano()
	regNo := fmt.Sprintf("REG-SEC-%d", uniqueTime%1000000)
	socParams := society.CreateSocietyParams{
		Name:               fmt.Sprintf("Sec Maint Society %d", uniqueTime),
		RegistrationNumber: &regNo,
		AddressLine1:       "500 Authorization Avenue",
		Locality:           "Viman Nagar",
		City:               "Pune",
		State:              "Maharashtra",
		Pincode:            "411014",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)
	socData := parseTestData(wSoc.Body.Bytes())
	socID := uuid.MustParse(socData["id"].(string))

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	// Create building via API
	bldReq := map[string]interface{}{
		"name":          "Wing Alpha",
		"building_code": fmt.Sprintf("WA%d", uniqueTime%10000),
		"total_floors":  2,
	}
	bldJSON, _ := json.Marshal(bldReq)
	reqBld := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", socID), bytes.NewReader(bldJSON))
	reqBld.Header.Set("Authorization", "Bearer "+superToken)
	reqBld.Header.Set("Content-Type", "application/json")
	wBld := httptest.NewRecorder()
	router.ServeHTTP(wBld, reqBld)
	bldData := parseTestData(wBld.Body.Bytes())
	bldID := bldData["id"].(string)

	// Generate 2 units
	genReq := society.GenerateUnitsParams{
		BuildingID:         uuid.MustParse(bldID),
		StartingFloor:      1,
		TotalFloors:        2,
		UnitsPerFloor:      1,
		StartingUnitNumber: 101,
		Prefix:             fmt.Sprintf("W%d", uniqueTime%1000),
	}
	genJSON, _ := json.Marshal(genReq)
	reqGenU := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/generate", socID), bytes.NewReader(genJSON))
	reqGenU.Header.Set("Authorization", "Bearer "+superToken)
	reqGenU.Header.Set("Content-Type", "application/json")
	wGenU := httptest.NewRecorder()
	router.ServeHTTP(wGenU, reqGenU)
	if wGenU.Code != http.StatusCreated {
		t.Fatalf("failed to generate units: %d %s", wGenU.Code, wGenU.Body.String())
	}

	units, _, err := societyRepo.ListUnits(ctx, society.ListUnitsFilter{SocietyID: socID, Limit: 10})
	if err != nil || len(units) < 2 {
		t.Fatalf("failed to list generated units: %v", err)
	}
	unit1 := units[0]
	unit2 := units[1]

	// Register 2 resident users
	resUser1ID := uuid.New()
	phone1 := fmt.Sprintf("+919811%06d", uniqueTime%1000000)
	normPhone1, _ := auth.NormalizePhone(phone1)
	_, _ = db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')`, resUser1ID, normPhone1)

	resUser2ID := uuid.New()
	phone2 := fmt.Sprintf("+919812%06d", uniqueTime%1000000)
	normPhone2, _ := auth.NormalizePhone(phone2)
	_, _ = db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')`, resUser2ID, normPhone2)

	mem1, err := societyRepo.RegisterMember(ctx, socID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "Resident",
		Phone:     phone1,
		ClientIP:  "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to register member 1: %v", err)
	}
	_, _ = db.Pool().Exec(ctx, `UPDATE society_members SET user_id = $1 WHERE id = $2`, resUser1ID, mem1.ID)

	mem2, err := societyRepo.RegisterMember(ctx, socID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "Resident",
		Phone:     phone2,
		ClientIP:  "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to register member 2: %v", err)
	}
	_, _ = db.Pool().Exec(ctx, `UPDATE society_members SET user_id = $1 WHERE id = $2`, resUser2ID, mem2.ID)

	// Associate Resident 1 with Unit 1, Resident 2 with Unit 2
	_, _ = societyRepo.CreateUnitRelationship(ctx, socID, unit1.ID, superAdminRole.UserID, society.CreateUnitRelationshipParams{
		MemberID:         mem1.ID,
		RelationshipType: "PRIMARY_OWNER",
		IsOccupant:       true,
		StartDate:        "2026-01-01",
		ClientIP:         "127.0.0.1",
		UserAgent:        "test-agent",
	})

	_, _ = societyRepo.CreateUnitRelationship(ctx, socID, unit2.ID, superAdminRole.UserID, society.CreateUnitRelationshipParams{
		MemberID:         mem2.ID,
		RelationshipType: "PRIMARY_OWNER",
		IsOccupant:       true,
		StartDate:        "2026-01-01",
		ClientIP:         "127.0.0.1",
		UserAgent:        "test-agent",
	})

	// Generate bills for both units
	cat, _ := societyRepo.CreateChargeCategory(ctx, socID, society.CreateChargeCategoryParams{Name: "General Maintenance"}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	effFrom, _ := time.Parse("2006-01-02", "2026-10-01")
	_, _ = societyRepo.CreateChargeConfig(ctx, socID, society.CreateChargeConfigParams{
		CategoryID:       cat.ID,
		Name:             "Flat Rate",
		ChargeType:       "FLAT_RECURRING",
		Amount:           4000.0,
		BillingFrequency: "MONTHLY",
		EffectiveFrom:    effFrom,
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")

	periodStart, _ := time.Parse("2006-01-02", "2026-10-01")
	periodEnd, _ := time.Parse("2006-01-02", "2026-10-31")
	dueDate, _ := time.Parse("2006-01-02", "2026-11-10")
	period, _ := societyRepo.CreateBillingPeriod(ctx, socID, society.CreateBillingPeriodParams{
		PeriodName:  "October 2026",
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		DueDate:     dueDate,
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")

	_, err = societyRepo.GenerateBills(ctx, socID, period.ID, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to generate bills: %v", err)
	}

	bills, _ := societyRepo.ListBills(ctx, socID, society.ListBillsFilter{Limit: 10})
	var billUnit1, billUnit2 society.MaintenanceBill
	for _, b := range bills.Items {
		if b.UnitID == unit1.ID {
			billUnit1 = b
		} else if b.UnitID == unit2.ID {
			billUnit2 = b
		}
	}

	tokenRes1 := generateToken(resUser1ID)
	tokenRes2 := generateToken(resUser2ID)

	// 1. Resident 1 queries /my-bills: must see Unit 1 bill ONLY!
	reqMy1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills", socID), nil)
	reqMy1.Header.Set("Authorization", "Bearer "+tokenRes1)
	recMy1 := httptest.NewRecorder()
	router.ServeHTTP(recMy1, reqMy1)

	if recMy1.Code != http.StatusOK {
		t.Fatalf("expected 200 for resident my-bills, got %d: %s", recMy1.Code, recMy1.Body.String())
	}

	var my1Resp struct {
		Success bool `json:"success"`
		Data    struct {
			Bills []society.MaintenanceBill `json:"bills"`
			Total int                       `json:"total"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recMy1.Body.Bytes(), &my1Resp)
	if my1Resp.Data.Total != 1 || len(my1Resp.Data.Bills) != 1 {
		t.Fatalf("expected Resident 1 to see exactly 1 bill, got %d", my1Resp.Data.Total)
	}
	if my1Resp.Data.Bills[0].UnitID != unit1.ID {
		t.Fatalf("expected Resident 1 bill to be for Unit 1, got unit_id=%s", my1Resp.Data.Bills[0].UnitID)
	}

	// 2. IDOR Protection: Resident 1 attempts to query Resident 2's bill details -> must return 403 Forbidden!
	reqIDOR := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s", socID, billUnit2.ID), nil)
	reqIDOR.Header.Set("Authorization", "Bearer "+tokenRes1)
	recIDOR := httptest.NewRecorder()
	router.ServeHTTP(recIDOR, reqIDOR)

	if recIDOR.Code != http.StatusForbidden {
		t.Fatalf("security violation: expected 403 Forbidden for cross-unit bill access, got %d", recIDOR.Code)
	}

	// 3. Resident 1 queries their own bill details -> 200 OK
	reqOwn1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s", socID, billUnit1.ID), nil)
	reqOwn1.Header.Set("Authorization", "Bearer "+tokenRes1)
	recOwn1 := httptest.NewRecorder()
	router.ServeHTTP(recOwn1, reqOwn1)

	if recOwn1.Code != http.StatusOK {
		t.Fatalf("expected 200 for resident 1 accessing own bill, got %d: %s", recOwn1.Code, recOwn1.Body.String())
	}

	// 4. Resident 2 queries their own bill details -> 200 OK
	reqOwn2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s", socID, billUnit2.ID), nil)
	reqOwn2.Header.Set("Authorization", "Bearer "+tokenRes2)
	recOwn2 := httptest.NewRecorder()
	router.ServeHTTP(recOwn2, reqOwn2)

	if recOwn2.Code != http.StatusOK {
		t.Fatalf("expected 200 for resident 2 accessing own bill, got %d: %s", recOwn2.Code, recOwn2.Body.String())
	}

	// 4. Resident attempts to generate bills or create categories -> 403 Forbidden
	reqUnauthGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods/%s/generate", socID, period.ID), nil)
	reqUnauthGen.Header.Set("Authorization", "Bearer "+tokenRes1)
	recUnauthGen := httptest.NewRecorder()
	router.ServeHTTP(recUnauthGen, reqUnauthGen)

	if recUnauthGen.Code != http.StatusForbidden {
		t.Fatalf("security violation: expected 403 Forbidden for resident accessing admin generation, got %d", recUnauthGen.Code)
	}
}
