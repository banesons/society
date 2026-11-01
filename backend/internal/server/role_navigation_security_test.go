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

func TestRoleNavigationSecurityAndIsolation(t *testing.T) {
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

	// Fetch canonical societies
	var puneSocID, mumbaiSocID uuid.UUID
	err = db.Pool().QueryRow(ctx, "SELECT id FROM societies WHERE city = 'Pune' AND status = 'ACTIVE' LIMIT 1").Scan(&puneSocID)
	if err != nil {
		t.Fatalf("failed to find canonical Pune society: %v", err)
	}
	err = db.Pool().QueryRow(ctx, "SELECT id FROM societies WHERE city = 'Mumbai' AND status = 'ACTIVE' LIMIT 1").Scan(&mumbaiSocID)
	if err != nil {
		t.Fatalf("failed to find canonical Mumbai society: %v", err)
	}

	// Helper to generate a valid JWT with an active session in user_sessions
	generateSessionToken := func(userID uuid.UUID) (string, uuid.UUID) {
		sessionID := uuid.New()
		_, err := db.Pool().Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at, created_at)
			VALUES ($1, $2, 'dummy-hash', NOW() + INTERVAL '1 hour', NOW())
		`, sessionID, userID)
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}

		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		return tok, sessionID
	}

	// Helper to get or create a user by phone
	getOrCreateUser := func(phone string) (uuid.UUID, string) {
		var uid uuid.UUID
		err := db.Pool().QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ($1, 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`, phone).Scan(&uid)
		if err != nil {
			t.Fatalf("failed to get/create user %s: %v", phone, err)
		}
		tok, _ := generateSessionToken(uid)
		return uid, tok
	}

	// Set up identities
	// 1. Super Admin (+919967089101)
	superAdminID, superAdminToken := getOrCreateUser("919967089101")
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status, assigned_by)
		VALUES ($1, 'SUPER_ADMIN', 'ACTIVE', $1)
		ON CONFLICT (user_id) DO UPDATE SET role = 'SUPER_ADMIN', status = 'ACTIVE'
	`, superAdminID)

	// 2. Platform Admin (+918422862001)
	platformAdminID, platformAdminToken := getOrCreateUser("918422862001")
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status, assigned_by)
		VALUES ($1, 'ADMIN', 'ACTIVE', $2)
		ON CONFLICT (user_id) DO UPDATE SET role = 'ADMIN', status = 'ACTIVE'
	`, platformAdminID, superAdminID)

	// 3. Resident of Pune Society (Ajit Kanitkar, Flat 109, +919822000109)
	residentID, residentToken := getOrCreateUser("919822000109")
	// Link resident to society_members if not already linked
	_, _ = db.Pool().Exec(ctx, `
		UPDATE society_members SET user_id = $1 WHERE phone = '919822000109' AND society_id = $2
	`, residentID, puneSocID)

	// 4. Gatekeeper of Pune Society (+919820020001)
	guardID, guardToken := getOrCreateUser("919820020001")
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)
		ON CONFLICT DO NOTHING
	`, puneSocID, guardID, superAdminID)

	// 5. Dedicated Society Admin of Pune Society (+919822000999)
	socAdminID, socAdminToken := getOrCreateUser("919822000999")
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by)
		VALUES ($1, $2, 'SOC_ADMIN', 'ACTIVE', $3)
		ON CONFLICT DO NOTHING
	`, puneSocID, socAdminID, superAdminID)

	// =========================================================================
	// 1. RESIDENT AUTHORIZATION & WORKFLOW VALIDATION
	// =========================================================================
	t.Run("Resident Permitted: Profile, Announcements, Bills, Documents, Units", func(t *testing.T) {
		endpoints := []struct {
			name   string
			method string
			path   string
		}{
			{"Society Profile", "GET", fmt.Sprintf("/api/v1/societies/%s/profile", puneSocID)},
			{"Announcements Feed", "GET", fmt.Sprintf("/api/v1/societies/%s/announcements/feed", puneSocID)},
			{"Unread Announcements Count", "GET", fmt.Sprintf("/api/v1/societies/%s/announcements/unread-count", puneSocID)},
			{"Resident My Bills", "GET", fmt.Sprintf("/api/v1/societies/%s/maintenance/my-bills", puneSocID)},
			{"Resident My Payments", "GET", fmt.Sprintf("/api/v1/societies/%s/maintenance/my-payments", puneSocID)},
			{"Resident Documents List", "GET", fmt.Sprintf("/api/v1/societies/%s/documents", puneSocID)},
			{"Resident Units", "GET", fmt.Sprintf("/api/v1/societies/%s/units/my", puneSocID)},
			{"Pending Visitor Approvals", "GET", fmt.Sprintf("/api/v1/societies/%s/visitors/pending", puneSocID)},
		}

		for _, ep := range endpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+residentToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("[%s] expected 200 OK for resident, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		}
	})

	t.Run("Resident Forbidden: Administrative & Perimeter APIs", func(t *testing.T) {
		adminEndpoints := []struct {
			name   string
			method string
			path   string
		}{
			{"Physical Structure Buildings", "GET", fmt.Sprintf("/api/v1/societies/%s/buildings", puneSocID)},
			{"Physical Structure Units", "GET", fmt.Sprintf("/api/v1/societies/%s/units", puneSocID)},
			{"Member Directory", "GET", fmt.Sprintf("/api/v1/societies/%s/members", puneSocID)},
			{"Committee Designations", "GET", fmt.Sprintf("/api/v1/societies/%s/designations", puneSocID)},
			{"Committee Appointments", "GET", fmt.Sprintf("/api/v1/societies/%s/appointments", puneSocID)},
			{"Committee Overview", "GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", puneSocID)},
			{"Gatekeeper Staffing List", "GET", fmt.Sprintf("/api/v1/societies/%s/gatekeepers", puneSocID)},
			{"Admin Maintenance Categories", "GET", fmt.Sprintf("/api/v1/societies/%s/maintenance/categories", puneSocID)},
			{"Admin Maintenance All Bills", "GET", fmt.Sprintf("/api/v1/societies/%s/maintenance/bills", puneSocID)},
			{"Society Audit Logs", "GET", fmt.Sprintf("/api/v1/societies/%s/audit-logs", puneSocID)},
			{"Society Admin Dashboard", "GET", fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", puneSocID)},
			{"Gate Operations Log", "GET", fmt.Sprintf("/api/v1/societies/%s/visitors", puneSocID)},
		}

		for _, ep := range adminEndpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+residentToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("[%s] expected 403 Forbidden for resident accessing admin API, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		}
	})

	// =========================================================================
	// 2. GATEKEEPER AUTHORIZATION VALIDATION
	// =========================================================================
	t.Run("Gatekeeper Permitted: Perimeter Operations & Broadcast Notices", func(t *testing.T) {
		endpoints := []struct {
			name   string
			method string
			path   string
		}{
			{"Gates List", "GET", fmt.Sprintf("/api/v1/societies/%s/gates", puneSocID)},
			{"Gate Shifts", "GET", fmt.Sprintf("/api/v1/societies/%s/shifts", puneSocID)},
			{"Duty Sessions", "GET", fmt.Sprintf("/api/v1/societies/%s/duty-sessions", puneSocID)},
			{"Visitor Operations", "GET", fmt.Sprintf("/api/v1/societies/%s/visitors", puneSocID)},
			{"Announcements Feed", "GET", fmt.Sprintf("/api/v1/societies/%s/announcements/feed", puneSocID)},
		}

		for _, ep := range endpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+guardToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("[%s] expected 200 OK for gatekeeper, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		}
	})

	t.Run("Gatekeeper Forbidden: Society Administration & Resident Private Records", func(t *testing.T) {
		forbiddenEndpoints := []struct {
			name   string
			method string
			path   string
		}{
			{"Member Directory", "GET", fmt.Sprintf("/api/v1/societies/%s/members", puneSocID)},
			{"Committee Governance", "GET", fmt.Sprintf("/api/v1/societies/%s/committee/overview", puneSocID)},
			{"Physical Structure", "GET", fmt.Sprintf("/api/v1/societies/%s/buildings", puneSocID)},
			{"Admin Billing Categories", "GET", fmt.Sprintf("/api/v1/societies/%s/maintenance/categories", puneSocID)},
			{"Society Audit Logs", "GET", fmt.Sprintf("/api/v1/societies/%s/audit-logs", puneSocID)},
			{"Society Dashboard", "GET", fmt.Sprintf("/api/v1/societies/%s/dashboard/summary", puneSocID)},
		}

		for _, ep := range forbiddenEndpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+guardToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("[%s] expected 403 Forbidden for gatekeeper accessing admin API, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		}
	})

	// =========================================================================
	// 3. CROSS-SOCIETY ACCESS ENFORCEMENT (TENANT ISOLATION)
	// =========================================================================
	t.Run("Cross-Society Isolation: Resident of Pune cannot access Mumbai endpoints", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/profile", mumbaiSocID), nil)
		req.Header.Set("Authorization", "Bearer "+residentToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for cross-society resident access, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Cross-Society Isolation: Pune SOC_ADMIN cannot access Mumbai society", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/buildings", mumbaiSocID), nil)
		req.Header.Set("Authorization", "Bearer "+socAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for cross-society admin access, got %d: %s", w.Code, w.Body.String())
		}
	})

	// =========================================================================
	// 4. SOCIETY ADMINISTRATOR CANNOT ACCESS PLATFORM-ONLY DOMAIN
	// =========================================================================
	t.Run("Platform Isolation: SOC_ADMIN cannot access platform endpoints", func(t *testing.T) {
		platformEndpoints := []struct {
			name   string
			method string
			path   string
		}{
			{"Platform Societies", "GET", "/api/v1/platform/societies"},
			{"Platform Administrators", "GET", "/api/v1/platform/admins"},
			{"Platform Audit Logs", "GET", "/api/v1/platform/audit-logs"},
			{"Platform Dashboard", "GET", "/api/v1/platform/dashboard/summary"},
		}

		for _, ep := range platformEndpoints {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			req.Header.Set("Authorization", "Bearer "+socAdminToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Errorf("[%s] expected 403 Forbidden for SOC_ADMIN accessing platform domain, got %d: %s", ep.name, w.Code, w.Body.String())
			}
		}
	})

	// =========================================================================
	// 5. SUPER_ADMIN VS PLATFORM ADMIN PRIVILEGE SEPARATION
	// =========================================================================
	t.Run("Privilege Separation: Platform ADMIN cannot create new platform administrators", func(t *testing.T) {
		body := map[string]string{
			"phone": "+919820999888",
			"role":  "ADMIN",
		}
		jsonBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/platform/admins", bytes.NewBuffer(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+platformAdminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for platform ADMIN creating new admin, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Privilege Separation: Platform ADMIN has NO implicit society access", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/buildings", puneSocID), nil)
		req.Header.Set("Authorization", "Bearer "+platformAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for unassigned platform ADMIN accessing society, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Privilege Separation: SUPER_ADMIN holds cross-society governance inspection access", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/buildings", puneSocID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK for SUPER_ADMIN governance access, got %d: %s", w.Code, w.Body.String())
		}
	})

	// =========================================================================
	// 6. SESSION REVOCATION & LOGOUT ENFORCEMENT
	// =========================================================================
	t.Run("Session Invalidation: Expired or deleted session is rejected", func(t *testing.T) {
		userXID, _ := getOrCreateUser("919820777111")
		tokenX, sessionXID := generateSessionToken(userXID)

		// Request works initially
		req1 := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req1.Header.Set("Authorization", "Bearer "+tokenX)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("expected 200 for fresh session, got %d", w1.Code)
		}

		// Delete session from user_sessions (simulate logout)
		_, err := db.Pool().Exec(ctx, "DELETE FROM user_sessions WHERE id = $1", sessionXID)
		if err != nil {
			t.Fatalf("failed to delete test session: %v", err)
		}

		// Subsequent request must fail with 401
		req2 := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		req2.Header.Set("Authorization", "Bearer "+tokenX)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for revoked session, got %d: %s", w2.Code, w2.Body.String())
		}
	})

	// =========================================================================
	// 7. INVALID SOCIETY IDENTIFIER HANDLING
	// =========================================================================
	t.Run("Invalid Society ID: Non-existent UUID returns 404", func(t *testing.T) {
		randomUUID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/societies/%s/profile", randomUUID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent society UUID, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Invalid Society ID: Malformed UUID returns 400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/societies/not-a-valid-uuid/profile", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for malformed society UUID, got %d: %s", w.Code, w.Body.String())
		}
	})
}
