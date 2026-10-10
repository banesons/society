package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenManager(t *testing.T) {
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}

	userID := uuid.New()
	sessionID := uuid.New()

	t.Run("generates and validates valid token", func(t *testing.T) {
		tokenStr, err := tm.GenerateAccessToken(userID, sessionID)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		claims, err := tm.ValidateAccessToken(tokenStr)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if claims.Subject != userID.String() {
			t.Errorf("expected subject %s, got %s", userID.String(), claims.Subject)
		}
		if claims.SessionID != sessionID {
			t.Errorf("expected session_id %s, got %s", sessionID.String(), claims.SessionID.String())
		}
	})

	t.Run("rejects token signed by another key", func(t *testing.T) {
		otherTM, _ := NewTokenManager("")
		tokenStr, _ := otherTM.GenerateAccessToken(userID, sessionID)

		_, err := tm.ValidateAccessToken(tokenStr)
		if err == nil {
			t.Errorf("expected validation failure for token signed with different key")
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		now := time.Now().Add(-20 * time.Minute)
		claims := Claims{
			SessionID: sessionID,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   userID.String(),
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
				Issuer:    "chs-backend",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		signed, err := token.SignedString(tm.privateKey)
		if err != nil {
			t.Fatalf("failed to sign expired token: %v", err)
		}

		_, err = tm.ValidateAccessToken(signed)
		if err != ErrTokenExpired {
			t.Errorf("expected ErrTokenExpired, got %v", err)
		}
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		_, err := tm.ValidateAccessToken("invalid.token.payload")
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got %v", err)
		}
	})
}
