package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"time"
)

const (
	// OTPValidityDuration is the lifetime of an issued OTP (5 minutes).
	OTPValidityDuration = 5 * time.Minute

	// MaxOTPVerificationAttempts is the maximum number of failed attempts before invalidation.
	MaxOTPVerificationAttempts = 3

	// OTPResendCooldownDuration is the minimum duration before another OTP can be requested (60 seconds).
	OTPResendCooldownDuration = 60 * time.Second
)

// GenerateNumericOTP generates a cryptographically secure 6-digit numeric string.
func GenerateNumericOTP() (string, error) {
	const max = 1000000
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random number: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashOTP computes a salted SHA-256 hash of the OTP tied to the canonical phone number.
func HashOTP(phone, otp string) string {
	hasher := sha256.New()
	// Salt with canonical phone to prevent precomputed rainbow tables across numbers
	hasher.Write([]byte(phone + ":" + otp + ":chs_otp_salt_v1"))
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyOTPHash compares an entered OTP against the stored hash in constant time.
func VerifyOTPHash(phone, candidateOTP, storedHash string) bool {
	computed := HashOTP(phone, candidateOTP)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1
}

// GenerateRandomOpaqueToken generates a cryptographically random hex string (e.g. for refresh tokens).
func GenerateRandomOpaqueToken(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken computes a SHA-256 hash of an opaque token for secure database storage.
func HashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
