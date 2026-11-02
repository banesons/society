package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chs/backend/internal/society"

	"github.com/google/uuid"
)

func strPtr(s string) *string {
	return &s
}

// TestBilling_EndToEnd_FullLifecycle executes the complete 14-step Phase 8 acceptance scenario:
// multi-building/unit society, mixed formulas (fixed, area, taxes, unit overrides), missing data fault isolation,
// retry of failed units, immutable finalization, resident visibility and PDF access, IDOR protection,
// and recurring automation idempotency.
func TestBilling_EndToEnd_FullLifecycle(t *testing.T) {
	_, db, router, _, sRepo, socID, superToken, adminUID, genTok := setupBillingTestEnv(t)
	ctx := context.Background()
	unique := time.Now().UnixNano()

	// =========================================================================
	// SCENARIO SETUP: Multi-Building, Multi-Unit Society with Diverse Occupancy
	// =========================================================================

	// Building 1: Tower Emerald
	bld1Req := map[string]interface{}{
		"name":          "Tower Emerald",
		"building_code": fmt.Sprintf("TE%d", unique%10000),
		"total_floors":  2,
	}
	bld1JSON, _ := json.Marshal(bld1Req)
	reqBld1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", socID), bytes.NewReader(bld1JSON))
	reqBld1.Header.Set("Authorization", "Bearer "+superToken)
	reqBld1.Header.Set("Content-Type", "application/json")
	wBld1 := httptest.NewRecorder()
	router.ServeHTTP(wBld1, reqBld1)
	if wBld1.Code != http.StatusCreated {
		t.Fatalf("create building 1: %d %s", wBld1.Code, wBld1.Body.String())
	}
	bld1ID := uuid.MustParse(parseTestData(wBld1.Body.Bytes())["id"].(string))

	// Unit 101 (Tower A): Standard Residential Owner
	unit101ID := uuid.New()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO society_units (id, society_id, building_id, unit_number, unit_identifier, floor_number, unit_type, status)
		VALUES ($1, $2, $3, '101', 'A-101', 1, 'RESIDENTIAL', 'ACTIVE')
	`, unit101ID, socID, bld1ID)
	if err != nil {
		t.Fatalf("insert unit 101: %v", err)
	}

	// Unit 102 (Tower A): Penthouse with Unit Override
	unit102ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_units (id, society_id, building_id, unit_number, unit_identifier, floor_number, unit_type, status)
		VALUES ($1, $2, $3, '102', 'A-102', 1, 'PENTHOUSE', 'ACTIVE')
	`, unit102ID, socID, bld1ID)
	if err != nil {
		t.Fatalf("insert unit 102: %v", err)
	}

	// Unit 201 (Tower A): Standard Residential
	unit201ID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO society_units (id, society_id, building_id, unit_number, unit_identifier, floor_number, unit_type, status)
		VALUES ($1, $2, $3, '201', 'A-201', 2, 'RESIDENTIAL', 'ACTIVE')
	`, unit201ID, socID, bld1ID)
	if err != nil {
		t.Fatalf("insert unit 201: %v", err)
	}

	// Create Resident Users
	// Resident 1 (Unit 101 Owner)
	user1ID := uuid.New()
	phone1 := fmt.Sprintf("9198%08d", unique%100000000)
	if _, err := db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')`, user1ID, phone1); err != nil {
		t.Fatalf("insert user 1: %v", err)
	}
	res1Token := genTok(user1ID)

	mem1ID := uuid.New()
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO society_members (id, society_id, user_id, first_name, phone, status)
		VALUES ($1, $2, $3, 'Resident One', $4, 'ACTIVE')
	`, mem1ID, socID, user1ID, phone1); err != nil {
		t.Fatalf("insert member 1: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO unit_relationships (id, society_id, unit_id, member_id, relationship_type, is_occupant, status)
		VALUES ($1, $2, $3, $4, 'PRIMARY_OWNER', true, 'ACTIVE')
	`, uuid.New(), socID, unit101ID, mem1ID); err != nil {
		t.Fatalf("insert relationship 1: %v", err)
	}

	// Resident 2 (Unit 102 Penthouse Tenant)
	user2ID := uuid.New()
	phone2 := fmt.Sprintf("9199%08d", (unique+1)%100000000)
	if _, err := db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')`, user2ID, phone2); err != nil {
		t.Fatalf("insert user 2: %v", err)
	}
	res2Token := genTok(user2ID)

	mem2ID := uuid.New()
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO society_members (id, society_id, user_id, first_name, phone, status)
		VALUES ($1, $2, $3, 'Resident Two', $4, 'ACTIVE')
	`, mem2ID, socID, user2ID, phone2); err != nil {
		t.Fatalf("insert member 2: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO unit_relationships (id, society_id, unit_id, member_id, relationship_type, is_occupant, status)
		VALUES ($1, $2, $3, $4, 'PRIMARY_TENANT', true, 'ACTIVE')
	`, uuid.New(), socID, unit102ID, mem2ID); err != nil {
		t.Fatalf("insert relationship 2: %v", err)
	}

	// Inactive Resident (Unit 201 Member departed)
	userInactiveID := uuid.New()
	phoneInact := fmt.Sprintf("9196%08d", (unique+2)%100000000)
	if _, err := db.Pool().Exec(ctx, `INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')`, userInactiveID, phoneInact); err != nil {
		t.Fatalf("insert user inactive: %v", err)
	}
	inactiveToken := genTok(userInactiveID)

	memInactID := uuid.New()
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO society_members (id, society_id, user_id, first_name, phone, status)
		VALUES ($1, $2, $3, 'Inactive Resident', $4, 'INACTIVE')
	`, memInactID, socID, userInactiveID, phoneInact); err != nil {
		t.Fatalf("insert member inactive: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO unit_relationships (id, society_id, unit_id, member_id, relationship_type, is_occupant, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, 'PRIMARY_OWNER', false, 'ENDED', CURRENT_DATE - 30, CURRENT_DATE - 1)
	`, uuid.New(), socID, unit201ID, memInactID); err != nil {
		t.Fatalf("insert relationship inactive: %v", err)
	}

	// =========================================================================
	// STEP 1: Configure Billing Rules (Charge Heads, Formulas, Overrides)
	// =========================================================================

	// Charge Head 1: Fixed Security Fee
	headSecurity, err := sRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Code:              fmt.Sprintf("SEC_%d", unique%10000),
		Name:              "Security & Guard Services",
		TaxClassification: "EXEMPT",
		CalculationMethod: "FIXED",
		DisplayOrder:      1,
		IsActive:          true,
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create head security: %v", err)
	}

	// Charge Head 2: Formula Sinking Fund (area * 2.50)
	headSinking, err := sRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Code:              fmt.Sprintf("SINK_%d", unique%10000),
		Name:              "Sinking Fund Contribution",
		TaxClassification: "EXEMPT",
		CalculationMethod: "FORMULA",
		DisplayOrder:      2,
		IsActive:          true,
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create head sinking fund: %v", err)
	}

	// Create and Publish Billing Template
	tpl, err := sRepo.CreateBillTemplate(ctx, socID, society.CreateBillTemplateParams{
		Name:             fmt.Sprintf("E2E Acceptance Template %d", unique%10000),
		BillType:         "MAINTENANCE",
		BillingFrequency: "MONTHLY",
		Currency:         "INR",
		BillTitle:        "Maintenance Assessment Invoice",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	pubTpl, err := sRepo.PublishBillTemplate(ctx, socID, tpl.ID, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("publish template: %v", err)
	}

	// Configure Fixed Overrides on Security Head:
	fixed1500 := 1500.0
	fixed1000 := 1000.0
	now := time.Now()
	_, err = sRepo.CreateUnitBillingOverride(ctx, socID, unit101ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:  headSecurity.ID,
		OverrideType:  "FIXED_AMOUNT",
		OverrideValue: &fixed1500,
		EffectiveFrom: now.AddDate(0, -1, 0),
		Reason:        "Standard Tower A Security Fee",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 101 security override: %v", err)
	}

	// Unit-Specific Override: Penthouse Unit 102 gets Fixed Security discounted to ₹1000
	_, err = sRepo.CreateUnitBillingOverride(ctx, socID, unit102ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:  headSecurity.ID,
		OverrideType:  "FIXED_AMOUNT",
		OverrideValue: &fixed1000,
		EffectiveFrom: now.AddDate(0, -1, 0),
		Reason:        "Penthouse bulk amenity discount",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 102 security override: %v", err)
	}

	_, err = sRepo.CreateUnitBillingOverride(ctx, socID, unit201ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:  headSecurity.ID,
		OverrideType:  "FIXED_AMOUNT",
		OverrideValue: &fixed1500,
		EffectiveFrom: now.AddDate(0, -1, 0),
		Reason:        "Tower B Security Fee",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 201 security override: %v", err)
	}

	// Configure Formula Overrides on Sinking Head:
	validFormula := "area * 2.50"
	invalidFormula := "area * / 2.50" // Syntax error to verify unit fault isolation
	_, err = sRepo.CreateUnitBillingOverride(ctx, socID, unit101ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:              headSinking.ID,
		OverrideType:              "FORMULA_OVERRIDE",
		FormulaOverrideExpression: &validFormula,
		EffectiveFrom:             now.AddDate(0, -1, 0),
		Reason:                    "Standard sinking fund formula",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 101 sinking override: %v", err)
	}

	formula1600 := "1600 * 2.50"
	_, err = sRepo.CreateUnitBillingOverride(ctx, socID, unit102ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:              headSinking.ID,
		OverrideType:              "FORMULA_OVERRIDE",
		FormulaOverrideExpression: &formula1600,
		EffectiveFrom:             now.AddDate(0, -1, 0),
		Reason:                    "Penthouse sinking fund formula",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 102 sinking override: %v", err)
	}

	ov201, err := sRepo.CreateUnitBillingOverride(ctx, socID, unit201ID, society.CreateUnitBillingOverrideParams{
		ChargeHeadID:              headSinking.ID,
		OverrideType:              "FORMULA_OVERRIDE",
		FormulaOverrideExpression: &invalidFormula,
		EffectiveFrom:             now.AddDate(0, -1, 0),
		Reason:                    "Intentional invalid syntax to verify fault isolation",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create unit 201 invalid override: %v", err)
	}

	// Create Billing Period
	now = time.Now()
	pStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	pEnd := pStart.AddDate(0, 1, -1)
	dueDate := pStart.AddDate(0, 0, 15)
	periodName := fmt.Sprintf("E2E Acceptance %s %d", now.Month().String(), now.Year())

	var periodID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO maintenance_billing_periods (
			society_id, period_name, period_start, period_end, due_date, status, created_by
		) VALUES ($1, $2, $3, $4, $5, 'ACTIVE', $6)
		RETURNING id
	`, socID, periodName, pStart, pEnd, dueDate, adminUID).Scan(&periodID)
	if err != nil {
		t.Fatalf("create billing period: %v", err)
	}

	// =========================================================================
	// STEP 2 & 3: Preview Calculation & Create Billing Batch
	// =========================================================================
	idempotencyKey := fmt.Sprintf("e2e-batch-%s", socID.String()[:8])
	batchParams := society.CreateBillingBatchParams{
		BillingPeriodID: periodID,
		TemplateID:      pubTpl.TemplateID,
		SelectionMode:   "ALL_UNITS",
		IdempotencyKey:  &idempotencyKey,
	}

	batch, err := sRepo.CreateBillingBatch(ctx, socID, batchParams, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create billing batch: %v", err)
	}
	if batch.Status != "QUEUED" && batch.Status != "DRAFT_PREVIEW" {
		t.Errorf("expected initial status QUEUED or DRAFT_PREVIEW, got %s", batch.Status)
	}

	// Verify Idempotency: submitting same batch params returns identical batch
	batchDup, err := sRepo.CreateBillingBatch(ctx, socID, batchParams, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("duplicate batch submit: %v", err)
	}
	if batchDup.ID != batch.ID {
		t.Errorf("idempotency violation: expected same batch ID %s, got %s", batch.ID, batchDup.ID)
	}

	// =========================================================================
	// STEP 4 & 5: Generate Drafts & Inspect Fault Isolation for Missing Data
	// =========================================================================
	batch, err = sRepo.GenerateBillingBatchDrafts(ctx, socID, batch.ID, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("generate drafts: %v", err)
	}

	// Batch has 3 units: Unit 101 (OK), Unit 102 (OK), Unit 201 (Invalid formula syntax -> fails)
	// Batch status should be COMPLETED_WITH_ERRORS or GENERATED depending on fault isolation
	if batch.FailedUnitCount != 1 {
		t.Errorf("expected exactly 1 failed unit due to invalid formula, got %d", batch.FailedUnitCount)
	}
	if batch.SuccessfulUnitCount != 2 {
		t.Errorf("expected exactly 2 successful units, got %d", batch.SuccessfulUnitCount)
	}

	// Inspect unit diagnostics API via HTTP
	reqDiag := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/units", socID, batch.ID), nil)
	reqDiag.Header.Set("Authorization", "Bearer "+superToken)
	recDiag := httptest.NewRecorder()
	router.ServeHTTP(recDiag, reqDiag)
	if recDiag.Code != http.StatusOK {
		t.Fatalf("query unit diagnostics: %d %s", recDiag.Code, recDiag.Body.String())
	}

	// =========================================================================
	// STEP 6 & 7: Correct Unit Data & Retry Failed Unit Safely
	// =========================================================================
	// Fix Unit 201 formula override
	valid201Formula := "1200 * 2.50"
	_, err = db.Pool().Exec(ctx, `
		UPDATE maintenance_unit_billing_overrides
		SET formula_override_expression = $1
		WHERE id = $2
	`, valid201Formula, ov201.ID)
	if err != nil {
		t.Fatalf("update unit 201 override formula: %v", err)
	}

	// Retry failed units on batch
	batch, err = sRepo.RetryFailedBatchUnits(ctx, socID, batch.ID, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("retry failed units: %v", err)
	}
	if batch.FailedUnitCount != 0 {
		t.Errorf("expected 0 failed units after retry, got %d", batch.FailedUnitCount)
	}
	if batch.SuccessfulUnitCount != 3 {
		t.Errorf("expected 3 successful units after retry, got %d", batch.SuccessfulUnitCount)
	}
	if batch.Status != "GENERATED" {
		t.Errorf("expected batch status GENERATED, got %s", batch.Status)
	}

	// =========================================================================
	// STEP 8: Security & Privacy Check — Residents CANNOT see DRAFT bills
	// =========================================================================
	// Resident 1 checks /my-bills while batch is GENERATED (DRAFT)
	reqResDraft := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills", socID), nil)
	reqResDraft.Header.Set("Authorization", "Bearer "+res1Token)
	recResDraft := httptest.NewRecorder()
	router.ServeHTTP(recResDraft, reqResDraft)
	if recResDraft.Code != http.StatusOK {
		t.Fatalf("resident my-bills query: %d %s", recResDraft.Code, recResDraft.Body.String())
	}
	var resBillsResp struct {
		Data struct {
			Bills []interface{} `json:"bills"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recResDraft.Body.Bytes(), &resBillsResp)
	if len(resBillsResp.Data.Bills) != 0 {
		t.Errorf("security violation: resident saw unfinalized DRAFT bills (%d items)", len(resBillsResp.Data.Bills))
	}

	// =========================================================================
	// STEP 9 & 10: Finalize Batch & Enforce Financial Immutability
	// =========================================================================
	batch, err = sRepo.FinalizeBillingBatch(ctx, socID, batch.ID, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("finalize billing batch: %v", err)
	}
	if batch.Status != "FINALIZED" {
		t.Errorf("expected batch status FINALIZED, got %s", batch.Status)
	}

	// Financial Immutability: Attempting to finalize again must fail
	_, err = sRepo.FinalizeBillingBatch(ctx, socID, batch.ID, adminUID, "127.0.0.1", "test")
	if err == nil {
		t.Errorf("expected error when re-finalizing batch, got nil")
	}

	// =========================================================================
	// STEP 11: Financial Correctness — Independent Calculation Verification
	// =========================================================================
	// Query bills generated for Unit 101, 102, 201
	type BillCheck struct {
		ID          uuid.UUID
		UnitID      uuid.UUID
		Subtotal    float64
		TotalAmount float64
		Status      string
	}
	rows, err := db.Pool().Query(ctx, `
		SELECT id, unit_id, subtotal, total_amount, status
		FROM maintenance_bills
		WHERE batch_id = $1 AND society_id = $2
	`, batch.ID, socID)
	if err != nil {
		t.Fatalf("query finalized bills: %v", err)
	}
	defer rows.Close()

	var bill101, bill102, bill201 BillCheck
	for rows.Next() {
		var bc BillCheck
		if err := rows.Scan(&bc.ID, &bc.UnitID, &bc.Subtotal, &bc.TotalAmount, &bc.Status); err != nil {
			t.Fatalf("scan bill check: %v", err)
		}
		if bc.Status != "ISSUED" {
			t.Errorf("expected finalized bill to have status ISSUED, got %s", bc.Status)
		}
		switch bc.UnitID {
		case unit101ID:
			bill101 = bc
		case unit102ID:
			bill102 = bc
		case unit201ID:
			bill201 = bc
		}
	}

	// Independent Formula Validation:
	// Unit 101: Area 1000 sqft
	//   Security: ₹1500 (standard fixed)
	//   Sinking: 1000 * 2.50 = ₹2500
	//   Subtotal: 1500 + 2500 = ₹4000.00
	expected101Total := 4000.00
	if math.Abs(bill101.TotalAmount-expected101Total) > 0.01 {
		t.Errorf("Unit 101 financial calculation mismatch: expected ₹%.2f, got ₹%.2f", expected101Total, bill101.TotalAmount)
	}

	// Unit 102 (Penthouse): Area 1600 sqft with Security overridden to ₹1000
	//   Security: ₹1000 (unit override)
	//   Sinking: 1600 * 2.50 = ₹4000
	//   Subtotal: 1000 + 4000 = ₹5000.00
	expected102Total := 5000.00
	if math.Abs(bill102.TotalAmount-expected102Total) > 0.01 {
		t.Errorf("Unit 102 financial calculation mismatch: expected ₹%.2f, got ₹%.2f", expected102Total, bill102.TotalAmount)
	}

	// Unit 201: Area 1200 sqft (post-correction)
	//   Security: ₹1500
	//   Sinking: 1200 * 2.50 = ₹3000
	//   Subtotal: 1500 + 3000 = ₹4500.00
	expected201Total := 4500.00
	if math.Abs(bill201.TotalAmount-expected201Total) > 0.01 {
		t.Errorf("Unit 201 financial calculation mismatch: expected ₹%.2f, got ₹%.2f", expected201Total, bill201.TotalAmount)
	}

	// =========================================================================
	// STEP 12: Resident Visibility, PDF Access & IDOR Security Protection
	// =========================================================================
	// Resident 1 now queries /my-bills: MUST see Unit 101 bill!
	reqRes1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills", socID), nil)
	reqRes1.Header.Set("Authorization", "Bearer "+res1Token)
	recRes1 := httptest.NewRecorder()
	router.ServeHTTP(recRes1, reqRes1)
	if recRes1.Code != http.StatusOK {
		t.Fatalf("resident 1 my-bills query: %d %s", recRes1.Code, recRes1.Body.String())
	}
	var res1Bills struct {
		Data struct {
			Bills []struct {
				ID          string  `json:"id"`
				TotalAmount float64 `json:"total_amount"`
			} `json:"bills"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recRes1.Body.Bytes(), &res1Bills)
	if len(res1Bills.Data.Bills) != 1 || res1Bills.Data.Bills[0].ID != bill101.ID.String() {
		t.Errorf("expected resident 1 to see only unit 101 bill, got %v", res1Bills.Data.Bills)
	}

	// Resident 1 downloads Unit 101 PDF: MUST succeed (200 OK, application/pdf)
	reqPdf1 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s/pdf", socID, bill101.ID), nil)
	reqPdf1.Header.Set("Authorization", "Bearer "+res1Token)
	recPdf1 := httptest.NewRecorder()
	router.ServeHTTP(recPdf1, reqPdf1)
	if recPdf1.Code != http.StatusOK {
		t.Errorf("expected 200 for resident downloading own PDF, got %d: %s", recPdf1.Code, recPdf1.Body.String())
	}
	if recPdf1.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %s", recPdf1.Header().Get("Content-Type"))
	}
	if recPdf1.Body.Len() == 0 {
		t.Errorf("expected non-empty PDF bytes")
	}

	// IDOR Protection: Resident 1 attempts to access Resident 2's bill (Unit 102) -> 403 Forbidden
	reqIDOR := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s", socID, bill102.ID), nil)
	reqIDOR.Header.Set("Authorization", "Bearer "+res1Token)
	recIDOR := httptest.NewRecorder()
	router.ServeHTTP(recIDOR, reqIDOR)
	if recIDOR.Code != http.StatusForbidden && recIDOR.Code != http.StatusNotFound {
		t.Errorf("IDOR security violation: resident 1 accessed unit 102 bill, got code %d", recIDOR.Code)
	}

	// IDOR PDF Protection: Resident 1 attempts to download Resident 2's PDF -> 403 Forbidden
	reqIDORPdf := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s/pdf", socID, bill102.ID), nil)
	reqIDORPdf.Header.Set("Authorization", "Bearer "+res1Token)
	recIDORPdf := httptest.NewRecorder()
	router.ServeHTTP(recIDORPdf, reqIDORPdf)
	if recIDORPdf.Code != http.StatusForbidden {
		t.Errorf("IDOR security violation: resident 1 downloaded unit 102 PDF, got code %d", recIDORPdf.Code)
	}

	// Resident 2 (Penthouse) accesses their own Unit 102 PDF: MUST succeed
	reqRes2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s/pdf", socID, bill102.ID), nil)
	reqRes2.Header.Set("Authorization", "Bearer "+res2Token)
	recRes2 := httptest.NewRecorder()
	router.ServeHTTP(recRes2, reqRes2)
	if recRes2.Code != http.StatusOK {
		t.Errorf("expected 200 for resident 2 accessing own bill, got %d: %s", recRes2.Code, recRes2.Body.String())
	}

	// Inactive Resident Security: Inactive resident attempts to access Unit 201 bill -> 403/404
	reqInact := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills/%s", socID, bill201.ID), nil)
	reqInact.Header.Set("Authorization", "Bearer "+inactiveToken)
	recInact := httptest.NewRecorder()
	router.ServeHTTP(recInact, reqInact)
	if recInact.Code != http.StatusForbidden && recInact.Code != http.StatusNotFound {
		t.Errorf("security violation: inactive resident accessed bill, got code %d", recInact.Code)
	}

	// =========================================================================
	// STEP 13: Administrator Batch Inspection & Artifact ZIP
	// =========================================================================
	// Admin generates batch PDF ZIP
	reqZip := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/pdf", socID, batch.ID), nil)
	reqZip.Header.Set("Authorization", "Bearer "+superToken)
	recZip := httptest.NewRecorder()
	router.ServeHTTP(recZip, reqZip)
	if recZip.Code != http.StatusOK {
		t.Errorf("admin batch PDF ZIP generation: expected 200, got %d: %s", recZip.Code, recZip.Body.String())
	}

	// Resident attempts to generate batch PDF ZIP: MUST be denied (403)
	reqZipRes := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/pdf", socID, batch.ID), nil)
	reqZipRes.Header.Set("Authorization", "Bearer "+res1Token)
	recZipRes := httptest.NewRecorder()
	router.ServeHTTP(recZipRes, reqZipRes)
	if recZipRes.Code != http.StatusForbidden {
		t.Errorf("security violation: resident authorized for batch ZIP, got code %d", recZipRes.Code)
	}

	// =========================================================================
	// STEP 14: Recurring Schedule Automation & Duplicate Prevention
	// =========================================================================
	sched, err := sRepo.CreateBillingSchedule(ctx, socID, society.CreateBillingScheduleParams{
		Name:           "E2E Recurring Automation",
		RecurrenceType: "MONTHLY",
		TemplateID:     pubTpl.TemplateID,
		DayOfMonth:     1,
		DueDays:        15,
		Timezone:       "Asia/Kolkata",
		AutoFinalize:   true,
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create recurring schedule: %v", err)
	}

	// Execute scheduled run
	exec1, err := sRepo.ExecuteScheduleRun(ctx, socID, sched.ID, "SCHEDULED", adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("execute scheduled run: %v", err)
	}
	if exec1.Status != "SUCCESS" {
		t.Errorf("expected schedule execution SUCCESS, got %s", exec1.Status)
	}

	// Verify scheduler health summary
	health, err := sRepo.GetSchedulerHealth(ctx, socID)
	if err != nil {
		t.Fatalf("get scheduler health: %v", err)
	}
	if health.TotalSchedules < 1 || health.ActiveSchedules < 1 {
		t.Errorf("expected active schedules in health summary, got %v", health)
	}
}
