package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccessResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	data := map[string]string{"foo": "bar"}
	meta := map[string]int{"count": 1}

	Success(rec, http.StatusOK, data, meta)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var env SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !env.Success {
		t.Errorf("expected success to be true")
	}

	dataMap, ok := env.Data.(map[string]interface{})
	if !ok || dataMap["foo"] != "bar" {
		t.Errorf("unexpected data payload: %v", env.Data)
	}
}

func TestErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()

	Error(rec, http.StatusBadRequest, "INVALID_INPUT", "Invalid payload received", "req-12345")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var env ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if env.Success {
		t.Errorf("expected success to be false")
	}

	if env.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected error code INVALID_INPUT, got %s", env.Error.Code)
	}

	if env.RequestID != "req-12345" {
		t.Errorf("expected request_id req-12345, got %s", env.RequestID)
	}
}
