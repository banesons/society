package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"chs/backend/internal/auth"
	"chs/backend/internal/response"
)

type mockSessionValidator struct {
	validateFunc func(ctx context.Context, userID, sessionID uuid.UUID) error
}

func (m *mockSessionValidator) ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, userID, sessionID)
	}
	return nil
}

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

	dummyProtectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := GetUserID(r.Context())
		if !ok || gotUserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, gotUserID)
		}
		gotSessionID, ok := GetSessionID(r.Context())
		if !ok || gotSessionID != sessionID {
			t.Errorf("expected session ID %s, got %s", sessionID, gotSessionID)
		}
		w.WriteHeader(http.StatusOK)
	})

	t.Run("succeeds with valid Bearer token and active session", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				if uID != userID || sID != sessionID {
					t.Errorf("validator received unexpected IDs: %s, %s", uID, sID)
				}
				return nil
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("fails without Authorization header", func(t *testing.T) {
		validator := &mockSessionValidator{}
		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "UNAUTHORIZED" {
			t.Errorf("expected UNAUTHORIZED, got %s", env.Error.Code)
		}
	})

	t.Run("fails with malformed token", func(t *testing.T) {
		validator := &mockSessionValidator{}
		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token-str")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "INVALID_TOKEN" {
			t.Errorf("expected INVALID_TOKEN, got %s", env.Error.Code)
		}
	})

	t.Run("fails with invalid signature JWT", func(t *testing.T) {
		tmOther, _ := auth.NewTokenManager("")
		wrongKeyToken, _ := tmOther.GenerateAccessToken(userID, sessionID)

		validator := &mockSessionValidator{}
		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+wrongKeyToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("rejects revoked session with 401 SESSION_REVOKED", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				return auth.ErrSessionRevoked
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "SESSION_REVOKED" {
			t.Errorf("expected SESSION_REVOKED, got %s", env.Error.Code)
		}
	})

	t.Run("rejects expired session with 401 SESSION_EXPIRED", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				return auth.ErrSessionExpired
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "SESSION_EXPIRED" {
			t.Errorf("expected SESSION_EXPIRED, got %s", env.Error.Code)
		}
	})

	t.Run("rejects session belonging to different user with 401 SESSION_INVALID", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				return auth.ErrSessionMismatch
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "SESSION_INVALID" {
			t.Errorf("expected SESSION_INVALID, got %s", env.Error.Code)
		}
	})

	t.Run("rejects missing session record with 401 SESSION_INVALID", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				return auth.ErrSessionNotFound
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "SESSION_INVALID" {
			t.Errorf("expected SESSION_INVALID, got %s", env.Error.Code)
		}
	})

	t.Run("fails closed on database validation error with 500 INTERNAL_ERROR", func(t *testing.T) {
		validator := &mockSessionValidator{
			validateFunc: func(ctx context.Context, uID, sID uuid.UUID) error {
				return errors.New("database connection timeout")
			},
		}

		handler := Authenticate(tm, validator)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "INTERNAL_ERROR" {
			t.Errorf("expected INTERNAL_ERROR, got %s", env.Error.Code)
		}
	})

	t.Run("fails closed if validator is nil with 500 INTERNAL_ERROR", func(t *testing.T) {
		handler := Authenticate(tm, nil)(dummyProtectedHandler)
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}

		var env response.ErrorEnvelope
		_ = json.NewDecoder(rec.Body).Decode(&env)
		if env.Error.Code != "INTERNAL_ERROR" {
			t.Errorf("expected INTERNAL_ERROR, got %s", env.Error.Code)
		}
	})
}
