package platform_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/platform"

	"github.com/google/uuid"
)

func TestSanitizeAuditDetails_Redaction(t *testing.T) {
	input := map[string]interface{}{
		"title":            "Official AGM Notice",
		"password":         "supersecret123",
		"token":            "eyJhbGciOi...",
		"access_token":     "token_abc123",
		"Refresh_Token":    "ref_987",
		"auth_header":      "Bearer secret_jwt",
		"Authorization":    "Bearer xyza",
		"service_role_key": "supabase_srv_key",
		"otp":              "123456",
		"pin":              "4321",
		"signed_url":       "https://bucket.storage.supabase.co/doc?token=xyz",
		"phone":            "+919820000101",
		"nested": map[string]interface{}{
			"secret":      "nested_secret_value",
			"category":    "legal",
			"inner_phone": "+919876543210",
		},
		"items": []interface{}{
			map[string]interface{}{
				"item_name": "Valid Item",
				"password":  "item_pass",
			},
			"token_in_slice",
			"safe_string",
		},
	}

	sanitized := platform.SanitizeAuditDetails(input)

	// Verify preserved safe fields
	if sanitized["title"] != "Official AGM Notice" {
		t.Errorf("expected title to be preserved, got %v", sanitized["title"])
	}

	// Verify redacted fields
	redactedKeys := []string{
		"password", "token", "access_token", "Refresh_Token",
		"auth_header", "Authorization", "service_role_key",
		"otp", "pin", "signed_url",
	}
	for _, key := range redactedKeys {
		if sanitized[key] != "[REDACTED]" {
			t.Errorf("expected %s to be [REDACTED], got %v", key, sanitized[key])
		}
	}

	// Verify phone masking
	if sanitized["phone"] != "+91 98*** ***01" {
		t.Errorf("expected masked phone +91 98*** ***01, got %v", sanitized["phone"])
	}

	// Verify nested map redaction
	nested, ok := sanitized["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested to be map[string]interface{}")
	}
	if nested["secret"] != "[REDACTED]" {
		t.Errorf("expected nested.secret to be [REDACTED], got %v", nested["secret"])
	}
	if nested["category"] != "legal" {
		t.Errorf("expected nested.category to be legal, got %v", nested["category"])
	}

	// Verify slice redaction
	items, ok := sanitized["items"].([]interface{})
	if !ok {
		t.Fatalf("expected items to be []interface{}")
	}
	itemMap, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected items[0] to be map[string]interface{}")
	}
	if itemMap["password"] != "[REDACTED]" {
		t.Errorf("expected items[0].password to be [REDACTED], got %v", itemMap["password"])
	}
	if itemMap["item_name"] != "Valid Item" {
		t.Errorf("expected items[0].item_name to be Valid Item, got %v", itemMap["item_name"])
	}
	if items[1] != "[REDACTED]" {
		t.Errorf("expected items[1] to be [REDACTED], got %v", items[1])
	}
	if items[2] != "safe_string" {
		t.Errorf("expected items[2] to be safe_string, got %v", items[2])
	}
}

func TestSanitizeAuditDetails_EdgeCases(t *testing.T) {
	// 1. Case variations of sensitive keys
	caseTests := map[string]interface{}{
		"PassWord":         "secret1",
		"ACCESS_TOKEN":     "secret2",
		"Private_Key":      "secret3",
		"Session_Secret":   "secret4",
		"auth_HEADER":      "secret5",
		"Otp":              "123456",
		"PIN":              "9999",
		"SERVICE_ROLE_KEY": "secret7",
		"Signed_Url":       "https://example.com/file?sig=abc",
	}
	sanitizedCases := platform.SanitizeAuditDetails(caseTests)
	for k := range caseTests {
		if sanitizedCases[k] != "[REDACTED]" {
			t.Errorf("expected case variation %s to be [REDACTED], got %v", k, sanitizedCases[k])
		}
	}

	// 2. Unexpected keys & arbitrary types
	arbitraryInput := map[string]interface{}{
		"custom_flag":   true,
		"int_metric":    42,
		"float_rate":    3.14159,
		"null_field":    nil,
		"unicode_key_🔥": "valid_unicode_val",
		"deeply_nested": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"password": "deep_password",
					"safe":     "deep_safe",
				},
			},
		},
	}
	sanitizedArbitrary := platform.SanitizeAuditDetails(arbitraryInput)
	if sanitizedArbitrary["custom_flag"] != true {
		t.Errorf("expected bool preserved")
	}
	if sanitizedArbitrary["unicode_key_🔥"] != "valid_unicode_val" {
		t.Errorf("expected unicode preserved")
	}
	deepLevel2 := sanitizedArbitrary["deeply_nested"].(map[string]interface{})["level1"].(map[string]interface{})["level2"].(map[string]interface{})
	if deepLevel2["password"] != "[REDACTED]" {
		t.Errorf("expected deep password redacted, got %v", deepLevel2["password"])
	}
	if deepLevel2["safe"] != "deep_safe" {
		t.Errorf("expected deep safe preserved")
	}

	// 3. Phone masking variations
	phoneTests := []struct {
		input    string
		expected string
	}{
		{"+919820000101", "+91 98*** ***01"},
		{"9820000101", "+91 98*** ***01"},
		{"+91 98200 00101", "+91 98*** ***01"},
		{"02025551234", "02***34"},
		{"", ""},
	}
	for _, pt := range phoneTests {
		res := platform.MaskPhone(pt.input)
		if res != pt.expected {
			t.Errorf("MaskPhone(%q) = %q, want %q", pt.input, res, pt.expected)
		}
	}

	// 4. Oversized metadata handling (1,000 keys and 100KB string value)
	oversized := make(map[string]interface{}, 1000)
	for i := 0; i < 1000; i++ {
		oversized[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("val_%d", i)
	}
	oversized["password"] = "leaked_pass"
	oversized["large_blob"] = strings.Repeat("x", 100*1024)

	sanitizedOversized := platform.SanitizeAuditDetails(oversized)
	if sanitizedOversized["password"] != "[REDACTED]" {
		t.Errorf("expected oversized password to be [REDACTED]")
	}
	if len(sanitizedOversized["large_blob"].(string)) != 100*1024 {
		t.Errorf("expected large blob preserved without crash")
	}
	if len(sanitizedOversized) != 1002 {
		t.Errorf("expected 1002 keys preserved, got %d", len(sanitizedOversized))
	}
}

func TestFormatActionLabel(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{"SUPER_ADMIN_BOOTSTRAPPED", "Super Admin Bootstrapped"},
		{"SOCIETY_CREATED", "Society Created"},
		{"MEMBER_REGISTERED", "Member Registered"},
		{"COMMITTEE_APPOINTMENT_CREATED", "Committee Appointment Created"},
		{"DOCUMENT_VERSION_UPLOADED", "Document Version Uploaded"},
		{"CUSTOM_ACTION_UNKNOWN", "Custom Action Unknown"},
	}

	for _, tt := range tests {
		got := platform.FormatActionLabel(tt.action)
		if got != tt.expected {
			t.Errorf("FormatActionLabel(%q) = %q; want %q", tt.action, got, tt.expected)
		}
	}
}

func TestListAuditLogsEnriched_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		cfg := config.Load()
		dbURL = cfg.DatabaseURL
	}
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping DB integration test")
	}

	ctx := context.Background()
	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repo := platform.NewRepository(db.Pool())

	// Test default pagination
	items, total, err := repo.ListAuditLogsEnriched(ctx, platform.AuditQueryFilters{
		Page:  1,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListAuditLogsEnriched failed: %v", err)
	}
	if total < 1 {
		t.Errorf("expected at least 1 baseline log, got %d", total)
	}
	if len(items) != 10 {
		t.Errorf("expected 10 items, got %d", len(items))
	}

	// Verify enriched fields
	first := items[0]
	if first.ID == uuid.Nil {
		t.Errorf("expected valid event ID")
	}
	if first.Action == "" {
		t.Errorf("expected action code")
	}
	if first.ActionLabel == "" {
		t.Errorf("expected action label")
	}
	if first.ActorName == "" {
		t.Errorf("expected actor name or fallback")
	}

	// Test action filter
	actionItems, actionTotal, err := repo.ListAuditLogsEnriched(ctx, platform.AuditQueryFilters{
		Page:   1,
		Limit:  10,
		Action: "SOCIETY_CREATED",
	})
	if err != nil {
		t.Fatalf("action filter query failed: %v", err)
	}
	if actionTotal == 0 {
		t.Errorf("expected non-zero SOCIETY_CREATED logs")
	}
	for _, it := range actionItems {
		if it.Action != "SOCIETY_CREATED" {
			t.Errorf("expected action SOCIETY_CREATED, got %s", it.Action)
		}
	}

	// Test GetAuditLogByID
	fetched, err := repo.GetAuditLogByID(ctx, first.ID, nil)
	if err != nil {
		t.Fatalf("GetAuditLogByID failed: %v", err)
	}
	if fetched.ID != first.ID {
		t.Errorf("expected id %v, got %v", first.ID, fetched.ID)
	}

	// Test cross-society access denial on GetAuditLogByID
	randomSocietyID := uuid.New()
	_, err = repo.GetAuditLogByID(ctx, first.ID, &randomSocietyID)
	if err == nil {
		t.Errorf("expected error when querying event with foreign society ID, got success")
	}
}
