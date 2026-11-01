package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"chs/backend/internal/database"
	"chs/backend/internal/society"
)

func setupFinalizedBillingFixture(t *testing.T) (
	*database.DB,
	*society.Repository,
	http.Handler,
	uuid.UUID,
	string,
	uuid.UUID,
	uuid.UUID,
	uuid.UUID,
	func(uuid.UUID) string,
) {
	t.Helper()
	_, db, router, _, societyRepo, socID, superToken, adminUserID, generateToken := setupBillingTestEnv(t)
	ctx := context.Background()
	uniqueTime := time.Now().UnixNano()

	// 1. Generate Building and Units
	bldReq := map[string]interface{}{
		"name":          "Chillon Tower",
		"building_code": fmt.Sprintf("CT%d", uniqueTime%10000),
		"total_floors":  2,
	}
	bldJSON, _ := json.Marshal(bldReq)
	reqBld := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/buildings", socID), bytes.NewReader(bldJSON))
	reqBld.Header.Set("Authorization", "Bearer "+superToken)
	reqBld.Header.Set("Content-Type", "application/json")
	wBld := httptest.NewRecorder()
	router.ServeHTTP(wBld, reqBld)
	if wBld.Code != http.StatusCreated {
		t.Fatalf("expected 201 for building, got %d: %s", wBld.Code, wBld.Body.String())
	}
	bldData := parseTestData(wBld.Body.Bytes())
	bldID := bldData["id"].(string)

	genReq := society.GenerateUnitsParams{
		BuildingID:         uuid.MustParse(bldID),
		StartingFloor:      1,
		TotalFloors:        1,
		UnitsPerFloor:      2,
		StartingUnitNumber: 101,
		Prefix:             fmt.Sprintf("P%d", uniqueTime%1000),
	}
	genJSON, _ := json.Marshal(genReq)
	reqGenU := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/units/generate", socID), bytes.NewReader(genJSON))
	reqGenU.Header.Set("Authorization", "Bearer "+superToken)
	reqGenU.Header.Set("Content-Type", "application/json")
	wGenU := httptest.NewRecorder()
	router.ServeHTTP(wGenU, reqGenU)
	if wGenU.Code != http.StatusCreated {
		t.Fatalf("expected 201 for unit generation, got %d: %s", wGenU.Code, wGenU.Body.String())
	}

	reqListU := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/units?limit=10", socID), nil)
	reqListU.Header.Set("Authorization", "Bearer "+superToken)
	wListU := httptest.NewRecorder()
	router.ServeHTTP(wListU, reqListU)
	var listResp struct {
		Data struct {
			Units []struct {
				ID string `json:"id"`
			} `json:"units"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wListU.Body.Bytes(), &listResp); err != nil || len(listResp.Data.Units) < 2 {
		t.Fatalf("failed to list generated units: %v", err)
	}
	unit1ID := uuid.MustParse(listResp.Data.Units[0].ID)
	_ = uuid.MustParse(listResp.Data.Units[1].ID)

	// 2. Create Charge Heads
	sac1 := "999511"
	_, err := societyRepo.CreateChargeHead(ctx, socID, society.CreateChargeHeadParams{
		Name:              "General Society Maintenance",
		Code:              "GEN_MAINT",
		SACCode:           &sac1,
		TaxClassification: "TAXABLE",
		DefaultTaxRate:    18.0,
		CalculationMethod: "AREA",
		DisplayOrder:      1,
		IsActive:          true,
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create charge head: %v", err)
	}

	// 3. Create & Publish Template with Watermark
	tpl, err := societyRepo.CreateBillTemplate(ctx, socID, society.CreateBillTemplateParams{
		Name:             fmt.Sprintf("PDF Test Template %d", uniqueTime),
		BillingFrequency: "MONTHLY",
		WatermarkEnabled: true,
	}, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	pubTpl, err := societyRepo.PublishBillTemplate(ctx, socID, tpl.ID, adminUserID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("failed to publish template: %v", err)
	}

	// 4. Create Billing Period
	validPeriod := map[string]interface{}{
		"period_name":  fmt.Sprintf("Nov 2026 PDF %d", uniqueTime%10000),
		"period_start": "2026-11-01",
		"period_end":   "2026-11-30",
		"due_date":     "2026-12-10",
	}
	valPBody, _ := json.Marshal(validPeriod)
	reqValP := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/maintenance/periods", socID), bytes.NewReader(valPBody))
	reqValP.Header.Set("Authorization", "Bearer "+superToken)
	reqValP.Header.Set("Content-Type", "application/json")
	recValP := httptest.NewRecorder()
	router.ServeHTTP(recValP, reqValP)
	if recValP.Code != http.StatusCreated {
		t.Fatalf("expected 201 for period, got %d: %s", recValP.Code, recValP.Body.String())
	}
	pData := parseTestData(recValP.Body.Bytes())
	periodMap := pData["period"].(map[string]interface{})
	periodID := periodMap["id"].(string)

	// 5. Create Billing Batch & Finalize
	batchReq := map[string]interface{}{
		"billing_period_id": periodID,
		"template_id":       pubTpl.TemplateID.String(),
		"batch_name":        "PDF Test Billing Batch",
	}
	batchJSON, _ := json.Marshal(batchReq)
	reqBatch := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches", socID), bytes.NewReader(batchJSON))
	reqBatch.Header.Set("Authorization", "Bearer "+superToken)
	reqBatch.Header.Set("Content-Type", "application/json")
	wBatch := httptest.NewRecorder()
	router.ServeHTTP(wBatch, reqBatch)
	if wBatch.Code != http.StatusCreated {
		t.Fatalf("expected 201 for batch, got %d: %s", wBatch.Code, wBatch.Body.String())
	}
	bData := parseTestData(wBatch.Body.Bytes())
	batchID := uuid.MustParse(bData["id"].(string))

	// Generate drafts
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/generate", socID, batchID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch draft generate, got %d: %s", wGen.Code, wGen.Body.String())
	}

	// Finalize batch
	reqFin := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/finalize", socID, batchID), nil)
	reqFin.Header.Set("Authorization", "Bearer "+superToken)
	wFin := httptest.NewRecorder()
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != http.StatusOK {
		t.Fatalf("expected 200 for batch finalize, got %d: %s", wFin.Code, wFin.Body.String())
	}

	// Fetch first bill ID
	var billID uuid.UUID
	err = db.Pool().QueryRow(ctx, `SELECT id FROM maintenance_bills WHERE batch_id = $1 ORDER BY bill_number ASC LIMIT 1`, batchID).Scan(&billID)
	if err != nil {
		t.Fatalf("failed to query bill id: %v", err)
	}

	return db, societyRepo, router, socID, superToken, batchID, billID, unit1ID, generateToken
}

func TestConfigurableBillingPDF_SingleBillGeneration(t *testing.T) {
	_, _, router, socID, superToken, _, billID, _, _ := setupFinalizedBillingFixture(t)

	// 1. POST generate PDF
	reqPost := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/bills/%s/pdf", socID, billID), nil)
	reqPost.Header.Set("Authorization", "Bearer "+superToken)
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for bill PDF generation, got %d: %s", wPost.Code, wPost.Body.String())
	}

	var postResp struct {
		Data struct {
			ID             string `json:"id"`
			FileName       string `json:"file_name"`
			SHA256Checksum string `json:"sha256_checksum"`
			FileSizeBytes  int64  `json:"file_size_bytes"`
			MimeType       string `json:"mime_type"`
			HasWatermark   bool   `json:"has_watermark"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("failed to parse artifact response: %v", err)
	}

	if postResp.Data.FileName == "" {
		t.Errorf("expected non-empty file_name")
	}
	if len(postResp.Data.SHA256Checksum) != 64 {
		t.Errorf("expected 64-character SHA-256 checksum, got %q", postResp.Data.SHA256Checksum)
	}
	if postResp.Data.FileSizeBytes <= 0 {
		t.Errorf("expected positive file size, got %d", postResp.Data.FileSizeBytes)
	}
	if !postResp.Data.HasWatermark {
		t.Errorf("expected has_watermark to be true")
	}
	if postResp.Data.MimeType != "application/pdf" {
		t.Errorf("expected application/pdf, got %s", postResp.Data.MimeType)
	}

	// 2. GET download PDF
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/bills/%s/pdf", socID, billID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+superToken)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for bill PDF download, got %d: %s", wGet.Code, wGet.Body.String())
	}

	if ct := wGet.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("expected Content-Type application/pdf, got %s", ct)
	}

	pdfBody := wGet.Body.Bytes()
	if !bytes.HasPrefix(pdfBody, []byte("%PDF-")) {
		t.Fatalf("expected PDF binary header %%PDF-, got %q", string(pdfBody[:min(len(pdfBody), 10)]))
	}

	// Verify SHA-256 match
	hash := sha256.Sum256(pdfBody)
	actualChecksum := hex.EncodeToString(hash[:])
	if actualChecksum != postResp.Data.SHA256Checksum {
		t.Errorf("checksum mismatch: expected %s, got %s", postResp.Data.SHA256Checksum, actualChecksum)
	}
}

func TestConfigurableBillingPDF_BatchZipPackaging(t *testing.T) {
	_, _, router, socID, superToken, batchID, _, _, _ := setupFinalizedBillingFixture(t)

	// 1. POST generate batch ZIP
	reqPost := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/pdf", socID, batchID), nil)
	reqPost.Header.Set("Authorization", "Bearer "+superToken)
	wPost := httptest.NewRecorder()
	router.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for batch PDF generation, got %d: %s", wPost.Code, wPost.Body.String())
	}

	// 2. GET download batch ZIP
	reqGet := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/batches/%s/pdf", socID, batchID), nil)
	reqGet.Header.Set("Authorization", "Bearer "+superToken)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for batch ZIP download, got %d: %s", wGet.Code, wGet.Body.String())
	}

	if ct := wGet.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("expected Content-Type application/zip, got %s", ct)
	}

	zipBytes := wGet.Body.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("failed to parse zip archive: %v", err)
	}

	if len(zipReader.File) != 2 {
		t.Fatalf("expected 2 bills in zip archive, got %d", len(zipReader.File))
	}

	for _, file := range zipReader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open file %s in zip: %v", file.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read content of %s: %v", file.Name, err)
		}
		if !bytes.HasPrefix(content, []byte("%PDF-")) {
			t.Errorf("file %s in zip is not a valid PDF", file.Name)
		}
	}
}

func TestConfigurableBillingPDF_ArtifactListingAndDownload(t *testing.T) {
	_, _, router, socID, superToken, _, billID, _, _ := setupFinalizedBillingFixture(t)

	// Generate PDF artifact
	reqGen := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/societies/%s/billing/bills/%s/pdf", socID, billID), nil)
	reqGen.Header.Set("Authorization", "Bearer "+superToken)
	wGen := httptest.NewRecorder()
	router.ServeHTTP(wGen, reqGen)
	if wGen.Code != http.StatusOK {
		t.Fatalf("failed to generate artifact: %d", wGen.Code)
	}

	// 1. List artifacts for bill
	reqList := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/bills/%s/artifacts", socID, billID), nil)
	reqList.Header.Set("Authorization", "Bearer "+superToken)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 for listing artifacts, got %d: %s", wList.Code, wList.Body.String())
	}

	var listResp struct {
		Data struct {
			Artifacts []struct {
				ID string `json:"id"`
			} `json:"artifacts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil || len(listResp.Data.Artifacts) == 0 {
		t.Fatalf("expected at least 1 artifact in listing: %v", err)
	}
	artID := listResp.Data.Artifacts[0].ID

	// 2. Get artifact details
	reqDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/artifacts/%s", socID, artID), nil)
	reqDetail.Header.Set("Authorization", "Bearer "+superToken)
	wDetail := httptest.NewRecorder()
	router.ServeHTTP(wDetail, reqDetail)
	if wDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 for artifact detail, got %d", wDetail.Code)
	}

	// 3. Download artifact via ?download=true
	reqDl := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/artifacts/%s?download=true", socID, artID), nil)
	reqDl.Header.Set("Authorization", "Bearer "+superToken)
	wDl := httptest.NewRecorder()
	router.ServeHTTP(wDl, reqDl)
	if wDl.Code != http.StatusOK {
		t.Fatalf("expected 200 for artifact download, got %d", wDl.Code)
	}
	if !bytes.HasPrefix(wDl.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("downloaded artifact is not a valid PDF")
	}
}

func TestConfigurableBillingPDF_AuthorizationSecurity(t *testing.T) {
	db, _, router, socID, _, _, billID, _, generateToken := setupFinalizedBillingFixture(t)
	ctx := context.Background()

	// Create another user not associated with the bill
	unauthorizedUserID := uuid.New()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO users (id, phone, status)
		VALUES ($1, $2, 'ACTIVE')
	`, unauthorizedUserID, fmt.Sprintf("+9197%08d", time.Now().UnixNano()%100000000))
	if err != nil {
		t.Fatalf("failed to insert unauthorized user: %v", err)
	}
	unauthorizedToken := generateToken(unauthorizedUserID)

	// Attempt download of bill PDF
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/societies/%s/billing/bills/%s/pdf", socID, billID), nil)
	req.Header.Set("Authorization", "Bearer "+unauthorizedToken)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unauthorized bill download, got %d: %s", w.Code, w.Body.String())
	}
}
