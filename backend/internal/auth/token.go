package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// AccessTokenLifespan is 15 minutes.
	AccessTokenLifespan = 15 * time.Minute

	// RefreshTokenLifespan is 30 days.
	RefreshTokenLifespan = 30 * 24 * time.Hour
)

var (
	ErrInvalidToken = errors.New("invalid or malformed access token")
	ErrTokenExpired = errors.New("access token has expired")
)

// Claims represents the standard CHS access token payload.
type Claims struct {
	SessionID uuid.UUID `json:"session_id"`
	jwt.RegisteredClaims
}

// TokenManager signs and validates JWTs using Ed25519.
type TokenManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewTokenManager initializes the TokenManager from an optional base64 private key string,
// or generates an ephemeral in-memory Ed25519 keypair if none is supplied (ideal for development).
func NewTokenManager(privateKeyB64 string) (*TokenManager, error) {
	privateKeyB64 = strings.TrimSpace(privateKeyB64)
	privateKeyB64 = strings.Trim(privateKeyB64, `"'`)
	if privateKeyB64 != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
		if err != nil {
			keyBytes, err = base64.RawStdEncoding.DecodeString(privateKeyB64)
		}
		if err != nil {
			keyBytes, err = base64.URLEncoding.DecodeString(privateKeyB64)
		}
		if err != nil {
			keyBytes, err = base64.RawURLEncoding.DecodeString(privateKeyB64)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 JWT private key: %w", err)
		}
		var priv ed25519.PrivateKey
		if len(keyBytes) == ed25519.SeedSize {
			priv = ed25519.NewKeyFromSeed(keyBytes)
		} else if len(keyBytes) == ed25519.PrivateKeySize {
			// Deriving from the first 32-byte seed guarantees the public key in priv[32:] is mathematically correct
			priv = ed25519.NewKeyFromSeed(keyBytes[:ed25519.SeedSize])
		} else {
			return nil, fmt.Errorf("invalid ed25519 private key size: expected %d or %d bytes, got %d", ed25519.SeedSize, ed25519.PrivateKeySize, len(keyBytes))
		}
		pub := priv.Public().(ed25519.PublicKey)
		return &TokenManager{privateKey: priv, publicKey: pub}, nil
	}

	// Auto-generate ephemeral Ed25519 keypair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 keypair: %w", err)
	}

	return &TokenManager{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// GenerateAccessToken creates a signed Ed25519 JWT for the given user and session.
func (tm *TokenManager) GenerateAccessToken(userID, sessionID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenLifespan)),
			Issuer:    "chs-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	signed, err := token.SignedString(tm.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signed, nil
}

// ValidateAccessToken verifies the token signature, algorithm, and claims.
func (tm *TokenManager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	tokenStr = strings.Trim(tokenStr, `"'`)
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Strict algorithm enforcement: must be EdDSA
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Header["alg"])
		}
		return tm.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
