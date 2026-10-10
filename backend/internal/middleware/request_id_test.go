package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := GetRequestID(r.Context())
		if reqID == "" {
			t.Errorf("expected request ID in context, got empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("generates new ID when header is missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get(RequestIDHeader)
		if respID == "" {
			t.Errorf("expected X-Request-ID header in response, got empty")
		}
	})

	t.Run("preserves valid incoming ID", func(t *testing.T) {
		customID := "test-request-id-12345"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(RequestIDHeader, customID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get(RequestIDHeader)
		if respID != customID {
			t.Errorf("expected preserved ID %q, got %q", customID, respID)
		}
	})

	t.Run("replaces malformed incoming ID", func(t *testing.T) {
		malformedID := "bad ID with spaces & <script>"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(RequestIDHeader, malformedID)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		respID := rec.Header().Get(RequestIDHeader)
		if respID == malformedID || respID == "" {
			t.Errorf("expected malformed ID to be sanitized/regenerated, got %q", respID)
		}
	})
}
