package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"chs/backend/internal/config"
	"chs/backend/internal/response"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:           "8080",
		Env:            "test",
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Test router without database connection
	router := SetupRouter(cfg, logger, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Verify X-Request-ID response header
	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Errorf("expected X-Request-ID header in response")
	}

	var env response.SuccessEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if !env.Success {
		t.Errorf("expected success to be true")
	}

	dataMap, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data map in health response, got %T", env.Data)
	}

	if dataMap["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", dataMap["status"])
	}

	dbMap, ok := dataMap["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database status map in health response, got %T", dataMap["database"])
	}

	if dbMap["connected"] != false {
		t.Errorf("expected connected=false when nil DB provided, got %v", dbMap["connected"])
	}
}

func TestNotFoundEndpoint(t *testing.T) {
	cfg := &config.Config{
		Port:           "8080",
		Env:            "test",
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := SetupRouter(cfg, logger, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown-endpoint", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var env response.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&env); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if env.Success {
		t.Errorf("expected success=false for 404")
	}

	if env.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND error code, got %s", env.Error.Code)
	}
}
