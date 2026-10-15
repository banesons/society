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
	"strings"
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

func TestMembersAndRelationshipsAPI(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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
		Name:         "S06 Test Society " + uniqueSuffixA,
		AddressLine1: "100 Residency Way",
		Locality:     "Bandra West",
		City:         "Mumbai",
		State:        "Maharashtra",
		Pincode:      "400050",
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

	// Activate society A
	actReq := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies/"+socAID.String()+"/activate", nil)
	actReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	actRec := httptest.NewRecorder()
	router.ServeHTTP(actRec, actReq)
	if actRec.Code != http.StatusOK {
		t.Fatalf("failed to activate society A: %d %s", actRec.Code, actRec.Body.String())
	}

	// Create building and units in Society A
	bReq := society.CreateBuildingParams{
		Name:         "Tower One",
		BuildingCode: "T1",
		TotalFloors:  5,
	}
	bJSON, _ := json.Marshal(bReq)
	bRec := httptest.NewRecorder()
	bHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+socAID.String()+"/buildings", bytes.NewReader(bJSON))
	bHttpReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	bHttpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(bRec, bHttpReq)
	if bRec.Code != http.StatusCreated {
		t.Fatalf("failed to create building: %d %s", bRec.Code, bRec.Body.String())
	}
	var bResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bRec.Body.Bytes(), &bResp)
	bID := uuid.MustParse(bResp.Data.ID)

	genReq := society.GenerateUnitsParams{
		BuildingID:         bID,
		StartingFloor:      1,
		TotalFloors:        2,
		UnitsPerFloor:      2,
		StartingUnitNumber: 101,
		Prefix:             "T1",
	}
	genJSON, _ := json.Marshal(genReq)
	genRec := httptest.NewRecorder()
	genHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/societies/"+socAID.String()+"/units/generate", bytes.NewReader(genJSON))
	genHttpReq.Header.Set("Authorization", "Bearer "+superAdminToken)
	genHttpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(genRec, genHttpReq)
	if genRec.Code != http.StatusCreated {
		t.Fatalf("failed to generate units: %d %s", genRec.Code, genRec.Body.String())
	}

	// List generated units to get IDs
	uRec := httptest.NewRecorder()
	uHttpReq := httptest.NewRequest(http.MethodGet, "/api/v1/societies/"+socAID.String()+"/units", nil)
	uHttpReq.Header.Set("Authorization", "Bearer "+superAdminToken)
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
	if len(uResp.Data.Units) < 2 {
		t.Fatalf("expected at least 2 units, got %d", len(uResp.Data.Units))
	}
	unit101ID := uuid.MustParse(uResp.Data.Units[0].ID)
	unit102ID := uuid.MustParse(uResp.Data.Units[1].ID)

	// Create test society B for cross-society tests
	createSocB := CreateSocietyRequest{
		Name:         "S06 Other Society " + uniqueSuffixA,
		AddressLine1: "200 Other Way",
		Locality:     "Andheri East",
		City:         "Mumbai",
		State:        "Maharashtra",
		Pincode:      "400069",
	}
	bJSON2, _ := json.Marshal(createSocB)
	reqB := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(bJSON2))
	reqB.Header.Set("Authorization", "Bearer "+superAdminToken)
	reqB.Header.Set("Content-Type", "application/json")
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusCreated {
		t.Fatalf("failed to create test society B: %d %s", recB.Code, recB.Body.String())
	}
	var socBResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recB.Body.Bytes(), &socBResp)
	socBID := uuid.MustParse(socBResp.Data.ID)
	_, _ = societyRepo.ActivateSociety(ctx, socBID, "Initial launch", superAdminRole.UserID, "127.0.0.1", "test")

	t.Run("Unauthenticated member request returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/members", socAID), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Unauthorized user returns 403", func(t *testing.T) {
		unauthorizedUser := uuid.New()
		unauthPhone := fmt.Sprintf("91991%07d", time.Now().UnixNano()%10000000)
		_, err := db.Pool().Exec(ctx, "INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')", unauthorizedUser, unauthPhone)
		if err != nil {
			t.Fatalf("failed to insert unauth user: %v", err)
		}
		tok := generateToken(unauthorizedUser)

		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/members", socAID), nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	var member1ID, member2ID uuid.UUID
	member1Phone := fmt.Sprintf("917%09d", time.Now().UnixNano()%1000000000)
	member2Phone := fmt.Sprintf("916%09d", time.Now().UnixNano()%1000000000)

	t.Run("Register member pre-registration without fake auth user", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Vikram",
			"last_name":  "Malhotra",
			"phone":      member1Phone,
			"email":      "vikram@example.com",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members", socAID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		m := data["member"].(map[string]interface{})
		member1ID = uuid.MustParse(m["id"].(string))

		// Ensure user_id is null (no fake auth user fabricated)
		if m["user_id"] != nil {
			t.Errorf("expected user_id to be null for pre-registered member, got %v", m["user_id"])
		}
	})

	t.Run("Duplicate active member registration fails with 409", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Vikram Second",
			"phone":      member1Phone,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members", socAID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Register second member", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Ananya",
			"last_name":  "Deshmukh",
			"phone":      member2Phone,
			"email":      "ananya@example.com",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members", socAID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		m := data["member"].(map[string]interface{})
		member2ID = uuid.MustParse(m["id"].(string))
	})

	t.Run("Safe identity linking connects verified user post-login", func(t *testing.T) {
		verifiedUser := uuid.New()
		_, err := db.Pool().Exec(ctx, "INSERT INTO users (id, phone, status) VALUES ($1, $2, 'ACTIVE')", verifiedUser, member1Phone)
		if err != nil {
			t.Fatalf("failed to insert user: %v", err)
		}

		err = societyRepo.SafeLinkUserMembers(ctx, verifiedUser, member1Phone)
		if err != nil {
			t.Fatalf("safe link failed: %v", err)
		}

		m, _, err := societyRepo.GetMemberByID(ctx, socAID, member1ID)
		if err != nil || m.UserID == nil || *m.UserID != verifiedUser {
			t.Errorf("expected member1 to be linked to verified user %v, got %v", verifiedUser, m.UserID)
		}
	})

	var ownerRelID uuid.UUID

	t.Run("Create unit relationship with absentee owner", func(t *testing.T) {
		share := 50.0
		payload := map[string]interface{}{
			"member_id":         member1ID.String(),
			"relationship_type": society.RelTypePrimaryOwner,
			"is_occupant":       false,
			"ownership_share":   share,
			"start_date":        "2026-01-01",
			"notes":             "Primary owner 50%",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		rel := data["relationship"].(map[string]interface{})
		ownerRelID = uuid.MustParse(rel["id"].(string))
	})

	t.Run("Duplicate identical active relationship fails with 409", func(t *testing.T) {
		share := 10.0
		payload := map[string]interface{}{
			"member_id":         member1ID.String(),
			"relationship_type": society.RelTypePrimaryOwner,
			"is_occupant":       false,
			"ownership_share":   share,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Ownership share exceeding 100 percent rejected with 400", func(t *testing.T) {
		// Existing is 50%, adding 60% exceeds 100%
		share := 60.0
		payload := map[string]interface{}{
			"member_id":         member2ID.String(),
			"relationship_type": society.RelTypeSecondaryOwner,
			"is_occupant":       false,
			"ownership_share":   share,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Ownership share on non-owner rejected with 400", func(t *testing.T) {
		share := 10.0
		payload := map[string]interface{}{
			"member_id":         member2ID.String(),
			"relationship_type": society.RelTypePrimaryTenant,
			"is_occupant":       true,
			"ownership_share":   share,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Tenant and owner legitimately coexist on same unit", func(t *testing.T) {
		payload := map[string]interface{}{
			"member_id":         member2ID.String(),
			"relationship_type": society.RelTypePrimaryTenant,
			"is_occupant":       true,
			"start_date":        "2026-03-01",
			"notes":             "Active lease",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		// List relationships on unit
		reqList := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socAID, unit101ID), nil)
		reqList.Header.Set("Authorization", "Bearer "+superAdminToken)
		recList := httptest.NewRecorder()
		router.ServeHTTP(recList, reqList)

		if recList.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", recList.Code, recList.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(recList.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		rels := data["relationships"].([]interface{})
		isVacant := data["is_vacant"].(bool)

		if len(rels) != 2 {
			t.Errorf("expected 2 coexisting relationships, got %d", len(rels))
		}
		if isVacant {
			t.Errorf("expected unit to not be vacant because tenant is occupant")
		}
	})

	var previewHash string

	t.Run("Preview member departure manifest and version hash", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/preview", socAID, member2ID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		data := resp["data"].(map[string]interface{})
		previewHash = data["version_hash"].(string)

		terminations := data["targeted_terminations"].([]interface{})
		if len(terminations) != 1 {
			t.Errorf("expected 1 targeted termination for tenant, got %d", len(terminations))
		}
	})

	t.Run("Departure confirmation validates reason_code and reason_details", func(t *testing.T) {
		// Missing reason_code
		body, _ := json.Marshal(map[string]interface{}{
			"expected_version_hash": previewHash,
			"end_date":              "2026-10-15",
		})
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing reason_code, got %d", rec.Code)
		}

		// Invalid reason_code
		body, _ = json.Marshal(map[string]interface{}{
			"expected_version_hash": previewHash,
			"end_date":              "2026-10-15",
			"reason_code":           "INVALID_CODE",
		})
		req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid reason_code, got %d", rec.Code)
		}

		// OTHER reason without reason_details
		body, _ = json.Marshal(map[string]interface{}{
			"expected_version_hash": previewHash,
			"end_date":              "2026-10-15",
			"reason_code":           "OTHER",
		})
		req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for OTHER without details, got %d", rec.Code)
		}

		// OTHER reason with details > 500 characters
		longDetails := strings.Repeat("A", 501)
		body, _ = json.Marshal(map[string]interface{}{
			"expected_version_hash": previewHash,
			"end_date":              "2026-10-15",
			"reason_code":           "OTHER",
			"reason_details":        longDetails,
		})
		req = httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for details > 500 chars, got %d", rec.Code)
		}
	})

	t.Run("Stale preview confirmation fails with 409", func(t *testing.T) {
		payload := map[string]interface{}{
			"expected_version_hash": "stale_hash_12345",
			"end_date":              "2026-10-15",
			"reason_code":           "TENANT_MOVED_OUT",
			"reason_details":        "Tenant moved out",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for stale preview, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("Confirm member departure terminates tenant and preserves history", func(t *testing.T) {
		detailsText := "Tenant relocated to another city"
		payload := map[string]interface{}{
			"expected_version_hash": previewHash,
			"end_date":              "2026-10-15",
			"reason_code":           "MEMBER_RELOCATED",
			"reason_details":        detailsText,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/members/%s/departure/confirm", socAID, member2ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		// Verify member status is LEFT and structured departure fields are persisted
		m, rels, err := societyRepo.GetMemberByID(ctx, socAID, member2ID)
		if err != nil || m.Status != society.MemberStatusLeft {
			t.Errorf("expected member status LEFT, got %v", m.Status)
		}
		if m.DepartureReasonCode == nil || *m.DepartureReasonCode != "MEMBER_RELOCATED" {
			t.Errorf("expected departure_reason_code 'MEMBER_RELOCATED', got %v", m.DepartureReasonCode)
		}
		if m.DepartureReasonDetails == nil || *m.DepartureReasonDetails != detailsText {
			t.Errorf("expected departure_reason_details '%s', got %v", detailsText, m.DepartureReasonDetails)
		}
		if m.DepartedAt == nil {
			t.Errorf("expected departed_at to be set")
		}

		// Verify tenancy relationship is ENDED with end_date preserved
		foundEnded := false
		for _, r := range rels {
			if r.RelationshipType == society.RelTypePrimaryTenant && r.Status == society.RelStatusEnded {
				foundEnded = true
				if r.EndDate == nil || *r.EndDate != "2026-10-15" {
					t.Errorf("expected end_date 2026-10-15, got %v", r.EndDate)
				}
			}
		}
		if !foundEnded {
			t.Errorf("expected ended tenancy record preserved in history")
		}

		// Verify owner on unit 101 is STILL ACTIVE and UNTOUCHED
		_, ownerRels, err := societyRepo.GetMemberByID(ctx, socAID, member1ID)
		if err != nil {
			t.Fatalf("failed to get owner: %v", err)
		}
		ownerActive := false
		for _, r := range ownerRels {
			if r.ID == ownerRelID && r.Status == society.RelStatusActive {
				ownerActive = true
			}
		}
		if !ownerActive {
			t.Errorf("expected owner relationship to remain ACTIVE after tenant departure")
		}
	})

	t.Run("Cross-society unit and member access rejected", func(t *testing.T) {
		// Attempt to create relationship in Society B pointing to member in Society A
		payload := map[string]interface{}{
			"member_id":         member1ID.String(),
			"relationship_type": society.RelTypePrimaryOwner,
			"is_occupant":       false,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/societies/%s/units/%s/relationships", socBID, unit102ID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound && rec.Code != http.StatusBadRequest {
			t.Errorf("expected 404 or 400 for cross-society reference, got %d", rec.Code)
		}
	})

	t.Run("Verify transactional audit logs for S06 operations", func(t *testing.T) {
		var auditCount int
		err := db.Pool().QueryRow(ctx, `
			SELECT COUNT(*) FROM platform_audit_logs 
			WHERE action IN ($1, $2, $3, $4)
		`, society.ActionMemberRegistered, society.ActionUnitRelCreated, society.ActionUnitRelEnded, society.ActionMemberDeparted).Scan(&auditCount)
		if err != nil || auditCount < 3 {
			t.Errorf("expected at least 3 audit log entries for S06 actions, got %d (err: %v)", auditCount, err)
		}

		// Verify departure audit log payload contains reason_code
		var departureDetails []byte
		err = db.Pool().QueryRow(ctx, `
			SELECT details FROM platform_audit_logs
			WHERE action = $1 AND details->>'reason_code' = 'MEMBER_RELOCATED'
			ORDER BY created_at DESC LIMIT 1
		`, society.ActionMemberDeparted).Scan(&departureDetails)
		if err != nil {
			t.Errorf("expected audit log for MEMBER_DEPARTED with reason_code: %v", err)
		}
	})
}
