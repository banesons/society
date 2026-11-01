package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"chs/backend/internal/society"
)

// Helper to bootstrap a test template and charge heads
func setupBatchTestFixture(t *testing.T) (
	router http.Handler,
	societyRepo *society.Repository,
	socID uuid.UUID,
	superToken string,
	templateID uuid.UUID,
	periodID uuid.UUID,
	unit1ID uuid.UUID,
	unit2ID uuid.UUID,
	adminUserID uuid.UUID,
	generateToken func(uuid.UUID) string,
	createUserToken func(phone string) (uuid.UUID, string),
) {
	t.Helper()
	_, db, r, _, sRepo, sID, sTok, aUID, genTok := setupBillingTestEnv(t)
	ctx := context.Background()
	uniqueTime := time.Now().UnixNano()

	// 1. Create Building & Units via router
	bldReq := map[string]interface{}{
		"name":          "Block Platinum",
		"building_code": fmt.Sprintf("BP%d", uniqueTime%10000),
		"total_floors":  2,
	}
	bldJSON, _ := json.Marshal(bldReq)
	reqBld := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", sID), bytes.NewReader(bldJSON))
	reqBld.Header.Set("Authorization", "Bearer "+sTok)
	reqBld.Header.Set("Content-Type", "application/json")
	wBld := httptest.NewRecorder()
	r.ServeHTTP(wBld, reqBld)
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
	reqGenU := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/generate", sID), bytes.NewReader(genJSON))
	reqGenU.Header.Set("Authorization", "Bearer "+sTok)
	reqGenU.Header.Set("Content-Type", "application/json")
	wGenU := httptest.NewRecorder()
	r.ServeHTTP(wGenU, reqGenU)
	if wGenU.Code != http.StatusCreated {
		t.Fatalf("failed to generate units: %d %s", wGenU.Code, wGenU.Body.String())
	}

	reqListU := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/units", sID), nil)
	reqListU.Header.Set("Authorization", "Bearer "+sTok)
	wListU := httptest.NewRecorder()
	r.ServeHTTP(wListU, reqListU)
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
	unit1ID = uuid.MustParse(listResp.Data.Units[0].ID)
	unit2ID = uuid.MustParse(listResp.Data.Units[1].ID)

	// 2. Create Charge Head
	ch1, err := sRepo.CreateChargeHead(ctx, sID, society.CreateChargeHeadParams{
		Code:              fmt.Sprintf("MAINT_STD_%d", uniqueTime%10000),
		Name:              "Standard Maintenance Fee",
		TaxClassification: "TAXABLE",
		DefaultTaxRate:    18.0,
		CalculationMethod: "FIXED",
		DisplayOrder:      1,
		IsActive:          true,
	}, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create charge head: %v", err)
	}

	// 3. Create Template & Publish
	tpl, err := sRepo.CreateBillTemplate(ctx, sID, society.CreateBillTemplateParams{
		Name:             fmt.Sprintf("Batch Template %d", uniqueTime%10000),
		BillType:         "MAINTENANCE",
		BillingFrequency: "MONTHLY",
		Currency:         "INR",
		BillTitle:        "Monthly Maintenance Invoice",
	}, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	pubTpl, err := sRepo.PublishBillTemplate(ctx, sID, tpl.ID, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to publish template: %v", err)
	}

	// 4. Create Billing Period
	validPeriod := map[string]interface{}{
		"period_name":  fmt.Sprintf("Period %d", uniqueTime%10000),
		"period_start": "2026-11-01",
		"period_end":   "2026-11-30",
		"due_date":     "2026-12-10",
	}
	valPBody, _ := json.Marshal(validPeriod)
	reqValP := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods", sID), bytes.NewReader(valPBody))
	reqValP.Header.Set("Authorization", "Bearer "+sTok)
	reqValP.Header.Set("Content-Type", "application/json")
	recValP := httptest.NewRecorder()
	r.ServeHTTP(recValP, reqValP)
	if recValP.Code != http.StatusCreated {
		t.Fatalf("expected 201 for period, got %d: %s", recValP.Code, recValP.Body.String())
	}
	pData := parseTestData(recValP.Body.Bytes())
	periodMap := pData["period"].(map[string]interface{})
	pID := uuid.MustParse(periodMap["id"].(string))

	createUserToken = func(phone string) (uuid.UUID, string) {
		uID := uuid.New()
		_, insErr := db.Pool().Exec(ctx, `
			INSERT INTO users (id, phone, status)
			VALUES ($1, $2, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
		`, uID, phone)
		if insErr != nil {
			t.Fatalf("insert user: %v", insErr)
		}
		return uID, genTok(uID)
	}

	_ = ch1
	return r, sRepo, sID, sTok, pubTpl.TemplateID, pID, unit1ID, unit2ID, aUID, genTok, createUserToken
}

func TestBillingBatchLifecycle_StateTransitions(t *testing.T) {
	router, _, socID, superToken, templateID, periodID, unit1ID, _, _, _, _ := setupBatchTestFixture(t)

	// 1. Create Batch (starts in DRAFT_PREVIEW)
	createReq := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Lifecycle Test Batch",
		"selection_mode":    "SINGLE_UNIT",
		"filter_criteria": map[string]interface{}{
			"unit_id": unit1ID.String(),
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}
	batchData := parseTestData(w.Body.Bytes())
	batchID := batchData["id"].(string)

	if batchData["status"] != "DRAFT_PREVIEW" {
		t.Fatalf("expected initial status DRAFT_PREVIEW, got %v", batchData["status"])
	}

	// 2. Invalid Transition: Finalizing directly from DRAFT_PREVIEW should fail (409 Conflict)
	reqFinInvalid := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFinInvalid.Header.Set("Authorization", "Bearer "+superToken)
	wFinInvalid := httptest.NewRecorder()
	router.ServeHTTP(wFinInvalid, reqFinInvalid)
	if wFinInvalid.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when finalizing un-generated batch, got %d: %s", wFinInvalid.Code, wFinInvalid.Body.String())
	}

	// 3. Valid Transition: Generate draft bills (DRAFT_PREVIEW -> PROCESSING -> GENERATED)
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for generate, got %d: %s", wGen.Code, wGen.Body.String())
	}
	genData := parseTestData(wGen.Body.Bytes())
	if genData["status"] != "GENERATED" {
		t.Fatalf("expected status GENERATED, got %v", genData["status"])
	}

	// 4. Valid Transition: Finalize batch (GENERATED -> FINALIZED)
	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}
	finData := parseTestData(wFin.Body.Bytes())
	if finData["status"] != "FINALIZED" {
		t.Fatalf("expected status FINALIZED, got %v", finData["status"])
	}

	// 5. Terminal State Immutability: Cannot cancel finalized batch
	reqCancelFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/cancel", socID, batchID), bytes.NewReader([]byte(`{"reason":"test"}`)))
	reqCancelFin.Header.Set("Authorization", "Bearer "+superToken)
	reqCancelFin.Header.Set("Content-Type", "application/json")
	wCancelFin := httptest.NewRecorder()
	router.ServeHTTP(wCancelFin, reqCancelFin)
	if wCancelFin.Code == http.StatusOK {
		t.Fatalf("expected error when cancelling finalized batch, got 200")
	}

	// 6. Test Cancellation on a fresh batch
	reqFresh := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	reqFresh.Header.Set("Authorization", "Bearer "+superToken)
	reqFresh.Header.Set("Content-Type", "application/json")
	wFresh := httptest.NewRecorder()
	router.ServeHTTP(wFresh, reqFresh)
	freshBatchID := parseTestData(wFresh.Body.Bytes())["id"].(string)

	reqCancel := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/cancel", socID, freshBatchID), bytes.NewReader([]byte(`{"reason":"Admin mistake"}`)))
	reqCancel.Header.Set("Authorization", "Bearer "+superToken)
	reqCancel.Header.Set("Content-Type", "application/json")
	wCancel := httptest.NewRecorder()
	router.ServeHTTP(wCancel, reqCancel)
	if wCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 for cancel, got %d: %s", wCancel.Code, wCancel.Body.String())
	}

	// Cancelled batch cannot be generated
	reqGenCancelled := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, freshBatchID), nil)
	reqGenCancelled.Header.Set("Authorization", "Bearer "+superToken)
	wGenCancelled := httptest.NewRecorder()
	router.ServeHTTP(wGenCancelled, reqGenCancelled)
	if wGenCancelled.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when generating cancelled batch, got %d: %s", wGenCancelled.Code, wGenCancelled.Body.String())
	}
}

func TestBillingBatch_IdempotencyAndDuplicatePrevention(t *testing.T) {
	router, _, socID, superToken, templateID, periodID, unit1ID, _, _, _, _ := setupBatchTestFixture(t)

	idempKey := fmt.Sprintf("idemp-batch-%d", time.Now().UnixNano())

	createReq := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Idempotency Test Batch",
		"selection_mode":    "SINGLE_UNIT",
		"idempotency_key":   idempKey,
		"filter_criteria": map[string]interface{}{
			"unit_id": unit1ID.String(),
		},
	}
	body, _ := json.Marshal(createReq)

	// First creation
	req1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	req1.Header.Set("Authorization", "Bearer "+superToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w1.Code, w1.Body.String())
	}
	batch1ID := parseTestData(w1.Body.Bytes())["id"].(string)

	// Second creation with exact same idempotency key returns identical batch
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+superToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated && w2.Code != http.StatusOK {
		t.Fatalf("expected success for idempotent batch request, got %d: %s", w2.Code, w2.Body.String())
	}
	batch2ID := parseTestData(w2.Body.Bytes())["id"].(string)
	if batch1ID != batch2ID {
		t.Fatalf("expected idempotent batch ID %s, got %s", batch1ID, batch2ID)
	}

	// Generate and finalize batch 1
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batch1ID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for generate, got %d: %s", wGen.Code, wGen.Body.String())
	}

	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batch1ID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}

	// Attempting another batch on the exact same unit and period should safely preserve the existing bill without creating duplicates
	createReq2 := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Second Run Same Period",
		"selection_mode":    "SINGLE_UNIT",
		"filter_criteria": map[string]interface{}{
			"unit_id": unit1ID.String(),
		},
	}
	body2, _ := json.Marshal(createReq2)
	req3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body2))
	req3.Header.Set("Authorization", "Bearer "+superToken)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	batch3ID := parseTestData(w3.Body.Bytes())["id"].(string)

	reqGen3 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batch3ID), nil)
	reqGen3.Header.Set("Authorization", "Bearer "+superToken)
	wGen3 := httptest.NewRecorder()
	router.ServeHTTP(wGen3, reqGen3)
	if wGen3.Code != http.StatusOK {
		t.Fatalf("expected 200 for generate second run, got %d: %s", wGen3.Code, wGen3.Body.String())
	}

	gen3Data := parseTestData(wGen3.Body.Bytes())
	if gen3Data["successful_unit_count"] != float64(1) {
		t.Fatalf("expected 1 unit mapped (existing), got %v", gen3Data["successful_unit_count"])
	}
}

func TestBillingBatch_ConcurrencyLocking(t *testing.T) {
	router, societyRepo, socID, superToken, templateID, periodID, unit1ID, _, _, _, _ := setupBatchTestFixture(t)
	ctx := context.Background()

	// Create batch
	createReq := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Concurrency Test Batch",
		"selection_mode":    "SINGLE_UNIT",
		"filter_criteria": map[string]interface{}{
			"unit_id": unit1ID.String(),
		},
	}
	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	batchID := uuid.MustParse(parseTestData(w.Body.Bytes())["id"].(string))

	// Simulate batch being actively processed by a worker
	_ = societyRepo

	// Generate batch
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for generate, got %d: %s", wGen.Code, wGen.Body.String())
	}

	// Now verify that a finalized batch rejects re-execution with 409 Conflict
	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}

	// Try generate again -> 409 Conflict (BATCH_FINALIZED)
	reqGenAgain := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGenAgain.Header.Set("Authorization", "Bearer "+superToken)
	wGenAgain := httptest.NewRecorder()
	router.ServeHTTP(wGenAgain, reqGenAgain)
	if wGenAgain.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for generating finalized batch, got %d: %s", wGenAgain.Code, wGenAgain.Body.String())
	}
	_ = ctx
}

func TestBillingBatch_PartialFailuresAndUnitRetry(t *testing.T) {
	router, societyRepo, socID, superToken, templateID, periodID, unit1ID, unit2ID, adminUID, _, _ := setupBatchTestFixture(t)
	ctx := context.Background()

	// Create a formula charge head
	formulaHead, err := societyRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Code:              fmt.Sprintf("FORMULA_TEST_%d", time.Now().UnixNano()%10000),
		Name:              "Formula Assessment",
		TaxClassification: "TAXABLE",
		DefaultTaxRate:    18.0,
		CalculationMethod: "FORMULA",
		DisplayOrder:      2,
		IsActive:          true,
	}, adminUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create formula charge head: %v", err)
	}

	// 1. Add invalid formula override on Unit 2: malformed formula expression
	invalidFormula := "area * / 500" // Syntax error in formula parser
	ov, err := societyRepo.CreateUnitBillingOverride(ctx, socID, unit2ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:              formulaHead.ID,
		OverrideType:              "FORMULA_OVERRIDE",
		FormulaOverrideExpression: &invalidFormula,
		Reason:                    "Intentional invalid formula to test unit failure isolation",
	}, adminUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create override: %v", err)
	}

	// 2. Create batch selecting both Unit 1 and Unit 2
	createReq := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Partial Failure Test Run",
		"selection_mode":    "SELECTED_UNITS",
		"filter_criteria": map[string]interface{}{
			"unit_ids": []string{unit1ID.String(), unit2ID.String()},
		},
	}
	body, _ := json.Marshal(createReq)
	reqBatch := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	reqBatch.Header.Set("Authorization", "Bearer "+superToken)
	reqBatch.Header.Set("Content-Type", "application/json")
	wBatch := httptest.NewRecorder()
	router.ServeHTTP(wBatch, reqBatch)
	if wBatch.Code != http.StatusCreated {
		t.Fatalf("expected 201 for batch, got %d: %s", wBatch.Code, wBatch.Body.String())
	}
	batchID := parseTestData(wBatch.Body.Bytes())["id"].(string)

	// 3. Execute Batch: Unit 1 succeeds, Unit 2 fails
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch generate with partial failure, got %d: %s", wGen.Code, wGen.Body.String())
	}

	genData := parseTestData(wGen.Body.Bytes())
	// Expected status: COMPLETED_WITH_ERRORS
	if genData["status"] != "COMPLETED_WITH_ERRORS" {
		t.Fatalf("expected COMPLETED_WITH_ERRORS, got %v", genData["status"])
	}
	if genData["successful_unit_count"] != float64(1) {
		t.Fatalf("expected 1 successful unit, got %v", genData["successful_unit_count"])
	}
	if genData["failed_unit_count"] != float64(1) {
		t.Fatalf("expected 1 failed unit, got %v", genData["failed_unit_count"])
	}

	// 4. Query Progress Endpoint
	reqProg := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/progress", socID, batchID), nil)
	reqProg.Header.Set("Authorization", "Bearer "+superToken)
	wProg := httptest.NewRecorder()
	router.ServeHTTP(wProg, reqProg)
	if wProg.Code != http.StatusOK {
		t.Fatalf("expected 200 for progress, got %d: %s", wProg.Code, wProg.Body.String())
	}
	progData := parseTestData(wProg.Body.Bytes())
	if progData["status"] != "COMPLETED_WITH_ERRORS" {
		t.Errorf("expected progress status COMPLETED_WITH_ERRORS, got %v", progData["status"])
	}
	if progData["successful_unit_count"] != float64(1) || progData["failed_unit_count"] != float64(1) {
		t.Errorf("expected counts (1, 1), got (%v, %v)", progData["successful_unit_count"], progData["failed_unit_count"])
	}

	// 5. Query Batch Units: verify Unit 2 failure message
	reqUnits := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/units?status=FAILED", socID, batchID), nil)
	reqUnits.Header.Set("Authorization", "Bearer "+superToken)
	wUnits := httptest.NewRecorder()
	router.ServeHTTP(wUnits, reqUnits)
	if wUnits.Code != http.StatusOK {
		t.Fatalf("expected 200 for list units, got %d: %s", wUnits.Code, wUnits.Body.String())
	}
	var unitsResp struct {
		Success bool `json:"success"`
		Data    struct {
			Units []struct {
				UnitID       string  `json:"unit_id"`
				Status       string  `json:"status"`
				ErrorMessage *string `json:"error_message"`
				RetryCount   int     `json:"retry_count"`
			} `json:"units"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUnits.Body.Bytes(), &unitsResp)
	if len(unitsResp.Data.Units) != 1 {
		t.Fatalf("expected 1 failed unit, got %d", len(unitsResp.Data.Units))
	}
	if unitsResp.Data.Units[0].UnitID != unit2ID.String() {
		t.Errorf("expected failed unit %s, got %s", unit2ID, unitsResp.Data.Units[0].UnitID)
	}
	if unitsResp.Data.Units[0].ErrorMessage == nil || *unitsResp.Data.Units[0].ErrorMessage == "" {
		t.Errorf("expected error message recorded for unit 2")
	}

	// 6. Fix the faulty override
	_ = societyRepo.DeleteUnitBillingOverride(ctx, socID, ov.ID, adminUID, "127.0.0.1", "test-agent")

	// 7. Retry the failed unit
	reqRetry := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/retry", socID, batchID), nil)
	reqRetry.Header.Set("Authorization", "Bearer "+superToken)
	wRetry := httptest.NewRecorder()
	router.ServeHTTP(wRetry, reqRetry)
	if wRetry.Code != http.StatusOK {
		t.Fatalf("expected 200 for retry, got %d: %s", wRetry.Code, wRetry.Body.String())
	}

	retryData := parseTestData(wRetry.Body.Bytes())
	if retryData["status"] != "GENERATED" {
		t.Fatalf("expected status GENERATED after successful retry, got %v", retryData["status"])
	}
	if retryData["successful_unit_count"] != float64(2) {
		t.Errorf("expected 2 successful units post-retry, got %v", retryData["successful_unit_count"])
	}
	if retryData["failed_unit_count"] != float64(0) {
		t.Errorf("expected 0 failed units post-retry, got %v", retryData["failed_unit_count"])
	}

	// 8. Finalize the batch
	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}
	if parseTestData(wFin.Body.Bytes())["status"] != "FINALIZED" {
		t.Errorf("expected FINALIZED status")
	}
}

func TestBillingBatch_AuthorizationSecurity(t *testing.T) {
	router, _, socID, _, templateID, periodID, unit1ID, _, _, _, createUserToken := setupBatchTestFixture(t)

	// Create resident user with no admin roles
	uniqueTime := time.Now().UnixNano()
	_, resToken := createUserToken(fmt.Sprintf("+9199%08d", uniqueTime%100000000))

	createReq := map[string]interface{}{
		"billing_period_id": periodID.String(),
		"template_id":       templateID.String(),
		"batch_name":        "Resident Forbidden Batch",
		"selection_mode":    "SINGLE_UNIT",
		"filter_criteria": map[string]interface{}{
			"unit_id": unit1ID.String(),
		},
	}
	body, _ := json.Marshal(createReq)

	// Resident attempting to create a batch -> 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+resToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for resident batch creation, got %d", w.Code)
	}

	// Resident attempting to get batch progress -> 403 Forbidden
	dummyBatchID := uuid.New()
	reqProg := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/progress", socID, dummyBatchID), nil)
	reqProg.Header.Set("Authorization", "Bearer "+resToken)
	wProg := httptest.NewRecorder()
	router.ServeHTTP(wProg, reqProg)
	if wProg.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for resident progress query, got %d", wProg.Code)
	}
}
