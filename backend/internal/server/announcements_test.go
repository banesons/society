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

func TestAnnouncementManagementLifecycle(t *testing.T) {
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

	// Create test society
	socName := fmt.Sprintf("Communication Test Society %d", time.Now().UnixNano())
	regNo := fmt.Sprintf("REG-COMM-%d", time.Now().UnixNano()%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "100 Notice Board Avenue",
		Locality:           "Bandra",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400050",
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

	// Create test society admin user
	adminPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/1000)%1000000000)
	adminNorm, _ := auth.NormalizePhone(adminPhone)
	var adminUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", adminNorm).Scan(&adminUserID)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	_, err = societyRepo.AssignSocietyAdmin(ctx, society.AssignSocietyAdminParams{
		SocietyID: socID,
		UserID:    &adminUserID,
		Role:      society.RoleSocietyAdmin,
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to assign society admin: %v", err)
	}
	adminToken := generateToken(adminUserID)

	// Create test resident user and member record
	residentPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/1000+1)%1000000000)
	residentNorm, _ := auth.NormalizePhone(residentPhone)
	var residentUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", residentNorm).Scan(&residentUserID)
	if err != nil {
		t.Fatalf("failed to create resident user: %v", err)
	}
	lastName := "Deshmukh"
	_, err = societyRepo.RegisterMember(ctx, socID, adminUserID, society.RegisterMemberParams{
		FirstName: "Rohan",
		LastName:  &lastName,
		Phone:     residentPhone,
		ClientIP:  "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to register resident member: %v", err)
	}
	residentToken := generateToken(residentUserID)

	// 1. Admin creates DRAFT announcement
	draftReqBody := map[string]interface{}{
		"title":         "Elevator Maintenance Notice",
		"body":          "Passenger lift in Wing A will be serviced tomorrow 10am to 1pm.",
		"category":      "MAINTENANCE",
		"priority":      "URGENT",
		"audience_type": "SOCIETY",
	}
	draftJSON, _ := json.Marshal(draftReqBody)
	createReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements", socID), bytes.NewReader(draftJSON))
	createReq.Header.Set("Authorization", "Bearer "+adminToken)
	createReq.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, createReq)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 created draft, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	createData := parseData(wCreate.Body.Bytes())
	announcementMap := createData["announcement"].(map[string]interface{})
	announcementID := announcementMap["id"].(string)

	if announcementMap["status"] != "DRAFT" {
		t.Errorf("expected status DRAFT, got %v", announcementMap["status"])
	}
	if announcementMap["published_at"] != nil {
		t.Errorf("expected published_at to be nil for draft, got %v", announcementMap["published_at"])
	}

	// 2. Resident cannot view DRAFT announcement via detail endpoint
	resGetReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/%s", socID, announcementID), nil)
	resGetReq.Header.Set("Authorization", "Bearer "+residentToken)
	wResGet := httptest.NewRecorder()
	router.ServeHTTP(wResGet, resGetReq)
	if wResGet.Code != http.StatusNotFound {
		t.Errorf("expected 404 for resident viewing draft, got %d: %s", wResGet.Code, wResGet.Body.String())
	}

	// 3. Resident feed does not include DRAFT announcement
	feedReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/feed", socID), nil)
	feedReq.Header.Set("Authorization", "Bearer "+residentToken)
	wFeed := httptest.NewRecorder()
	router.ServeHTTP(wFeed, feedReq)
	if wFeed.Code != http.StatusOK {
		t.Fatalf("expected 200 feed, got %d: %s", wFeed.Code, wFeed.Body.String())
	}
	feedData := parseData(wFeed.Body.Bytes())
	if total, ok := feedData["total"].(float64); !ok || total != 0 {
		t.Errorf("expected 0 items in resident feed for draft, got %v", feedData["total"])
	}

	// 4. Admin edits draft announcement
	updateReqBody := map[string]interface{}{
		"title":    "Elevator Maintenance Notice (Updated)",
		"priority": "CRITICAL",
	}
	updateJSON, _ := json.Marshal(updateReqBody)
	updReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/societies/%s/announcements/%s", socID, announcementID), bytes.NewReader(updateJSON))
	updReq.Header.Set("Authorization", "Bearer "+adminToken)
	updReq.Header.Set("Content-Type", "application/json")
	wUpd := httptest.NewRecorder()
	router.ServeHTTP(wUpd, updReq)
	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 update draft, got %d: %s", wUpd.Code, wUpd.Body.String())
	}
	updData := parseData(wUpd.Body.Bytes())
	updAnn := updData["announcement"].(map[string]interface{})
	if updAnn["title"] != "Elevator Maintenance Notice (Updated)" {
		t.Errorf("expected updated title, got %v", updAnn["title"])
	}

	// 5. Admin publishes announcement
	pubReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/publish", socID, announcementID), nil)
	pubReq.Header.Set("Authorization", "Bearer "+adminToken)
	wPub := httptest.NewRecorder()
	router.ServeHTTP(wPub, pubReq)
	if wPub.Code != http.StatusOK {
		t.Fatalf("expected 200 published, got %d: %s", wPub.Code, wPub.Body.String())
	}
	pubData := parseData(wPub.Body.Bytes())
	pubAnn := pubData["announcement"].(map[string]interface{})
	if pubAnn["status"] != "PUBLISHED" {
		t.Errorf("expected status PUBLISHED, got %v", pubAnn["status"])
	}
	if pubAnn["published_at"] == nil {
		t.Errorf("expected non-nil published_at")
	}

	// 6. Resident feed now contains the published announcement
	wFeed2 := httptest.NewRecorder()
	router.ServeHTTP(wFeed2, feedReq)
	if wFeed2.Code != http.StatusOK {
		t.Fatalf("expected 200 feed, got %d: %s", wFeed2.Code, wFeed2.Body.String())
	}
	feedData2 := parseData(wFeed2.Body.Bytes())
	if total, ok := feedData2["total"].(float64); !ok || total != 1 {
		t.Errorf("expected 1 item in resident feed, got %v", feedData2["total"])
	}
	feedItems := feedData2["announcements"].([]interface{})
	firstItem := feedItems[0].(map[string]interface{})
	if isRead, ok := firstItem["is_read"].(bool); !ok || isRead {
		t.Errorf("expected is_read to be false initially, got %v", firstItem["is_read"])
	}

	// 7. Check unread count -> should be 1
	unreadReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/unread-count", socID), nil)
	unreadReq.Header.Set("Authorization", "Bearer "+residentToken)
	wUnread := httptest.NewRecorder()
	router.ServeHTTP(wUnread, unreadReq)
	if wUnread.Code != http.StatusOK {
		t.Fatalf("expected 200 unread count, got %d: %s", wUnread.Code, wUnread.Body.String())
	}
	unreadData := parseData(wUnread.Body.Bytes())
	if count, ok := unreadData["unread_count"].(float64); !ok || count != 1 {
		t.Errorf("expected unread_count = 1, got %v", unreadData["unread_count"])
	}

	// 8. Resident marks announcement as read
	markReadReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/read", socID, announcementID), nil)
	markReadReq.Header.Set("Authorization", "Bearer "+residentToken)
	wMark := httptest.NewRecorder()
	router.ServeHTTP(wMark, markReadReq)
	if wMark.Code != http.StatusOK {
		t.Fatalf("expected 200 mark read, got %d: %s", wMark.Code, wMark.Body.String())
	}

	// 9. Repeat mark-as-read (Idempotency test)
	wMark2 := httptest.NewRecorder()
	markReadReq2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/read", socID, announcementID), nil)
	markReadReq2.Header.Set("Authorization", "Bearer "+residentToken)
	router.ServeHTTP(wMark2, markReadReq2)
	if wMark2.Code != http.StatusOK {
		t.Errorf("expected 200 idempotent mark read, got %d: %s", wMark2.Code, wMark2.Body.String())
	}

	// 10. Check unread count -> now 0
	wUnread2 := httptest.NewRecorder()
	router.ServeHTTP(wUnread2, unreadReq)
	unreadData2 := parseData(wUnread2.Body.Bytes())
	if count, ok := unreadData2["unread_count"].(float64); !ok || count != 0 {
		t.Errorf("expected unread_count = 0 after reading, got %v", unreadData2["unread_count"])
	}

	// 11. Admin archives announcement
	archReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/archive", socID, announcementID), nil)
	archReq.Header.Set("Authorization", "Bearer "+adminToken)
	wArch := httptest.NewRecorder()
	router.ServeHTTP(wArch, archReq)
	if wArch.Code != http.StatusOK {
		t.Fatalf("expected 200 archive, got %d: %s", wArch.Code, wArch.Body.String())
	}

	// 12. Resident feed excludes archived announcements
	wFeed3 := httptest.NewRecorder()
	router.ServeHTTP(wFeed3, feedReq)
	feedData3 := parseData(wFeed3.Body.Bytes())
	if total, ok := feedData3["total"].(float64); !ok || total != 0 {
		t.Errorf("expected 0 items in resident feed after archival, got %v", feedData3["total"])
	}

	// 13. Admin can list archived announcements via status filter
	adminListReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/admin?status=ARCHIVED", socID), nil)
	adminListReq.Header.Set("Authorization", "Bearer "+adminToken)
	wAdminList := httptest.NewRecorder()
	router.ServeHTTP(wAdminList, adminListReq)
	if wAdminList.Code != http.StatusOK {
		t.Fatalf("expected 200 admin list, got %d: %s", wAdminList.Code, wAdminList.Body.String())
	}
	adminListData := parseData(wAdminList.Body.Bytes())
	if total, ok := adminListData["total"].(float64); !ok || total != 1 {
		t.Errorf("expected 1 archived item in admin list, got %v", adminListData["total"])
	}

	// 14. Verify audit log entries were recorded
	var auditCount int
	err = db.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM platform_audit_logs 
		WHERE details->>'announcement_id' = $1
	`, announcementID).Scan(&auditCount)
	if err != nil {
		t.Fatalf("failed to query audit logs: %v", err)
	}
	if auditCount < 4 {
		t.Errorf("expected at least 4 audit logs (create, update, publish, archive), got %d", auditCount)
	}
}

func TestAnnouncementAudienceTargetingAndSecurity(t *testing.T) {
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

	// Create test society
	socName := fmt.Sprintf("Targeting Test Society %d", time.Now().UnixNano())
	regNo := fmt.Sprintf("REG-TARG-%d", time.Now().UnixNano()%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "200 Hierarchy Blvd",
		Locality:           "Kothrud",
		City:               "Pune",
		State:              "Maharashtra",
		Pincode:            "411038",
	}
	socJSON, _ := json.Marshal(socParams)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	socData := parseData(w.Body.Bytes())
	socID := uuid.MustParse(socData["id"].(string))

	// Activate
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	// Create buildings
	bldA, err := societyRepo.CreateBuilding(ctx, socID, society.CreateBuildingParams{Name: "Tower A", BuildingCode: "TOW-A", TotalFloors: 1}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create building A: %v", err)
	}
	bldB, err := societyRepo.CreateBuilding(ctx, socID, society.CreateBuildingParams{Name: "Tower B", BuildingCode: "TOW-B", TotalFloors: 1}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create building B: %v", err)
	}

	// Create units in Building A and Building B
	_, err = societyRepo.BulkGenerateUnits(ctx, socID, society.GenerateUnitsParams{
		BuildingID:    bldA.ID,
		TotalFloors:   1,
		UnitsPerFloor: 1,
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to generate unit in A: %v", err)
	}
	_, err = societyRepo.BulkGenerateUnits(ctx, socID, society.GenerateUnitsParams{
		BuildingID:    bldB.ID,
		TotalFloors:   1,
		UnitsPerFloor: 1,
		Prefix:        "B-",
	}, superAdminRole.UserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to generate unit in B: %v", err)
	}

	var unitA, unitB uuid.UUID
	_ = db.Pool().QueryRow(ctx, "SELECT id FROM society_units WHERE building_id = $1 LIMIT 1", bldA.ID).Scan(&unitA)
	_ = db.Pool().QueryRow(ctx, "SELECT id FROM society_units WHERE building_id = $1 LIMIT 1", bldB.ID).Scan(&unitB)

	// Create resident in Tower A
	resAPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/1000+2)%1000000000)
	resANorm, _ := auth.NormalizePhone(resAPhone)
	var resAUserID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", resANorm).Scan(&resAUserID)
	memA, err := societyRepo.RegisterMember(ctx, socID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "A Resident",
		Phone:     resAPhone,
		ClientIP:  "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to register member A: %v", err)
	}
	_, err = societyRepo.CreateUnitRelationship(ctx, socID, unitA, superAdminRole.UserID, society.CreateUnitRelationshipParams{
		MemberID:         memA.ID,
		RelationshipType: "PRIMARY_OWNER",
		IsOccupant:       true,
		StartDate:        "2026-01-01",
		ClientIP:         "127.0.0.1",
		UserAgent:        "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to assign resident A to unit A: %v", err)
	}
	tokenA := generateToken(resAUserID)

	// Create resident in Tower B
	resBPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/1000+3)%1000000000)
	resBNorm, _ := auth.NormalizePhone(resBPhone)
	var resBUserID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", resBNorm).Scan(&resBUserID)
	memB, err := societyRepo.RegisterMember(ctx, socID, superAdminRole.UserID, society.RegisterMemberParams{
		FirstName: "B Resident",
		Phone:     resBPhone,
		ClientIP:  "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to register member B: %v", err)
	}
	_, err = societyRepo.CreateUnitRelationship(ctx, socID, unitB, superAdminRole.UserID, society.CreateUnitRelationshipParams{
		MemberID:         memB.ID,
		RelationshipType: "PRIMARY_OWNER",
		IsOccupant:       true,
		StartDate:        "2026-01-01",
		ClientIP:         "127.0.0.1",
		UserAgent:        "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to assign resident B to unit B: %v", err)
	}
	tokenB := generateToken(resBUserID)

	// 1. Cross-society target rejection test: foreign building ID
	foreignBldID := uuid.New()
	badReqBody := map[string]interface{}{
		"title":              "Cross society notice",
		"body":               "Will fail due to foreign building",
		"audience_type":      "BUILDING",
		"target_building_id": foreignBldID,
	}
	badJSON, _ := json.Marshal(badReqBody)
	badReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements", socID), bytes.NewReader(badJSON))
	badReq.Header.Set("Authorization", "Bearer "+superToken)
	badReq.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, badReq)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for cross-society target, got %d: %s", wBad.Code, wBad.Body.String())
	}

	// 2. Create Tower A targeted announcement
	targetReqBody := map[string]interface{}{
		"title":              "Tower A Water Shutdown",
		"body":               "Shutdown applies only to Tower A",
		"category":           "MAINTENANCE",
		"priority":           "NORMAL",
		"audience_type":      "BUILDING",
		"target_building_id": bldA.ID,
	}
	targetJSON, _ := json.Marshal(targetReqBody)
	tReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements", socID), bytes.NewReader(targetJSON))
	tReq.Header.Set("Authorization", "Bearer "+superToken)
	tReq.Header.Set("Content-Type", "application/json")
	wT := httptest.NewRecorder()
	router.ServeHTTP(wT, tReq)
	if wT.Code != http.StatusCreated {
		t.Fatalf("expected 201 created target announcement, got %d: %s", wT.Code, wT.Body.String())
	}
	tData := parseData(wT.Body.Bytes())
	tAnnID := tData["announcement"].(map[string]interface{})["id"].(string)

	// Publish Tower A announcement
	tPubReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/publish", socID, tAnnID), nil)
	tPubReq.Header.Set("Authorization", "Bearer "+superToken)
	wTPub := httptest.NewRecorder()
	router.ServeHTTP(wTPub, tPubReq)
	if wTPub.Code != http.StatusOK {
		t.Fatalf("expected 200 publish, got %d: %s", wTPub.Code, wTPub.Body.String())
	}

	// 3. Resident A (in Tower A) CAN see announcement
	feedReqA := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/feed", socID), nil)
	feedReqA.Header.Set("Authorization", "Bearer "+tokenA)
	wFeedA := httptest.NewRecorder()
	router.ServeHTTP(wFeedA, feedReqA)
	feedDataA := parseData(wFeedA.Body.Bytes())
	if total, ok := feedDataA["total"].(float64); !ok || total != 1 {
		t.Errorf("expected resident A to see 1 announcement, got %v", feedDataA["total"])
	}

	// 4. Resident B (in Tower B) CANNOT see announcement
	feedReqB := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/announcements/feed", socID), nil)
	feedReqB.Header.Set("Authorization", "Bearer "+tokenB)
	wFeedB := httptest.NewRecorder()
	router.ServeHTTP(wFeedB, feedReqB)
	feedDataB := parseData(wFeedB.Body.Bytes())
	if total, ok := feedDataB["total"].(float64); !ok || total != 0 {
		t.Errorf("expected resident B to see 0 announcements, got %v", feedDataB["total"])
	}

	// 5. Resident B cannot mark Tower A announcement as read
	markBReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/read", socID, tAnnID), nil)
	markBReq.Header.Set("Authorization", "Bearer "+tokenB)
	wMarkB := httptest.NewRecorder()
	router.ServeHTTP(wMarkB, markBReq)
	if wMarkB.Code != http.StatusForbidden {
		t.Errorf("expected 403 when resident B attempts to read Tower A announcement, got %d: %s", wMarkB.Code, wMarkB.Body.String())
	}

	// 6. Resident cannot create announcement (negative auth check)
	resCreateReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements", socID), bytes.NewReader(targetJSON))
	resCreateReq.Header.Set("Authorization", "Bearer "+tokenA)
	resCreateReq.Header.Set("Content-Type", "application/json")
	wResCreate := httptest.NewRecorder()
	router.ServeHTTP(wResCreate, resCreateReq)
	if wResCreate.Code != http.StatusForbidden {
		t.Errorf("expected 403 when resident attempts to create announcement, got %d", wResCreate.Code)
	}

	// 7. Invalid lifecycle: trying to publish already published announcement
	doublePubReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/announcements/%s/publish", socID, tAnnID), nil)
	doublePubReq.Header.Set("Authorization", "Bearer "+superToken)
	wDoublePub := httptest.NewRecorder()
	router.ServeHTTP(wDoublePub, doublePubReq)
	if wDoublePub.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when publishing already published announcement, got %d: %s", wDoublePub.Code, wDoublePub.Body.String())
	}
}
