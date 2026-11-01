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

func setupBillingTestEnv(t *testing.T) (
	*config.Config,
	*database.DB,
	http.Handler,
	*platform.Repository,
	*society.Repository,
	uuid.UUID,
	string,
	uuid.UUID,
	func(uuid.UUID) string,
) {
	t.Helper()
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
		sessCtx, cancelSess := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelSess()
		_, err := db.Pool().Exec(sessCtx, `
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

	// Create test society
	uniqueTime := time.Now().UnixNano()
	regNo := fmt.Sprintf("REG-BILL-%d", uniqueTime%1000000)
	socParams := society.CreateSocietyParams{
		Name:               fmt.Sprintf("Billing Test Society %d", uniqueTime),
		RegistrationNumber: &regNo,
		AddressLine1:       "500 Invoice Avenue",
		Locality:           "Nariman Point",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400021",
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

	return cfg, db, router, platformRepo, societyRepo, socID, superToken, superAdminRole.UserID, generateToken
}

func TestConfigurableBillingTemplateLifecycle(t *testing.T) {
	_, _, router, _, _, socID, superToken, _, _ := setupBillingTestEnv(t)

	// 1. Create Template
	createPayload := map[string]interface{}{
		"name":                 "Standard Society Invoice",
		"description":          "Standard monthly maintenance billing template",
		"billing_frequency":    "MONTHLY",
		"currency":             "INR",
		"payment_instructions": "Please pay via NEFT/RTGS to HDFC Bank",
		"footer_text":          "Cheques subject to realization. 2% interest per month on delayed payment.",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/templates", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for template creation, got %d: %s", w.Code, w.Body.String())
	}

	tplData := parseTestData(w.Body.Bytes())
	templateID := tplData["id"].(string)
	if tplData["status"] != "DRAFT" {
		t.Fatalf("expected initial status DRAFT, got %v", tplData["status"])
	}

	// 2. Retrieve Template
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s", socID, templateID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+superToken)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 for template get, got %d: %s", wGet.Code, wGet.Body.String())
	}

	// 3. Update Draft Template
	newDesc := "Updated monthly billing template description"
	updatePayload := map[string]interface{}{
		"description": newDesc,
	}
	updBody, _ := json.Marshal(updatePayload)
	reqUpd := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s", socID, templateID), bytes.NewReader(updBody))
	reqUpd.Header.Set("Authorization", "Bearer "+superToken)
	reqUpd.Header.Set("Content-Type", "application/json")
	wUpd := httptest.NewRecorder()
	router.ServeHTTP(wUpd, reqUpd)
	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 for template update, got %d: %s", wUpd.Code, wUpd.Body.String())
	}
	updData := parseTestData(wUpd.Body.Bytes())
	if updData["description"] != newDesc {
		t.Fatalf("expected description %q, got %v", newDesc, updData["description"])
	}

	// 4. Duplicate Template
	dupPayload := map[string]interface{}{
		"new_name": "Standard Society Invoice (Duplicated)",
	}
	dupBody, _ := json.Marshal(dupPayload)
	reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s/duplicate", socID, templateID), bytes.NewReader(dupBody))
	reqDup.Header.Set("Authorization", "Bearer "+superToken)
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	router.ServeHTTP(wDup, reqDup)
	if wDup.Code != http.StatusCreated {
		t.Fatalf("expected 201 for template duplicate, got %d: %s", wDup.Code, wDup.Body.String())
	}

	// 5. Publish Template
	reqPub := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s/publish", socID, templateID), nil)
	reqPub.Header.Set("Authorization", "Bearer "+superToken)
	wPub := httptest.NewRecorder()
	router.ServeHTTP(wPub, reqPub)
	if wPub.Code != http.StatusOK {
		t.Fatalf("expected 200 for template publish, got %d: %s", wPub.Code, wPub.Body.String())
	}
	pubData := parseTestData(wPub.Body.Bytes())
	if pubData["version"] != float64(1) {
		t.Fatalf("expected published version 1, got %v", pubData["version"])
	}

	// 6. Immutability: Modifying an active/published template is rejected with 409 Conflict
	modDesc := "Modifying published template directly should be rejected"
	modPayload := map[string]interface{}{
		"description": modDesc,
	}
	modBody, _ := json.Marshal(modPayload)
	reqMod := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s", socID, templateID), bytes.NewReader(modBody))
	reqMod.Header.Set("Authorization", "Bearer "+superToken)
	reqMod.Header.Set("Content-Type", "application/json")
	wMod := httptest.NewRecorder()
	router.ServeHTTP(wMod, reqMod)
	if wMod.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when updating published template, got %d: %s", wMod.Code, wMod.Body.String())
	}

	// 7. List Versions
	reqVer := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s/versions", socID, templateID), nil)
	reqVer.Header.Set("Authorization", "Bearer "+superToken)
	wVer := httptest.NewRecorder()
	router.ServeHTTP(wVer, reqVer)
	if wVer.Code != http.StatusOK {
		t.Fatalf("expected 200 for listing versions, got %d: %s", wVer.Code, wVer.Body.String())
	}

	// 8. Archive Template
	reqArch := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/templates/%s/archive", socID, templateID), nil)
	reqArch.Header.Set("Authorization", "Bearer "+superToken)
	wArch := httptest.NewRecorder()
	router.ServeHTTP(wArch, reqArch)
	if wArch.Code != http.StatusOK {
		t.Fatalf("expected 200 for template archive, got %d: %s", wArch.Code, wArch.Body.String())
	}
}

func TestConfigurableBillingChargeHeads(t *testing.T) {
	_, _, router, _, _, socID, superToken, _, _ := setupBillingTestEnv(t)

	// 1. Create Area-based charge head
	ch1Payload := map[string]interface{}{
		"name":               "Square Foot Maintenance",
		"code":               "MAINT_AREA",
		"description":        "Monthly maintenance charged per square foot",
		"sac_code":           "999511",
		"tax_classification": "TAXABLE",
		"default_tax_rate":   18.0,
		"calculation_method": "AREA",
		"display_order":      1,
		"is_active":          true,
	}
	b1, _ := json.Marshal(ch1Payload)
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/charge-heads", socID), bytes.NewReader(b1))
	req1.Header.Set("Authorization", "Bearer "+superToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for charge head 1, got %d: %s", w1.Code, w1.Body.String())
	}
	ch1Data := parseTestData(w1.Body.Bytes())
	headID := ch1Data["id"].(string)

	// 2. Create Sinking Fund (Exempt, Fixed)
	ch2Payload := map[string]interface{}{
		"name":               "Sinking Fund",
		"code":               "SINKING_FUND",
		"description":        "Statutory reserve fund",
		"sac_code":           "999512",
		"tax_classification": "EXEMPT",
		"default_tax_rate":   0.0,
		"calculation_method": "FIXED",
		"display_order":      2,
		"is_active":          true,
	}
	b2, _ := json.Marshal(ch2Payload)
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/charge-heads", socID), bytes.NewReader(b2))
	req2.Header.Set("Authorization", "Bearer "+superToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for charge head 2, got %d: %s", w2.Code, w2.Body.String())
	}

	// 3. List Charge Heads
	reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/charge-heads", socID), nil)
	reqList.Header.Set("Authorization", "Bearer "+superToken)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 for list charge heads, got %d: %s", wList.Code, wList.Body.String())
	}

	// 4. Update Charge Head
	updDesc := "Updated sinking fund description"
	updPayload := map[string]interface{}{
		"description": updDesc,
	}
	bUpd, _ := json.Marshal(updPayload)
	reqUpd := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/billing/charge-heads/%s", socID, headID), bytes.NewReader(bUpd))
	reqUpd.Header.Set("Authorization", "Bearer "+superToken)
	reqUpd.Header.Set("Content-Type", "application/json")
	wUpd := httptest.NewRecorder()
	router.ServeHTTP(wUpd, reqUpd)
	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 for charge head update, got %d: %s", wUpd.Code, wUpd.Body.String())
	}

	// 5. Archive Charge Head
	reqArch := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/societies/%s/billing/charge-heads/%s", socID, headID), nil)
	reqArch.Header.Set("Authorization", "Bearer "+superToken)
	wArch := httptest.NewRecorder()
	router.ServeHTTP(wArch, reqArch)
	if wArch.Code != http.StatusOK {
		t.Fatalf("expected 200 for charge head archive, got %d: %s", wArch.Code, wArch.Body.String())
	}
}

func TestConfigurableBillingBatchPipelineAndOverrides(t *testing.T) {
	_, db, router, _, societyRepo, socID, superToken, adminUserID, _ := setupBillingTestEnv(t)
	ctx := context.Background()

	// 1. Create Building & Units
	uniqueTime := time.Now().UnixNano()
	bldReq := map[string]interface{}{
		"name":          "Block Platinum",
		"building_code": fmt.Sprintf("BP%d", uniqueTime%10000),
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

	genReq := society.GenerateUnitsParams{
		BuildingID:         uuid.MustParse(bldID),
		StartingFloor:      1,
		TotalFloors:        1,
		UnitsPerFloor:      2,
		StartingUnitNumber: 101,
		Prefix:             fmt.Sprintf("P%d", uniqueTime%1000),
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

	// Fetch generated units
	reqListU := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/units", socID), nil)
	reqListU.Header.Set("Authorization", "Bearer "+superToken)
	wListU := httptest.NewRecorder()
	router.ServeHTTP(wListU, reqListU)
	var listResp struct {
		Success bool `json:"success"`
		Data    struct {
			Units []struct {
				ID string `json:"id"`
			} `json:"units"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wListU.Body.Bytes(), &listResp)
	if len(listResp.Data.Units) < 2 {
		t.Fatalf("expected at least 2 units, got %d", len(listResp.Data.Units))
	}
	unit1ID := uuid.MustParse(listResp.Data.Units[0].ID)
	unit2ID := uuid.MustParse(listResp.Data.Units[1].ID)

	// 2. Create Charge Heads
	sac1 := "999511"
	ch1, err := societyRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Name:              "Base Maintenance Fee",
		Code:              "BASE_MAINT",
		SACCode:           &sac1,
		TaxClassification: "TAXABLE",
		DefaultTaxRate:    18.0,
		CalculationMethod: "AREA",
		DisplayOrder:      1,
		IsActive:          true,
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create charge head 1: %v", err)
	}

	sac2 := "999512"
	ch2, err := societyRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Name:              "Sinking Fund",
		Code:              "SINKING_FUND",
		SACCode:           &sac2,
		TaxClassification: "EXEMPT",
		DefaultTaxRate:    0.0,
		CalculationMethod: "FIXED",
		DisplayOrder:      2,
		IsActive:          true,
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create charge head 2: %v", err)
	}

	// 3. Create & Publish Template
	tpl, err := societyRepo.CreateBillTemplate(ctx, socID, society.CreateBillTemplateParams{
		Name:             fmt.Sprintf("Residential Monthly Maintenance %d", uniqueTime),
		BillingFrequency: "MONTHLY",
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	pubTpl, err := societyRepo.PublishBillTemplate(ctx, socID, tpl.ID, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to publish template: %v", err)
	}

	// 4. Unit Overrides on Unit 2:
	// a) Head 1: Rate override of 2.80 per sq ft (1000 * 2.80 = 2800)
	rateVal := 2.80
	_, err = societyRepo.CreateUnitBillingOverride(ctx, socID, unit2ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:  ch1.ID,
		OverrideType:  "RATE_OVERRIDE",
		OverrideValue: &rateVal,
		Reason:        "Special discounted rate for ground floor unit",
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create unit rate override: %v", err)
	}

	// b) Head 2: Exemption from Sinking Fund
	_, err = societyRepo.CreateUnitBillingOverride(ctx, socID, unit2ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID: ch2.ID,
		OverrideType: "EXEMPTION",
		Reason:       "Builder unit statutory exemption agreement",
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create unit exemption override: %v", err)
	}

	// 5. Create Billing Period
	validPeriod := map[string]interface{}{
		"period_name":  fmt.Sprintf("Nov 2026 Batch %d", uniqueTime%10000),
		"period_start": "2026-11-01",
		"period_end":   "2026-11-30",
		"due_date":     "2026-12-10",
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
	pData := parseTestData(recValP.Body.Bytes())
	periodMap := pData["period"].(map[string]interface{})
	periodID := periodMap["id"].(string)

	// 6. Create Billing Batch
	batchReq := map[string]interface{}{
		"billing_period_id": periodID,
		"template_id":       pubTpl.TemplateID.String(),
		"batch_name":        "November 2026 Maintenance Billing Run",
		"notes":             "Automated batch generation test",
	}
	batchJSON, _ := json.Marshal(batchReq)
	reqBatch := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(batchJSON))
	reqBatch.Header.Set("Authorization", "Bearer "+superToken)
	reqBatch.Header.Set("Content-Type", "application/json")
	wBatch := httptest.NewRecorder()
	router.ServeHTTP(wBatch, reqBatch)
	if wBatch.Code != http.StatusCreated {
		t.Fatalf("expected 201 for billing batch create, got %d: %s", wBatch.Code, wBatch.Body.String())
	}
	batchData := parseTestData(wBatch.Body.Bytes())
	batchID := batchData["id"].(string)

	// 7. Preview Batch Calculations
	reqPrev := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/preview", socID, batchID), nil)
	reqPrev.Header.Set("Authorization", "Bearer "+superToken)
	wPrev := httptest.NewRecorder()
	router.ServeHTTP(wPrev, reqPrev)
	if wPrev.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch preview, got %d: %s", wPrev.Code, wPrev.Body.String())
	}

	var prevResp struct {
		Success bool `json:"success"`
		Data    struct {
			EligibleUnitCount int     `json:"eligible_unit_count"`
			TotalPayable      float64 `json:"total_payable"`
			TotalCGST         float64 `json:"total_cgst"`
			TotalSGST         float64 `json:"total_sgst"`
			Units             []struct {
				UnitID        string  `json:"unit_id"`
				Subtotal      float64 `json:"subtotal"`
				TaxableAmount float64 `json:"taxable_amount"`
				TotalTax      float64 `json:"total_tax_amount"`
				PayableAmount float64 `json:"payable_amount"`
			} `json:"units"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wPrev.Body.Bytes(), &prevResp)

	if prevResp.Data.EligibleUnitCount != 2 {
		t.Fatalf("expected 2 units in preview, got %d", prevResp.Data.EligibleUnitCount)
	}

	// Verify calculations:
	// Unit 1: 1000 sq ft * 3.50 = 3500 (taxable at 18% -> 315 CGST + 315 SGST = 630 tax). Sinking fund = 500 (exempt).
	// Subtotal = 4000. Total tax = 630. Total payable = 4630.
	// Unit 2: 1000 sq ft * 2.80 = 2800 (taxable at 18% -> 252 CGST + 252 SGST = 504 tax). Sinking fund = 0 (exempt override).
	// Subtotal = 2800. Total tax = 504. Total payable = 3304.
	for _, u := range prevResp.Data.Units {
		if u.UnitID == unit1ID.String() {
			if u.Subtotal != 4000.00 {
				t.Errorf("Unit 1 expected subtotal 4000.00, got %.2f", u.Subtotal)
			}
			if u.TaxableAmount != 3500.00 {
				t.Errorf("Unit 1 expected taxable amount 3500.00, got %.2f", u.TaxableAmount)
			}
			if u.TotalTax != 630.00 {
				t.Errorf("Unit 1 expected total tax 630.00, got %.2f", u.TotalTax)
			}
			if u.PayableAmount != 4630.00 {
				t.Errorf("Unit 1 expected payable amount 4630.00, got %.2f", u.PayableAmount)
			}
		} else if u.UnitID == unit2ID.String() {
			if u.Subtotal != 2800.00 {
				t.Errorf("Unit 2 expected subtotal 2800.00 (with exempt sinking fund), got %.2f", u.Subtotal)
			}
			if u.TaxableAmount != 2800.00 {
				t.Errorf("Unit 2 expected taxable amount 2800.00, got %.2f", u.TaxableAmount)
			}
			if u.TotalTax != 504.00 {
				t.Errorf("Unit 2 expected total tax 504.00, got %.2f", u.TotalTax)
			}
			if u.PayableAmount != 3304.00 {
				t.Errorf("Unit 2 expected payable amount 3304.00, got %.2f", u.PayableAmount)
			}
		}
	}

	// 8. Generate Draft Bills
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch draft generate, got %d: %s", wGen.Code, wGen.Body.String())
	}
	genData := parseTestData(wGen.Body.Bytes())
	if genData["status"] != "GENERATED" {
		t.Fatalf("expected batch status GENERATED, got %v", genData["status"])
	}

	// Verify bills in database are DRAFT
	var billCount int
	err = db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_bills WHERE batch_id = $1 AND status = 'DRAFT'`, batchID).Scan(&billCount)
	if err != nil || billCount != 2 {
		t.Fatalf("expected 2 draft bills in database, got count %d, err %v", billCount, err)
	}

	// 9. Finalize Batch
	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}
	finData := parseTestData(wFin.Body.Bytes())
	if finData["status"] != "FINALIZED" {
		t.Fatalf("expected batch status FINALIZED, got %v", finData["status"])
	}

	// Verify bills transitioned to ISSUED
	var issuedCount int
	err = db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_bills WHERE batch_id = $1 AND status = 'ISSUED'`, batchID).Scan(&issuedCount)
	if err != nil || issuedCount != 2 {
		t.Fatalf("expected 2 issued bills in database, got count %d, err %v", issuedCount, err)
	}

	// 10. Immutability Trigger Test: Tampering with finalized bill must fail via DB trigger
	var billID uuid.UUID
	err = db.Pool().QueryRow(ctx, `SELECT id FROM maintenance_bills WHERE batch_id = $1 LIMIT 1`, batchID).Scan(&billID)
	if err != nil {
		t.Fatalf("failed to query bill id: %v", err)
	}
	_, tamperErr := db.Pool().Exec(ctx, `UPDATE maintenance_bills SET total_payable = 1.00 WHERE id = $1`, billID)
	if tamperErr == nil {
		t.Fatalf("expected database immutability trigger to reject updating finalized bill")
	}

	// 11. Duplicate Billing Prevention: Unique constraint blocks generating another active bill for same unit & period
	_, dupErr := db.Pool().Exec(ctx, `
		INSERT INTO maintenance_bills (
			society_id, billing_period_id, unit_id, bill_number, bill_type, bill_date, due_date,
			charge_subtotal, cgst_amount, sgst_amount, igst_amount, total_tax, total_payable, status
		) VALUES (
			$1, $2, $3, 'DUP-BILL-001', 'REGULAR', CURRENT_DATE, CURRENT_DATE + INTERVAL '10 days',
			100, 0, 0, 0, 0, 100, 'DRAFT'
		)
	`, socID, periodID, unit1ID)
	if dupErr == nil {
		t.Fatalf("expected partial unique index uq_idx_maintenance_bills_active_unit_period_type to block duplicate active bill")
	}
}

func TestConfigurableBillingAuthorization(t *testing.T) {
	_, db, router, _, _, socID, _, _, generateToken := setupBillingTestEnv(t)
	ctx := context.Background()

	// Create a regular user who is NOT an admin of this society
	nonAdminUserID := uuid.New()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO users (id, phone, status)
		VALUES ($1, $2, 'ACTIVE')
	`, nonAdminUserID, fmt.Sprintf("+9198%08d", time.Now().UnixNano()%100000000))
	if err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	userToken := generateToken(nonAdminUserID)

	// Attempt to list templates as regular non-admin user
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/templates", socID), nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 403 Forbidden or 401 Unauthorized for non-admin, got %d: %s", w.Code, w.Body.String())
	}
}
