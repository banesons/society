package server

import (
	"bytes"
	"context"
	"encoding/json"
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
	"chs/backend/internal/response"
	"chs/backend/internal/sms"
	"chs/backend/internal/society"
)

func TestProfilePersistenceAndIdentityResolution(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping live server integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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

	router := SetupRouter(cfg, logger, db, authService, tm, platformRepo, societyRepo)

	// Fetch SUPER_ADMIN phone 919967089101
	superAdminPhone := "919967089101"
	var superAdminID uuid.UUID
	err = db.Pool().QueryRow(ctx, "SELECT id FROM users WHERE phone = $1", superAdminPhone).Scan(&superAdminID)
	if err != nil {
		t.Fatalf("failed to find SUPER_ADMIN user %s: %v", superAdminPhone, err)
	}

	// Create valid access token for SUPER_ADMIN
	sessionID := uuid.New()
	_, err = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, 'dummy_hash', NOW() + INTERVAL '1 hour')
	`, sessionID, superAdminID)
	if err != nil {
		t.Fatalf("failed to provision session for super admin: %v", err)
	}
	defer func() {
		_, _ = db.Pool().Exec(context.Background(), `DELETE FROM user_sessions WHERE id = $1`, sessionID)
	}()

	superAdminToken, err := tm.GenerateAccessToken(superAdminID, sessionID)
	if err != nil {
		t.Fatalf("failed to generate access token for super admin: %v", err)
	}

	// Always restore canonical seed profile data after test execution
	t.Cleanup(func() {
		_, _ = db.Pool().Exec(context.Background(), `
			UPDATE user_profiles
			SET first_name = 'Shuubham', last_name = 'Bane', email = 'shubham@banesons.com', profile_photo_url = NULL, updated_at = NOW()
			WHERE user_id = $1
		`, superAdminID)
	})

	// TEST A: Platform SUPER_ADMIN without society assignments
	t.Run("TEST A - Platform SUPER_ADMIN without society assignments persists profile", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Shuubham",
			"last_name":  "Bane",
			"email":      "shubham@banesons.com",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var env response.SuccessEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		data, ok := env.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected data object, got %T", env.Data)
		}

		if data["first_name"] != "Shuubham" {
			t.Errorf("expected first_name 'Shuubham', got %v", data["first_name"])
		}
		if data["last_name"] != "Bane" {
			t.Errorf("expected last_name 'Bane', got %v", data["last_name"])
		}
		if data["email"] != "shubham@banesons.com" {
			t.Errorf("expected email 'shubham@banesons.com', got %v", data["email"])
		}
		if data["display_name"] != "Shuubham Bane" {
			t.Errorf("expected display_name 'Shuubham Bane', got %v", data["display_name"])
		}
		if data["display_name"] == "Verified Resident" {
			t.Errorf("display_name must never be hardcoded Verified Resident")
		}

		// Verify user still has NO society assignments
		assignments, ok := data["society_assignments"].([]interface{})
		if !ok || len(assignments) != 0 {
			t.Errorf("expected empty society_assignments for platform super admin, got %v", data["society_assignments"])
		}

		// Verify NO society membership records were created in the database
		var memberCount int
		err = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM society_members WHERE user_id = $1", superAdminID).Scan(&memberCount)
		if err != nil {
			t.Fatalf("failed to count society_members: %v", err)
		}
		if memberCount != 0 {
			t.Errorf("expected 0 society_members rows for super admin, got %d", memberCount)
		}
	})

	// TEST B: Persistence after refresh
	t.Run("TEST B - Persistence after refresh via GET /auth/profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d (body: %s)", rec.Code, rec.Body.String())
		}

		var env response.SuccessEnvelope
		if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		data := env.Data.(map[string]interface{})
		if data["first_name"] != "Shuubham" || data["last_name"] != "Bane" || data["email"] != "shubham@banesons.com" {
			t.Errorf("persisted profile does not match saved values: %v", data)
		}
		if data["display_name"] != "Shuubham Bane" {
			t.Errorf("expected display_name 'Shuubham Bane', got %v", data["display_name"])
		}
	})

	// TEST C: Display name resolution precedence
	t.Run("TEST C - Display name resolution handles first name only", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Shubham",
			"last_name":  "",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var env response.SuccessEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		data := env.Data.(map[string]interface{})
		if data["display_name"] != "Shubham" {
			t.Errorf("expected single name 'Shubham', got %v", data["display_name"])
		}

		// Restore full name
		restorePayload := map[string]interface{}{
			"first_name": "Shuubham",
			"last_name":  "Bane",
			"email":      "shubham@banesons.com",
		}
		body, _ = json.Marshal(restorePayload)
		req = httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	})

	// TEST D: Validation rejects invalid inputs
	t.Run("TEST D - Validation rejects empty first name and malformed email", func(t *testing.T) {
		// Empty first name
		badNamePayload := map[string]interface{}{
			"first_name": "   ",
		}
		body, _ := json.Marshal(badNamePayload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for empty first name, got %d", rec.Code)
		}

		// Bad email
		badEmail := "not-an-email"
		badEmailPayload := map[string]interface{}{
			"first_name": "ValidName",
			"email":      badEmail,
		}
		body, _ = json.Marshal(badEmailPayload)
		req = httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec = httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for bad email, got %d", rec.Code)
		}
	})

	// TEST E: Phone immutability
	t.Run("TEST E - Phone immutability verified", func(t *testing.T) {
		payload := map[string]interface{}{
			"first_name": "Shuubham",
			"phone":      "918888888888", // attempt to modify phone
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var env response.SuccessEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		data := env.Data.(map[string]interface{})

		// Phone must remain original verified phone
		if data["phone_raw"] != superAdminPhone {
			t.Errorf("verified phone must not be modified: got %v, want %s", data["phone_raw"], superAdminPhone)
		}
	})

	// TEST F: Cross-user isolation (server-derived user ID strictly enforced)
	t.Run("TEST F - Cross-user isolation: body cannot supply different user ID", func(t *testing.T) {
		otherUserID := uuid.New()
		payload := map[string]interface{}{
			"id":         otherUserID.String(),
			"user_id":    otherUserID.String(),
			"first_name": "Hacker",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		var env response.SuccessEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		data := env.Data.(map[string]interface{})

		// Response must be for the authenticated user, not the injected ID
		if data["id"] != superAdminID.String() {
			t.Errorf("endpoint allowed identity spoofing: got id %v, want %s", data["id"], superAdminID.String())
		}

		// Immediate restoration to avoid dirtying live seed state
		restorePayload := map[string]interface{}{
			"first_name": "Shuubham",
			"last_name":  "Bane",
			"email":      "shubham@banesons.com",
		}
		resBody, _ := json.Marshal(restorePayload)
		resReq := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(resBody))
		resReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		resReq.Header.Set("X-Session-ID", sessionID.String())
		resRec := httptest.NewRecorder()
		router.ServeHTTP(resRec, resReq)
	})

	// TEST G: Society-member isolation: editing global profile does NOT overwrite society-specific membership records
	t.Run("TEST G - Society-member isolation: global profile update preserves tenant-specific member data", func(t *testing.T) {
		// Find a member who has a linked user_id and active membership
		var testMemberID, testUserID, testSocID uuid.UUID
		var originalMemberFirst, originalMemberLast string
		var testUserPhone string

		query := `
			SELECT sm.id, sm.user_id, sm.society_id, sm.first_name, COALESCE(sm.last_name, ''), u.phone
			FROM society_members sm
			JOIN users u ON sm.user_id = u.id
			WHERE sm.status = 'ACTIVE' AND sm.user_id IS NOT NULL
			LIMIT 1
		`
		err = db.Pool().QueryRow(ctx, query).Scan(
			&testMemberID, &testUserID, &testSocID,
			&originalMemberFirst, &originalMemberLast, &testUserPhone,
		)
		if err != nil {
			t.Skipf("no linked society members available for isolation test: %v", err)
			return
		}

		// Generate token for test user
		memberSessionID := uuid.New()
		_, err = db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at)
			VALUES ($1, $2, 'dummy_hash', NOW() + INTERVAL '1 hour')
		`, memberSessionID, testUserID)
		if err != nil {
			t.Fatalf("failed to provision session for member test: %v", err)
		}
		defer func() {
			_, _ = db.Pool().Exec(context.Background(), `DELETE FROM user_sessions WHERE id = $1`, memberSessionID)
		}()

		memberToken, err := tm.GenerateAccessToken(testUserID, memberSessionID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// Update global profile with a distinctly different name
		newGlobalFirst := "GlobalUniqueName"
		newGlobalLast := "GlobalUniqueLast"
		payload := map[string]interface{}{
			"first_name": newGlobalFirst,
			"last_name":  newGlobalLast,
			"email":      "global.unique@example.com",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+memberToken)
		req.Header.Set("X-Session-ID", memberSessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		// Verify global profile updated
		var dbGlobalFirst, dbGlobalLast string
		err = db.Pool().QueryRow(ctx, "SELECT first_name, last_name FROM user_profiles WHERE user_id = $1", testUserID).
			Scan(&dbGlobalFirst, &dbGlobalLast)
		if err != nil || dbGlobalFirst != newGlobalFirst {
			t.Errorf("global profile was not persisted: %v", err)
		}

		// CRITICAL: verify society_members table was NOT altered!
		var memberCurrentFirst, memberCurrentLast string
		err = db.Pool().QueryRow(ctx, "SELECT first_name, COALESCE(last_name, '') FROM society_members WHERE id = $1", testMemberID).
			Scan(&memberCurrentFirst, &memberCurrentLast)
		if err != nil {
			t.Fatalf("failed to query member: %v", err)
		}

		if memberCurrentFirst != originalMemberFirst || memberCurrentLast != originalMemberLast {
			t.Errorf("tenant isolation violation! society_members record was altered: got %s %s, want original %s %s",
				memberCurrentFirst, memberCurrentLast, originalMemberFirst, originalMemberLast)
		}

		// Restore user_profiles record for test user
		_, _ = db.Pool().Exec(ctx, `
			UPDATE user_profiles
			SET first_name = $1, last_name = $2, updated_at = NOW()
			WHERE user_id = $3
		`, originalMemberFirst, originalMemberLast, testUserID)
	})

	// TEST H: Persistence failure for non-existent user
	t.Run("TEST H - Persistence failure handles non-existent user gracefully", func(t *testing.T) {
		fakeUserID := uuid.New()
		fakeToken, err := tm.GenerateAccessToken(fakeUserID, uuid.New())
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		payload := map[string]interface{}{
			"first_name": "Ghost",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+fakeToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		// Must return 400 or 401 or 404, never 200 success!
		if rec.Code == http.StatusOK {
			t.Errorf("endpoint reported success for non-existent user!")
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code == "" {
			t.Errorf("expected structured error code for failure")
		}
	})

	// TEST I: Profile photo URL persistence and removal
	t.Run("TEST I - Profile photo URL persistence and nullable removal", func(t *testing.T) {
		testPhotoURL := "https://example.com/portraits/shubham.jpg"
		payload := map[string]interface{}{
			"first_name":        "Shuubham",
			"last_name":         "Bane",
			"email":             "shubham@banesons.com",
			"profile_photo_url": testPhotoURL,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		// Verify on GET /auth/profile
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
		getReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		getReq.Header.Set("X-Session-ID", sessionID.String())
		getRec := httptest.NewRecorder()
		router.ServeHTTP(getRec, getReq)

		var getEnv response.SuccessEnvelope
		_ = json.NewDecoder(getRec.Body).Decode(&getEnv)
		getData := getEnv.Data.(map[string]interface{})
		if getData["profile_photo_url"] != testPhotoURL {
			t.Errorf("expected photo URL %s, got %v", testPhotoURL, getData["profile_photo_url"])
		}

		// Remove photo URL by sending empty string
		clearPayload := map[string]interface{}{
			"first_name":        "Shuubham",
			"last_name":         "Bane",
			"email":             "shubham@banesons.com",
			"profile_photo_url": "",
		}
		clearBody, _ := json.Marshal(clearPayload)
		clearReq := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/profile", bytes.NewReader(clearBody))
		clearReq.Header.Set("Authorization", "Bearer "+superAdminToken)
		clearReq.Header.Set("X-Session-ID", sessionID.String())
		clearRec := httptest.NewRecorder()
		router.ServeHTTP(clearRec, clearReq)
		if clearRec.Code != http.StatusOK {
			t.Fatalf("expected 200 on photo clear, got %d", clearRec.Code)
		}

		// Verify GET returns nil
		getReq2 := httptest.NewRequest(http.MethodGet, "/api/v1/auth/profile", nil)
		getReq2.Header.Set("Authorization", "Bearer "+superAdminToken)
		getReq2.Header.Set("X-Session-ID", sessionID.String())
		getRec2 := httptest.NewRecorder()
		router.ServeHTTP(getRec2, getReq2)

		var getEnv2 response.SuccessEnvelope
		_ = json.NewDecoder(getRec2.Body).Decode(&getEnv2)
		getData2 := getEnv2.Data.(map[string]interface{})
		if getData2["profile_photo_url"] != nil {
			t.Errorf("expected nil photo URL after clear, got %v", getData2["profile_photo_url"])
		}
	})

	// TEST J: GET /auth/me returns canonical profile identity
	t.Run("TEST J - GET /auth/me returns canonical profile identity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("X-Session-ID", sessionID.String())
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var env response.SuccessEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		data := env.Data.(map[string]interface{})
		if data["display_name"] != "Shuubham Bane" {
			t.Errorf("expected display_name 'Shuubham Bane' on /me, got %v", data["display_name"])
		}
		if data["first_name"] != "Shuubham" {
			t.Errorf("expected first_name 'Shuubham' on /me, got %v", data["first_name"])
		}
		if data["email"] != "shubham@banesons.com" {
			t.Errorf("expected email 'shubham@banesons.com' on /me, got %v", data["email"])
		}
	})
}
