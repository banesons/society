package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	handler := CORS([]string{"https://custom.banesons.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	testCases := []struct {
		origin      string
		expectAllow bool
	}{
		{"http://localhost:5173", true},
		{"http://localhost:4173", true},
		{"http://localhost:4174", true},
		{"http://127.0.0.1:4174", true},
		{"https://society.banesons.com", true},
		{"https://custom.banesons.com", true},
		{"https://malicious-site.com", false},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
		if tc.expectAllow && allowOrigin != tc.origin {
			t.Errorf("origin %s: expected Access-Control-Allow-Origin=%s, got %s", tc.origin, tc.origin, allowOrigin)
		}
		if !tc.expectAllow && allowOrigin != "" {
			t.Errorf("origin %s: expected empty allow origin, got %s", tc.origin, allowOrigin)
		}
	}
}
