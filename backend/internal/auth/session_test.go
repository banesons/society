package auth

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/sms"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := config.Load()
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping live database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	if !db.IsConnected() {
		t.Skip("database not connected; skipping")
	}

	return db.Pool()
}

func setupTestAuthService(t *testing.T, pool *pgxpool.Pool) (*Service, *sms.MockProvider, *TokenManager) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tm, err := NewTokenManager("")
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}
	smsProvider := sms.NewMockProvider(logger)
	rateLimiter := NewRateLimiter()

	svc := NewService(
		pool,
		tm,
		smsProvider,
		rateLimiter,
		ServiceConfig{
			DevTestPhones: []string{"+919967089101", "+918422862001"},
			DevTestOTP:    "123456",
		},
		logger,
	)

	return svc, smsProvider, tm
}

func cleanupUser(t *testing.T, pool *pgxpool.Pool, phone string) {
	t.Helper()
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM auth_otps WHERE phone = $1`, phone)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE phone = $1`, phone)
}

func countActiveSessions(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	ctx := context.Background()
	var count int
	query := `
		SELECT count(*)
		FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	err := pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count active sessions: %v", err)
	}
	return count
}

func sendAndVerifyOTP(t *testing.T, svc *Service, pool *pgxpool.Pool, phone, userAgent, ip string) *AuthResult {
	t.Helper()
	ctx := context.Background()

	otpCode := "123456"
	otpHash := HashOTP(phone, otpCode)
	expiresAt := time.Now().Add(5 * time.Minute)

	_, err := pool.Exec(ctx, `
		INSERT INTO auth_otps (phone, otp_hash, attempts, expires_at)
		VALUES ($1, $2, 0, $3)
	`, phone, otpHash, expiresAt)
	if err != nil {
		t.Fatalf("failed to seed OTP for %s: %v", phone, err)
	}

	res, err := svc.VerifyOTP(ctx, phone, otpCode, userAgent, ip)
	if err != nil {
		t.Fatalf("failed to verify OTP for %s: %v", phone, err)
	}
	return res
}

func TestFIFOSessionReplacement(t *testing.T) {
	pool := getTestPool(t)
	svc, smsMock, tm := setupTestAuthService(t, pool)
	ctx := context.Background()

	// A, B, C: Zero sessions, One session, Two sessions -> FIFO replacement
	t.Run("A_B_C: Zero, One, and Two sessions FIFO replacement lifecycle", func(t *testing.T) {
		phone := "+919876543001"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		// A: Zero existing sessions -> Login succeeds, 1 active session exists
		res1 := sendAndVerifyOTP(t, svc, pool, phone, "Device-1", "127.0.0.1")
		if res1.SessionReplaced {
			t.Errorf("expected SessionReplaced=false for first session")
		}
		if countActiveSessions(t, pool, res1.User.ID) != 1 {
			t.Fatalf("expected exactly 1 active session, got %d", countActiveSessions(t, pool, res1.User.ID))
		}

		// B: One existing session -> Login succeeds, 2 active sessions exist
		time.Sleep(10 * time.Millisecond) // ensure distinct created_at
		res2 := sendAndVerifyOTP(t, svc, pool, phone, "Device-2", "127.0.0.1")
		if res2.SessionReplaced {
			t.Errorf("expected SessionReplaced=false for second session")
		}
		if countActiveSessions(t, pool, res1.User.ID) != 2 {
			t.Fatalf("expected exactly 2 active sessions, got %d", countActiveSessions(t, pool, res1.User.ID))
		}

		// C: Two existing sessions -> 3rd login succeeds, oldest session revoked, exactly 2 active remain
		time.Sleep(10 * time.Millisecond)
		res3 := sendAndVerifyOTP(t, svc, pool, phone, "Device-3", "127.0.0.1")
		if !res3.SessionReplaced {
			t.Errorf("expected SessionReplaced=true for 3rd session (FIFO replacement)")
		}

		if countActiveSessions(t, pool, res1.User.ID) != 2 {
			t.Fatalf("expected exactly 2 active sessions after FIFO, got %d", countActiveSessions(t, pool, res1.User.ID))
		}

		// Verify that Device-1's session was revoked, while Device-2 and Device-3 remain active
		claims1, _ := tm.ValidateAccessToken(res1.AccessToken)
		claims2, _ := tm.ValidateAccessToken(res2.AccessToken)
		claims3, _ := tm.ValidateAccessToken(res3.AccessToken)

		var revokedAt1, revokedAt2, revokedAt3 *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claims1.SessionID).Scan(&revokedAt1)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claims2.SessionID).Scan(&revokedAt2)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claims3.SessionID).Scan(&revokedAt3)

		if revokedAt1 == nil {
			t.Errorf("expected session 1 (oldest) to be revoked")
		}
		if revokedAt2 != nil {
			t.Errorf("expected session 2 to remain active, but was revoked")
		}
		if revokedAt3 != nil {
			t.Errorf("expected session 3 (newest) to be active, but was revoked")
		}
	})

	// D: Legacy state with 3 or more active sessions safely restored to max 2
	t.Run("D: Legacy state with 3 active sessions restores to limit safely", func(t *testing.T) {
		phone := "+919876543002"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		// Create user
		var userID uuid.UUID
		err := pool.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id`, phone).Scan(&userID)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Insert 3 active sessions manually with sequential timestamps
		baseTime := time.Now().Add(-1 * time.Hour)
		var sIDs []uuid.UUID
		for i := 0; i < 3; i++ {
			var sid uuid.UUID
			tStamp := baseTime.Add(time.Duration(i*10) * time.Minute)
			err = pool.QueryRow(ctx, `
				INSERT INTO user_sessions (user_id, refresh_token_hash, user_agent, client_ip, expires_at, created_at)
				VALUES ($1, $2, $3, '127.0.0.1', $4, $5)
				RETURNING id
			`, userID, HashToken(uuid.NewString()), "LegacyDevice", time.Now().Add(RefreshTokenLifespan), tStamp).Scan(&sid)
			if err != nil {
				t.Fatalf("failed to insert legacy session: %v", err)
			}
			sIDs = append(sIDs, sid)
		}

		if countActiveSessions(t, pool, userID) != 3 {
			t.Fatalf("expected 3 legacy active sessions, got %d", countActiveSessions(t, pool, userID))
		}

		// Perform login
		res := sendAndVerifyOTP(t, svc, pool, phone, "NewDevice", "127.0.0.1")
		if !res.SessionReplaced {
			t.Errorf("expected SessionReplaced=true")
		}

		// Must restore to exactly 2 active sessions
		if countActiveSessions(t, pool, userID) != 2 {
			t.Fatalf("expected exactly 2 active sessions, got %d", countActiveSessions(t, pool, userID))
		}

		// The 2 oldest sessions (sIDs[0] and sIDs[1]) must be revoked
		var r0, r1, r2 *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, sIDs[0]).Scan(&r0)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, sIDs[1]).Scan(&r1)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, sIDs[2]).Scan(&r2)

		if r0 == nil || r1 == nil {
			t.Errorf("expected oldest 2 sessions to be revoked, got r0=%v, r1=%v", r0, r1)
		}
		if r2 != nil {
			t.Errorf("expected sIDs[2] to remain active, but was revoked")
		}
	})

	// E: Deterministic ordering with identical created_at uses id ASC tie-breaker
	t.Run("E: Deterministic ordering uses id ASC tie-breaker", func(t *testing.T) {
		phone := "+919876543003"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		var userID uuid.UUID
		err := pool.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id`, phone).Scan(&userID)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		sameTime := time.Now().Add(-10 * time.Minute)
		// Small UUID and Large UUID
		smallID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		largeID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

		_, err = pool.Exec(ctx, `
			INSERT INTO user_sessions (id, user_id, refresh_token_hash, user_agent, client_ip, expires_at, created_at)
			VALUES ($1, $2, $3, 'Tie-1', '127.0.0.1', $4, $5),
			       ($6, $2, $7, 'Tie-2', '127.0.0.1', $4, $5)
		`, smallID, userID, HashToken("h1"), time.Now().Add(RefreshTokenLifespan), sameTime, largeID, HashToken("h2"))
		if err != nil {
			t.Fatalf("failed to insert sessions with identical timestamps: %v", err)
		}

		res := sendAndVerifyOTP(t, svc, pool, phone, "NewTieDevice", "127.0.0.1")
		if !res.SessionReplaced {
			t.Errorf("expected SessionReplaced=true")
		}

		// smallID must be revoked, largeID must remain active
		var smallRevoked, largeRevoked *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, smallID).Scan(&smallRevoked)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, largeID).Scan(&largeRevoked)

		if smallRevoked == nil {
			t.Errorf("expected smaller UUID session to be revoked by tie-breaker")
		}
		if largeRevoked != nil {
			t.Errorf("expected larger UUID session to remain active")
		}
		if countActiveSessions(t, pool, userID) != 2 {
			t.Fatalf("expected exactly 2 active sessions, got %d", countActiveSessions(t, pool, userID))
		}
	})

	// F: Transaction rollback: simulated failure does not leave old session revoked
	t.Run("F: Transaction rollback preserves existing sessions", func(t *testing.T) {
		phone := "+919876543004"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		res1 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-1", "127.0.0.1")
		res2 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-2", "127.0.0.1")
		claims1, _ := tm.ValidateAccessToken(res1.AccessToken)

		// Simulate an atomic transaction attempt that revokes oldest session but encounters error before commit
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}
		// Revoke session 1
		_, err = tx.Exec(ctx, `UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`, claims1.SessionID)
		if err != nil {
			t.Fatalf("failed to exec update: %v", err)
		}
		// Explicitly rollback
		_ = tx.Rollback(ctx)

		// Verify session 1 is NOT revoked
		var revokedAt *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claims1.SessionID).Scan(&revokedAt)
		if revokedAt != nil {
			t.Errorf("expected session 1 to remain active after rollback, but revoked_at was set")
		}
		if countActiveSessions(t, pool, res2.User.ID) != 2 {
			t.Fatalf("expected 2 active sessions, got %d", countActiveSessions(t, pool, res2.User.ID))
		}
	})

	// G: Concurrency: simultaneous logins for the same user never exceed 2 active sessions
	t.Run("G: Concurrent logins strictly adhere to 2-session maximum", func(t *testing.T) {
		phone := "+919876543005"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		// Provision user first
		var userID uuid.UUID
		err := pool.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ($1, 'ACTIVE') RETURNING id`, phone).Scan(&userID)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Create 5 distinct valid OTP records
		const concurrentWorkers = 5
		type otpTask struct {
			otp  string
			hash string
		}
		var tasks []otpTask
		for i := 0; i < concurrentWorkers; i++ {
			code := "11111" + string(rune('0'+i))
			h := HashOTP(phone, code)
			tasks = append(tasks, otpTask{otp: code, hash: h})
			_, err = pool.Exec(ctx, `
				INSERT INTO auth_otps (phone, otp_hash, attempts, expires_at)
				VALUES ($1, $2, 0, $3)
			`, phone, h, time.Now().Add(5*time.Minute))
			if err != nil {
				t.Fatalf("failed to seed OTP: %v", err)
			}
		}

		var wg sync.WaitGroup
		wg.Add(concurrentWorkers)

		for i := 0; i < concurrentWorkers; i++ {
			go func(idx int) {
				defer wg.Done()
				_, _ = svc.VerifyOTP(context.Background(), phone, tasks[idx].otp, "ConcurrentWorker", "127.0.0.1")
			}(i)
		}

		wg.Wait()

		// Crucial verification: count active sessions must be <= 2
		finalActive := countActiveSessions(t, pool, userID)
		if finalActive > 2 {
			t.Fatalf("CONCURRENCY VIOLATION: active sessions exceeded limit: got %d, max is 2", finalActive)
		}
		if finalActive < 1 {
			t.Fatalf("expected at least 1 active session, got %d", finalActive)
		}
	})

	// H: User isolation: login for User A never revokes User B's sessions
	t.Run("H: User isolation across session management", func(t *testing.T) {
		phoneA := "+919876543006"
		phoneB := "+919876543007"
		cleanupUser(t, pool, phoneA)
		cleanupUser(t, pool, phoneB)
		defer cleanupUser(t, pool, phoneA)
		defer cleanupUser(t, pool, phoneB)

		// User A has 2 sessions
		resA1 := sendAndVerifyOTP(t, svc, pool, phoneA, "Dev-A1", "127.0.0.1")
		_ = sendAndVerifyOTP(t, svc, pool, phoneA, "Dev-A2", "127.0.0.1")

		// User B has 2 sessions
		resB1 := sendAndVerifyOTP(t, svc, pool, phoneB, "Dev-B1", "127.0.0.1")
		resB2 := sendAndVerifyOTP(t, svc, pool, phoneB, "Dev-B2", "127.0.0.1")

		claimsB1, _ := tm.ValidateAccessToken(resB1.AccessToken)
		claimsB2, _ := tm.ValidateAccessToken(resB2.AccessToken)

		// User A performs 3rd login (FIFO replacement for User A)
		resA3 := sendAndVerifyOTP(t, svc, pool, phoneA, "Dev-A3", "127.0.0.1")
		if !resA3.SessionReplaced {
			t.Errorf("expected User A session replaced")
		}

		// User A's oldest is revoked
		claimsA1, _ := tm.ValidateAccessToken(resA1.AccessToken)
		var revA1 *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claimsA1.SessionID).Scan(&revA1)
		if revA1 == nil {
			t.Errorf("expected User A1 session to be revoked")
		}

		// User B's sessions MUST remain completely untouched and active
		var revB1, revB2 *time.Time
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claimsB1.SessionID).Scan(&revB1)
		_ = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, claimsB2.SessionID).Scan(&revB2)
		if revB1 != nil || revB2 != nil {
			t.Errorf("User B sessions corrupted by User A login: revB1=%v, revB2=%v", revB1, revB2)
		}
		if countActiveSessions(t, pool, resB1.User.ID) != 2 {
			t.Errorf("expected User B to maintain 2 active sessions, got %d", countActiveSessions(t, pool, resB1.User.ID))
		}
	})

	// I: OTP semantics: Invalid, Expired, and Consumed OTPs rejected
	t.Run("I: OTP verification failure semantics", func(t *testing.T) {
		phone := "+919876543008"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		_, err := svc.SendOTP(ctx, phone, "127.0.0.1")
		if err != nil {
			t.Fatalf("failed to send OTP: %v", err)
		}
		correctOTP, _ := smsMock.GetLastOTP(phone)

		// 1. Invalid OTP
		_, err = svc.VerifyOTP(ctx, phone, "000000", "Dev", "127.0.0.1")
		if err != ErrOTPInvalid {
			t.Errorf("expected ErrOTPInvalid, got %v", err)
		}

		// 2. Correct OTP works
		res, err := svc.VerifyOTP(ctx, phone, correctOTP, "Dev", "127.0.0.1")
		if err != nil {
			t.Fatalf("expected verify success, got %v", err)
		}

		// 3. Consumed OTP reuse rejected
		_, err = svc.VerifyOTP(ctx, phone, correctOTP, "Dev", "127.0.0.1")
		if err != ErrOTPInvalid {
			t.Errorf("expected ErrOTPInvalid on consumed OTP reuse, got %v", err)
		}

		// 4. Expired OTP rejected
		expiredCode := "999999"
		_, _ = pool.Exec(ctx, `
			INSERT INTO auth_otps (phone, otp_hash, attempts, expires_at)
			VALUES ($1, $2, 0, $3)
		`, phone, HashOTP(phone, expiredCode), time.Now().Add(-10*time.Minute))

		_, err = svc.VerifyOTP(ctx, phone, expiredCode, "Dev", "127.0.0.1")
		if err != ErrOTPExpired {
			t.Errorf("expected ErrOTPExpired, got %v", err)
		}

		// User only has the 1 session from the valid verification
		if countActiveSessions(t, pool, res.User.ID) != 1 {
			t.Errorf("expected 1 active session, got %d", countActiveSessions(t, pool, res.User.ID))
		}
	})

	// J: Logout and refresh: revoked sessions cannot refresh, logout-all works
	t.Run("J: Refresh and logout semantics on revoked sessions", func(t *testing.T) {
		phone := "+919876543009"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		res1 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-1", "127.0.0.1")
		time.Sleep(10 * time.Millisecond)
		res2 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-2", "127.0.0.1")
		time.Sleep(10 * time.Millisecond)
		_ = sendAndVerifyOTP(t, svc, pool, phone, "Dev-3", "127.0.0.1") // causes res1 to be FIFO-revoked

		// Attempt to refresh res1 (FIFO revoked)
		_, err := svc.RefreshToken(ctx, res1.RefreshToken, "Dev-1", "127.0.0.1")
		if err != ErrSessionRevoked {
			t.Errorf("expected ErrSessionRevoked on FIFO-revoked session refresh, got %v", err)
		}

		// Refreshing res2 (active) succeeds and rotates token
		refRes, err := svc.RefreshToken(ctx, res2.RefreshToken, "Dev-2", "127.0.0.1")
		if err != nil {
			t.Fatalf("expected active session refresh to succeed, got %v", err)
		}
		if refRes.AccessToken == "" || refRes.RefreshToken == "" {
			t.Errorf("expected rotated tokens returned")
		}

		// Logout-all revokes all remaining sessions for user
		err = svc.LogoutAll(ctx, res1.User.ID)
		if err != nil {
			t.Fatalf("logout-all failed: %v", err)
		}
		if countActiveSessions(t, pool, res1.User.ID) != 0 {
			t.Errorf("expected 0 active sessions after logout-all, got %d", countActiveSessions(t, pool, res1.User.ID))
		}

		// Now refreshed token is also revoked and cannot be refreshed
		_, err = svc.RefreshToken(ctx, refRes.RefreshToken, "Dev-2", "127.0.0.1")
		if err != ErrSessionRevoked {
			t.Errorf("expected ErrSessionRevoked after logout-all, got %v", err)
		}
	})

	// K: Access JWT revocation hardening: FIFO evicted, logged out, and logout-all sessions rejected immediately
	t.Run("K: Hardened session revocation rejects existing access JWTs immediately", func(t *testing.T) {
		phone := "+919876543010"
		cleanupUser(t, pool, phone)
		defer cleanupUser(t, pool, phone)

		res1 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-1", "127.0.0.1")
		time.Sleep(10 * time.Millisecond)
		res2 := sendAndVerifyOTP(t, svc, pool, phone, "Dev-2", "127.0.0.1")
		time.Sleep(10 * time.Millisecond)
		_ = sendAndVerifyOTP(t, svc, pool, phone, "Dev-3", "127.0.0.1") // res1 FIFO revoked

		claims1, err := tm.ValidateAccessToken(res1.AccessToken)
		if err != nil {
			t.Fatalf("expected JWT signature to remain valid: %v", err)
		}
		claims2, err := tm.ValidateAccessToken(res2.AccessToken)
		if err != nil {
			t.Fatalf("expected JWT signature to remain valid: %v", err)
		}

		// 1. Active session (res2) succeeds active validation
		if err := svc.ValidateSession(ctx, res2.User.ID, claims2.SessionID); err != nil {
			t.Errorf("expected active session validation to succeed, got %v", err)
		}

		// 2. FIFO-evicted session (res1) is rejected immediately with ErrSessionRevoked
		if err := svc.ValidateSession(ctx, res1.User.ID, claims1.SessionID); err != ErrSessionRevoked {
			t.Errorf("expected FIFO-evicted session to return ErrSessionRevoked immediately, got %v", err)
		}

		// 3. User mismatch rejected
		randomUser := uuid.New()
		if err := svc.ValidateSession(ctx, randomUser, claims2.SessionID); err != ErrSessionMismatch {
			t.Errorf("expected ErrSessionMismatch for wrong user, got %v", err)
		}

		// 4. Unknown session rejected
		if err := svc.ValidateSession(ctx, res2.User.ID, uuid.New()); err != ErrSessionNotFound {
			t.Errorf("expected ErrSessionNotFound for nonexistent session, got %v", err)
		}

		// 5. Explicit logout revokes session immediately
		if err := svc.Logout(ctx, claims2.SessionID); err != nil {
			t.Fatalf("logout failed: %v", err)
		}
		if err := svc.ValidateSession(ctx, res2.User.ID, claims2.SessionID); err != ErrSessionRevoked {
			t.Errorf("expected logged-out session to return ErrSessionRevoked immediately, got %v", err)
		}
	})
}
