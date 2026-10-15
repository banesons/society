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

func TestDesignationsAndAppointmentsAPI(t *testing.T) {
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
			t.Fatalf("failed to create test session: %v", err)
		}
		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return tok
	}

	superAdminToken := generateToken(superAdminRole.UserID)

	// Create test society A via HTTP
	uniqueSuffixA := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	createSocA := CreateSocietyRequest{
		Name:         "S07 Test Society " + uniqueSuffixA,
		AddressLine1: "200 Committee Road",
		Locality:     "Shivajinagar",
		City:         "Pune",
		State:        "Maharashtra",
		Pincode:      "411005",
	}
	bodyJSON, _ := json.Marshal(createSocA)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+superAdminToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create test society A: %d %s", rec.Code, rec.Body.String())
	}
	var socAResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &socAResp)
	socAID := uuid.MustParse(socAResp.Data.ID)

	// Activate Society A
	actReqA := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+socAID.String()+"/activate", nil)
	actReqA.Header.Set("Authorization", "Bearer "+superAdminToken)
	actRecA := httptest.NewRecorder()
	router.ServeHTTP(actRecA, actReqA)
	if actRecA.Code != http.StatusOK {
		t.Fatalf("failed to activate society A: %d %s", actRecA.Code, actRecA.Body.String())
	}

	// Create test society B
	createSocB := CreateSocietyRequest{
		Name:         "S07 Cross Society " + uniqueSuffixA,
		AddressLine1: "300 External Road",
		Locality:     "Aundh",
		City:         "Pune",
		State:        "Maharashtra",
		Pincode:      "411007",
	}
	bodyJSON, _ = json.Marshal(createSocB)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+superAdminToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var socBResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &socBResp)
	socBID := uuid.MustParse(socBResp.Data.ID)

	// Activate Society B
	actReqB := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+socBID.String()+"/activate", nil)
	actReqB.Header.Set("Authorization", "Bearer "+superAdminToken)
	actRecB := httptest.NewRecorder()
	router.ServeHTTP(actRecB, actReqB)

	// Register 2 test members in Society A
	member1, err := societyRepo.RegisterMember(ctx, socAID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "Virender",
		Phone:     "919830000001",
		ClientIP:  "127.0.0.1",
		UserAgent: "test",
	})
	if err != nil {
		t.Fatalf("failed to register member 1: %v", err)
	}

	member2, err := societyRepo.RegisterMember(ctx, socAID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "Rahul",
		Phone:     "919830000002",
		ClientIP:  "127.0.0.1",
		UserAgent: "test",
	})
	if err != nil {
		t.Fatalf("failed to register member 2: %v", err)
	}

	var chairmanDesigID, secDesigID, commMemberDesigID uuid.UUID

	t.Run("Unauthenticated request returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Platform ADMIN without society role returns 403", func(t *testing.T) {
		adminPhone := "919830099999"
		var adminUID uuid.UUID
		_ = db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, adminPhone).Scan(&adminUID)
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status, assigned_by)
			VALUES ($1, 'ADMIN', 'ACTIVE', $2)
			ON CONFLICT (user_id) DO UPDATE SET role = 'ADMIN', status = 'ACTIVE'
		`, adminUID, superAdminRole.UserID)
		adminToken := generateToken(adminUID)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for platform admin, got %d", rec.Code)
		}
	})

	t.Run("List designations automatically seeds standard defaults", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Designations []society.SocietyDesignation `json:"designations"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)

		if len(resp.Data.Designations) < 4 {
			t.Fatalf("expected at least 4 default designations, got %d", len(resp.Data.Designations))
		}

		foundCodes := make(map[string]bool)
		for _, d := range resp.Data.Designations {
			foundCodes[d.Code] = true
			if d.Code == society.DesignationCodeChairman {
				chairmanDesigID = d.ID
				if d.AllowsMultipleMembers {
					t.Errorf("expected CHAIRMAN to have allows_multiple_members = false")
				}
			}
			if d.Code == society.DesignationCodeSecretary {
				secDesigID = d.ID
			}
			if d.Code == society.DesignationCodeCommitteeMember {
				commMemberDesigID = d.ID
				if !d.AllowsMultipleMembers {
					t.Errorf("expected COMMITTEE_MEMBER to have allows_multiple_members = true")
				}
			}
		}

		if !foundCodes["CHAIRMAN"] || !foundCodes["SECRETARY"] || !foundCodes["TREASURER"] || !foundCodes["COMMITTEE_MEMBER"] {
			t.Errorf("missing standard default designations: %v", foundCodes)
		}
	})

	var customDesigID uuid.UUID
	t.Run("Create custom designation and reject duplicate code", func(t *testing.T) {
		// Valid creation
		payload := CreateDesignationRequest{
			Code:                  "VICE_CHAIRMAN",
			Name:                  "Vice Chairman",
			AllowsMultipleMembers: false,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Designation society.SocietyDesignation `json:"designation"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		customDesigID = resp.Data.Designation.ID

		// Duplicate code rejected with 409
		reqDup := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), bytes.NewReader(body))
		reqDup.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqDup.Header.Set("Content-Type", "application/json")
		recDup := httptest.NewRecorder()
		router.ServeHTTP(recDup, reqDup)
		if recDup.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate designation code, got %d", recDup.Code)
		}

		// Invalid code format rejected with 400
		badPayload := CreateDesignationRequest{Code: "bad code!", Name: "Bad Code"}
		badBody, _ := json.Marshal(badPayload)
		reqBad := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), bytes.NewReader(badBody))
		reqBad.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqBad.Header.Set("Content-Type", "application/json")
		recBad := httptest.NewRecorder()
		router.ServeHTTP(recBad, reqBad)
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for invalid code regex, got %d", recBad.Code)
		}
	})

	t.Run("Update designation and toggle status", func(t *testing.T) {
		// Update name
		updatePayload := UpdateDesignationRequest{
			Name: "Executive Vice Chairman",
		}
		body, _ := json.Marshal(updatePayload)
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/societies/%s/designations/%s", socAID, customDesigID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		// Deactivate designation
		statusPayload := SetDesignationStatusRequest{IsActive: false}
		sBody, _ := json.Marshal(statusPayload)
		reqStatus := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/societies/%s/designations/%s/status", socAID, customDesigID), bytes.NewReader(sBody))
		reqStatus.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqStatus.Header.Set("Content-Type", "application/json")
		recStatus := httptest.NewRecorder()
		router.ServeHTTP(recStatus, reqStatus)
		if recStatus.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", recStatus.Code, recStatus.Body.String())
		}

		// Verify attempting to appoint to inactive designation fails with 400
		aptPayload := CreateAppointmentRequest{
			DesignationID: customDesigID.String(),
			MemberID:      member1.ID.String(),
			StartDate:     "2026-05-01",
		}
		aBody, _ := json.Marshal(aptPayload)
		reqApt := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(aBody))
		reqApt.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqApt.Header.Set("Content-Type", "application/json")
		recApt := httptest.NewRecorder()
		router.ServeHTTP(recApt, reqApt)
		if recApt.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for appointment to inactive designation, got %d: %s", recApt.Code, recApt.Body.String())
		}

		// Reactivate designation
		statusPayload.IsActive = true
		sBody, _ = json.Marshal(statusPayload)
		reqReactivate := httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/societies/%s/designations/%s/status", socAID, customDesigID), bytes.NewReader(sBody))
		reqReactivate.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqReactivate.Header.Set("Content-Type", "application/json")
		recReactivate := httptest.NewRecorder()
		router.ServeHTTP(recReactivate, reqReactivate)
		if recReactivate.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", recReactivate.Code)
		}
	})

	var chairmanAptID uuid.UUID
	t.Run("Create singleton appointment and enforce cardinality", func(t *testing.T) {
		// 1. Appoint Member 1 as CHAIRMAN
		aptPayload := CreateAppointmentRequest{
			DesignationID: chairmanDesigID.String(),
			MemberID:      member1.ID.String(),
			StartDate:     "2026-04-01",
		}
		body, _ := json.Marshal(aptPayload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for appointment, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				Appointment society.CommitteeAppointment `json:"appointment"`
			} `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		chairmanAptID = resp.Data.Appointment.ID

		// 2. Attempt to appoint Member 2 as CHAIRMAN (must fail with 409 DESIGNATION_ALREADY_APPOINTED)
		apt2Payload := CreateAppointmentRequest{
			DesignationID: chairmanDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-04-01",
		}
		body2, _ := json.Marshal(apt2Payload)
		req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(body2))
		req2.Header.Set("Authorization", "Bearer "+superAdminToken)
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for concurrent singleton appointment, got %d: %s", rec2.Code, rec2.Body.String())
		}

		// 3. Attempt to re-appoint Member 1 as CHAIRMAN (must fail with 409 MEMBER_ALREADY_APPOINTED or DESIGNATION_ALREADY_APPOINTED)
		req3 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(body))
		req3.Header.Set("Authorization", "Bearer "+superAdminToken)
		req3.Header.Set("Content-Type", "application/json")
		rec3 := httptest.NewRecorder()
		router.ServeHTTP(rec3, req3)
		if rec3.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for duplicate appointment of same member, got %d", rec3.Code)
		}
	})

	t.Run("Multi-member designation permits multiple active appointments", func(t *testing.T) {
		// Appoint Member 1 as Committee Member
		apt1 := CreateAppointmentRequest{
			DesignationID: commMemberDesigID.String(),
			MemberID:      member1.ID.String(),
			StartDate:     "2026-04-01",
		}
		b1, _ := json.Marshal(apt1)
		req1 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(b1))
		req1.Header.Set("Authorization", "Bearer "+superAdminToken)
		req1.Header.Set("Content-Type", "application/json")
		rec1 := httptest.NewRecorder()
		router.ServeHTTP(rec1, req1)
		if rec1.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for committee member 1, got %d: %s", rec1.Code, rec1.Body.String())
		}

		// Appoint Member 2 as Committee Member (must succeed because allows_multiple_members = true)
		apt2 := CreateAppointmentRequest{
			DesignationID: commMemberDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-04-01",
		}
		b2, _ := json.Marshal(apt2)
		req2 := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(b2))
		req2.Header.Set("Authorization", "Bearer "+superAdminToken)
		req2.Header.Set("Content-Type", "application/json")
		rec2 := httptest.NewRecorder()
		router.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for committee member 2, got %d: %s", rec2.Code, rec2.Body.String())
		}
	})

	t.Run("End and Revoke appointment preserves historical records", func(t *testing.T) {
		// Invalid end date (end date < start date)
		badEnd := EndAppointmentRequest{
			Action:  "ENDED",
			EndDate: "2025-01-01",
		}
		bBad, _ := json.Marshal(badEnd)
		reqBad := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments/%s/end", socAID, chairmanAptID), bytes.NewReader(bBad))
		reqBad.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqBad.Header.Set("Content-Type", "application/json")
		recBad := httptest.NewRecorder()
		router.ServeHTTP(recBad, reqBad)
		if recBad.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for end date before start date, got %d", recBad.Code)
		}

		// Valid end
		notes := "Completed tenure early"
		validEnd := EndAppointmentRequest{
			Action:  "ENDED",
			EndDate: "2026-09-30",
			Notes:   &notes,
		}
		bValid, _ := json.Marshal(validEnd)
		reqValid := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments/%s/end", socAID, chairmanAptID), bytes.NewReader(bValid))
		reqValid.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqValid.Header.Set("Content-Type", "application/json")
		recValid := httptest.NewRecorder()
		router.ServeHTTP(recValid, reqValid)
		if recValid.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for ending appointment, got %d: %s", recValid.Code, recValid.Body.String())
		}

		// Verify conflicting date range overlapping historical term [2026-04-01, 2026-09-30] is rejected
		badOverlap := CreateAppointmentRequest{
			DesignationID: chairmanDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-08-01",
		}
		bOverlap, _ := json.Marshal(badOverlap)
		reqOverlap := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bOverlap))
		reqOverlap.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqOverlap.Header.Set("Content-Type", "application/json")
		recOverlap := httptest.NewRecorder()
		router.ServeHTTP(recOverlap, reqOverlap)
		if recOverlap.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict for appointment overlapping historical term, got %d: %s", recOverlap.Code, recOverlap.Body.String())
		}

		// Verify Chairman is now vacant for a subsequent non-overlapping term [2026-10-01, ...)
		reAppoint := CreateAppointmentRequest{
			DesignationID: chairmanDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-10-01",
		}
		bRe, _ := json.Marshal(reAppoint)
		reqRe := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bRe))
		reqRe.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqRe.Header.Set("Content-Type", "application/json")
		recRe := httptest.NewRecorder()
		router.ServeHTTP(recRe, reqRe)
		if recRe.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for re-appointment to vacated singleton, got %d: %s", recRe.Code, recRe.Body.String())
		}

		// Check historical appointments list includes ENDED status
		reqHist := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/appointments?status=all", socAID), nil)
		reqHist.Header.Set("Authorization", "Bearer "+superAdminToken)
		recHist := httptest.NewRecorder()
		router.ServeHTTP(recHist, reqHist)
		var histResp struct {
			Data struct {
				Appointments []society.CommitteeAppointment `json:"appointments"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recHist.Body.Bytes(), &histResp)
		foundEndedRecord := false
		for _, a := range histResp.Data.Appointments {
			if a.ID == chairmanAptID && a.Status == "ENDED" {
				foundEndedRecord = true
			}
		}
		if !foundEndedRecord {
			t.Errorf("expected historical appointment record to be preserved with status = ENDED")
		}
	})

	t.Run("Member departure automatically concludes active committee appointment", func(t *testing.T) {
		// Appoint member 1 as SECRETARY
		secApt := CreateAppointmentRequest{
			DesignationID: secDesigID.String(),
			MemberID:      member1.ID.String(),
			StartDate:     "2026-04-01",
		}
		bSec, _ := json.Marshal(secApt)
		reqSec := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bSec))
		reqSec.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqSec.Header.Set("Content-Type", "application/json")
		recSec := httptest.NewRecorder()
		router.ServeHTTP(recSec, reqSec)
		if recSec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for secretary appointment, got %d", recSec.Code)
		}

		// Preview departure for member 1
		reqPrev := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/preview", socAID, member1.ID), nil)
		reqPrev.Header.Set("Authorization", "Bearer "+superAdminToken)
		recPrev := httptest.NewRecorder()
		router.ServeHTTP(recPrev, reqPrev)
		var prevResp struct {
			Data struct {
				VersionHash string `json:"version_hash"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recPrev.Body.Bytes(), &prevResp)

		// Confirm departure for member 1
		confirmPayload := ConfirmMemberDepartureRequest{
			ExpectedVersionHash: prevResp.Data.VersionHash,
			EndDate:             "2026-10-15",
			ReasonCode:          society.DepartureReasonMemberRelocated,
		}
		bConf, _ := json.Marshal(confirmPayload)
		reqConf := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member1.ID), bytes.NewReader(bConf))
		reqConf.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqConf.Header.Set("Content-Type", "application/json")
		recConf := httptest.NewRecorder()
		router.ServeHTTP(recConf, reqConf)
		if recConf.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for confirming departure, got %d: %s", recConf.Code, recConf.Body.String())
		}

		// Verify Secretary appointment is now ENDED
		var secStatus string
		err := db.Pool().QueryRow(ctx, `
			SELECT status FROM society_committee_appointments 
			WHERE member_id = $1 AND designation_id = $2
		`, member1.ID, secDesigID).Scan(&secStatus)
		if err != nil || secStatus != "ENDED" {
			t.Errorf("expected secretary appointment to be terminated upon member departure, got %s (err: %v)", secStatus, err)
		}

		// Verify attempting to appoint departed member rejects with 400 MEMBER_NOT_ACTIVE
		reqDepApt := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bSec))
		reqDepApt.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqDepApt.Header.Set("Content-Type", "application/json")
		recDepApt := httptest.NewRecorder()
		router.ServeHTTP(recDepApt, reqDepApt)
		if recDepApt.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for appointing departed member, got %d", recDepApt.Code)
		}
	})

	t.Run("Cross-society appointment rejected", func(t *testing.T) {
		// Attempt to appoint member from Society A into Society B
		crossPayload := CreateAppointmentRequest{
			DesignationID: chairmanDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-10-01",
		}
		bCross, _ := json.Marshal(crossPayload)
		reqCross := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socBID), bytes.NewReader(bCross))
		reqCross.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqCross.Header.Set("Content-Type", "application/json")
		recCross := httptest.NewRecorder()
		router.ServeHTTP(recCross, reqCross)
		if recCross.Code != http.StatusNotFound && recCross.Code != http.StatusBadRequest {
			t.Errorf("expected 404 or 400 for cross-society appointment, got %d", recCross.Code)
		}
	})

	t.Run("Suspended society rejects appointment mutations", func(t *testing.T) {
		// Suspend Society A
		reasonID := int16(1)
		_, _ = societyRepo.SuspendSociety(ctx, socAID, society.SuspendSocietyParams{
			ReasonID: &reasonID,
		}, superAdminRole.UserID, "127.0.0.1", "test")

		// Assign a SOC_ADMIN to Society A
		socAdminPhone := "919830088888"
		var socAdminUID uuid.UUID
		_ = db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, socAdminPhone).Scan(&socAdminUID)
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
			VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $3)
			ON CONFLICT (society_id, user_id, role) DO UPDATE SET status = 'ACTIVE'
		`, socAID, socAdminUID, superAdminRole.UserID)
		socAdminToken := generateToken(socAdminUID)

		// SOC_ADMIN attempting to create appointment on suspended society
		apt := CreateAppointmentRequest{
			DesignationID: commMemberDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-10-01",
		}
		b, _ := json.Marshal(apt)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+socAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for suspended society mutation, got %d: %s", rec.Code, rec.Body.String())
		}

		// Reactivate Society A
		_, _ = societyRepo.ReactivateSociety(ctx, socAID, "Reactivation note", superAdminRole.UserID, "127.0.0.1", "test")
	})

	t.Run("Get committee overview returns accurate active counts and office bearers", func(t *testing.T) {
		// 1. Authorized overview request with existing appointments -> 200
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp struct {
			Data society.CommitteeOverview `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		overview := resp.Data

		if overview.ActiveAppointments <= 0 {
			t.Errorf("expected > 0 active appointments in committee overview, got %d", overview.ActiveAppointments)
		}
		if !overview.KeyPositionsFilled["CHAIRMAN"] {
			t.Errorf("expected CHAIRMAN to be filled in overview")
		}
		if len(overview.OfficeBearers) == 0 {
			t.Errorf("expected office bearers to be populated, got %d", len(overview.OfficeBearers))
		}
		var foundChairman bool
		for _, ob := range overview.OfficeBearers {
			if ob.DesignationCode == "CHAIRMAN" {
				foundChairman = true
				if ob.MemberID == nil {
					t.Errorf("expected Chairman to be occupied")
				}
				if ob.MemberName == nil || *ob.MemberName == "" {
					t.Errorf("expected Chairman member name to be populated")
				}
			}
		}
		if !foundChairman {
			t.Errorf("expected CHAIRMAN in office bearers list")
		}

		// 2. Unauthenticated request -> 401
		unauthReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", socAID), nil)
		unauthRec := httptest.NewRecorder()
		router.ServeHTTP(unauthRec, unauthReq)
		if unauthRec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated overview request, got %d", unauthRec.Code)
		}

		// 3. Unauthorized request -> 403
		unassignedUserID := uuid.New()
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')
		`, unassignedUserID, fmt.Sprintf("9198%08d", time.Now().UnixNano()%100000000))
		_, _ = db.Pool().Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status, granted_by) VALUES ($1, 'ADMIN', 'ACTIVE', $2)
		`, unassignedUserID, superAdminRole.UserID)
		unassignedToken := generateToken(unassignedUserID)
		unauthSocReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", socAID), nil)
		unauthSocReq.Header.Set("Authorization", "Bearer "+unassignedToken)
		unauthSocRec := httptest.NewRecorder()
		router.ServeHTTP(unauthSocRec, unauthSocReq)
		if unauthSocRec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for unassigned admin overview request, got %d", unauthSocRec.Code)
		}

		// 4. Authorized overview request for society with zero appointments -> 200 with empty collections (socBID)
		socBReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", socBID), nil)
		socBReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		socBRec := httptest.NewRecorder()
		router.ServeHTTP(socBRec, socBReq)
		if socBRec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for society with zero appointments, got %d: %s", socBRec.Code, socBRec.Body.String())
		} else {
			var zeroResp struct {
				Data society.CommitteeOverview `json:"data"`
			}
			_ = json.Unmarshal(socBRec.Body.Bytes(), &zeroResp)
			if zeroResp.Data.ActiveAppointments != 0 {
				t.Errorf("expected 0 active appointments, got %d", zeroResp.Data.ActiveAppointments)
			}
			if zeroResp.Data.CurrentAppointments == nil {
				t.Errorf("expected non-nil CurrentAppointments slice")
			}
			if len(zeroResp.Data.OfficeBearers) != 3 {
				t.Errorf("expected 3 default office bearers (all vacant), got %d", len(zeroResp.Data.OfficeBearers))
			}
			for _, ob := range zeroResp.Data.OfficeBearers {
				if ob.MemberID != nil {
					t.Errorf("expected vacant office bearer, got member %v", ob.MemberID)
				}
			}
		}

		// 5. Nonexistent endpoint -> 404
		badReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/committee/nonexistent", socAID), nil)
		badReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		badRec := httptest.NewRecorder()
		router.ServeHTTP(badRec, badReq)
		if badRec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for nonexistent route, got %d", badRec.Code)
		}
	})

	t.Run("Multiple designations per member rule enforcement", func(t *testing.T) {
		// 1. Create a custom designation that disallows holding other positions
		exclPayload := CreateDesignationRequest{
			Code:                             "PRESIDENT",
			Name:                             "Society President",
			AllowsMultipleMembers:            false,
			AllowsMultiplePositionsPerMember: false,
		}
		bExcl, _ := json.Marshal(exclPayload)
		reqExcl := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), bytes.NewReader(bExcl))
		reqExcl.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqExcl.Header.Set("Content-Type", "application/json")
		recExcl := httptest.NewRecorder()
		router.ServeHTTP(recExcl, reqExcl)
		if recExcl.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for PRESIDENT designation, got %d: %s", recExcl.Code, recExcl.Body.String())
		}
		var exclResp struct {
			Data struct {
				Designation society.SocietyDesignation `json:"designation"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recExcl.Body.Bytes(), &exclResp)
		presDesigID := exclResp.Data.Designation.ID

		// 2. Member 2 already has active appointment as Chairman.
		// Attempting to appoint Member 2 as PRESIDENT (which disallows multiple positions) must be rejected.
		badAppt := CreateAppointmentRequest{
			DesignationID: presDesigID.String(),
			MemberID:      member2.ID.String(),
			StartDate:     "2026-10-15",
		}
		bBad, _ := json.Marshal(badAppt)
		reqBad := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bBad))
		reqBad.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqBad.Header.Set("Content-Type", "application/json")
		recBad := httptest.NewRecorder()
		router.ServeHTTP(recBad, reqBad)
		if recBad.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when member with active appointment attempts exclusive designation, got %d: %s", recBad.Code, recBad.Body.String())
		}

		// 3. Register a clean member 3
		mem3Phone := "919876543299"
		m3, err := societyRepo.RegisterMember(ctx, socAID, superAdminRole.UserID, society.RegisterMemberParams{
			FirstName: "Clean",
			Phone:     mem3Phone,
			ClientIP:  "127.0.0.1",
			UserAgent: "test",
		})
		if err != nil {
			t.Fatalf("failed to register member 3: %v", err)
		}

		// 4. Appoint clean Member 3 to PRESIDENT (must succeed)
		goodAppt := CreateAppointmentRequest{
			DesignationID: presDesigID.String(),
			MemberID:      m3.ID.String(),
			StartDate:     "2026-10-15",
		}
		bGood, _ := json.Marshal(goodAppt)
		reqGood := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bGood))
		reqGood.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqGood.Header.Set("Content-Type", "application/json")
		recGood := httptest.NewRecorder()
		router.ServeHTTP(recGood, reqGood)
		if recGood.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for clean member appointed to PRESIDENT, got %d: %s", recGood.Code, recGood.Body.String())
		}

		// 5. Member 3 (now holding PRESIDENT with allows_multiple_positions_per_member = false)
		// attempts to also be appointed as Committee Member (must be rejected)
		badSecondAppt := CreateAppointmentRequest{
			DesignationID: commMemberDesigID.String(),
			MemberID:      m3.ID.String(),
			StartDate:     "2026-10-16",
		}
		bBadSec, _ := json.Marshal(badSecondAppt)
		reqBadSec := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(bBadSec))
		reqBadSec.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqBadSec.Header.Set("Content-Type", "application/json")
		recBadSec := httptest.NewRecorder()
		router.ServeHTTP(recBadSec, reqBadSec)
		if recBadSec.Code != http.StatusConflict {
			t.Errorf("expected 409 Conflict when member holding exclusive designation attempts second role, got %d: %s", recBadSec.Code, recBadSec.Body.String())
		}
	})

	t.Run("Genuine concurrent appointment conflict on singleton designation", func(t *testing.T) {
		// 1. Create a fresh singleton designation
		desigReq := CreateDesignationRequest{
			Code:                             "TREASURER_EXECUTIVE",
			Name:                             "Executive Treasurer",
			AllowsMultipleMembers:            false,
			AllowsMultiplePositionsPerMember: true,
		}
		bDesig, _ := json.Marshal(desigReq)
		reqDesig := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/designations", socAID), bytes.NewReader(bDesig))
		reqDesig.Header.Set("Authorization", "Bearer "+superAdminToken)
		reqDesig.Header.Set("Content-Type", "application/json")
		recDesig := httptest.NewRecorder()
		router.ServeHTTP(recDesig, reqDesig)
		if recDesig.Code != http.StatusCreated {
			t.Fatalf("failed to create concurrent test designation: %d %s", recDesig.Code, recDesig.Body.String())
		}
		var desResp struct {
			Data struct {
				Designation society.SocietyDesignation `json:"designation"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recDesig.Body.Bytes(), &desResp)
		targetDesigID := desResp.Data.Designation.ID

		// 2. Register Member 4 and Member 5
		m4, err := societyRepo.RegisterMember(ctx, socAID, superAdminRole.UserID, society.RegisterMemberParams{
			FirstName: "Candidate",
			Phone:     "919876543284",
			ClientIP:  "127.0.0.1",
			UserAgent: "test",
		})
		if err != nil {
			t.Fatalf("failed to register member 4: %v", err)
		}
		m5, err := societyRepo.RegisterMember(ctx, socAID, superAdminRole.UserID, society.RegisterMemberParams{
			FirstName: "Candidate",
			Phone:     "919876543285",
			ClientIP:  "127.0.0.1",
			UserAgent: "test",
		})
		if err != nil {
			t.Fatalf("failed to register member 5: %v", err)
		}

		// 3. Launch two concurrent goroutines attempting to appoint Member 4 and Member 5
		// to the exact same singleton designation at the same time
		var wg sync.WaitGroup
		results := make(chan int, 2)

		sendAppt := func(targetMemberID uuid.UUID) {
			defer wg.Done()
			payload := CreateAppointmentRequest{
				DesignationID: targetDesigID.String(),
				MemberID:      targetMemberID.String(),
				StartDate:     "2026-11-01",
			}
			b, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/appointments", socAID), bytes.NewReader(b))
			req.Header.Set("Authorization", "Bearer "+superAdminToken)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			results <- rec.Code
		}

		wg.Add(2)
		go sendAppt(m4.ID)
		go sendAppt(m5.ID)
		wg.Wait()
		close(results)

		statusCodes := make([]int, 0, 2)
		for code := range results {
			statusCodes = append(statusCodes, code)
		}

		count201 := 0
		count409 := 0
		for _, code := range statusCodes {
			if code == http.StatusCreated {
				count201++
			} else if code == http.StatusConflict {
				count409++
			}
		}

		if count201 != 1 || count409 != 1 {
			t.Errorf("expected exactly one 201 Created and one 409 Conflict, got status codes %v", statusCodes)
		}

		// 4. Verify in DB that exactly 1 active appointment exists for targetDesigID
		var activeCount int
		err = db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM society_committee_appointments
			WHERE designation_id = $1 AND society_id = $2 AND status = 'ACTIVE'
		`, targetDesigID, socAID).Scan(&activeCount)
		if err != nil || activeCount != 1 {
			t.Errorf("expected exactly 1 active appointment in database, got %d (err: %v)", activeCount, err)
		}
	})

	t.Run("Transactional audit logs written for S07 operations", func(t *testing.T) {
		var auditCount int
		err := db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM platform_audit_logs 
			WHERE action IN ($1, $2, $3, $4, $5, $6)
		`, society.ActionDesignationCreated, society.ActionDesignationUpdated, society.ActionDesignationStatusChanged,
			society.ActionCommitteeAppointmentCreated, society.ActionCommitteeAppointmentEnded, society.ActionCommitteeAppointmentRevoked).Scan(&auditCount)
		if err != nil || auditCount < 4 {
			t.Errorf("expected at least 4 audit log entries for S07 actions, got %d (err: %v)", auditCount, err)
		}
	})
}
