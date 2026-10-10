package platform

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
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

func TestPlatformDatabaseIntegration(t *testing.T) {
	pool := getTestPool(t)
	repo := NewRepository(pool)
	ctx := context.Background()

	// 1. Test Super Admin Bootstrap
	t.Run("BootstrapSuperAdmin is idempotent and does not overwrite", func(t *testing.T) {
		testSuperAdminPhone := "+919967089101"

		role1, err := repo.BootstrapSuperAdmin(ctx, testSuperAdminPhone)
		if err != nil {
			t.Fatalf("bootstrap failed: %v", err)
		}
		if role1.Role != RoleSuperAdmin {
			t.Errorf("expected role %s, got %s", RoleSuperAdmin, role1.Role)
		}
		if role1.Status != StatusActive {
			t.Errorf("expected status %s, got %s", StatusActive, role1.Status)
		}

		// Idempotent re-run should return existing record without error
		role2, err := repo.BootstrapSuperAdmin(ctx, testSuperAdminPhone)
		if err != nil {
			t.Fatalf("second bootstrap failed: %v", err)
		}
		if role1.ID != role2.ID {
			t.Errorf("expected same role ID %s, got %s", role1.ID, role2.ID)
		}
	})

	// 2. Test Single Role Per User Constraint
	t.Run("Enforces strictly one platform role per user", func(t *testing.T) {
		// Create a test user
		var testUserID uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ('+919111222333', 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`).Scan(&testUserID)
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}

		// Insert first role
		_, err = pool.Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status)
			VALUES ($1, 'ADMIN', 'ACTIVE')
			ON CONFLICT (user_id) DO NOTHING
		`, testUserID)
		if err != nil {
			t.Fatalf("failed to insert initial role: %v", err)
		}

		// Attempt inserting a second role for the same user should fail UNIQUE constraint
		_, err = pool.Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status)
			VALUES ($1, 'SUPER_ADMIN', 'ACTIVE')
		`, testUserID)
		if err == nil {
			t.Errorf("expected UNIQUE constraint violation when assigning second platform role to user")
		}
	})

	// 3. Test Audit Log Immutability Trigger
	t.Run("Audit log trigger forbids UPDATE and DELETE", func(t *testing.T) {
		var actorID uuid.UUID
		err := pool.QueryRow(ctx, `SELECT id FROM users LIMIT 1`).Scan(&actorID)
		if err != nil {
			t.Fatalf("failed to fetch user: %v", err)
		}

		var auditID uuid.UUID
		err = pool.QueryRow(ctx, `
			INSERT INTO platform_audit_logs (actor_id, action, details)
			VALUES ($1, 'TEST_AUDIT', '{"test": true}')
			RETURNING id
		`, actorID).Scan(&auditID)
		if err != nil {
			t.Fatalf("failed to insert audit log: %v", err)
		}

		// Attempting UPDATE should be blocked by trigger
		_, err = pool.Exec(ctx, `UPDATE platform_audit_logs SET action = 'MODIFIED' WHERE id = $1`, auditID)
		if err == nil {
			t.Errorf("expected trigger exception on UPDATE of platform_audit_logs, got success")
		}

		// Attempting DELETE should be blocked by trigger
		_, err = pool.Exec(ctx, `DELETE FROM platform_audit_logs WHERE id = $1`, auditID)
		if err == nil {
			t.Errorf("expected trigger exception on DELETE of platform_audit_logs, got success")
		}
	})

	// 4. Test DeactivateAdmin: status disabled, sessions revoked, users.status stays active
	t.Run("DeactivateAdmin revokes sessions while keeping users.status ACTIVE", func(t *testing.T) {
		var superAdminID uuid.UUID
		err := pool.QueryRow(ctx, `
			SELECT user_id FROM platform_roles WHERE role = 'SUPER_ADMIN' LIMIT 1
		`).Scan(&superAdminID)
		if err != nil {
			t.Fatalf("no super admin found: %v", err)
		}

		// Create target admin
		var targetUserID uuid.UUID
		err = pool.QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ('+919444555666', 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`).Scan(&targetUserID)
		if err != nil {
			t.Fatalf("failed to create target user: %v", err)
		}

		// Assign ADMIN role
		_, err = pool.Exec(ctx, `
			INSERT INTO platform_roles (user_id, role, status, created_by)
			VALUES ($1, 'ADMIN', 'ACTIVE', $2)
			ON CONFLICT (user_id) DO UPDATE SET status = 'ACTIVE'
		`, targetUserID, superAdminID)
		if err != nil {
			t.Fatalf("failed to assign admin role: %v", err)
		}

		// Create an active session
		var sessionID uuid.UUID
		err = pool.QueryRow(ctx, `
			INSERT INTO user_sessions (user_id, refresh_token_hash, expires_at)
			VALUES ($1, 'dummy_hash_123', NOW() + INTERVAL '1 hour')
			RETURNING id
		`, targetUserID).Scan(&sessionID)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Execute DeactivateAdmin
		updated, err := repo.UpdateAdminStatus(ctx, superAdminID, targetUserID, StatusDisabled, "Security suspension", "127.0.0.1", "test-agent")
		if err != nil {
			t.Fatalf("DeactivateAdmin failed: %v", err)
		}
		if updated.Status != StatusDisabled {
			t.Errorf("expected role status %s, got %s", StatusDisabled, updated.Status)
		}

		// Verify session was revoked
		var revokedAt *time.Time
		err = pool.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id = $1`, sessionID).Scan(&revokedAt)
		if err != nil || revokedAt == nil {
			t.Errorf("expected session to be revoked, got err: %v, revokedAt: %v", err, revokedAt)
		}

		// Verify base identity users.status remains ACTIVE
		var userStatus string
		err = pool.QueryRow(ctx, `SELECT status FROM users WHERE id = $1`, targetUserID).Scan(&userStatus)
		if err != nil || userStatus != "ACTIVE" {
			t.Errorf("expected users.status to remain ACTIVE, got %s", userStatus)
		}
	})

	// 5. Test Transaction Rollback on Audit Failure
	t.Run("Audit failure rolls back administrative mutation", func(t *testing.T) {
		nonExistentActorID := uuid.New() // FK violation on platform_audit_logs.actor_id
		var candidateUserID uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO users (phone, status)
			VALUES ('+919999888777', 'ACTIVE')
			ON CONFLICT (phone) DO UPDATE SET status = 'ACTIVE'
			RETURNING id
		`).Scan(&candidateUserID)
		if err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		// Clean up any existing role
		_, _ = pool.Exec(ctx, `DELETE FROM platform_roles WHERE user_id = $1`, candidateUserID)

		// Attempt CreateAdmin with non-existent actorID to trigger FK violation during audit log write
		// Note: CreateAdmin inserts platform_roles first, then platform_audit_logs with actor_id
		// If audit insert fails due to FK violation, the transaction MUST rollback and platform_roles MUST NOT exist
		_, err = repo.CreateAdmin(ctx, nonExistentActorID, "+919999888777", "127.0.0.1", "test-agent")
		if err == nil {
			t.Fatalf("expected error due to invalid actor_id in audit log, got nil")
		}

		// Verify that platform_roles was NOT committed (rolled back)
		var roleCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM platform_roles WHERE user_id = $1`, candidateUserID).Scan(&roleCount)
		if err != nil {
			t.Fatalf("failed to query platform_roles: %v", err)
		}
		if roleCount != 0 {
			t.Errorf("expected 0 platform_roles after rollback, found %d", roleCount)
		}
	})
}
