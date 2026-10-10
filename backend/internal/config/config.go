package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all centralized backend configuration.
type Config struct {
	Port            string
	Env             string
	AllowedOrigins  []string
	DatabaseURL     string
	SupabaseURL     string
	SupabaseAnonKey string
	DevTestPhones   []string
	DevTestOTP      string
	JWTPrivateKey   string
	SuperAdminPhone string
}

// Load reads configuration from environment variables, optionally reading .env if present.
func Load() *Config {
	// Attempt to load .env, searching upward if running inside a subpackage test
	for _, envPath := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			break
		}
	}

	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")
	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:4173")
	databaseURL := os.Getenv("DATABASE_URL")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	devTestPhonesStr := getEnv("DEV_TEST_PHONES", "+919967089101,+918422862001")
	devTestOTP := getEnv("DEV_TEST_OTP", "123456")
	jwtPrivateKey := os.Getenv("JWT_PRIVATE_KEY")
	superAdminPhone := getEnv("SUPER_ADMIN_PHONE", "+919967089101")

	var allowedOrigins []string
	for _, o := range strings.Split(originsStr, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}

	var devTestPhones []string
	for _, p := range strings.Split(devTestPhonesStr, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			devTestPhones = append(devTestPhones, trimmed)
		}
	}

	return &Config{
		Port:            port,
		Env:             env,
		AllowedOrigins:  allowedOrigins,
		DatabaseURL:     databaseURL,
		SupabaseURL:     supabaseURL,
		SupabaseAnonKey: supabaseAnonKey,
		DevTestPhones:   devTestPhones,
		DevTestOTP:      devTestOTP,
		JWTPrivateKey:   jwtPrivateKey,
		SuperAdminPhone: superAdminPhone,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
