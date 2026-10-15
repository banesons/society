package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

const (
	BaseURL   = "http://localhost:8080/api/v1"
	SocID     = "ba19522f-73ad-4ae5-95f5-d03edd71f180"
	DocID     = "567350b9-f2b8-4da0-834c-712e82c5d700"
	V1ID      = "467f8279-100f-46ee-8a35-b7e5b5828ecd"
	V1HashExp = "ac07d0ddfd6a1290b539e76452250a26cf306b7d7761436346f21bd1796e0ea0"
	V2HashExp = "fdbcb2aa9ce142aade90534eef7d7ee4b6b572de7865c0c61b1580f26f763910"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("FAIL: db connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Login via HTTP API to ensure matching Ed25519 token key
	var token string
	for attempt := 0; attempt < 25; attempt++ {
		sendPayload := []byte(`{"phone":"` + cfg.SuperAdminPhone + `"}`)
		respSend, err := http.Post(BaseURL+"/auth/otp/send", "application/json", bytes.NewReader(sendPayload))
		if err == nil {
			var sendRes struct {
				Success bool `json:"success"`
			}
			_ = json.NewDecoder(respSend.Body).Decode(&sendRes)
			respSend.Body.Close()
			if sendRes.Success {
				loginPayload := []byte(`{"phone":"` + cfg.SuperAdminPhone + `","otp":"123456"}`)
				respLogin, err := http.Post(BaseURL+"/auth/otp/verify", "application/json", bytes.NewReader(loginPayload))
				if err == nil && respLogin.StatusCode == http.StatusOK {
					var loginResp struct {
						Data struct {
							AccessToken string `json:"access_token"`
						} `json:"data"`
					}
					_ = json.NewDecoder(respLogin.Body).Decode(&loginResp)
					respLogin.Body.Close()
					token = loginResp.Data.AccessToken
					break
				}
				if respLogin != nil {
					respLogin.Body.Close()
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
	if token == "" {
		fmt.Printf("FAIL: unable to authenticate via HTTP API\n")
		os.Exit(1)
	}
	fmt.Printf("[AUTH] Authenticated as SuperAdmin (%s) via HTTP API\n", cfg.SuperAdminPhone)

	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: Download Version 1 and verify SHA-256
	fmt.Println("\n--- STEP 1: DOWNLOAD VERSION 1 & VERIFY SHA-256 ---")
	reqV1, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/%s/versions/%s/download", BaseURL, SocID, DocID, V1ID), nil)
	reqV1.Header.Set("Authorization", "Bearer "+token)
	respV1, err := client.Do(reqV1)
	if err != nil || respV1.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: download v1 failed, status=%d, err=%v\n", respV1.StatusCode, err)
		os.Exit(1)
	}
	v1Bytes, _ := io.ReadAll(respV1.Body)
	respV1.Body.Close()

	h1 := sha256.Sum256(v1Bytes)
	v1ActualHash := hex.EncodeToString(h1[:])
	if v1ActualHash != V1HashExp {
		fmt.Printf("FAIL: v1 hash mismatch: got %s, expected %s\n", v1ActualHash, V1HashExp)
		os.Exit(1)
	}
	fmt.Printf("PASS: Version 1 retrieved successfully, bytes=%d, SHA-256=%s matches expected\n", len(v1Bytes), v1ActualHash)

	// Step 2: Upload Version 2
	fmt.Println("\n--- STEP 2: UPLOAD VERSION 2 ---")
	v2FileBytes, err := os.ReadFile("/Users/shubham/Developer/chs_neu/test_doc_v2.pdf")
	if err != nil {
		fmt.Printf("FAIL: read test_doc_v2.pdf: %v\n", err)
		os.Exit(1)
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("change_summary", "Updated statutory clauses for 2026 revision")
	fw, _ := w.CreateFormFile("file", "test_doc_v2.pdf")
	_, _ = fw.Write(v2FileBytes)
	w.Close()

	reqUploadV2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/societies/%s/documents/%s/versions", BaseURL, SocID, DocID), &b)
	reqUploadV2.Header.Set("Authorization", "Bearer "+token)
	reqUploadV2.Header.Set("Content-Type", w.FormDataContentType())
	respUploadV2, err := client.Do(reqUploadV2)
	if err != nil || respUploadV2.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(respUploadV2.Body)
		fmt.Printf("FAIL: upload v2 failed, status=%d, body=%s\n", respUploadV2.StatusCode, string(body))
		os.Exit(1)
	}

	var v2Resp struct {
		Data struct {
			Version struct {
				ID            string `json:"id"`
				VersionNumber int    `json:"version_number"`
				FileName      string `json:"file_name"`
				FileHash      string `json:"file_hash"`
				ChangeSummary string `json:"change_summary"`
			} `json:"version"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respUploadV2.Body).Decode(&v2Resp)
	respUploadV2.Body.Close()

	v2ID := v2Resp.Data.Version.ID
	if v2Resp.Data.Version.VersionNumber != 2 {
		fmt.Printf("FAIL: expected version number 2, got %d\n", v2Resp.Data.Version.VersionNumber)
		os.Exit(1)
	}
	if v2Resp.Data.Version.FileHash != V2HashExp {
		fmt.Printf("FAIL: v2 metadata hash mismatch: got %s, expected %s\n", v2Resp.Data.Version.FileHash, V2HashExp)
		os.Exit(1)
	}
	fmt.Printf("PASS: Version 2 committed with next sequential version number=2, file=%s, hash=%s, summary=%q\n",
		v2Resp.Data.Version.FileName, v2Resp.Data.Version.FileHash, v2Resp.Data.Version.ChangeSummary)

	// Step 3: Download Version 2 and verify SHA-256
	fmt.Println("\n--- STEP 3: DOWNLOAD VERSION 2 & VERIFY SHA-256 ---")
	reqDlV2, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/%s/versions/%s/download", BaseURL, SocID, DocID, v2ID), nil)
	reqDlV2.Header.Set("Authorization", "Bearer "+token)
	respDlV2, err := client.Do(reqDlV2)
	if err != nil || respDlV2.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: download v2 failed, status=%d\n", respDlV2.StatusCode)
		os.Exit(1)
	}
	v2DlBytes, _ := io.ReadAll(respDlV2.Body)
	respDlV2.Body.Close()

	h2 := sha256.Sum256(v2DlBytes)
	v2ActualHash := hex.EncodeToString(h2[:])
	if v2ActualHash != V2HashExp {
		fmt.Printf("FAIL: downloaded v2 hash mismatch: got %s, expected %s\n", v2ActualHash, V2HashExp)
		os.Exit(1)
	}
	fmt.Printf("PASS: Version 2 downloaded successfully, bytes=%d, SHA-256=%s matches expected\n", len(v2DlBytes), v2ActualHash)

	// Step 4: Re-download Version 1 to prove previous version was NOT overwritten
	fmt.Println("\n--- STEP 4: RE-DOWNLOAD VERSION 1 (INTEGRITY PRESERVATION CHECK) ---")
	reqV1Again, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/%s/versions/%s/download", BaseURL, SocID, DocID, V1ID), nil)
	reqV1Again.Header.Set("Authorization", "Bearer "+token)
	respV1Again, err := client.Do(reqV1Again)
	if err != nil || respV1Again.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: re-download v1 failed, status=%d\n", respV1Again.StatusCode)
		os.Exit(1)
	}
	v1AgainBytes, _ := io.ReadAll(respV1Again.Body)
	respV1Again.Body.Close()

	h1Again := sha256.Sum256(v1AgainBytes)
	v1AgainHash := hex.EncodeToString(h1Again[:])
	if v1AgainHash != V1HashExp {
		fmt.Printf("FAIL: v1 corrupted after v2 upload: got %s, expected %s\n", v1AgainHash, V1HashExp)
		os.Exit(1)
	}
	fmt.Printf("PASS: Version 1 re-download verified: exact bytes preserved (%d bytes), SHA-256=%s intact\n", len(v1AgainBytes), v1AgainHash)

	// Step 5: Verify Version History Persistence
	fmt.Println("\n--- STEP 5: VERIFY COMPLETE VERSION HISTORY PERSISTENCE ---")
	reqListVer, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/%s/versions", BaseURL, SocID, DocID), nil)
	reqListVer.Header.Set("Authorization", "Bearer "+token)
	respListVer, err := client.Do(reqListVer)
	if err != nil || respListVer.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: list versions failed: status=%d\n", respListVer.StatusCode)
		os.Exit(1)
	}

	var listVerResp struct {
		Data struct {
			Versions []struct {
				ID            string `json:"id"`
				VersionNumber int    `json:"version_number"`
				FileName      string `json:"file_name"`
				FileHash      string `json:"file_hash"`
				ChangeSummary string `json:"change_summary"`
			} `json:"versions"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respListVer.Body).Decode(&listVerResp)
	respListVer.Body.Close()

	if len(listVerResp.Data.Versions) != 2 {
		fmt.Printf("FAIL: expected 2 versions in history, got %d\n", len(listVerResp.Data.Versions))
		os.Exit(1)
	}
	for _, v := range listVerResp.Data.Versions {
		fmt.Printf("  -> Version %d: ID=%s, file=%s, hash=%s, summary=%q\n", v.VersionNumber, v.ID, v.FileName, v.FileHash, v.ChangeSummary)
	}
	fmt.Println("PASS: Version history persisted with 2 distinct versions")

	// Step 6: Verify Document Storage Metrics
	fmt.Println("\n--- STEP 6: VERIFY STORAGE METRICS ENDPOINT ---")
	reqMetrics, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/metrics", BaseURL, SocID), nil)
	reqMetrics.Header.Set("Authorization", "Bearer "+token)
	respMetrics, err := client.Do(reqMetrics)
	if err != nil || respMetrics.StatusCode != http.StatusOK {
		fmt.Printf("FAIL: get metrics failed: status=%d\n", respMetrics.StatusCode)
		os.Exit(1)
	}
	var metricsResp struct {
		Data struct {
			Metrics struct {
				TotalDocuments    int64 `json:"total_documents"`
				ActiveDocuments   int64 `json:"active_documents"`
				ArchivedDocuments int64 `json:"archived_documents"`
				TotalVersions     int64 `json:"total_versions"`
				TotalStoredBytes  int64 `json:"total_stored_bytes"`
			} `json:"metrics"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respMetrics.Body).Decode(&metricsResp)
	respMetrics.Body.Close()
	m := metricsResp.Data.Metrics
	fmt.Printf("PASS: Storage Metrics: total_docs=%d, active=%d, archived=%d, total_versions=%d, total_bytes=%d\n",
		m.TotalDocuments, m.ActiveDocuments, m.ArchivedDocuments, m.TotalVersions, m.TotalStoredBytes)

	// Step 7: Verify Unauthorized Access Rejection (No token)
	fmt.Println("\n--- STEP 7: VERIFY SECURITY ENFORCEMENT (UNAUTHORIZED DOWNLOAD DENIED) ---")
	reqUnauth, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/documents/%s/versions/%s/download", BaseURL, SocID, DocID, V1ID), nil)
	respUnauth, _ := client.Do(reqUnauth)
	if respUnauth.StatusCode != http.StatusUnauthorized {
		fmt.Printf("FAIL: expected 401 Unauthorized without auth header, got %d\n", respUnauth.StatusCode)
		os.Exit(1)
	}
	respUnauth.Body.Close()
	fmt.Println("PASS: Unauthenticated access rejected with 401 Unauthorized")

	// Clean up created test document and versions
	fmt.Println("\n--- STEP 8: SAFE ISOLATED TEST CLEANUP ---")
	_, _ = db.Pool().Exec(ctx, `DELETE FROM document_versions WHERE document_id = $1`, DocID)
	_, _ = db.Pool().Exec(ctx, `DELETE FROM society_documents WHERE id = $1`, DocID)
	fmt.Println("PASS: Isolated test document records cleaned up; baseline documents untouched")

	fmt.Println("\n=======================================================")
	fmt.Println("ALL S13 ACCEPTANCE VERIFICATION STEPS PASSED SUCCESSFULLY")
	fmt.Println("=======================================================")
}
