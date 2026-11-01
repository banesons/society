package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"chs/backend/internal/sms"
)

var (
	ErrOTPInvalid          = errors.New("invalid verification code")
	ErrOTPExpired          = errors.New("verification code has expired")
	ErrOTPAttemptsExceeded = errors.New("maximum verification attempts exceeded")
	ErrSessionLimitReached = errors.New("maximum active devices reached, please log out from another device")
	ErrSessionRevoked      = errors.New("session has been revoked")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExpired      = errors.New("session has expired")
	ErrSessionMismatch     = errors.New("session does not belong to user")
	ErrRefreshTokenReused  = errors.New("refresh token was already used or revoked")
	ErrUserNotFound        = errors.New("user not found")
)

// SessionValidator abstracts server-side session status checks.
type SessionValidator interface {
	ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error
}

// User represents the core platform user identity.
type User struct {
	ID            uuid.UUID `json:"id"`
	Phone         string    `json:"phone"`
	Status        string    `json:"status"`
	PlatformRoles []string  `json:"platform_roles,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AuthResult contains the issued tokens and authenticated user info.
type AuthResult struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresIn       int64  `json:"expires_in"` // seconds (900 = 15m)
	User            User   `json:"user"`
	SessionReplaced bool   `json:"session_replaced,omitempty"`
}

// TokenRefreshResult holds the new access token and rotated refresh token.
type TokenRefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// ActiveSessionInfo contains client-safe metadata for active session management.
type ActiveSessionInfo struct {
	ID           uuid.UUID `json:"id"`
	DeviceLabel  string    `json:"device_label"`
	Browser      string    `json:"browser"`
	OS           string    `json:"os"`
	IsCurrent    bool      `json:"is_current"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
}

// ParseUserAgent derives human-readable device, browser, and OS labels without fabricating values.
func ParseUserAgent(ua string) (deviceLabel, browser, os string) {
	uaClean := strings.TrimSpace(ua)
	if uaClean == "" {
		return "Unknown Device", "Unknown", "Unknown"
	}

	// Detect OS
	switch {
	case strings.Contains(uaClean, "iPhone") || strings.Contains(uaClean, "iPad") || strings.Contains(uaClean, "iOS"):
		os = "iOS"
	case strings.Contains(uaClean, "Macintosh") || strings.Contains(uaClean, "Mac OS X"):
		os = "macOS"
	case strings.Contains(uaClean, "Android"):
		os = "Android"
	case strings.Contains(uaClean, "Windows"):
		os = "Windows"
	case strings.Contains(uaClean, "Linux"):
		os = "Linux"
	default:
		os = "Unknown"
	}

	// Detect Browser
	switch {
	case strings.Contains(uaClean, "Edg/") || strings.Contains(uaClean, "Edge/"):
		browser = "Edge"
	case strings.Contains(uaClean, "Chrome/") || strings.Contains(uaClean, "CriOS/"):
		browser = "Chrome"
	case strings.Contains(uaClean, "Firefox/") || strings.Contains(uaClean, "FxiOS/"):
		browser = "Firefox"
	case strings.Contains(uaClean, "Safari/") && !strings.Contains(uaClean, "Chrome") && !strings.Contains(uaClean, "Chromium"):
		browser = "Safari"
	case strings.Contains(uaClean, "curl/") || strings.Contains(uaClean, "PostmanRuntime"):
		browser = "API Client"
	default:
		if strings.Contains(uaClean, "Mobile") {
			browser = "Mobile Browser"
		} else {
			browser = "Browser"
		}
	}

	if os != "Unknown" && browser != "Browser" && browser != "Unknown" {
		deviceLabel = fmt.Sprintf("%s on %s", browser, os)
	} else if os != "Unknown" {
		deviceLabel = fmt.Sprintf("%s Device", os)
	} else if browser != "Unknown" {
		deviceLabel = browser
	} else {
		deviceLabel = "Unknown Device"
	}

	return deviceLabel, browser, os
}

// ServiceConfig contains runtime configuration for the authentication service.
type ServiceConfig struct {
	DevTestPhones      []string
	DevTestOTP         string
	VerificationSecret string
}

// Service coordinates all authentication domain workflows.
type Service struct {
	pool                       *pgxpool.Pool
	tokenManager               *TokenManager
	smsProvider                sms.Provider
	rateLimiter                *RateLimiter
	config                     ServiceConfig
	logger                     *slog.Logger
	consumedVerificationNonces sync.Map
	ephemeralSecret            []byte
}

// NewService initializes an authentication service instance.
func NewService(
	pool *pgxpool.Pool,
	tm *TokenManager,
	smsProvider sms.Provider,
	limiter *RateLimiter,
	cfg ServiceConfig,
	logger *slog.Logger,
) *Service {
	ephemeral := make([]byte, 32)
	if _, err := rand.Read(ephemeral); err != nil {
		ephemeral = nil
	}
	return &Service{
		pool:            pool,
		tokenManager:    tm,
		smsProvider:     smsProvider,
		rateLimiter:     limiter,
		config:          cfg,
		logger:          logger,
		ephemeralSecret: ephemeral,
	}
}

// RateLimiter returns the underlying rate limiter instance.
func (s *Service) RateLimiter() *RateLimiter {
	return s.rateLimiter
}

// SendOTP normalizes the phone number, checks rate limits, generates/stores the OTP, and dispatches SMS.
// In development mode, the generated OTP code is returned for testing inspection.
func (s *Service) SendOTP(ctx context.Context, rawPhone, clientIP string) (int, string, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return 0, "", err
	}

	// Rate limiting by IP
	if err := s.rateLimiter.CheckAndRecordIP(clientIP); err != nil {
		return 0, "", err
	}

	// Determine OTP code: check test numbers in dev
	otpCode := ""
	isTestNumber := false
	for _, testPhone := range s.config.DevTestPhones {
		if testPhone == phone {
			isTestNumber = true
			break
		}
	}

	// Rate limiting by phone (cooldown & hourly cap)
	resendAfter, err := s.rateLimiter.CheckAndRecordPhone(phone)
	if err != nil {
		return resendAfter, "", err
	}

	if isTestNumber && s.config.DevTestOTP != "" {
		otpCode = s.config.DevTestOTP
		s.logger.Info("Using configured development test OTP", slog.String("phone", MaskPhone(phone)))
	} else {
		generated, err := GenerateNumericOTP()
		if err != nil {
			return 0, "", fmt.Errorf("failed to generate secure OTP: %w", err)
		}
		otpCode = generated
	}

	otpHash := HashOTP(phone, otpCode)
	expiresAt := time.Now().Add(OTPValidityDuration)

	// Persist OTP in database (if pool connected)
	if s.pool != nil {
		query := `
			INSERT INTO auth_otps (phone, otp_hash, attempts, expires_at)
			VALUES ($1, $2, 0, $3)
		`
		if _, err := s.pool.Exec(ctx, query, phone, otpHash, expiresAt); err != nil {
			return 0, "", fmt.Errorf("failed to record OTP: %w", err)
		}
	}

	// Dispatch SMS
	if err := s.smsProvider.SendOTP(ctx, phone, otpCode); err != nil {
		s.logger.Error("SMS dispatch failed", slog.String("phone", MaskPhone(phone)), slog.String("error", err.Error()))
		return 0, "", fmt.Errorf("failed to deliver verification SMS")
	}

	s.logger.Info("OTP generated and dispatched", slog.String("phone", MaskPhone(phone)))
	return resendAfter, otpCode, nil
}

// VerifyOTP verifies candidate code, provisions user (JIT), checks 2-device limit, and issues tokens.
func (s *Service) VerifyOTP(ctx context.Context, rawPhone, candidateOTP, userAgent, clientIP string) (*AuthResult, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return nil, err
	}

	if s.rateLimiter != nil {
		if err := s.rateLimiter.CheckVerifyIP(clientIP); err != nil {
			return nil, err
		}
	}

	if s.pool == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	// 1. Fetch latest active OTP record
	queryOTP := `
		SELECT id, otp_hash, attempts, expires_at, consumed_at
		FROM auth_otps
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var (
		otpID      uuid.UUID
		storedHash string
		attempts   int
		expiresAt  time.Time
		consumedAt *time.Time
	)

	err = s.pool.QueryRow(ctx, queryOTP, phone).Scan(&otpID, &storedHash, &attempts, &expiresAt, &consumedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if s.rateLimiter != nil {
				s.rateLimiter.RecordVerifyFailureIP(clientIP)
			}
			return nil, ErrOTPInvalid
		}
		return nil, fmt.Errorf("failed to query OTP record: %w", err)
	}

	if consumedAt != nil {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return nil, ErrOTPInvalid
	}

	if time.Now().After(expiresAt) {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return nil, ErrOTPExpired
	}

	if attempts >= MaxOTPVerificationAttempts {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return nil, ErrOTPAttemptsExceeded
	}

	// Verify constant-time hash
	if !VerifyOTPHash(phone, candidateOTP, storedHash) {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		// Increment attempts
		_, _ = s.pool.Exec(ctx, `UPDATE auth_otps SET attempts = attempts + 1, updated_at = NOW() WHERE id = $1`, otpID)
		if attempts+1 >= MaxOTPVerificationAttempts {
			return nil, ErrOTPAttemptsExceeded
		}
		return nil, ErrOTPInvalid
	}

	// Reset IP verify failures on successful verification
	if s.rateLimiter != nil {
		s.rateLimiter.ResetVerifyIP(clientIP)
	}

	// Mark consumed
	_, _ = s.pool.Exec(ctx, `UPDATE auth_otps SET consumed_at = NOW(), updated_at = NOW() WHERE id = $1`, otpID)

	// 2. Just-In-Time user lookup or creation
	var user User
	findUserQuery := `SELECT id, phone, status, created_at, updated_at FROM users WHERE phone = $1`
	err = s.pool.QueryRow(ctx, findUserQuery, phone).Scan(&user.ID, &user.Phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Auto-create new user
			insertUserQuery := `
				INSERT INTO users (phone, status)
				VALUES ($1, 'ACTIVE')
				RETURNING id, phone, status, created_at, updated_at
			`
			err = s.pool.QueryRow(ctx, insertUserQuery, phone).Scan(&user.ID, &user.Phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to provision user: %w", err)
			}
			s.logger.Info("New user auto-created via OTP", slog.String("user_id", user.ID.String()), slog.String("phone", MaskPhone(user.Phone)))
		} else {
			return nil, fmt.Errorf("failed to query user: %w", err)
		}
	}

	// 3. Concurrency-safe session management with FIFO replacement (Max 2 devices)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin session transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Acquire exclusive user-level row lock on users table to serialize concurrent logins
	var lockedUserID uuid.UUID
	lockQuery := `SELECT id FROM users WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRow(ctx, lockQuery, user.ID).Scan(&lockedUserID); err != nil {
		return nil, fmt.Errorf("failed to acquire user session lock: %w", err)
	}

	// Read currently active sessions ordered deterministically:
	// Earliest created_at first (FIFO), with session ID as secondary tie-breaker.
	// Lock active session rows.
	activeSessionsQuery := `
		SELECT id, created_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at ASC, id ASC
		FOR UPDATE
	`
	rows, err := tx.Query(ctx, activeSessionsQuery, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query active sessions: %w", err)
	}

	type activeSessionRow struct {
		id        uuid.UUID
		createdAt time.Time
	}
	var activeSessions []activeSessionRow
	for rows.Next() {
		var asr activeSessionRow
		if err := rows.Scan(&asr.id, &asr.createdAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}
		activeSessions = append(activeSessions, asr)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading active session rows: %w", err)
	}

	// Enforce 2-device active session limit with FIFO replacement
	const maxActiveSessions = 2
	var revokedSessionIDs []uuid.UUID

	if len(activeSessions) >= maxActiveSessions {
		excess := len(activeSessions) - maxActiveSessions + 1
		toRevoke := activeSessions[:excess]

		for _, sess := range toRevoke {
			revokeQuery := `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`
			if _, err := tx.Exec(ctx, revokeQuery, sess.id); err != nil {
				return nil, fmt.Errorf("failed to revoke oldest session %s: %w", sess.id, err)
			}
			revokedSessionIDs = append(revokedSessionIDs, sess.id)
		}
	}

	// 4. Create new session and tokens inside the same transaction
	refreshToken, err := GenerateRandomOpaqueToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenHash := HashToken(refreshToken)
	sessionExpiresAt := time.Now().Add(RefreshTokenLifespan)

	var sessionID uuid.UUID
	createSessionQuery := `
		INSERT INTO user_sessions (user_id, refresh_token_hash, user_agent, client_ip, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err = tx.QueryRow(ctx, createSessionQuery, user.ID, refreshTokenHash, userAgent, clientIP, sessionExpiresAt).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit session transaction: %w", err)
	}

	sessionReplaced := len(revokedSessionIDs) > 0
	if sessionReplaced {
		for _, revID := range revokedSessionIDs {
			s.logger.Info("FIFO session replacement performed",
				slog.String("event_type", "fifo_session_replacement"),
				slog.String("user_id", user.ID.String()),
				slog.String("revoked_session_id", revID.String()),
				slog.String("replacement_session_id", sessionID.String()),
				slog.String("reason", "FIFO_SESSION_LIMIT"),
			)

			auditDetails := map[string]interface{}{
				"reason":                 "FIFO_SESSION_LIMIT",
				"revoked_session_id":     revID.String(),
				"replacement_session_id": sessionID.String(),
			}
			detailsJSON, _ := json.Marshal(auditDetails)
			_, _ = s.pool.Exec(ctx, `
				INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
				VALUES ($1, 'SESSION_ROTATED', $1, $2, $3, $4)
			`, user.ID, detailsJSON, clientIP, userAgent)
		}
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue access token: %w", err)
	}

	s.logger.Info("User session created successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("session_id", sessionID.String()),
		slog.Bool("session_replaced", sessionReplaced),
	)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(AccessTokenLifespan.Seconds()),
		User: User{
			ID:        user.ID,
			Phone:     MaskPhone(user.Phone),
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		SessionReplaced: sessionReplaced,
	}, nil
}

// RefreshToken validates an active refresh token, rotates it, and issues a new access token.
func (s *Service) RefreshToken(ctx context.Context, candidateRefreshToken, userAgent, clientIP string) (*TokenRefreshResult, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	tokenHash := HashToken(candidateRefreshToken)

	var (
		sessionID uuid.UUID
		userID    uuid.UUID
		expiresAt time.Time
		revokedAt *time.Time
	)

	query := `
		SELECT id, user_id, expires_at, revoked_at
		FROM user_sessions
		WHERE refresh_token_hash = $1
	`
	err := s.pool.QueryRow(ctx, query, tokenHash).Scan(&sessionID, &userID, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to look up session: %w", err)
	}

	// Replay / Revocation check: if this session was revoked (by FIFO or logout), reject refresh
	if revokedAt != nil {
		s.logger.Warn("Session refresh rejected: session was revoked", slog.String("session_id", sessionID.String()))
		return nil, ErrSessionRevoked
	}

	if time.Now().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	// Rotate refresh token
	newRefreshToken, err := GenerateRandomOpaqueToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate rotated refresh token: %w", err)
	}
	newHash := HashToken(newRefreshToken)

	updateQuery := `
		UPDATE user_sessions
		SET refresh_token_hash = $1, user_agent = $2, client_ip = $3, last_active_at = NOW()
		WHERE id = $4
	`
	if _, err := s.pool.Exec(ctx, updateQuery, newHash, userAgent, clientIP, sessionID); err != nil {
		return nil, fmt.Errorf("failed to update session tokens: %w", err)
	}

	newAccessToken, err := s.tokenManager.GenerateAccessToken(userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &TokenRefreshResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(AccessTokenLifespan.Seconds()),
	}, nil
}

// Logout revokes the specified session.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if s.pool == nil {
		return nil
	}
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, query, sessionID)
	return err
}

// LogoutAll revokes all active sessions for a user and records a persistent security audit entry.
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if s.pool == nil {
		return nil
	}
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	tag, err := s.pool.Exec(ctx, query, userID)
	if err == nil && tag.RowsAffected() > 0 {
		auditDetails := map[string]interface{}{
			"revoked_count": tag.RowsAffected(),
		}
		detailsJSON, _ := json.Marshal(auditDetails)
		_, _ = s.pool.Exec(ctx, `
			INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
			VALUES ($1, 'USER_LOGOUT_ALL', $1, $2)
		`, userID, detailsJSON)
	}
	return err
}

// ListActiveSessions returns the authenticated user's currently active sessions (maximum 2).
func (s *Service) ListActiveSessions(ctx context.Context, userID, currentSessionID uuid.UUID) ([]ActiveSessionInfo, error) {
	if s.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, user_agent, created_at, last_active_at
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC, id DESC
		LIMIT 2
	`
	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ActiveSessionInfo
	for rows.Next() {
		var (
			id           uuid.UUID
			ua           *string
			createdAt    time.Time
			lastActiveAt time.Time
		)
		if err := rows.Scan(&id, &ua, &createdAt, &lastActiveAt); err != nil {
			return nil, fmt.Errorf("failed to scan active session: %w", err)
		}

		uaStr := ""
		if ua != nil {
			uaStr = *ua
		}
		label, browser, os := ParseUserAgent(uaStr)

		sessions = append(sessions, ActiveSessionInfo{
			ID:           id,
			DeviceLabel:  label,
			Browser:      browser,
			OS:           os,
			IsCurrent:    id == currentSessionID,
			CreatedAt:    createdAt,
			LastActiveAt: lastActiveAt,
		})
	}

	if sessions == nil {
		sessions = []ActiveSessionInfo{}
	}

	return sessions, nil
}

// RevokeSession revokes a specific active session belonging to the user.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if s.pool == nil {
		return errors.New("database pool unavailable")
	}

	query := `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`
	tag, err := s.pool.Exec(ctx, query, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	auditDetails := map[string]interface{}{
		"revoked_session_id": sessionID.String(),
	}
	detailsJSON, _ := json.Marshal(auditDetails)
	_, _ = s.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, 'SESSION_REVOKED', $1, $2)
	`, userID, detailsJSON)

	return nil
}

// GetUser loads user details by UUID.
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*User, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	var user User
	query := `SELECT id, phone, status, created_at, updated_at FROM users WHERE id = $1`
	err := s.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Phone, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.Phone = MaskPhone(user.Phone)
	return &user, nil
}

// ValidateSession verifies that the session exists, belongs to userID, is not revoked, and is not expired.
func (s *Service) ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	if s.pool == nil {
		return errors.New("database pool unavailable")
	}

	var (
		dbUserID  uuid.UUID
		expiresAt time.Time
		revokedAt *time.Time
	)

	query := `
		SELECT user_id, expires_at, revoked_at
		FROM user_sessions
		WHERE id = $1
	`
	err := s.pool.QueryRow(ctx, query, sessionID).Scan(&dbUserID, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("failed to query session status: %w", err)
	}

	if dbUserID != userID {
		return ErrSessionMismatch
	}

	if revokedAt != nil {
		return ErrSessionRevoked
	}

	if time.Now().After(expiresAt) {
		return ErrSessionExpired
	}

	return nil
}

// VisitorPhoneVerificationDuration defines how long an OTP verification token remains valid for registration (15 min).
const VisitorPhoneVerificationDuration = 15 * time.Minute

// getVerificationHMACKey derives a key for phone verification HMAC tokens.
// Prioritizes explicit VerificationSecret; otherwise derives via HMAC domain separation
// from Ed25519 signing seed; otherwise falls back to a process-lifetime ephemeral secret.
// Never uses a static or guessable hardcoded string.
func (s *Service) getVerificationHMACKey() []byte {
	if s.config.VerificationSecret != "" {
		hash := sha256.Sum256([]byte(s.config.VerificationSecret))
		return hash[:]
	}
	if s.tokenManager != nil && len(s.tokenManager.privateKey) >= 32 {
		mac := hmac.New(sha256.New, []byte("chs_neu:visitor_verification_hmac_v1"))
		mac.Write(s.tokenManager.privateKey.Seed())
		return mac.Sum(nil)
	}
	if len(s.ephemeralSecret) >= 32 {
		return s.ephemeralSecret
	}
	return nil
}

// pruneExpiredNonces iterates consumed nonces and purges entries past their expiration timestamp,
// ensuring the in-memory map has a bounded memory lifecycle in long-running processes.
func (s *Service) pruneExpiredNonces(nowUnix int64) {
	s.consumedVerificationNonces.Range(func(key, value interface{}) bool {
		if exp, ok := value.(int64); ok && exp < nowUnix {
			s.consumedVerificationNonces.Delete(key)
		}
		return true
	})
}

// GeneratePhoneVerificationTokenWithContext issues an opaque HMAC-signed token attesting phone verification,
// cryptographically bound to a specific QR context and containing a single-use random nonce.
func (s *Service) GeneratePhoneVerificationTokenWithContext(phone, qrToken string) string {
	hmacKey := s.getVerificationHMACKey()
	if len(hmacKey) == 0 {
		return ""
	}

	expiresAt := time.Now().Add(VisitorPhoneVerificationDuration).Unix()

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		timestamp := time.Now().UnixNano()
		nonceBytes = []byte(fmt.Sprintf("%x%x", timestamp, expiresAt))
	}
	nonceHex := hex.EncodeToString(nonceBytes)

	qrHash := sha256.Sum256([]byte(qrToken))
	qrHashHex := hex.EncodeToString(qrHash[:8])

	payload := fmt.Sprintf("%s|%s|%s|%d", phone, qrHashHex, nonceHex, expiresAt)

	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	rawToken := fmt.Sprintf("%s|%s", payload, sig)
	return base64.URLEncoding.EncodeToString([]byte(rawToken))
}

// GeneratePhoneVerificationToken issues an opaque HMAC-signed token attesting phone verification (backward compatible).
func (s *Service) GeneratePhoneVerificationToken(phone string) string {
	return s.GeneratePhoneVerificationTokenWithContext(phone, "")
}

// ConsumePhoneVerificationToken validates that the token was signed by the server, has not expired,
// matches the submitted phone number and QR context, and has NOT been consumed in a prior registration.
// On success, it atomically records the token's nonce as consumed to prevent replay attacks.
func (s *Service) ConsumePhoneVerificationToken(rawPhone, qrToken, token string) (bool, string) {
	hmacKey := s.getVerificationHMACKey()
	if len(hmacKey) == 0 {
		return false, ""
	}

	normPhone, err := NormalizePhone(rawPhone)
	if err != nil {
		return false, ""
	}

	decodedBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false, ""
	}
	parts := strings.Split(string(decodedBytes), "|")
	nowUnix := time.Now().Unix()

	// Prune expired nonces to maintain bounded memory footprint
	s.pruneExpiredNonces(nowUnix)

	// 5-part format: phone|qrHashHex|nonceHex|expiresAt|sig
	if len(parts) == 5 {
		tokenPhone := parts[0]
		tokenQRHash := parts[1]
		tokenNonce := parts[2]
		expiryStr := parts[3]
		tokenSig := parts[4]

		expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
		if err != nil || nowUnix > expiryUnix {
			return false, "" // Expired or malformed
		}

		if tokenPhone != normPhone {
			return false, "" // Phone mismatch
		}

		// Verify QR context binding if token was generated with a QR token
		if tokenQRHash != "" {
			expectedQRHash := sha256.Sum256([]byte(qrToken))
			expectedQRHashHex := hex.EncodeToString(expectedQRHash[:8])
			if !hmac.Equal([]byte(tokenQRHash), []byte(expectedQRHashHex)) {
				return false, "" // QR context mismatch
			}
		}

		// Verify HMAC signature
		payload := fmt.Sprintf("%s|%s|%s|%s", tokenPhone, tokenQRHash, tokenNonce, expiryStr)
		mac := hmac.New(sha256.New, hmacKey)
		mac.Write([]byte(payload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(tokenSig), []byte(expectedSig)) {
			return false, "" // Invalid signature
		}

		// Enforce single-use consumption (Replay Protection)
		if _, loaded := s.consumedVerificationNonces.LoadOrStore(tokenNonce, expiryUnix); loaded {
			return false, "" // Already consumed (replay detected)
		}

		return true, normPhone
	}

	// Legacy 3-part format: phone|expiresAt|sig
	if len(parts) == 3 {
		tokenPhone := parts[0]
		expiryStr := parts[1]
		tokenSig := parts[2]

		expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
		if err != nil || nowUnix > expiryUnix {
			return false, ""
		}

		if tokenPhone != normPhone {
			return false, ""
		}

		payload := fmt.Sprintf("%s|%s", tokenPhone, expiryStr)
		mac := hmac.New(sha256.New, s.getVerificationHMACKey())
		mac.Write([]byte(payload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(tokenSig), []byte(expectedSig)) {
			return false, ""
		}

		// Enforce single-use consumption on legacy signature
		if _, loaded := s.consumedVerificationNonces.LoadOrStore(tokenSig, expiryUnix); loaded {
			return false, ""
		}

		return true, normPhone
	}

	return false, ""
}

// ValidatePhoneVerificationToken validates that the token was signed by the server, has not expired,
// and matches the submitted phone number without consuming the token.
func (s *Service) ValidatePhoneVerificationToken(rawPhone, token string) (bool, string) {
	hmacKey := s.getVerificationHMACKey()
	if len(hmacKey) == 0 {
		return false, ""
	}

	normPhone, err := NormalizePhone(rawPhone)
	if err != nil {
		return false, ""
	}

	decodedBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return false, ""
	}
	parts := strings.Split(string(decodedBytes), "|")
	nowUnix := time.Now().Unix()

	if len(parts) == 5 {
		tokenPhone := parts[0]
		tokenQRHash := parts[1]
		tokenNonce := parts[2]
		expiryStr := parts[3]
		tokenSig := parts[4]

		expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
		if err != nil || nowUnix > expiryUnix {
			return false, ""
		}

		if tokenPhone != normPhone {
			return false, ""
		}

		payload := fmt.Sprintf("%s|%s|%s|%s", tokenPhone, tokenQRHash, tokenNonce, expiryStr)
		mac := hmac.New(sha256.New, hmacKey)
		mac.Write([]byte(payload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(tokenSig), []byte(expectedSig)) {
			return false, ""
		}

		return true, normPhone
	}

	if len(parts) == 3 {
		tokenPhone := parts[0]
		expiryStr := parts[1]
		tokenSig := parts[2]

		expiryUnix, err := strconv.ParseInt(expiryStr, 10, 64)
		if err != nil || nowUnix > expiryUnix {
			return false, ""
		}

		if tokenPhone != normPhone {
			return false, ""
		}

		payload := fmt.Sprintf("%s|%s", tokenPhone, expiryStr)
		mac := hmac.New(sha256.New, hmacKey)
		mac.Write([]byte(payload))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(tokenSig), []byte(expectedSig)) {
			return false, ""
		}

		return true, normPhone
	}

	return false, ""
}

// VerifyVisitorOTPWithContext verifies candidate OTP code for public visitor self-registration
// without creating a permanent account or user session. Returns an HMAC phone verification token bound to qrToken.
func (s *Service) VerifyVisitorOTPWithContext(ctx context.Context, rawPhone, candidateOTP, qrToken, clientIP string) (string, string, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return "", "", err
	}

	if s.rateLimiter != nil {
		if err := s.rateLimiter.CheckVerifyIP(clientIP); err != nil {
			return "", "", err
		}
	}

	if s.pool == nil {
		return "", "", fmt.Errorf("database connection unavailable")
	}

	queryOTP := `
		SELECT id, otp_hash, attempts, expires_at, consumed_at
		FROM auth_otps
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	var (
		otpID      uuid.UUID
		storedHash string
		attempts   int
		expiresAt  time.Time
		consumedAt *time.Time
	)

	err = s.pool.QueryRow(ctx, queryOTP, phone).Scan(&otpID, &storedHash, &attempts, &expiresAt, &consumedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if s.rateLimiter != nil {
				s.rateLimiter.RecordVerifyFailureIP(clientIP)
			}
			return "", "", ErrOTPInvalid
		}
		return "", "", fmt.Errorf("failed to query OTP record: %w", err)
	}

	if consumedAt != nil {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return "", "", ErrOTPInvalid
	}

	if time.Now().After(expiresAt) {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return "", "", ErrOTPExpired
	}

	if attempts >= MaxOTPVerificationAttempts {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		return "", "", ErrOTPAttemptsExceeded
	}

	// Validate OTP Hash
	candidateHash := HashOTP(phone, candidateOTP)
	if candidateHash != storedHash {
		if s.rateLimiter != nil {
			s.rateLimiter.RecordVerifyFailureIP(clientIP)
		}
		_, _ = s.pool.Exec(ctx, `UPDATE auth_otps SET attempts = attempts + 1, updated_at = NOW() WHERE id = $1`, otpID)
		if attempts+1 >= MaxOTPVerificationAttempts {
			return "", "", ErrOTPAttemptsExceeded
		}
		return "", "", ErrOTPInvalid
	}

	// Reset IP verify failures on successful verification
	if s.rateLimiter != nil {
		s.rateLimiter.ResetVerifyIP(clientIP)
	}

	// Mark consumed
	_, _ = s.pool.Exec(ctx, `UPDATE auth_otps SET consumed_at = NOW(), updated_at = NOW() WHERE id = $1`, otpID)

	// Issue phone verification token bound to QR context
	verificationToken := s.GeneratePhoneVerificationTokenWithContext(phone, qrToken)
	maskedPhone := MaskPhone(phone)

	s.logger.Info("Visitor phone verified via OTP", slog.String("phone", maskedPhone))
	return verificationToken, maskedPhone, nil
}

// VerifyVisitorOTP verifies candidate OTP code for public visitor self-registration (backward compatible).
func (s *Service) VerifyVisitorOTP(ctx context.Context, rawPhone, candidateOTP, clientIP string) (string, string, error) {
	return s.VerifyVisitorOTPWithContext(ctx, rawPhone, candidateOTP, "", clientIP)
}

