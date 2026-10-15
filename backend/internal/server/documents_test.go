package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDocumentsAndRecordsLifecycle(t *testing.T) {
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
			VALUES ($1, $2, $3, 'test-agent', '127.0.0.1', $4)
		`, sessionID, userID, "dummy-hash", time.Now().Add(1*time.Hour))
		if err != nil {
			t.Fatalf("failed to insert test session: %v", err)
		}
		tok, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate access token: %v", err)
		}
		return tok
	}

	superToken := generateToken(superAdminRole.UserID)

	// Create isolated test society
	socName := fmt.Sprintf("Doc Test Society %d", time.Now().UnixNano())
	regNo := fmt.Sprintf("REG-DOC-%d", time.Now().UnixNano()%1000000)
	socParams := society.CreateSocietyParams{
		Name:               socName,
		RegistrationNumber: &regNo,
		AddressLine1:       "200 Records Boulevard",
		Locality:           "Shivajinagar",
		City:               "Pune",
		State:              "Maharashtra",
		Pincode:            "411005",
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

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &socResp)
	socID := uuid.MustParse(socResp.Data.ID)

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
	adminToken := generateToken(adminUserID)

	assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader([]byte(fmt.Sprintf(`{"user_id":"%s"}`, adminUserID))))
	assignReq.Header.Set("Authorization", "Bearer "+superToken)
	assignReq.Header.Set("Content-Type", "application/json")
	wAssign := httptest.NewRecorder()
	router.ServeHTTP(wAssign, assignReq)
	if wAssign.Code != http.StatusCreated {
		t.Fatalf("failed to assign admin: %d: %s", wAssign.Code, wAssign.Body.String())
	}

	// Create test resident member
	resPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/1000+1)%1000000000)
	resNorm, _ := auth.NormalizePhone(resPhone)
	var residentUserID uuid.UUID
	err = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", resNorm).Scan(&residentUserID)
	if err != nil {
		t.Fatalf("failed to create resident user: %v", err)
	}
	residentToken := generateToken(residentUserID)

	var residentMemberID uuid.UUID
	err = db.Pool().QueryRow(ctx, `
		INSERT INTO society_members (society_id, user_id, first_name, last_name, phone, status)
		VALUES ($1, $2, 'Ramesh', 'Patil', $3, 'ACTIVE')
		RETURNING id
	`, socID, residentUserID, resNorm).Scan(&residentMemberID)
	if err != nil {
		t.Fatalf("failed to create member: %v", err)
	}

	// Test 1: List Categories
	var targetCatID uuid.UUID
	t.Run("List Categories", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/categories", socID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Categories []society.DocumentCategory `json:"categories"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Data.Categories) != 8 {
			t.Fatalf("expected 8 seeded categories, got %d", len(resp.Data.Categories))
		}
		targetCatID = resp.Data.Categories[0].ID
	})

	// Test 2: Reject Invalid File Type (.txt)
	t.Run("Reject Invalid File Type", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Invalid Document Test")
		_ = w.WriteField("category_id", targetCatID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "test.txt")
		_, _ = fw.Write([]byte("plain text content not allowed"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for txt file, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// Test 3: Reject Oversized File (>10MB)
	t.Run("Reject Oversized File", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Oversized Document Test")
		_ = w.WriteField("category_id", targetCatID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "huge.pdf")
		oversized := make([]byte, society.MaxDocumentFileSize+100)
		copy(oversized, []byte("%PDF-1.4"))
		_, _ = fw.Write(oversized)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for oversized file, got %d", rec.Code)
		}
	})

	// Test 4: Create Valid Document (Version 1 PDF)
	var createdDocID uuid.UUID
	var createdVersionID uuid.UUID
	t.Run("Create Valid Document (PDF)", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Official Test Bylaws 2026")
		_ = w.WriteField("description", "Registered society rules and constitution")
		_ = w.WriteField("category_id", targetCatID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		_ = w.WriteField("change_summary", "Initial registered edition")
		fw, _ := w.CreateFormFile("file", "bylaws_v1.pdf")
		_, _ = fw.Write([]byte("%PDF-1.4 sample test pdf document bytes"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Document society.SocietyDocument `json:"document"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Data.Document.Title != "Official Test Bylaws 2026" {
			t.Errorf("unexpected title: %s", resp.Data.Document.Title)
		}
		if resp.Data.Document.CurrentVersion == nil {
			t.Fatalf("expected CurrentVersion to be populated")
		}
		if resp.Data.Document.CurrentVersion.VersionNumber != 1 {
			t.Errorf("expected version 1, got %d", resp.Data.Document.CurrentVersion.VersionNumber)
		}

		createdDocID = resp.Data.Document.ID
		createdVersionID = resp.Data.Document.CurrentVersion.ID
	})

	// Test 5: Upload New Version (Version 2)
	var version2ID uuid.UUID
	t.Run("Upload New Version", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("change_summary", "Updated clause 4b with parking rules")
		fw, _ := w.CreateFormFile("file", "bylaws_v2.pdf")
		_, _ = fw.Write([]byte("%PDF-1.4 revised sample test pdf document bytes"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions", socID, createdDocID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for version 2, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Version society.DocumentVersion `json:"version"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Data.Version.VersionNumber != 2 {
			t.Errorf("expected version number 2, got %d", resp.Data.Version.VersionNumber)
		}
		version2ID = resp.Data.Version.ID
	})

	// Test 6: List Versions (Verify both v1 and v2 exist)
	t.Run("List Versions", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions", socID, createdDocID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Versions []society.DocumentVersion `json:"versions"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp.Data.Versions) != 2 {
			t.Fatalf("expected 2 versions, got %d", len(resp.Data.Versions))
		}
		if resp.Data.Versions[0].VersionNumber != 2 || resp.Data.Versions[1].VersionNumber != 1 {
			t.Errorf("versions not sorted in descending order: %v, %v", resp.Data.Versions[0].VersionNumber, resp.Data.Versions[1].VersionNumber)
		}
	})

	// Test 7: Download Version
	t.Run("Download Version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions/%s/download?inline=true", socID, createdDocID, version2ID), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for download, got %d: %s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Content-Type") != "application/pdf" {
			t.Errorf("expected application/pdf content type, got: %s", rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("expected nosniff header")
		}
	})

	// Test 8: Archive and Restore Document
	t.Run("Archive and Restore Document", func(t *testing.T) {
		// Archive without reason -> 400
		reqNoReason := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/archive", socID, createdDocID), bytes.NewReader([]byte(`{"reason":""}`)))
		reqNoReason.Header.Set("Authorization", "Bearer "+adminToken)
		recNoReason := httptest.NewRecorder()
		router.ServeHTTP(recNoReason, reqNoReason)
		if recNoReason.Code != http.StatusBadRequest {
			t.Errorf("expected 400 when reason is empty, got %d", recNoReason.Code)
		}

		// Archive with reason -> 200
		reqArchive := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/archive", socID, createdDocID), bytes.NewReader([]byte(`{"reason":"Superseded by 2027 revision"}`)))
		reqArchive.Header.Set("Authorization", "Bearer "+adminToken)
		recArchive := httptest.NewRecorder()
		router.ServeHTTP(recArchive, reqArchive)
		if recArchive.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on archive, got %d: %s", recArchive.Code, recArchive.Body.String())
		}

		// Resident cannot see archived document in list
		reqResList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents", socID), nil)
		reqResList.Header.Set("Authorization", "Bearer "+residentToken)
		recResList := httptest.NewRecorder()
		router.ServeHTTP(recResList, reqResList)
		if recResList.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on resident list, got %d: %s", recResList.Code, recResList.Body.String())
		}
		var listResp struct {
			Data struct {
				Documents []society.SocietyDocument `json:"documents"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recResList.Body.Bytes(), &listResp)
		for _, d := range listResp.Data.Documents {
			if d.ID == createdDocID {
				t.Fatalf("archived document leaked into resident list")
			}
		}

		// Restore -> 200
		reqRestore := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/restore", socID, createdDocID), nil)
		reqRestore.Header.Set("Authorization", "Bearer "+adminToken)
		recRestore := httptest.NewRecorder()
		router.ServeHTTP(recRestore, reqRestore)
		if recRestore.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on restore, got %d: %s", recRestore.Code, recRestore.Body.String())
		}
	})

	// Test 9: Classification RBAC (Create Admin-Only Doc and verify resident is denied)
	t.Run("Classification RBAC Enforced", func(t *testing.T) {
		// Create SOC_ADMIN_ONLY document
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Admin Confidential Financial Audit")
		_ = w.WriteField("category_id", targetCatID.String())
		_ = w.WriteField("classification", "SOC_ADMIN_ONLY")
		fw, _ := w.CreateFormFile("file", "audit_confidential.pdf")
		_, _ = fw.Write([]byte("%PDF-1.4 confidential audit bytes"))
		w.Close()

		reqCreate := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		reqCreate.Header.Set("Content-Type", w.FormDataContentType())
		reqCreate.Header.Set("Authorization", "Bearer "+adminToken)
		recCreate := httptest.NewRecorder()
		router.ServeHTTP(recCreate, reqCreate)
		if recCreate.Code != http.StatusCreated {
			t.Fatalf("failed to create admin doc: %d: %s", recCreate.Code, recCreate.Body.String())
		}

		var resp struct {
			Data struct {
				Document society.SocietyDocument `json:"document"`
			} `json:"data"`
		}
		_ = json.Unmarshal(recCreate.Body.Bytes(), &resp)
		adminDocID := resp.Data.Document.ID
		adminDocVerID := resp.Data.Document.CurrentVersion.ID

		// Resident tries to get details -> 403 Forbidden
		reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/%s", socID, adminDocID), nil)
		reqGet.Header.Set("Authorization", "Bearer "+residentToken)
		recGet := httptest.NewRecorder()
		router.ServeHTTP(recGet, reqGet)
		if recGet.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for resident accessing SOC_ADMIN_ONLY doc, got %d", recGet.Code)
		}

		// Resident tries to download -> 403 Forbidden
		reqDl := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions/%s/download", socID, adminDocID, adminDocVerID), nil)
		reqDl.Header.Set("Authorization", "Bearer "+residentToken)
		recDl := httptest.NewRecorder()
		router.ServeHTTP(recDl, reqDl)
		if recDl.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden for resident downloading SOC_ADMIN_ONLY doc, got %d", recDl.Code)
		}

		// Clean up admin document
		_, _ = db.Pool().Exec(ctx, `DELETE FROM document_versions WHERE document_id = $1`, adminDocID)
		_, _ = db.Pool().Exec(ctx, `DELETE FROM society_documents WHERE id = $1`, adminDocID)
	})

	// Test 10: Cross-Society IDOR Protection
	t.Run("Cross-Society Document Access Rejected", func(t *testing.T) {
		otherSocID := uuid.New()
		reqOther := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/%s", otherSocID, createdDocID), nil)
		reqOther.Header.Set("Authorization", "Bearer "+adminToken)
		recOther := httptest.NewRecorder()
		router.ServeHTTP(recOther, reqOther)

		if recOther.Code != http.StatusNotFound && recOther.Code != http.StatusForbidden {
			t.Errorf("expected 404 or 403 for cross-society document access, got %d", recOther.Code)
		}
	})

	// Test 11: Document Storage Metrics Endpoint
	t.Run("Document Storage Metrics", func(t *testing.T) {
		reqMetrics := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/documents/metrics", socID), nil)
		reqMetrics.Header.Set("Authorization", "Bearer "+adminToken)
		recMetrics := httptest.NewRecorder()
		router.ServeHTTP(recMetrics, reqMetrics)

		if recMetrics.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for storage metrics, got %d: %s", recMetrics.Code, recMetrics.Body.String())
		}

		var metricsResp struct {
			Data struct {
				Metrics society.DocumentStorageMetrics `json:"metrics"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recMetrics.Body.Bytes(), &metricsResp); err != nil {
			t.Fatalf("failed to decode metrics response: %v", err)
		}
		if metricsResp.Data.Metrics.TotalDocuments < 1 {
			t.Errorf("expected at least 1 document in metrics, got %d", metricsResp.Data.Metrics.TotalDocuments)
		}
	})

	// Test 12: File exceeding accepted 5MB limit is rejected
	t.Run("File Exceeding 5MB Rejected", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Large Document")
		_ = w.WriteField("category_id", targetCatID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "large.pdf")
		largeBytes := make([]byte, 6*1024*1024)
		copy(largeBytes, []byte("%PDF-1.4"))
		_, _ = fw.Write(largeBytes)
		w.Close()

		reqLarge := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		reqLarge.Header.Set("Content-Type", w.FormDataContentType())
		reqLarge.Header.Set("Authorization", "Bearer "+adminToken)
		recLarge := httptest.NewRecorder()
		router.ServeHTTP(recLarge, reqLarge)

		if recLarge.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for 6MB unoptimized PDF, got %d", recLarge.Code)
		}
	})

	// Clean up created test document and versions
	_ = createdVersionID
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_versions WHERE document_id = $1`, createdDocID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM society_documents WHERE id = $1`, createdDocID)
}

func TestDocumentVersionLimitEnforcement(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration tests")
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	sessionID := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, sessionID, superAdminRole.UserID, time.Now().Add(1*time.Hour))
	superToken, _ := tm.GenerateAccessToken(superAdminRole.UserID, sessionID)

	// Create test society
	socName := fmt.Sprintf("Limit Test Society %d", time.Now().UnixNano())
	socParams := society.CreateSocietyParams{
		Name:         socName,
		AddressLine1: "Limit Test Road",
		Locality:     "Kothrud",
		City:         "Pune",
		State:        "Maharashtra",
		Pincode:      "411038",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSoc.Body.Bytes(), &socResp)
	socID := uuid.MustParse(socResp.Data.ID)

	// Activate society
	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	// Create admin
	adminPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/777)%1000000000)
	adminNorm, _ := auth.NormalizePhone(adminPhone)
	var adminUserID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", adminNorm).Scan(&adminUserID)
	adminSess := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, adminSess, adminUserID, time.Now().Add(1*time.Hour))
	adminToken, _ := tm.GenerateAccessToken(adminUserID, adminSess)

	assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader([]byte(fmt.Sprintf(`{"user_id":"%s"}`, adminUserID))))
	assignReq.Header.Set("Authorization", "Bearer "+superToken)
	assignReq.Header.Set("Content-Type", "application/json")
	wAssign := httptest.NewRecorder()
	router.ServeHTTP(wAssign, assignReq)

	// Create document category
	cat, err := societyRepo.CreateCategory(ctx, socID, "Legal", nil, 1, adminUserID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Create test document (version 1)
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("title", "Version Limit Document")
	_ = w.WriteField("category_id", cat.ID.String())
	_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
	fw, _ := w.CreateFormFile("file", "v1.pdf")
	_, _ = fw.Write([]byte("%PDF-1.4 test version 1 content"))
	w.Close()

	reqDoc := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
	reqDoc.Header.Set("Content-Type", w.FormDataContentType())
	reqDoc.Header.Set("Authorization", "Bearer "+adminToken)
	recDoc := httptest.NewRecorder()
	router.ServeHTTP(recDoc, reqDoc)
	if recDoc.Code != http.StatusCreated {
		t.Fatalf("failed to create document: %d: %s", recDoc.Code, recDoc.Body.String())
	}

	var createResp struct {
		Data struct {
			Document struct {
				ID string `json:"id"`
			} `json:"document"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recDoc.Body.Bytes(), &createResp)
	docID := uuid.MustParse(createResp.Data.Document.ID)

	// Insert versions 2 through 20 directly to reach the 20-version ceiling
	for v := 2; v <= 20; v++ {
		dummyVerID := uuid.New()
		dummyPath := fmt.Sprintf("societies/%s/documents/%s/versions/%s/v%d.pdf", socID, docID, dummyVerID, v)
		_, err := db.Pool().Exec(ctx, `
			INSERT INTO document_versions (
				id, society_id, document_id, version_number, storage_path, file_name,
				file_size, mime_type, file_hash, change_summary, uploaded_by
			) VALUES ($1, $2, $3, $4, $5, $6, 100, 'application/pdf', 'hash', 'test', $7)
		`, dummyVerID, socID, docID, v, dummyPath, fmt.Sprintf("v%d.pdf", v), adminUserID)
		if err != nil {
			t.Fatalf("failed to insert dummy version %d: %v", v, err)
		}
	}

	// Now attempting version 21 via HTTP must be rejected by backend version limit check (ErrMaxVersionsReached)
	var b21 bytes.Buffer
	w21 := multipart.NewWriter(&b21)
	_ = w21.WriteField("change_summary", "Attempting version 21 beyond ceiling")
	fw21, _ := w21.CreateFormFile("file", "v21.pdf")
	_, _ = fw21.Write([]byte("%PDF-1.4 test version 21 content"))
	w21.Close()

	req21 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions", socID, docID), &b21)
	req21.Header.Set("Content-Type", w21.FormDataContentType())
	req21.Header.Set("Authorization", "Bearer "+adminToken)
	rec21 := httptest.NewRecorder()
	router.ServeHTTP(rec21, req21)

	if rec21.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when exceeding 20 versions, got %d: %s", rec21.Code, rec21.Body.String())
	}

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec21.Body.Bytes(), &errResp)
	if errResp.Error.Code != "MAX_VERSIONS_REACHED" {
		t.Errorf("expected error code MAX_VERSIONS_REACHED, got: %s", errResp.Error.Code)
	}

	// Direct DB insert of version 21 should also fail via the database trigger
	v21ID := uuid.New()
	_, triggerErr := db.Pool().Exec(ctx, `
		INSERT INTO document_versions (
			id, society_id, document_id, version_number, storage_path, file_name,
			file_size, mime_type, file_hash, change_summary, uploaded_by
		) VALUES ($1, $2, $3, 21, 'dummy', 'v21.pdf', 100, 'application/pdf', 'hash', 'test', $4)
	`, v21ID, socID, docID, adminUserID)
	if triggerErr == nil {
		t.Error("expected DB trigger to reject version 21 insert directly")
	}

	// Clean up isolated test document and society
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_versions WHERE document_id = $1`, docID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM society_documents WHERE id = $1`, docID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_categories WHERE society_id = $1`, socID)
}

func TestDocumentUploadLimits_OversizedAndMalformed(t *testing.T) {
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

	sessionID := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, sessionID, superAdminRole.UserID, time.Now().Add(1*time.Hour))
	superToken, _ := tm.GenerateAccessToken(superAdminRole.UserID, sessionID)

	socName := fmt.Sprintf("Upload Limits Society %d", time.Now().UnixNano())
	socParams := society.CreateSocietyParams{
		Name:         socName,
		AddressLine1: "Upload Road",
		Locality:     "Shivajinagar",
		City:         "Pune",
		State:        "Maharashtra",
		Pincode:      "411005",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSoc.Body.Bytes(), &socResp)
	socID := uuid.MustParse(socResp.Data.ID)

	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	adminPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/432)%1000000000)
	adminNorm, _ := auth.NormalizePhone(adminPhone)
	var adminUserID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", adminNorm).Scan(&adminUserID)
	adminSess := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, adminSess, adminUserID, time.Now().Add(1*time.Hour))
	adminToken, _ := tm.GenerateAccessToken(adminUserID, adminSess)

	assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader([]byte(fmt.Sprintf(`{"user_id":"%s"}`, adminUserID))))
	assignReq.Header.Set("Authorization", "Bearer "+superToken)
	assignReq.Header.Set("Content-Type", "application/json")
	wAssign := httptest.NewRecorder()
	router.ServeHTTP(wAssign, assignReq)

	cat, err := societyRepo.CreateCategory(ctx, socID, "Limits Category", nil, 1, adminUserID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// Subtest 1: Oversized request body (>12MB) rejected with FILE_TOO_LARGE
	t.Run("Oversized Multipart Request (>12MB)", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "Oversized Body")
		_ = w.WriteField("category_id", cat.ID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "oversized.pdf")
		// 13MB payload
		huge := make([]byte, 13*1024*1024)
		copy(huge, []byte("%PDF-1.4"))
		_, _ = fw.Write(huge)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("expected 400 or 413 for >12MB request body, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Error.Code != "FILE_TOO_LARGE" {
			t.Errorf("expected FILE_TOO_LARGE error code, got: %s", errResp.Error.Code)
		}
	})

	// Subtest 2: Truncated / Malformed multipart payload rejected with BAD_REQUEST
	t.Run("Malformed Truncated Multipart Payload", func(t *testing.T) {
		truncatedBody := "--sampleBoundary\r\nContent-Disposition: form-data; name=\"title\"\r\n\r\nIncomplete"
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), strings.NewReader(truncatedBody))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=sampleBoundary")
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for truncated multipart, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Error.Code != "BAD_REQUEST" {
			t.Errorf("expected BAD_REQUEST error code, got: %s", errResp.Error.Code)
		}
	})

	// Subtest 3: File at 10MB + 100 bytes (within 12MB total request body) rejected with FILE_TOO_LARGE
	t.Run("File Exceeding 10MB Original Limit", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "10MB+100 File")
		_ = w.WriteField("category_id", cat.ID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "too_large.pdf")
		oversized := make([]byte, society.MaxOriginalFileSize+100)
		copy(oversized, []byte("%PDF-1.4"))
		_, _ = fw.Write(oversized)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Error.Code != "FILE_TOO_LARGE" {
			t.Errorf("expected FILE_TOO_LARGE code, got: %s", errResp.Error.Code)
		}
	})

	// Subtest 4: File at 10MB exactly (PDF cannot optimize) rejected with FILE_EXCEEDS_ACCEPTED_LIMIT
	t.Run("File at 10MB Exactly Rejected By 5MB Accepted Limit", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("title", "10MB Exact File")
		_ = w.WriteField("category_id", cat.ID.String())
		_ = w.WriteField("classification", "SOCIETY_RESIDENTS")
		fw, _ := w.CreateFormFile("file", "exact10mb.pdf")
		exact10MB := make([]byte, society.MaxOriginalFileSize)
		copy(exact10MB, []byte("%PDF-1.4"))
		_, _ = fw.Write(exact10MB)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
		}
		var errResp struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
		if errResp.Error.Code != "FILE_EXCEEDS_ACCEPTED_LIMIT" {
			t.Errorf("expected FILE_EXCEEDS_ACCEPTED_LIMIT code, got: %s", errResp.Error.Code)
		}
	})

	// Clean up
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_categories WHERE society_id = $1`, socID)
}

func TestDocumentVersion_ConcurrencyAndIntegrity(t *testing.T) {
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

	sessionID := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, sessionID, superAdminRole.UserID, time.Now().Add(1*time.Hour))
	superToken, _ := tm.GenerateAccessToken(superAdminRole.UserID, sessionID)

	socName := fmt.Sprintf("Concurrency Test Society %d", time.Now().UnixNano())
	socParams := society.CreateSocietyParams{
		Name:         socName,
		AddressLine1: "Concurrent Lane",
		Locality:     "Aundh",
		City:         "Pune",
		State:        "Maharashtra",
		Pincode:      "411007",
	}
	socJSON, _ := json.Marshal(socParams)
	reqSoc := httptest.NewRequest(http.MethodPost, "/api/v1/platform/societies", bytes.NewReader(socJSON))
	reqSoc.Header.Set("Authorization", "Bearer "+superToken)
	reqSoc.Header.Set("Content-Type", "application/json")
	wSoc := httptest.NewRecorder()
	router.ServeHTTP(wSoc, reqSoc)

	var socResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSoc.Body.Bytes(), &socResp)
	socID := uuid.MustParse(socResp.Data.ID)

	actReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/activate", socID), nil)
	actReq.Header.Set("Authorization", "Bearer "+superToken)
	wAct := httptest.NewRecorder()
	router.ServeHTTP(wAct, actReq)

	adminPhone := fmt.Sprintf("+919%09d", (time.Now().UnixNano()/888)%1000000000)
	adminNorm, _ := auth.NormalizePhone(adminPhone)
	var adminUserID uuid.UUID
	_ = db.Pool().QueryRow(ctx, "INSERT INTO users (phone) VALUES ($1) RETURNING id", adminNorm).Scan(&adminUserID)
	adminSess := uuid.New()
	_, _ = db.Pool().Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, 'dummy', 'test-agent', '127.0.0.1', $3)
	`, adminSess, adminUserID, time.Now().Add(1*time.Hour))
	adminToken, _ := tm.GenerateAccessToken(adminUserID, adminSess)

	assignReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/platform/societies/%s/admins", socID), bytes.NewReader([]byte(fmt.Sprintf(`{"user_id":"%s"}`, adminUserID))))
	assignReq.Header.Set("Authorization", "Bearer "+superToken)
	assignReq.Header.Set("Content-Type", "application/json")
	wAssign := httptest.NewRecorder()
	router.ServeHTTP(wAssign, assignReq)

	cat, err := societyRepo.CreateCategory(ctx, socID, "Concurrency Cat", nil, 1, adminUserID, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	// 1. Create document (version 1)
	var b1 bytes.Buffer
	w1 := multipart.NewWriter(&b1)
	_ = w1.WriteField("title", "Concurrent Document")
	_ = w1.WriteField("category_id", cat.ID.String())
	_ = w1.WriteField("classification", "SOCIETY_RESIDENTS")
	fw1, _ := w1.CreateFormFile("file", "v1.pdf")
	v1Content := []byte("%PDF-1.4 initial version 1 payload")
	_, _ = fw1.Write(v1Content)
	w1.Close()

	reqDoc := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents", socID), &b1)
	reqDoc.Header.Set("Content-Type", w1.FormDataContentType())
	reqDoc.Header.Set("Authorization", "Bearer "+adminToken)
	recDoc := httptest.NewRecorder()
	router.ServeHTTP(recDoc, reqDoc)
	if recDoc.Code != http.StatusCreated {
		t.Fatalf("failed to create document: %d: %s", recDoc.Code, recDoc.Body.String())
	}

	var createResp struct {
		Data struct {
			Document struct {
				ID string `json:"id"`
			} `json:"document"`
		} `json:"data"`
	}
	_ = json.Unmarshal(recDoc.Body.Bytes(), &createResp)
	docID := uuid.MustParse(createResp.Data.Document.ID)

	// Fetch initial version 1 record to check preservation later
	var v1Record society.DocumentVersion
	err = db.Pool().QueryRow(ctx, `
		SELECT id, version_number, file_name, file_hash
		FROM document_versions WHERE document_id = $1 AND version_number = 1
	`, docID).Scan(&v1Record.ID, &v1Record.VersionNumber, &v1Record.FileName, &v1Record.FileHash)
	if err != nil {
		t.Fatalf("failed to query v1 record: %v", err)
	}

	// 2. Perform 4 concurrent uploads (v2, v3, v4, v5) simultaneously
	concurrency := 4
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)

	for i := 1; i <= concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			_ = w.WriteField("change_summary", fmt.Sprintf("Concurrent revision %d", workerID))
			fw, _ := w.CreateFormFile("file", fmt.Sprintf("concurrent_v%d.pdf", workerID))
			_, _ = fw.Write([]byte(fmt.Sprintf("%%PDF-1.4 concurrent worker %d content %d", workerID, time.Now().UnixNano())))
			w.Close()

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/documents/%s/versions", socID, docID), &b)
			req.Header.Set("Content-Type", w.FormDataContentType())
			req.Header.Set("Authorization", "Bearer "+adminToken)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				errCh <- fmt.Errorf("worker %d failed with status %d: %s", workerID, rec.Code, rec.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent upload failed: %v", err)
	}

	// 3. Verify that all version numbers are strictly distinct and sequential [1, 2, 3, 4, 5]
	rows, err := db.Pool().Query(ctx, `
		SELECT version_number, file_name, file_hash
		FROM document_versions
		WHERE document_id = $1
		ORDER BY version_number ASC
	`, docID)
	if err != nil {
		t.Fatalf("failed to query versions: %v", err)
	}
	defer rows.Close()

	var versions []struct {
		Number int
		Name   string
		Hash   string
	}
	for rows.Next() {
		var v struct {
			Number int
			Name   string
			Hash   string
		}
		if err := rows.Scan(&v.Number, &v.Name, &v.Hash); err != nil {
			t.Fatalf("scan version failed: %v", err)
		}
		versions = append(versions, v)
	}

	if len(versions) != concurrency+1 {
		t.Fatalf("expected %d versions, got %d", concurrency+1, len(versions))
	}

	for i, v := range versions {
		expectedNum := i + 1
		if v.Number != expectedNum {
			t.Errorf("version index %d: expected version_number %d, got %d", i, expectedNum, v.Number)
		}
	}

	// 4. Verify version 1 remains strictly unchanged (immutability check)
	if versions[0].Number != 1 || versions[0].Hash != v1Record.FileHash {
		t.Errorf("version 1 was mutated! Got hash %s, expected %s", versions[0].Hash, v1Record.FileHash)
	}

	// Clean up
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_versions WHERE document_id = $1`, docID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM society_documents WHERE id = $1`, docID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_categories WHERE society_id = $1`, socID)
}
