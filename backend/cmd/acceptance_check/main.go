package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID                 string  `json:"id"`
		Phone              string  `json:"phone"`
		Role               *string `json:"platform_role"`
		SocietyAssignments []struct {
			SocietyID   string `json:"society_id"`
			SocietyName string `json:"society_name"`
			Role        string `json:"role"`
		} `json:"society_assignments"`
	} `json:"user"`
}

type ResidentUnit struct {
	ID                      string `json:"id"`
	UnitIdentifier          string `json:"unit_identifier"`
	RequiresVisitorApproval bool   `json:"requires_visitor_approval"`
}

type VisitResponse struct {
	Visit struct {
		ID                       string `json:"id"`
		Status                   string `json:"status"`
		RequiresResidentApproval bool   `json:"requires_resident_approval"`
	} `json:"visit"`
}

type VisitorPassResponse struct {
	Pass struct {
		ID       string `json:"id"`
		PassCode string `json:"pass_code"`
		Status   string `json:"status"`
	} `json:"pass"`
}

const BaseURL = "http://localhost:8080/api/v1"

func main() {
	fmt.Println("================================================================================")
	fmt.Println("           S09 FINAL ACCEPTANCE CRITERIA VERIFICATION SUITE")
	fmt.Println("================================================================================")

	cfg := config.Load()
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("FAIL: Connect error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	conn := db.Pool()
	ctx := context.Background()

	var puneSocID, mumbaiSocID string
	_ = conn.QueryRow(ctx, "SELECT id FROM societies WHERE city = 'Pune'").Scan(&puneSocID)
	_ = conn.QueryRow(ctx, "SELECT id FROM societies WHERE city = 'Mumbai'").Scan(&mumbaiSocID)

	var flat101ID, flat102ID string
	_ = conn.QueryRow(ctx, "SELECT id FROM society_units WHERE society_id = $1 AND unit_identifier = '101'", puneSocID).Scan(&flat101ID)
	_ = conn.QueryRow(ctx, "SELECT id FROM society_units WHERE society_id = $1 AND unit_identifier = '102'", puneSocID).Scan(&flat102ID)

	var puneGateID string
	_ = conn.QueryRow(ctx, "SELECT id FROM society_gates WHERE society_id = $1 LIMIT 1", puneSocID).Scan(&puneGateID)

	fmt.Printf("[CONTEXT] Pune Society ID: %s | Flat 101 ID: %s | Flat 102 ID: %s | Gate: %s\n", puneSocID, flat101ID, flat102ID, puneGateID)
	fmt.Printf("[CONTEXT] Mumbai Society ID: %s\n\n", mumbaiSocID)

	// Clean any previous test visits for clean run
	_, _ = conn.Exec(ctx, "DELETE FROM society_gate_events WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visitor_passes WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visit_approvals WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visits WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "UPDATE society_units SET requires_visitor_approval = FALSE WHERE id IN ($1, $2)", flat101ID, flat102ID)

	// =========================================================================
	// SCENARIO 1: MEMBER LOGIN & SOCIETY ASSIGNMENT RESOLUTION (AC-S09-F01)
	// =========================================================================
	fmt.Println("--- SCENARIO 1: MEMBER LOGIN & SOCIETY ASSIGNMENT RESOLUTION ---")
	vijayToken, _, err := loginOTP(conn, "919822000101", "123456")
	if err != nil {
		fatalf("Vijay Godbole OTP login failed: %v", err)
	}
	fmt.Printf("PASS: Vijay Godbole authenticated via OTP (+919822000101)\n")

	status, body, err := doRequest("GET", fmt.Sprintf("%s/auth/me", BaseURL), vijayToken, nil)
	if status != 200 {
		fatalf("GET /auth/me failed (%d): %s", status, body)
	}
	var meRes struct {
		SocietyAssignments []struct {
			SocietyID   string `json:"society_id"`
			SocietyName string `json:"society_name"`
			Role        string `json:"role"`
		} `json:"society_assignments"`
	}
	decodeEnvelopeData(body, &meRes)

	hasPuneAssignment := false
	for _, a := range meRes.SocietyAssignments {
		if a.SocietyID == puneSocID && a.Role == "RESIDENT" {
			hasPuneAssignment = true
		}
	}
	if !hasPuneAssignment {
		fatalf("Expected Pune society assignment with role RESIDENT, got %+v", meRes.SocietyAssignments)
	}
	fmt.Printf("PASS: User resolved society assignment: Greenview Residency with role RESIDENT\n")

	// Verify unaffiliated society access denied
	status, _, _ = doRequest("GET", fmt.Sprintf("%s/societies/%s/units/my", BaseURL, mumbaiSocID), vijayToken, nil)
	if status == 200 {
		fatalf("Vijay should NOT have access to Mumbai society, got %d", status)
	}
	fmt.Printf("PASS: Unaffiliated society access denied (HTTP %d)\n", status)

	// =========================================================================
	// SCENARIO 2: RESIDENT PREFERENCE DEFAULT OFF & PERSISTENCE (AC-S09-F02, F05)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 2: RESIDENT PREFERENCE DEFAULT OFF & PERSISTENCE ---")
	status, body, err = doRequest("GET", fmt.Sprintf("%s/societies/%s/units/my", BaseURL, puneSocID), vijayToken, nil)
	if status != 200 {
		fatalf("GET /units/my failed (%d): %s", status, body)
	}
	var myUnitsRes struct {
		Units []ResidentUnit `json:"units"`
	}
	decodeEnvelopeData(body, &myUnitsRes)
	if len(myUnitsRes.Units) == 0 {
		fatalf("Vijay has no authorized units")
	}
	if myUnitsRes.Units[0].RequiresVisitorApproval != false {
		fatalf("Expected default requires_visitor_approval = false, got true")
	}
	fmt.Printf("PASS: Resident preference defaults to OFF (requires_visitor_approval = false)\n")

	// Enable preference
	status, body, _ = doRequest("PATCH", fmt.Sprintf("%s/societies/%s/units/%s/visitor-policy", BaseURL, puneSocID, flat101ID), vijayToken, map[string]bool{
		"requires_visitor_approval": true,
	})
	if status != 200 {
		fatalf("PATCH /units/policy failed (%d): %s", status, body)
	}
	fmt.Printf("PASS: Resident enabled approval requirement (requires_visitor_approval = true)\n")

	// Verify persistence
	_, body, _ = doRequest("GET", fmt.Sprintf("%s/societies/%s/units/my", BaseURL, puneSocID), vijayToken, nil)
	decodeEnvelopeData(body, &myUnitsRes)
	if myUnitsRes.Units[0].RequiresVisitorApproval != true {
		fatalf("Preference did not persist as true")
	}
	fmt.Printf("PASS: Preference persists across queries\n")

	// =========================================================================
	// SCENARIO 3: WALK-IN REGISTRATION WITH APPROVAL ON (AC-S09-F03, F07)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 3: WALK-IN REGISTRATION WITH APPROVAL ON (BACKEND RESOLUTION) ---")
	guardToken, _, err := loginOTP(conn, "919820020001", "123456")
	if err != nil {
		fatalf("Guard OTP login failed: %v", err)
	}
	fmt.Printf("PASS: Pune Gatekeeper authenticated (+919820020001)\n")

	// Gatekeeper attempts client override requires_approval = false, backend MUST resolve from unit (true)
	walkInPayload := map[string]interface{}{
		"unit_id":           flat101ID,
		"visitor_name":      "Courier Boy Suresh",
		"visitor_phone":     "+919820112233",
		"purpose":           "DELIVERY",
		"requires_approval": false, // Ignored by backend!
	}
	status, body, err = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/walk-in", BaseURL, puneSocID), guardToken, walkInPayload)
	if status != 201 {
		fatalf("Walk-in creation failed (%d): %s", status, body)
	}
	var visit1 VisitResponse
	decodeEnvelopeData(body, &visit1)
	if visit1.Visit.Status != "PENDING_APPROVAL" {
		fatalf("Expected visit status PENDING_APPROVAL, got %s", visit1.Visit.Status)
	}
	fmt.Printf("PASS: Walk-in held in PENDING_APPROVAL per destination unit policy (client override ignored)\n")

	// =========================================================================
	// SCENARIO 4: RESIDENT APPROVAL WORKFLOW (AC-S09-F06)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 4: RESIDENT APPROVAL WORKFLOW ---")
	status, body, _ = doRequest("GET", fmt.Sprintf("%s/societies/%s/visitors/pending", BaseURL, puneSocID), vijayToken, nil)
	var pendingRes struct {
		Visits []struct {
			ID string `json:"id"`
		} `json:"visits"`
	}
	decodeEnvelopeData(body, &pendingRes)
	if len(pendingRes.Visits) == 0 || pendingRes.Visits[0].ID != visit1.Visit.ID {
		fatalf("Pending visit not visible to authorized resident")
	}
	fmt.Printf("PASS: Pending visit visible in resident queue\n")

	// Approve visit
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/approve", BaseURL, puneSocID, visit1.Visit.ID), vijayToken, map[string]string{
		"notes": "Approved by flat occupant Vijay",
	})
	if status != 200 {
		fatalf("Approve visit failed (%d): %s", status, body)
	}
	var approvedVisit VisitResponse
	decodeEnvelopeData(body, &approvedVisit)
	if approvedVisit.Visit.Status != "APPROVED" {
		fatalf("Expected status APPROVED, got %s", approvedVisit.Visit.Status)
	}
	fmt.Printf("PASS: Resident approved visit (status transitioned to APPROVED)\n")

	// =========================================================================
	// SCENARIO 5: RESIDENT REJECTION WORKFLOW (AC-S09-F06)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 5: RESIDENT REJECTION WORKFLOW ---")
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/walk-in", BaseURL, puneSocID), guardToken, map[string]interface{}{
		"unit_id":       flat101ID,
		"visitor_name":  "Unsolicited Salesman",
		"visitor_phone": "+919820445566",
		"purpose":       "OTHER",
	})
	var visit2 VisitResponse
	decodeEnvelopeData(body, &visit2)

	// Reject visit
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/reject", BaseURL, puneSocID, visit2.Visit.ID), vijayToken, map[string]string{
		"reason": "Not expected. Denied entry.",
	})
	if status != 200 {
		fatalf("Reject visit failed (%d): %s", status, body)
	}
	var rejectedVisit VisitResponse
	decodeEnvelopeData(body, &rejectedVisit)
	if rejectedVisit.Visit.Status != "REJECTED" {
		fatalf("Expected status REJECTED, got %s", rejectedVisit.Visit.Status)
	}
	fmt.Printf("PASS: Resident rejected visit (status transitioned to REJECTED)\n")

	// Gatekeeper check-in must be rejected
	status, _, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-in", BaseURL, puneSocID, visit2.Visit.ID), guardToken, map[string]string{
		"gate_id": puneGateID,
	})
	if status == 200 {
		fatalf("Rejected visit must NOT be checked in")
	}
	fmt.Printf("PASS: Check-in denied for rejected visitor (HTTP %d)\n", status)

	// =========================================================================
	// SCENARIO 6: CROSS-UNIT AUTHORIZATION ENFORCEMENT (AC-S09-F06)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 6: CROSS-UNIT AUTHORIZATION ENFORCEMENT ---")
	// Vijay attempts to update policy for Flat 102
	status, _, _ = doRequest("PATCH", fmt.Sprintf("%s/societies/%s/units/%s/visitor-policy", BaseURL, puneSocID, flat102ID), vijayToken, map[string]bool{
		"requires_visitor_approval": true,
	})
	if status != 403 {
		fatalf("Vijay should NOT be allowed to update Flat 102 policy, got HTTP %d", status)
	}
	fmt.Printf("PASS: Cross-unit preference modification denied (HTTP 403 OCCUPANT_UNAUTHORIZED)\n")

	// Guard registers visit for Flat 102
	_, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/walk-in", BaseURL, puneSocID), guardToken, map[string]interface{}{
		"unit_id":       flat102ID,
		"visitor_name":  "Flat 102 Guest",
		"visitor_phone": "+919820778899",
		"purpose":       "GUEST",
	})
	var visit102 VisitResponse
	decodeEnvelopeData(body, &visit102)

	// Vijay attempts to approve or reject Flat 102 visit
	status, _, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/approve", BaseURL, puneSocID, visit102.Visit.ID), vijayToken, map[string]string{
		"notes": "Unauthorized approval attempt",
	})
	if status != 403 {
		fatalf("Vijay should NOT be allowed to approve Flat 102 visit, got HTTP %d", status)
	}
	fmt.Printf("PASS: Cross-unit visitor approval denied (HTTP 403 OCCUPANT_UNAUTHORIZED)\n")

	// Gatekeeper attempts to approve visit
	status, _, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/approve", BaseURL, puneSocID, visit102.Visit.ID), guardToken, map[string]string{})
	if status != 403 {
		fatalf("Gatekeeper should NOT be allowed to approve visit, got HTTP %d", status)
	}
	fmt.Printf("PASS: Gatekeeper visitor approval denied (HTTP 403 GATEKEEPER_UNAUTHORIZED)\n")

	// =========================================================================
	// SCENARIO 7: DIRECT ENTRY WHEN APPROVAL OFF (AC-S09-F02)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 7: DIRECT ENTRY WHEN APPROVAL OFF ---")
	// Turn preference back OFF for Flat 101
	status, _, _ = doRequest("PATCH", fmt.Sprintf("%s/societies/%s/units/%s/visitor-policy", BaseURL, puneSocID, flat101ID), vijayToken, map[string]bool{
		"requires_visitor_approval": false,
	})
	if status != 200 {
		fatalf("Failed to turn preference OFF")
	}

	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/walk-in", BaseURL, puneSocID), guardToken, map[string]interface{}{
		"unit_id":       flat101ID,
		"visitor_name":  "Direct Walk-In Visitor",
		"visitor_phone": "+919820888999",
		"purpose":       "GUEST",
	})
	var directVisit VisitResponse
	decodeEnvelopeData(body, &directVisit)
	if directVisit.Visit.Status != "APPROVED" {
		fatalf("Expected direct approval when policy OFF, got %s", directVisit.Visit.Status)
	}
	fmt.Printf("PASS: Direct walk-in automatically APPROVED when unit approval is OFF\n")

	// =========================================================================
	// SCENARIO 8: ACTIVE DUTY-SESSION ENFORCEMENT (AC-S09-F10)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 8: ACTIVE DUTY-SESSION ENFORCEMENT ---")
	// Mumbai Guard has active session at Mumbai gate, but NOT at Pune gate
	mumbaiGuardToken, _, err := loginOTP(conn, "919820010001", "123456")
	if err != nil {
		fatalf("Mumbai guard login failed: %v", err)
	}

	// Mumbai Guard attempts to check in visitor at Pune gate
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-in", BaseURL, puneSocID, directVisit.Visit.ID), mumbaiGuardToken, map[string]string{
		"gate_id": puneGateID,
	})
	if status != 403 {
		fatalf("Expected 403 DUTY_SESSION_REQUIRED or access denied for guard at wrong gate, got %d", status)
	}
	fmt.Printf("PASS: Check-in denied without active duty session at specified gate (HTTP %d)\n", status)

	// Pune Guard HAS active duty session at Pune gate
	status, body, err = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-in", BaseURL, puneSocID, directVisit.Visit.ID), guardToken, map[string]string{
		"gate_id": puneGateID,
	})
	if status != 200 {
		fatalf("Check-in with active duty session failed (%d): %s", status, body)
	}
	fmt.Printf("PASS: Check-in succeeded with verified active duty session (status transitioned to CHECKED_IN)\n")

	// Check-out
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-out", BaseURL, puneSocID, directVisit.Visit.ID), guardToken, map[string]string{
		"gate_id": puneGateID,
	})
	if status != 200 {
		fatalf("Check-out failed (%d): %s", status, body)
	}
	fmt.Printf("PASS: Check-out succeeded with verified active duty session (status transitioned to CHECKED_OUT)\n")

	// =========================================================================
	// SCENARIO 9: PASS VERIFICATION AND CHECK-IN ATOMICITY (AC-S09-F08, F09)
	// =========================================================================
	fmt.Println("\n--- SCENARIO 9: PASS VERIFICATION AND ATOMICITY ---")
	// Pre-register expected visit
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/expected", BaseURL, puneSocID), vijayToken, map[string]interface{}{
		"unit_id":       flat101ID,
		"visitor_name":  "Expected Guest Priya",
		"visitor_phone": "+919820334455",
		"purpose":       "GUEST",
	})
	if status != 201 {
		fatalf("Expected visit creation failed (%d): %s", status, body)
	}
	var expVisit VisitResponse
	decodeEnvelopeData(body, &expVisit)

	// Issue pass
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/passes", BaseURL, puneSocID, expVisit.Visit.ID), vijayToken, map[string]interface{}{})
	if status != 201 {
		fatalf("Pass issuance failed (%d): %s", status, body)
	}
	var passRes VisitorPassResponse
	decodeEnvelopeData(body, &passRes)
	fmt.Printf("PASS: Pass issued (Pass Code: %s)\n", passRes.Pass.PassCode)

	// Verify pass (read-only verification)
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/passes/verify", BaseURL, puneSocID), guardToken, map[string]string{
		"token_or_code": passRes.Pass.PassCode,
	})
	if status != 200 {
		fatalf("Pass verification failed (%d): %s", status, body)
	}
	fmt.Printf("PASS: Pass verification succeeded without premature consumption\n")

	// Check-in consumes pass
	status, body, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-in", BaseURL, puneSocID, expVisit.Visit.ID), guardToken, map[string]string{
		"gate_id":   puneGateID,
		"pass_code": passRes.Pass.PassCode,
	})
	if status != 200 {
		fatalf("Check-in with pass failed (%d): %s", status, body)
	}
	fmt.Printf("PASS: Visitor checked in with pass\n")

	// Verify pass cannot be reused
	var passStatus string
	_ = conn.QueryRow(ctx, "SELECT status FROM society_visitor_passes WHERE id = $1", passRes.Pass.ID).Scan(&passStatus)
	if passStatus != "CONSUMED" {
		fatalf("Expected pass status CONSUMED, got %s", passStatus)
	}
	fmt.Printf("PASS: Pass status atomically updated to CONSUMED\n")

	// Second check-in must fail with 409
	status, _, _ = doRequest("POST", fmt.Sprintf("%s/societies/%s/visitors/%s/check-in", BaseURL, puneSocID, expVisit.Visit.ID), guardToken, map[string]string{
		"gate_id": puneGateID,
	})
	if status != 409 {
		fatalf("Expected 409 on duplicate check-in, got %d", status)
	}
	fmt.Printf("PASS: Duplicate check-in rejected with HTTP 409 Conflict\n")

	// =========================================================================
	// CLEANUP TEST VISITS & PRESERVE DEVELOPMENT BASELINE (AC-S09-F14)
	// =========================================================================
	_, _ = conn.Exec(ctx, "DELETE FROM society_gate_events WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visitor_passes WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visit_approvals WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "DELETE FROM society_visits WHERE society_id = $1", puneSocID)
	_, _ = conn.Exec(ctx, "UPDATE society_units SET requires_visitor_approval = FALSE WHERE id IN ($1, $2)", flat101ID, flat102ID)

	var socCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM societies").Scan(&socCount)
	if socCount != 2 {
		fatalf("Database corrupt: expected exactly 2 societies, found %d", socCount)
	}
	fmt.Printf("PASS: Baseline seed preserved (Exactly %d societies in database)\n", socCount)

	fmt.Println("\n================================================================================")
	fmt.Println("       ALL S09 ACCEPTANCE SCENARIOS PASSED WITH LIVE API & DB EVIDENCE")
	fmt.Println("================================================================================")
}

func loginOTP(conn *pgxpool.Pool, phone, code string) (string, AuthTokens, error) {
	if conn != nil {
		_, _ = conn.Exec(context.Background(), "DELETE FROM otp_challenges WHERE phone = $1", phone)
	}
	sendPayload, _ := json.Marshal(map[string]string{"phone": phone})
	var resp *http.Response
	var err error
	var sendBytes []byte
	for attempt := 0; attempt < 12; attempt++ {
		resp, err = http.Post(fmt.Sprintf("%s/auth/otp/send", BaseURL), "application/json", bytes.NewBuffer(sendPayload))
		if err != nil {
			return "", AuthTokens{}, fmt.Errorf("failed to send OTP: %w", err)
		}
		sendBytes, _ = io.ReadAll(resp.Body)
		if resp.StatusCode == 429 {
			time.Sleep(5 * time.Second)
			continue
		}
		break
	}
	if resp.StatusCode != 200 {
		return "", AuthTokens{}, fmt.Errorf("send OTP returned %d: %s", resp.StatusCode, string(sendBytes))
	}

	var sendEnv struct {
		Data struct {
			OTP string `json:"otp"`
		} `json:"data"`
	}
	_ = json.Unmarshal(sendBytes, &sendEnv)
	if sendEnv.Data.OTP != "" {
		code = sendEnv.Data.OTP
	}

	verifyPayload, _ := json.Marshal(map[string]string{"phone": phone, "otp": code})
	vResp, err := http.Post(fmt.Sprintf("%s/auth/otp/verify", BaseURL), "application/json", bytes.NewBuffer(verifyPayload))
	if err != nil {
		return "", AuthTokens{}, fmt.Errorf("failed to verify OTP: %w", err)
	}
	defer vResp.Body.Close()
	if vResp.StatusCode != 200 {
		b, _ := io.ReadAll(vResp.Body)
		return "", AuthTokens{}, fmt.Errorf("verify OTP returned %d: %s", vResp.StatusCode, string(b))
	}

	var env APIEnvelope
	if err := json.NewDecoder(vResp.Body).Decode(&env); err != nil {
		return "", AuthTokens{}, fmt.Errorf("failed to decode response envelope: %w", err)
	}

	var tokens AuthTokens
	if err := json.Unmarshal(env.Data, &tokens); err != nil {
		return "", AuthTokens{}, fmt.Errorf("failed to unmarshal auth tokens: %w", err)
	}

	return tokens.AccessToken, tokens, nil
}

func doRequest(method, url string, token string, bodyData interface{}) (int, string, error) {
	var bodyReader io.Reader
	if bodyData != nil {
		b, _ := json.Marshal(bodyData)
		bodyReader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBytes), nil
}

func decodeEnvelopeData(body string, target interface{}) {
	var env APIEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		fatalf("Failed to decode envelope: %v", err)
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		fatalf("Failed to decode data inside envelope: %v", err)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Printf("FAIL: "+format+"\n", args...)
	os.Exit(1)
}
