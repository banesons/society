package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	t.Run("development default passes validation", func(t *testing.T) {
		cfg := &Config{
			Env:         "development",
			DatabaseURL: "postgres://test",
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected error in dev: %v", err)
		}
	})

	t.Run("production requires DATABASE_URL", func(t *testing.T) {
		cfg := &Config{
			Env:           "production",
			DatabaseURL:   "",
			JWTPrivateKey: "dummy",
		}
		if err := cfg.Validate(); err == nil || err.Error() != "DATABASE_URL is required in production" {
			t.Fatalf("expected DATABASE_URL required error, got: %v", err)
		}
	})

	t.Run("production requires valid 64-byte Ed25519 JWT_PRIVATE_KEY", func(t *testing.T) {
		cfg := &Config{
			Env:           "production",
			DatabaseURL:   "postgres://prod-db",
			JWTPrivateKey: "invalid-key",
		}
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected error for invalid JWT_PRIVATE_KEY, got nil")
		}
	})

	t.Run("production rejects DEV_TEST_OTP", func(t *testing.T) {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate ed25519 key: %v", err)
		}
		privB64 := base64.StdEncoding.EncodeToString(priv)

		cfg := &Config{
			Env:           "production",
			DatabaseURL:   "postgres://prod-db",
			JWTPrivateKey: privB64,
			DevTestOTP:    "123456",
		}
		if err := cfg.Validate(); err == nil || err.Error() != "DEV_TEST_OTP must be empty in production" {
			t.Fatalf("expected DEV_TEST_OTP rejection, got: %v", err)
		}
	})

	t.Run("production rejects mock or empty SMS provider", func(t *testing.T) {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		privB64 := base64.StdEncoding.EncodeToString(priv)

		cfg := &Config{
			Env:           "production",
			DatabaseURL:   "postgres://prod-db",
			JWTPrivateKey: privB64,
			SMSProvider:   "mock",
		}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "SMS_PROVIDER cannot be mock") {
			t.Fatalf("expected mock SMS provider rejection in production, got: %v", err)
		}
	})

	t.Run("production requires MSG91AuthKey and MSG91TemplateID when SMSProvider is msg91", func(t *testing.T) {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		privB64 := base64.StdEncoding.EncodeToString(priv)

		cfg := &Config{
			Env:           "production",
			DatabaseURL:   "postgres://prod-db",
			JWTPrivateKey: privB64,
			SMSProvider:   "msg91",
		}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "MSG91_AUTH_KEY is required") {
			t.Fatalf("expected MSG91_AUTH_KEY error, got: %v", err)
		}

		cfg.MSG91AuthKey = "secret-key"
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "MSG91_TEMPLATE_ID is required") {
			t.Fatalf("expected MSG91_TEMPLATE_ID error, got: %v", err)
		}
	})

	t.Run("production valid configuration with MSG91 passes", func(t *testing.T) {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("failed to generate ed25519 key: %v", err)
		}
		privB64 := base64.StdEncoding.EncodeToString(priv)

		cfg := &Config{
			Env:             "production",
			DatabaseURL:     "postgres://prod-db",
			JWTPrivateKey:   privB64,
			SMSProvider:     "msg91",
			MSG91AuthKey:    "test-auth-key",
			MSG91TemplateID: "DLT_12345",
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected validation error in production: %v", err)
		}
	})

	t.Run("production load does not populate test otp default", func(t *testing.T) {
		os.Setenv("ENV", "production")
		os.Unsetenv("DEV_TEST_OTP")
		os.Unsetenv("DEV_TEST_PHONES")
		defer os.Setenv("ENV", "development")

		cfg := Load()
		if cfg.DevTestOTP != "" {
			t.Errorf("expected empty DevTestOTP in production, got: %s", cfg.DevTestOTP)
		}
		if len(cfg.DevTestPhones) != 0 {
			t.Errorf("expected empty DevTestPhones in production, got: %v", cfg.DevTestPhones)
		}
	})
}
