package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// ServiceConfig contains runtime configuration for the authentication service.
type ServiceConfig struct {
	DevTestPhones []string
	DevTestOTP    string
}

// Service coordinates all authentication domain workflows.
type Service struct {
	pool         *pgxpool.Pool
	tokenManager *TokenManager
	smsProvider  sms.Provider
	rateLimiter  *RateLimiter
	config       ServiceConfig
	logger       *slog.Logger
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
	return &Service{
		pool:         pool,
		tokenManager: tm,
		smsProvider:  smsProvider,
		rateLimiter:  limiter,
		config:       cfg,
		logger:       logger,
	}
}

// SendOTP normalizes the phone number, checks rate limits, generates/stores the OTP, and dispatches SMS.
func (s *Service) SendOTP(ctx context.Context, rawPhone, clientIP string) (int, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return 0, err
	}

	// Rate limiting by IP
	if err := s.rateLimiter.CheckAndRecordIP(clientIP); err != nil {
		return 0, err
	}

	// Rate limiting by phone (cooldown & hourly cap)
	resendAfter, err := s.rateLimiter.CheckAndRecordPhone(phone)
	if err != nil {
		return resendAfter, err
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

	if isTestNumber && s.config.DevTestOTP != "" {
		otpCode = s.config.DevTestOTP
		s.logger.Info("Using configured development test OTP", slog.String("phone", MaskPhone(phone)))
	} else {
		generated, err := GenerateNumericOTP()
		if err != nil {
			return 0, fmt.Errorf("failed to generate secure OTP: %w", err)
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
			return 0, fmt.Errorf("failed to record OTP: %w", err)
		}
	}

	// Dispatch SMS
	if err := s.smsProvider.SendOTP(ctx, phone, otpCode); err != nil {
		s.logger.Error("SMS dispatch failed", slog.String("phone", MaskPhone(phone)), slog.String("error", err.Error()))
		return 0, fmt.Errorf("failed to deliver verification SMS")
	}

	s.logger.Info("OTP generated and dispatched", slog.String("phone", MaskPhone(phone)))
	return resendAfter, nil
}

// VerifyOTP verifies candidate code, provisions user (JIT), checks 2-device limit, and issues tokens.
func (s *Service) VerifyOTP(ctx context.Context, rawPhone, candidateOTP, userAgent, clientIP string) (*AuthResult, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return nil, err
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
			return nil, ErrOTPInvalid
		}
		return nil, fmt.Errorf("failed to query OTP record: %w", err)
	}

	if consumedAt != nil {
		return nil, ErrOTPInvalid
	}

	if time.Now().After(expiresAt) {
		return nil, ErrOTPExpired
	}

	if attempts >= MaxOTPVerificationAttempts {
		return nil, ErrOTPAttemptsExceeded
	}

	// Verify constant-time hash
	if !VerifyOTPHash(phone, candidateOTP, storedHash) {
		// Increment attempts
		_, _ = s.pool.Exec(ctx, `UPDATE auth_otps SET attempts = attempts + 1, updated_at = NOW() WHERE id = $1`, otpID)
		if attempts+1 >= MaxOTPVerificationAttempts {
			return nil, ErrOTPAttemptsExceeded
		}
		return nil, ErrOTPInvalid
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

// LogoutAll revokes all active sessions for a user.
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if s.pool == nil {
		return nil
	}
	query := `UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := s.pool.Exec(ctx, query, userID)
	return err
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
