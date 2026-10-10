package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"chs/backend/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	tm, err := auth.NewTokenManager("")
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	userID := uuid.New()
	sessionID := uuid.New()

	validToken, err := tm.GenerateAccessToken(userID, sessionID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	testHandler := Authenticate(tm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := GetUserID(r.Context())
		if !ok || gotUserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, gotUserID)
		}
		gotSessionID, ok := GetSessionID(r.Context())
		if !ok || gotSessionID != sessionID {
			t.Errorf("expected session ID %s, got %s", sessionID, gotSessionID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("succeeds with valid Bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		testHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("fails without Authorization header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()

		testHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("fails with malformed token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-str")
		rec := httptest.NewRecorder()

		testHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})
}
