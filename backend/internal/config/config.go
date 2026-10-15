package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all centralized backend configuration.
type Config struct {
	Port                          string
	Env                           string
	AllowedOrigins                []string
	DatabaseURL                   string
	SupabaseURL                   string
	SupabaseAnonKey               string
	DevTestPhones                 []string
	DevTestOTP                    string
	JWTPrivateKey                 string
	SuperAdminPhone               string
	ResendAPIKey                  string
	ResendFromEmail               string
	FirebaseProjectID             string
	FirebaseCredentialsFile       string
	FirebaseWebPushVAPIDPublicKey string
	SMSProvider                   string
	MSG91AuthKey                  string
	MSG91TemplateID               string
	MSG91SenderID                 string
	MSG91EntityID                 string
	MSG91BaseURL                  string
}

// Load reads configuration from environment variables, optionally reading .env if present.
func Load() *Config {
	// In production, environment variables are injected by the host/container runtime; skip local .env
	if strings.ToLower(strings.TrimSpace(os.Getenv("ENV"))) != "production" {
		for _, envPath := range []string{".env", "../.env", "../../.env"} {
			if err := godotenv.Load(envPath); err == nil {
				break
			}
		}
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:4173,https://society.banesons.com")
	databaseURL := os.Getenv("DATABASE_URL")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	var devTestPhonesStr string
	var devTestOTP string
	if strings.ToLower(strings.TrimSpace(env)) != "production" {
		devTestPhonesStr = getEnv("DEV_TEST_PHONES", "919967089101,918422862001,919820010001,919820010002,919820020001,919822000101")
		devTestOTP = getEnv("DEV_TEST_OTP", "123456")
	} else {
		devTestPhonesStr = os.Getenv("DEV_TEST_PHONES")
		devTestOTP = os.Getenv("DEV_TEST_OTP")
	}
	jwtPrivateKey := os.Getenv("JWT_PRIVATE_KEY")
	superAdminPhoneRaw := getEnv("SUPER_ADMIN_PHONE", "919967089101")

	// Strip non-digits from superAdminPhone
	superAdminPhone := ""
	for _, ch := range superAdminPhoneRaw {
		if ch >= '0' && ch <= '9' {
			superAdminPhone += string(ch)
		}
	}

	var allowedOrigins []string
	for _, o := range strings.Split(originsStr, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}

	var devTestPhones []string
	for _, p := range strings.Split(devTestPhonesStr, ",") {
		digits := ""
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				digits += string(ch)
			}
		}
		if digits != "" {
			devTestPhones = append(devTestPhones, digits)
		}
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")
	resendFromEmail := getEnv("RESEND_FROM_EMAIL", "onboarding@resend.dev")
	firebaseProjectID := getEnv("FIREBASE_PROJECT_ID", "chsneu-2021")
	firebaseCredentialsFile := os.Getenv("FIREBASE_CREDENTIALS_FILE")
	firebaseWebPushVAPIDPublicKey := os.Getenv("FIREBASE_WEB_PUSH_VAPID_PUBLIC_KEY")
	smsProvider := getEnv("SMS_PROVIDER", "mock")
	msg91AuthKey := os.Getenv("MSG91_AUTH_KEY")
	msg91TemplateID := os.Getenv("MSG91_TEMPLATE_ID")
	msg91SenderID := getEnv("MSG91_SENDER_ID", "CHSNEU")
	msg91EntityID := os.Getenv("MSG91_ENTITY_ID")
	msg91BaseURL := getEnv("MSG91_BASE_URL", "https://control.msg91.com/api/v5/otp")

	return &Config{
		Port:                          port,
		Env:                           env,
		AllowedOrigins:                allowedOrigins,
		DatabaseURL:                   databaseURL,
		SupabaseURL:                   supabaseURL,
		SupabaseAnonKey:               supabaseAnonKey,
		DevTestPhones:                 devTestPhones,
		DevTestOTP:                    devTestOTP,
		JWTPrivateKey:                 jwtPrivateKey,
		SuperAdminPhone:               superAdminPhone,
		ResendAPIKey:                  resendAPIKey,
		ResendFromEmail:               resendFromEmail,
		FirebaseProjectID:             firebaseProjectID,
		FirebaseCredentialsFile:       firebaseCredentialsFile,
		FirebaseWebPushVAPIDPublicKey: firebaseWebPushVAPIDPublicKey,
		SMSProvider:                   smsProvider,
		MSG91AuthKey:                  msg91AuthKey,
		MSG91TemplateID:               msg91TemplateID,
		MSG91SenderID:                 msg91SenderID,
		MSG91EntityID:                 msg91EntityID,
		MSG91BaseURL:                  msg91BaseURL,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// IsProduction returns true if ENV is set to production.
func (c *Config) IsProduction() bool {
	return strings.ToLower(strings.TrimSpace(c.Env)) == "production"
}

// Validate performs strict security and operational integrity checks on loaded configuration.
func (c *Config) Validate() error {
	if c.IsProduction() {
		if strings.TrimSpace(c.DatabaseURL) == "" {
			return errors.New("DATABASE_URL is required in production")
		}
		if strings.TrimSpace(c.JWTPrivateKey) == "" {
			return errors.New("JWT_PRIVATE_KEY is required in production (ephemeral keys forbidden)")
		}
		keyBytes, err := base64.StdEncoding.DecodeString(c.JWTPrivateKey)
		if err != nil || len(keyBytes) != ed25519.PrivateKeySize {
			return errors.New("JWT_PRIVATE_KEY must be a valid 64-byte base64-encoded Ed25519 private key")
		}
		if c.DevTestOTP != "" {
			return errors.New("DEV_TEST_OTP must be empty in production")
		}
		if len(c.DevTestPhones) > 0 {
			return errors.New("DEV_TEST_PHONES must be empty in production")
		}
		if c.SMSProvider == "mock" || c.SMSProvider == "" {
			return errors.New("SMS_PROVIDER cannot be mock or empty in production; a valid provider (e.g. msg91) is required")
		}
		if c.SMSProvider == "msg91" {
			if strings.TrimSpace(c.MSG91AuthKey) == "" {
				return errors.New("MSG91_AUTH_KEY is required in production")
			}
			if strings.TrimSpace(c.MSG91TemplateID) == "" {
				return errors.New("MSG91_TEMPLATE_ID is required in production")
			}
		}
	}
	return nil
}
