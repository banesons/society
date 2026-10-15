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

func parseData(body []byte) map[string]interface{} {
	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)
	if d, ok := raw["data"].(map[string]interface{}); ok {
		return d
	}
	return raw
}

func TestGateFoundationAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
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

	// Create a test society
	socName := fmt.Sprintf("Gate Security Society %d", time.Now().UnixNano())
	regNo := fmt.Sprintf("REG-GATE-%d", time.Now().UnixNano()%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "100 Perimeter Way",
		Locality:           "Hiranandani",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400076",
	}
	socJSON, _ := json.Marshal(socParams)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}
	socData := parseData(w.Body.Bytes())
	socID := uuid.MustParse(socData["id"].(string))

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)
	if wAct.Code != http.StatusOK {
		t.Fatalf("expected 200 activated, got %d: %s", wAct.Code, wAct.Body.String())
	}

	// Create test gates in the society
	gate1Params := society.CreateGateParams{
		Name:                  "North Gate 1",
		GateCode:              "NG-01",
		GateType:              "VEHICLE_AND_PEDESTRIAN",
		RequiredStaffPerShift: 2,
	}
	g1JSON, _ := json.Marshal(gate1Params)
	reqG1 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates", socID), bytes.NewReader(g1JSON))
	reqG1.Header.Set("Authorization", "Bearer "+superToken)
	reqG1.Header.Set("Content-Type", "application/json")
	wG1 := httptest.NewRecorder()
	router.ServeHTTP(wG1, reqG1)
	if wG1.Code != http.StatusCreated {
		t.Fatalf("expected 201 for gate 1, got %d: %s", wG1.Code, wG1.Body.String())
	}
	gate1ID := uuid.MustParse(parseData(wG1.Body.Bytes())["id"].(string))

	gate2Params := society.CreateGateParams{
		Name:                  "South Service Gate",
		GateCode:              "SG-02",
		GateType:              "SERVICE",
		RequiredStaffPerShift: 1,
	}
	g2JSON, _ := json.Marshal(gate2Params)
	reqG2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates", socID), bytes.NewReader(g2JSON))
	reqG2.Header.Set("Authorization", "Bearer "+superToken)
	reqG2.Header.Set("Content-Type", "application/json")
	wG2 := httptest.NewRecorder()
	router.ServeHTTP(wG2, reqG2)
	if wG2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for gate 2, got %d: %s", wG2.Code, wG2.Body.String())
	}
	gate2ID := uuid.MustParse(parseData(wG2.Body.Bytes())["id"].(string))

	// Subtest 1: Update Gate Operational Configuration
	t.Run("Update Gate Operational Configuration", func(t *testing.T) {
		name := "North Main Gate (Inbound)"
		dir := "INGRESS_ONLY"
		opStatus := "ACTIVE"
		staff := 3
		desc := "Main vehicle inflow lane"
		updateParams := society.UpdateGateParams{
			Name:                  &name,
			Directionality:        &dir,
			OperationalStatus:     &opStatus,
			RequiredStaffPerShift: &staff,
			Description:           &desc,
		}
		uJSON, _ := json.Marshal(updateParams)
		reqU := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/gates/%s", socID, gate1ID), bytes.NewReader(uJSON))
		reqU.Header.Set("Authorization", "Bearer "+superToken)
		reqU.Header.Set("Content-Type", "application/json")
		wU := httptest.NewRecorder()
		router.ServeHTTP(wU, reqU)

		if wU.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wU.Code, wU.Body.String())
		}

		gateData := parseData(wU.Body.Bytes())["gate"].(map[string]interface{})
		if gateData["directionality"] != "INGRESS_ONLY" || gateData["required_staff_per_shift"].(float64) != 3 {
			t.Errorf("unexpected updated gate values: %+v", gateData)
		}
	})

	// Subtest 2: Assign Gatekeeper Staff
	guardPhone := fmt.Sprintf("+9198200%05d", time.Now().UnixNano()%100000)
	var guardUserID uuid.UUID
	var guardToken string

	t.Run("Assign Gatekeeper by phone", func(t *testing.T) {
		assignPayload := map[string]string{"phone": guardPhone}
		aJSON, _ := json.Marshal(assignPayload)
		reqA := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), bytes.NewReader(aJSON))
		reqA.Header.Set("Authorization", "Bearer "+superToken)
		reqA.Header.Set("Content-Type", "application/json")
		wA := httptest.NewRecorder()
		router.ServeHTTP(wA, reqA)

		if wA.Code != http.StatusCreated {
			t.Fatalf("expected 201 created, got %d: %s", wA.Code, wA.Body.String())
		}

		gkData := parseData(wA.Body.Bytes())["gatekeeper"].(map[string]interface{})
		guardUserID = uuid.MustParse(gkData["user_id"].(string))
		guardToken = generateToken(guardUserID)

		if gkData["role"] != "GATEKEEPER" || gkData["status"] != "ACTIVE" {
			t.Errorf("unexpected gatekeeper assignment: %+v", gkData)
		}
	})

	// Subtest 3: Reject Duplicate Gatekeeper Assignment
	t.Run("Reject duplicate Gatekeeper assignment", func(t *testing.T) {
		assignPayload := map[string]string{"phone": guardPhone}
		aJSON, _ := json.Marshal(assignPayload)
		reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), bytes.NewReader(aJSON))
		reqDup.Header.Set("Authorization", "Bearer "+superToken)
		reqDup.Header.Set("Content-Type", "application/json")
		wDup := httptest.NewRecorder()
		router.ServeHTTP(wDup, reqDup)

		if wDup.Code != http.StatusConflict {
			t.Fatalf("expected 409 conflict, got %d: %s", wDup.Code, wDup.Body.String())
		}
	})

	// Subtest 4: List Gatekeepers
	t.Run("List Gatekeepers", func(t *testing.T) {
		reqL := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), nil)
		reqL.Header.Set("Authorization", "Bearer "+superToken)
		wL := httptest.NewRecorder()
		router.ServeHTTP(wL, reqL)

		if wL.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wL.Code, wL.Body.String())
		}

		items := parseData(wL.Body.Bytes())["gatekeepers"].([]interface{})
		if len(items) == 0 {
			t.Fatalf("expected at least 1 gatekeeper, got 0")
		}
	})

	// Subtest 5: Role Segregation — GATEKEEPER cannot access admin endpoints
	t.Run("GATEKEEPER cannot access admin endpoints", func(t *testing.T) {
		// Attempt to access members list
		reqM := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/members", socID), nil)
		reqM.Header.Set("Authorization", "Bearer "+guardToken)
		wM := httptest.NewRecorder()
		router.ServeHTTP(wM, reqM)

		if wM.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden for gatekeeper accessing members, got %d: %s", wM.Code, wM.Body.String())
		}

		// Attempt to modify gate
		dir := "EGRESS_ONLY"
		uJSON, _ := json.Marshal(society.UpdateGateParams{Directionality: &dir})
		reqMod := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/gates/%s", socID, gate1ID), bytes.NewReader(uJSON))
		reqMod.Header.Set("Authorization", "Bearer "+guardToken)
		reqMod.Header.Set("Content-Type", "application/json")
		wMod := httptest.NewRecorder()
		router.ServeHTTP(wMod, reqMod)

		if wMod.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden for gatekeeper updating gate, got %d: %s", wMod.Code, wMod.Body.String())
		}
	})

	// Subtest 6: Create & List Shifts
	var shiftID uuid.UUID
	t.Run("Create & List Shifts", func(t *testing.T) {
		shiftParams := society.CreateGateShiftParams{
			Name:      "Morning Shift",
			StartTime: "06:00:00",
			EndTime:   "14:00:00",
		}
		sJSON, _ := json.Marshal(shiftParams)
		reqS := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/shifts", socID), bytes.NewReader(sJSON))
		reqS.Header.Set("Authorization", "Bearer "+superToken)
		reqS.Header.Set("Content-Type", "application/json")
		wS := httptest.NewRecorder()
		router.ServeHTTP(wS, reqS)

		if wS.Code != http.StatusCreated {
			t.Fatalf("expected 201 created, got %d: %s", wS.Code, wS.Body.String())
		}
		shiftID = uuid.MustParse(parseData(wS.Body.Bytes())["shift"].(map[string]interface{})["id"].(string))

		// Duplicate shift name rejected
		wDup := httptest.NewRecorder()
		reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/shifts", socID), bytes.NewReader(sJSON))
		reqDup.Header.Set("Authorization", "Bearer "+superToken)
		reqDup.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(wDup, reqDup)
		if wDup.Code != http.StatusConflict {
			t.Fatalf("expected 409 conflict, got %d: %s", wDup.Code, wDup.Body.String())
		}

		// Gatekeeper can list shifts
		reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/shifts", socID), nil)
		reqList.Header.Set("Authorization", "Bearer "+guardToken)
		wList := httptest.NewRecorder()
		router.ServeHTTP(wList, reqList)
		if wList.Code != http.StatusOK {
			t.Fatalf("expected 200 for guard listing shifts, got %d: %s", wList.Code, wList.Body.String())
		}
	})

	// Subtest 7: Gatekeeper Check-In to Active Duty Session
	t.Run("Gatekeeper Check-In to Duty Session", func(t *testing.T) {
		checkInBody := society.CheckInDutyParams{ShiftID: &shiftID}
		ciJSON, _ := json.Marshal(checkInBody)
		reqCI := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, gate1ID), bytes.NewReader(ciJSON))
		reqCI.Header.Set("Authorization", "Bearer "+guardToken)
		reqCI.Header.Set("Content-Type", "application/json")
		wCI := httptest.NewRecorder()
		router.ServeHTTP(wCI, reqCI)

		if wCI.Code != http.StatusCreated {
			t.Fatalf("expected 201 check-in, got %d: %s", wCI.Code, wCI.Body.String())
		}

		sessData := parseData(wCI.Body.Bytes())["session"].(map[string]interface{})
		if sessData["status"] != "ACTIVE" || sessData["gate_name"] != "North Main Gate (Inbound)" {
			t.Errorf("unexpected session data: %+v", sessData)
		}
	})

	// Subtest 8: Concurrency — Prevent Double-Booking on Multiple Gates
	t.Run("Prevent Guard Double-Booking Concurrent Sessions", func(t *testing.T) {
		// Guard attempts to check in at Gate 2 while still active at Gate 1
		ciJSON, _ := json.Marshal(society.CheckInDutyParams{})
		reqDup := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, gate2ID), bytes.NewReader(ciJSON))
		reqDup.Header.Set("Authorization", "Bearer "+guardToken)
		reqDup.Header.Set("Content-Type", "application/json")
		wDup := httptest.NewRecorder()
		router.ServeHTTP(wDup, reqDup)

		if wDup.Code != http.StatusConflict {
			t.Fatalf("expected 409 conflict, got %d: %s", wDup.Code, wDup.Body.String())
		}
	})

	// Subtest 9: Inactive Gate Rejects Duty Check-In
	t.Run("Inactive Gate Rejects Duty Check-In", func(t *testing.T) {
		// Deactivate gate 2
		inact := "INACTIVE"
		uJSON, _ := json.Marshal(society.UpdateGateParams{OperationalStatus: &inact})
		reqU := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/gates/%s", socID, gate2ID), bytes.NewReader(uJSON))
		reqU.Header.Set("Authorization", "Bearer "+superToken)
		reqU.Header.Set("Content-Type", "application/json")
		wU := httptest.NewRecorder()
		router.ServeHTTP(wU, reqU)
		if wU.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wU.Code, wU.Body.String())
		}

		// Another guard checks into gate 2
		otherGuardPhone := fmt.Sprintf("+9198200%05d", (time.Now().UnixNano()+100)%100000)
		aJSON, _ := json.Marshal(map[string]string{"phone": otherGuardPhone})
		reqA := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), bytes.NewReader(aJSON))
		reqA.Header.Set("Authorization", "Bearer "+superToken)
		reqA.Header.Set("Content-Type", "application/json")
		wA := httptest.NewRecorder()
		router.ServeHTTP(wA, reqA)
		if wA.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", wA.Code, wA.Body.String())
		}
		otherGuardID := uuid.MustParse(parseData(wA.Body.Bytes())["gatekeeper"].(map[string]interface{})["user_id"].(string))
		otherToken := generateToken(otherGuardID)

		reqCI := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, gate2ID), bytes.NewReader([]byte("{}")))
		reqCI.Header.Set("Authorization", "Bearer "+otherToken)
		reqCI.Header.Set("Content-Type", "application/json")
		wCI := httptest.NewRecorder()
		router.ServeHTTP(wCI, reqCI)

		if wCI.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for inactive gate, got %d: %s", wCI.Code, wCI.Body.String())
		}
	})

	// Subtest 10: List Duty Sessions Roster
	t.Run("List Duty Sessions Roster", func(t *testing.T) {
		reqDS := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/duty-sessions", socID), nil)
		reqDS.Header.Set("Authorization", "Bearer "+guardToken)
		wDS := httptest.NewRecorder()
		router.ServeHTTP(wDS, reqDS)

		if wDS.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wDS.Code, wDS.Body.String())
		}

		sessions := parseData(wDS.Body.Bytes())["sessions"].([]interface{})
		if len(sessions) != 1 {
			t.Fatalf("expected exactly 1 active duty session, got %d", len(sessions))
		}
	})

	// Subtest 11: Gatekeeper Check-Out (Conclude Duty Session)
	t.Run("Gatekeeper Check-Out Concludes Session with Handover Notes", func(t *testing.T) {
		notes := "Perimeter secured, perimeter lights operational, handing over post"
		coBody := society.CheckOutDutyParams{HandoverNotes: &notes}
		coJSON, _ := json.Marshal(coBody)
		reqCO := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-out", socID, gate1ID), bytes.NewReader(coJSON))
		reqCO.Header.Set("Authorization", "Bearer "+guardToken)
		reqCO.Header.Set("Content-Type", "application/json")
		wCO := httptest.NewRecorder()
		router.ServeHTTP(wCO, reqCO)

		if wCO.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", wCO.Code, wCO.Body.String())
		}

		sessData := parseData(wCO.Body.Bytes())["session"].(map[string]interface{})
		if sessData["status"] != "ENDED" || sessData["ended_at"] == nil {
			t.Errorf("expected session ended, got: %+v", sessData)
		}
	})

	// Subtest 12: Revoking Gatekeeper Terminates Active Sessions Automatically
	t.Run("Revoking Gatekeeper Terminates Active Sessions", func(t *testing.T) {
		// Reactivate gate 2
		act := "ACTIVE"
		uJSON, _ := json.Marshal(society.UpdateGateParams{OperationalStatus: &act})
		reqU := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/societies/%s/gates/%s", socID, gate2ID), bytes.NewReader(uJSON))
		reqU.Header.Set("Authorization", "Bearer "+superToken)
		reqU.Header.Set("Content-Type", "application/json")
		wU := httptest.NewRecorder()
		router.ServeHTTP(wU, reqU)

		// Guard checks into Gate 2
		reqCI := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, gate2ID), bytes.NewReader([]byte("{}")))
		reqCI.Header.Set("Authorization", "Bearer "+guardToken)
		reqCI.Header.Set("Content-Type", "application/json")
		wCI := httptest.NewRecorder()
		router.ServeHTTP(wCI, reqCI)
		if wCI.Code != http.StatusCreated {
			t.Fatalf("expected 201 check-in, got %d: %s", wCI.Code, wCI.Body.String())
		}

		// Find guard assignment ID
		var assignID uuid.UUID
		err := db.Pool().QueryRow(ctx, `SELECT id FROM society_user_roles WHERE society_id = $1 AND user_id = $2 AND status = 'ACTIVE'`, socID, guardUserID).Scan(&assignID)
		if err != nil {
			t.Fatalf("failed to find assignment ID: %v", err)
		}

		// Revoke gatekeeper
		reqRev := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/societies/%s/gatekeepers/%s", socID, assignID), nil)
		reqRev.Header.Set("Authorization", "Bearer "+superToken)
		wRev := httptest.NewRecorder()
		router.ServeHTTP(wRev, reqRev)
		if wRev.Code != http.StatusOK {
			t.Fatalf("expected 200 for revoke, got %d: %s", wRev.Code, wRev.Body.String())
		}

		// Verify guard has 0 active sessions now
		var countActive int
		_ = db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM society_gate_sessions WHERE society_id = $1 AND user_id = $2 AND status = 'ACTIVE'`, socID, guardUserID).Scan(&countActive)
		if countActive != 0 {
			t.Fatalf("expected 0 active sessions after revocation, got %d", countActive)
		}
	})

	// Subtest 13: Suspended Society Rejects Gate Operations
	t.Run("Suspended Society Rejects Gate Operations", func(t *testing.T) {
		reasonID := int16(1)
		_, err := societyRepo.SuspendSociety(ctx, socID, society.SuspendSocietyParams{
			ReasonID: &reasonID,
		}, superAdminRole.UserID, "127.0.0.1", "test")
		if err != nil {
			t.Fatalf("failed to suspend society: %v", err)
		}

		// Attempt to check in
		reqCI := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, gate1ID), bytes.NewReader([]byte("{}")))
		reqCI.Header.Set("Authorization", "Bearer "+superToken)
		reqCI.Header.Set("Content-Type", "application/json")
		wCI := httptest.NewRecorder()
		router.ServeHTTP(wCI, reqCI)

		if wCI.Code != http.StatusForbidden {
			t.Fatalf("expected 403 forbidden on suspended society, got %d: %s", wCI.Code, wCI.Body.String())
		}
	})

	// Subtest 14: Genuine Concurrency Conflict on Single Guard Check-In
	t.Run("Concurrent check-in by same guard results in at most one success", func(t *testing.T) {
		// Reactivate society
		_, err := societyRepo.ReactivateSociety(ctx, socID, "Suspension lifted", superAdminRole.UserID, "127.0.0.1", "test")
		if err != nil {
			t.Fatalf("failed to reactivate society: %v", err)
		}

		// Provision third guard
		guard3Phone := fmt.Sprintf("+9198200%05d", (time.Now().UnixNano()+200)%100000)
		aJSON, _ := json.Marshal(map[string]string{"phone": guard3Phone})
		reqA := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gatekeepers", socID), bytes.NewReader(aJSON))
		reqA.Header.Set("Authorization", "Bearer "+superToken)
		reqA.Header.Set("Content-Type", "application/json")
		wA := httptest.NewRecorder()
		router.ServeHTTP(wA, reqA)
		if wA.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", wA.Code, wA.Body.String())
		}
		g3ID := uuid.MustParse(parseData(wA.Body.Bytes())["gatekeeper"].(map[string]interface{})["user_id"].(string))
		g3Token := generateToken(g3ID)

		var wg sync.WaitGroup
		wg.Add(2)
		results := make([]int, 2)

		for i, gate := range []uuid.UUID{gate1ID, gate2ID} {
			idx := i
			targetGate := gate
			go func() {
				defer wg.Done()
				reqCI := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/gates/%s/sessions/check-in", socID, targetGate), bytes.NewReader([]byte("{}")))
				reqCI.Header.Set("Authorization", "Bearer "+g3Token)
				reqCI.Header.Set("Content-Type", "application/json")
				wCI := httptest.NewRecorder()
				router.ServeHTTP(wCI, reqCI)
				results[idx] = wCI.Code
			}()
		}

		wg.Wait()

		successCount := 0
		conflictCount := 0
		for _, code := range results {
			if code == http.StatusCreated {
				successCount++
			} else if code == http.StatusConflict {
				conflictCount++
			}
		}

		if successCount != 1 || conflictCount != 1 {
			t.Fatalf("expected exactly 1 success and 1 conflict, got %v", results)
		}
	})
}
