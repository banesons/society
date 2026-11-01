package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"chs/backend/internal/society"

	"github.com/google/uuid"
)

// Unit tests for recurrence date arithmetic, month-end edge cases, leap years, and timezones.
func TestRecurrenceCalculations(t *testing.T) {
	kolkata, err := society.ValidateTimezone("Asia/Kolkata")
	if err != nil {
		t.Fatalf("load Asia/Kolkata timezone: %v", err)
	}

	t.Run("Monthly Recurrence and Month-End Dates", func(t *testing.T) {
		// Run in January 2026: Target month Jan 2026
		janDate := time.Date(2026, 1, 15, 10, 0, 0, 0, kolkata)
		pStart, pEnd, dueDate, pName := society.CalculateBillingPeriodDates(janDate, "MONTHLY", 15, kolkata)

		if pStart.Day() != 1 || pStart.Month() != time.January || pStart.Year() != 2026 {
			t.Errorf("expected Jan 1 2026, got %v", pStart)
		}
		if pEnd.Day() != 31 || pEnd.Month() != time.January || pEnd.Year() != 2026 {
			t.Errorf("expected Jan 31 2026, got %v", pEnd)
		}
		if dueDate.Day() != 16 || dueDate.Month() != time.January {
			t.Errorf("expected Jan 16 2026, got %v", dueDate)
		}
		if pName != "January 2026 Maintenance" {
			t.Errorf("expected 'January 2026 Maintenance', got %q", pName)
		}

		// Leap Year: February in 2028 has 29 days
		leapFeb := time.Date(2028, 2, 10, 10, 0, 0, 0, kolkata)
		_, febEnd, _, _ := society.CalculateBillingPeriodDates(leapFeb, "MONTHLY", 15, kolkata)
		if febEnd.Day() != 29 {
			t.Errorf("expected Feb 29 2028 (leap year), got day %d", febEnd.Day())
		}

		// Non-Leap Year: February in 2026 has 28 days
		nonLeapFeb := time.Date(2026, 2, 10, 10, 0, 0, 0, kolkata)
		_, nonFebEnd, _, _ := society.CalculateBillingPeriodDates(nonLeapFeb, "MONTHLY", 15, kolkata)
		if nonFebEnd.Day() != 28 {
			t.Errorf("expected Feb 28 2026, got day %d", nonFebEnd.Day())
		}
	})

	t.Run("Next Run Clamping for 31st Day", func(t *testing.T) {
		// From Jan 31, monthly schedule for 31st day should advance to Feb 28 in 2026
		jan31 := time.Date(2026, 1, 31, 9, 0, 0, 0, kolkata)
		nextRun := society.CalculateNextRunAt(jan31, "MONTHLY", 31, kolkata)
		if nextRun.Month() != time.February || nextRun.Day() != 28 {
			t.Errorf("expected Feb 28 next run clamped, got %v", nextRun)
		}

		// In 2028 leap year, should clamp to Feb 29
		jan31Leap := time.Date(2028, 1, 31, 9, 0, 0, 0, kolkata)
		nextRunLeap := society.CalculateNextRunAt(jan31Leap, "MONTHLY", 31, kolkata)
		if nextRunLeap.Month() != time.February || nextRunLeap.Day() != 29 {
			t.Errorf("expected Feb 29 next run clamped, got %v", nextRunLeap)
		}
	})

	t.Run("Quarterly Recurrence", func(t *testing.T) {
		// Run in May (Q2: Apr 1 - Jun 30)
		mayDate := time.Date(2026, 5, 20, 10, 0, 0, 0, kolkata)
		pStart, pEnd, _, pName := society.CalculateBillingPeriodDates(mayDate, "QUARTERLY", 15, kolkata)

		if pStart.Month() != time.April || pStart.Day() != 1 {
			t.Errorf("expected Apr 1 for Q2, got %v", pStart)
		}
		if pEnd.Month() != time.June || pEnd.Day() != 30 {
			t.Errorf("expected Jun 30 for Q2, got %v", pEnd)
		}
		if pName != "Q2 2026 Maintenance" {
			t.Errorf("expected 'Q2 2026 Maintenance', got %q", pName)
		}

		// Next run advances by 3 months
		nextRun := society.CalculateNextRunAt(mayDate, "QUARTERLY", 1, kolkata)
		if nextRun.Month() != time.August || nextRun.Day() != 1 {
			t.Errorf("expected Aug 1 for next quarterly run, got %v", nextRun)
		}
	})

	t.Run("Annual Recurrence", func(t *testing.T) {
		annDate := time.Date(2026, 7, 10, 10, 0, 0, 0, kolkata)
		pStart, pEnd, _, pName := society.CalculateBillingPeriodDates(annDate, "ANNUAL", 30, kolkata)

		if pStart.Year() != 2026 || pStart.Month() != time.January || pStart.Day() != 1 {
			t.Errorf("expected Jan 1 2026, got %v", pStart)
		}
		if pEnd.Year() != 2026 || pEnd.Month() != time.December || pEnd.Day() != 31 {
			t.Errorf("expected Dec 31 2026, got %v", pEnd)
		}
		if pName != "FY 2026 Maintenance" {
			t.Errorf("expected 'FY 2026 Maintenance', got %q", pName)
		}

		nextRun := society.CalculateNextRunAt(annDate, "ANNUAL", 1, kolkata)
		if nextRun.Year() != 2027 || nextRun.Month() != time.July || nextRun.Day() != 1 {
			t.Errorf("expected Jul 1 2027 for annual next run, got %v", nextRun)
		}
	})

	t.Run("Timezone Conversion and Validation", func(t *testing.T) {
		utc, err := society.ValidateTimezone("UTC")
		if err != nil || utc.String() != "UTC" {
			t.Errorf("expected valid UTC timezone, got %v, err=%v", utc, err)
		}

		invalid, err := society.ValidateTimezone("Invalid/Unknown_Zone")
		if err == nil || invalid != nil {
			t.Errorf("expected error for invalid timezone, got %v", err)
		}
	})
}

// Database-backed fixture setup for scheduler tests.
func setupSchedulerFixture(t *testing.T) (
	*society.Repository,
	uuid.UUID, // societyID
	string,    // superAdminToken
	uuid.UUID, // templateID
	http.Handler,
	func(phone string) (uuid.UUID, string),
	uuid.UUID, // adminUserID
) {
	t.Helper()
	_, db, r, _, sRepo, sID, sTok, aUID, genTok := setupBillingTestEnv(t)
	ctx := context.Background()
	uniqueTime := time.Now().UnixNano()

	// 1. Create Building & Units via router
	bldReq := map[string]interface{}{
		"name":          "Block Platinum",
		"building_code": fmt.Sprintf("BS%d", uniqueTime%10000),
		"total_floors":  1,
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
		Prefix:             fmt.Sprintf("S%d", uniqueTime%1000),
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

	// 2. Create Charge Head
	_, err := sRepo.CreateChargeHead(ctx, sID, society.CreateChargeHeadParams{
		Code:              fmt.Sprintf("SCH_MAINT_%d", uniqueTime%10000),
		Name:              "Scheduler Maintenance Fee",
		TaxClassification: "EXEMPT",
		CalculationMethod: "FIXED",
		DisplayOrder:      1,
		IsActive:          true,
	}, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create charge head: %v", err)
	}

	// 3. Create Template & Publish
	tpl, err := sRepo.CreateBillTemplate(ctx, sID, society.CreateBillTemplateParams{
		Name:             fmt.Sprintf("Scheduler Template %d", uniqueTime%10000),
		BillType:         "MAINTENANCE",
		BillingFrequency: "MONTHLY",
		Currency:         "INR",
		BillTitle:        "Monthly Invoice",
	}, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	pubTpl, err := sRepo.PublishBillTemplate(ctx, sID, tpl.ID, aUID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to publish template: %v", err)
	}

	createUserToken := func(phone string) (uuid.UUID, string) {
		uID := uuid.New()
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
		`, uID, phone)
		return uID, genTok(uID)
	}

	return sRepo, sID, sTok, pubTpl.TemplateID, r, createUserToken, aUID
}

func TestBillingSchedule_LifecycleAndAdminControls(t *testing.T) {
	repo, socID, superToken, templateID, router, _, _ := setupSchedulerFixture(t)

	// 1. Create Schedule
	createPayload := map[string]interface{}{
		"name": "Monthly Auto Billing",
		"recurrence_type": "MONTHLY",
		"template_id": templateID.String(),
		"selection_mode": "ALL_UNITS",
		"day_of_month": 1,
		"due_days": 15,
		"timezone": "Asia/Kolkata",
		"auto_finalize": false,
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/schedules", socID), bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createResp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &createResp)
	schedData := createResp["data"].(map[string]interface{})
	schedID := schedData["id"].(string)

	if schedData["status"] != "ACTIVE" {
		t.Errorf("expected status ACTIVE, got %v", schedData["status"])
	}

	// Duplicate schedule name rejected with 409
	reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/schedules", socID), bytes.NewReader(body))
	reqDup.Header.Set("Authorization", "Bearer "+superToken)
	reqDup.Header.Set("Content-Type", "application/json")
	recDup := httptest.NewRecorder()
	router.ServeHTTP(recDup, reqDup)
	if recDup.Code != http.StatusConflict {
		t.Errorf("expected 409 conflict for duplicate name, got %d", recDup.Code)
	}

	// 2. List Schedules
	reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/schedules", socID), nil)
	reqList.Header.Set("Authorization", "Bearer "+superToken)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 ok for list schedules, got %d", recList.Code)
	}

	// 3. Pause Schedule
	reqPause := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/schedules/%s/pause", socID, schedID), nil)
	reqPause.Header.Set("Authorization", "Bearer "+superToken)
	recPause := httptest.NewRecorder()
	router.ServeHTTP(recPause, reqPause)
	if recPause.Code != http.StatusOK {
		t.Fatalf("expected 200 ok for pause, got %d", recPause.Code)
	}
	sPaused, _ := repo.GetBillingSchedule(context.Background(), socID, uuid.MustParse(schedID))
	if sPaused.Status != "PAUSED" {
		t.Errorf("expected PAUSED, got %s", sPaused.Status)
	}

	// 4. Resume Schedule
	reqResume := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/schedules/%s/resume", socID, schedID), nil)
	reqResume.Header.Set("Authorization", "Bearer "+superToken)
	recResume := httptest.NewRecorder()
	router.ServeHTTP(recResume, reqResume)
	if recResume.Code != http.StatusOK {
		t.Fatalf("expected 200 ok for resume, got %d", recResume.Code)
	}
	sResumed, _ := repo.GetBillingSchedule(context.Background(), socID, uuid.MustParse(schedID))
	if sResumed.Status != "ACTIVE" {
		t.Errorf("expected ACTIVE, got %s", sResumed.Status)
	}

	// 5. Trigger Manual Run Now
	reqRun := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/schedules/%s/run-now", socID, schedID), nil)
	reqRun.Header.Set("Authorization", "Bearer "+superToken)
	recRun := httptest.NewRecorder()
	router.ServeHTTP(recRun, reqRun)
	if recRun.Code != http.StatusOK {
		t.Fatalf("expected 200 ok for run-now, got %d: %s", recRun.Code, recRun.Body.String())
	}

	var runResp map[string]interface{}
	_ = json.Unmarshal(recRun.Body.Bytes(), &runResp)
	execData := runResp["data"].(map[string]interface{})
	if execData["status"] != "SUCCESS" {
		t.Errorf("expected execution status SUCCESS, got %v", execData["status"])
	}
	if execData["batch_id"] == nil {
		t.Errorf("expected batch_id to be populated")
	}

	// 6. List Executions History
	reqExecs := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/schedules/%s/executions", socID, schedID), nil)
	reqExecs.Header.Set("Authorization", "Bearer "+superToken)
	recExecs := httptest.NewRecorder()
	router.ServeHTTP(recExecs, reqExecs)
	if recExecs.Code != http.StatusOK {
		t.Fatalf("expected 200 ok for executions list, got %d", recExecs.Code)
	}

	var execsResp map[string]interface{}
	_ = json.Unmarshal(recExecs.Body.Bytes(), &execsResp)
	execList := execsResp["data"].(map[string]interface{})["executions"].([]interface{})
	if len(execList) == 0 {
		t.Errorf("expected at least 1 execution record, got 0")
	}
}

func TestBillingSchedule_WorkerClaimingAndConcurrency(t *testing.T) {
	repo, socID, superToken, templateID, _, _, adminUID := setupSchedulerFixture(t)
	ctx := context.Background()

	// Create a schedule that is immediately due (next_run_at in past)
	pastTime := time.Now().Add(-10 * time.Minute)
	sched, err := repo.CreateBillingSchedule(ctx, socID, society.CreateBillingScheduleParams{
		Name:           "Immediately Due Schedule",
		RecurrenceType: "MONTHLY",
		TemplateID:     templateID,
		SelectionMode:  "ALL_UNITS",
		DayOfMonth:     1,
		DueDays:        15,
		Timezone:       "Asia/Kolkata",
		AutoFinalize:   false,
		FirstRunAt:     &pastTime,
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create due schedule: %v", err)
	}

	// Test concurrent claiming across 2 simulated workers
	worker1 := "worker-instance-01"
	worker2 := "worker-instance-02"

	var wg sync.WaitGroup
	var claimed1, claimed2 []society.BillingSchedule
	var err1, err2 error

	wg.Add(2)
	go func() {
		defer wg.Done()
		claimed1, err1 = repo.ClaimDueSchedules(ctx, worker1, 10)
	}()
	go func() {
		defer wg.Done()
		claimed2, err2 = repo.ClaimDueSchedules(ctx, worker2, 10)
	}()
	wg.Wait()

	if err1 != nil {
		t.Errorf("worker 1 claim error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("worker 2 claim error: %v", err2)
	}

	totalClaimed := len(claimed1) + len(claimed2)
	if totalClaimed != 1 {
		t.Errorf("FOR UPDATE SKIP LOCKED failed: expected exactly 1 worker to claim schedule, got total %d (worker1: %d, worker2: %d)",
			totalClaimed, len(claimed1), len(claimed2))
	}

	// Verify the claimed schedule has locked_by set
	claimedSched, _ := repo.GetBillingSchedule(ctx, socID, sched.ID)
	if claimedSched.LockedBy == nil {
		t.Errorf("expected locked_by to be set on claimed schedule")
	}

	// Execute the claimed schedule
	exec, execErr := repo.ExecuteScheduleRun(ctx, socID, sched.ID, "SCHEDULED", adminUID, "127.0.0.1", "worker")
	if execErr != nil {
		t.Fatalf("execute schedule run: %v", execErr)
	}
	if exec.Status != "SUCCESS" {
		t.Errorf("expected SUCCESS, got %s", exec.Status)
	}

	// Verify lock released and next_run_at advanced
	refreshedSched, _ := repo.GetBillingSchedule(ctx, socID, sched.ID)
	if refreshedSched.LockedBy != nil {
		t.Errorf("expected locked_by to be cleared after execution")
	}
	if !refreshedSched.NextRunAt.After(time.Now()) {
		t.Errorf("expected next_run_at to advance into future, got %v", refreshedSched.NextRunAt)
	}
	_ = superToken
}

func TestBillingSchedule_AuthorizationAndTenantIsolation(t *testing.T) {
	repo, socA_ID, superToken, templateA_ID, router, createUserToken, adminUID := setupSchedulerFixture(t)
	ctx := context.Background()

	// Create resident user token
	resPhone := fmt.Sprintf("9197%08d", time.Now().UnixNano()%100000000)
	_, resToken := createUserToken(resPhone)

	// Resident should be denied access (403)
	reqRes := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/schedules", socA_ID), nil)
	reqRes.Header.Set("Authorization", "Bearer "+resToken)
	recRes := httptest.NewRecorder()
	router.ServeHTTP(recRes, reqRes)
	if recRes.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for resident, got %d", recRes.Code)
	}

	// Create second society B via platform endpoint
	uniqueTime := time.Now().UnixNano()
	regNo := fmt.Sprintf("REG-ISOL-%d", uniqueTime%1000000)
	socBParams := society.CreateSocietyParams{
		Name:               fmt.Sprintf("Isolation Society %d", uniqueTime),
		RegistrationNumber: &regNo,
		AddressLine1:       "456 Isolation Rd",
		Locality:           "Cyber City",
		City:               "Gurugram",
		State:              "Haryana",
		Pincode:            "122002",
	}
	socBJSON, _ := json.Marshal(socBParams)
	reqSocB := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socBJSON))
	reqSocB.Header.Set("Authorization", "Bearer "+superToken)
	reqSocB.Header.Set("Content-Type", "application/json")
	wSocB := httptest.NewRecorder()
	router.ServeHTTP(wSocB, reqSocB)
	if wSocB.Code != http.StatusCreated {
		t.Fatalf("expected 201 created society B, got %d: %s", wSocB.Code, wSocB.Body.String())
	}
	socBData := parseTestData(wSocB.Body.Bytes())
	socB_ID := uuid.MustParse(socBData["id"].(string))

	// Create schedule in Society A
	schedA, err := repo.CreateBillingSchedule(ctx, socA_ID, society.CreateBillingScheduleParams{
		Name:           "Society A Schedule",
		RecurrenceType: "MONTHLY",
		TemplateID:     templateA_ID,
		DayOfMonth:     1,
		DueDays:        15,
		Timezone:       "Asia/Kolkata",
	}, adminUID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("create schedule A: %v", err)
	}

	// Attempting to access Society A's schedule via Society B's endpoint should return 404
	reqCross := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/schedules/%s", socB_ID, schedA.ID), nil)
	reqCross.Header.Set("Authorization", "Bearer "+superToken)
	recCross := httptest.NewRecorder()
	router.ServeHTTP(recCross, reqCross)
	if recCross.Code != http.StatusNotFound && recCross.Code != http.StatusForbidden {
		t.Errorf("tenant isolation failure: expected 403 or 404 when querying Society A schedule under Society B, got %d", recCross.Code)
	}
}
