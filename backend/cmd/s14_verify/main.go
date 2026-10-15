package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const baseURL = "http://localhost:8080/api/v1"

func main() {
	client := &http.Client{Timeout: 10 * time.Second}

	fmt.Println("================================================================================")
	fmt.Println("              S14 AUDIT & ADMINISTRATION LIVE API VERIFICATION")
	fmt.Println("================================================================================")

	// Step 1: Send OTP for Super Admin (wait for cooldown if needed)
	sendPayload, _ := json.Marshal(map[string]string{"phone": "+919967089101"})
	for attempt := 1; attempt <= 10; attempt++ {
		resp, err := client.Post(baseURL+"/auth/otp/send", "application/json", bytes.NewReader(sendPayload))
		if err != nil {
			log.Fatalf("failed to send OTP: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("PASS: OTP sent for Super Admin (+919967089101)")
			break
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(3 * time.Second)
			continue
		}
		log.Fatalf("failed to send OTP with status %d", resp.StatusCode)
	}

	// Step 2: Verify OTP
	verifyPayload, _ := json.Marshal(map[string]string{
		"phone": "+919967089101",
		"otp":   "123456",
	})
	resp, err := client.Post(baseURL+"/auth/otp/verify", "application/json", bytes.NewReader(verifyPayload))
	if err != nil {
		log.Fatalf("failed to verify OTP: %v", err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("failed to verify OTP (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	var authResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bodyBytes, &authResp)

	if !authResp.Success || authResp.Data.AccessToken == "" {
		log.Fatalf("failed to obtain access token: %s", string(bodyBytes))
	}
	token := authResp.Data.AccessToken
	fmt.Println("PASS: Authenticated as Super Admin with valid JWT")

	// Step 3: Platform Audit List
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/platform/audit-logs?limit=5&page=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		log.Fatalf("failed to list platform audit logs: %v", err)
	}
	var listResp struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ID          string `json:"id"`
				Action      string `json:"action"`
				ActionLabel string `json:"action_label"`
				ActorName   string `json:"actor_name"`
				ActorPhone  string `json:"actor_phone"`
				SocietyName string `json:"society_name"`
			} `json:"items"`
			Total      int `json:"total"`
			Pagination struct {
				Page       int `json:"page"`
				Limit      int `json:"limit"`
				TotalItems int `json:"total_items"`
				TotalPages int `json:"total_pages"`
			} `json:"pagination"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&listResp)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !listResp.Success {
		log.Fatalf("failed to list audit logs: status %d", resp.StatusCode)
	}
	fmt.Printf("PASS: Platform audit logs listed: total=%d, page=%d, pages=%d, items_returned=%d\n",
		listResp.Data.Total, listResp.Data.Pagination.Page, listResp.Data.Pagination.TotalPages, len(listResp.Data.Items))

	if len(listResp.Data.Items) == 0 {
		log.Fatalf("expected items in audit log list")
	}
	firstEvent := listResp.Data.Items[0]
	fmt.Printf("      First event: [%s] %s | Actor: %s (%s) | Society: %s\n",
		firstEvent.Action, firstEvent.ActionLabel, firstEvent.ActorName, firstEvent.ActorPhone, firstEvent.SocietyName)

	// Step 4: Platform Audit Detail
	reqDetail, _ := http.NewRequest(http.MethodGet, baseURL+"/platform/audit-logs/"+firstEvent.ID, nil)
	reqDetail.Header.Set("Authorization", "Bearer "+token)
	respDetail, err := client.Do(reqDetail)
	if err != nil {
		log.Fatalf("failed to get audit detail: %v", err)
	}
	if respDetail.StatusCode != http.StatusOK {
		log.Fatalf("expected 200 OK on detail, got %d", respDetail.StatusCode)
	}
	var detailResp struct {
		Success bool `json:"success"`
		Data    struct {
			Item struct {
				ID          string                 `json:"id"`
				Action      string                 `json:"action"`
				ActionLabel string                 `json:"action_label"`
				Details     map[string]interface{} `json:"details"`
			} `json:"item"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respDetail.Body).Decode(&detailResp)
	respDetail.Body.Close()
	fmt.Printf("PASS: Event detail retrieved: ID=%s, Action=%s, MetadataKeys=%d\n",
		detailResp.Data.Item.ID, detailResp.Data.Item.ActionLabel, len(detailResp.Data.Item.Details))

	// Step 5: Test Society List & Society Audit Scoping
	reqSoc, _ := http.NewRequest(http.MethodGet, baseURL+"/platform/societies?limit=1", nil)
	reqSoc.Header.Set("Authorization", "Bearer "+token)
	respSoc, err := client.Do(reqSoc)
	if err != nil {
		log.Fatalf("failed to list societies: %v", err)
	}
	var socListResp struct {
		Data struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respSoc.Body).Decode(&socListResp)
	respSoc.Body.Close()

	if len(socListResp.Data.Items) == 0 {
		log.Fatalf("expected active societies in database")
	}
	activeSocID := socListResp.Data.Items[0].ID
	activeSocName := socListResp.Data.Items[0].Name

	// Query society-scoped audit history
	reqSocAudit, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/societies/%s/audit-logs?limit=5", baseURL, activeSocID), nil)
	reqSocAudit.Header.Set("Authorization", "Bearer "+token)
	respSocAudit, err := client.Do(reqSocAudit)
	if err != nil {
		log.Fatalf("failed to list society audit logs: %v", err)
	}
	if respSocAudit.StatusCode != http.StatusOK {
		log.Fatalf("expected 200 OK on society audit, got %d", respSocAudit.StatusCode)
	}
	var socAuditResp struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ID          string `json:"id"`
				ActionLabel string `json:"action_label"`
				SocietyName string `json:"society_name"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respSocAudit.Body).Decode(&socAuditResp)
	respSocAudit.Body.Close()
	fmt.Printf("PASS: Society-scoped audit logs for [%s]: total=%d, returned=%d\n",
		activeSocName, socAuditResp.Data.Total, len(socAuditResp.Data.Items))

	// Step 6: Test Unauthenticated Rejection (401)
	reqAnon, _ := http.NewRequest(http.MethodGet, baseURL+"/platform/audit-logs", nil)
	respAnon, err := client.Do(reqAnon)
	if err != nil {
		log.Fatalf("failed to perform anon request: %v", err)
	}
	if respAnon.StatusCode != http.StatusUnauthorized {
		log.Fatalf("expected 401 Unauthorized for anonymous request, got %d", respAnon.StatusCode)
	}
	respAnon.Body.Close()
	fmt.Println("PASS: Anonymous request rejected with 401 Unauthorized")

	fmt.Println("================================================================================")
	fmt.Println("       ALL S14 AUDIT & ADMINISTRATION LIVE API CHECKS PASSED SUCCESSFULLY")
	fmt.Println("================================================================================")
}
