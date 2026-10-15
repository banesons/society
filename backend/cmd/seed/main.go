package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

const (
	ExpectedProjectRef = "glpjpqpbhdcmgmwlnkay"

	SuperAdminPhone    = "919967089101"
	PlatformAdminPhone = "918422862001"
)

func main() {
	confirmReset := flag.Bool("confirm-dev-reset", false, "Explicit safety confirmation to reset and reseed development database")
	flag.Parse()

	cfg := config.Load()

	// 1. Refuse to run in production environment
	if strings.ToLower(cfg.Env) == "production" || strings.ToLower(os.Getenv("NODE_ENV")) == "production" {
		log.Fatalf("FATAL: Development reset command refused to run in production environment (ENV=%s)", cfg.Env)
	}

	// 2. Validate confirmation flag
	if !*confirmReset {
		fmt.Println("================================================================================")
		fmt.Println("                   DEVELOPMENT DATABASE RESET SAFETY CHECK")
		fmt.Println("================================================================================")
		fmt.Println("This is a DESTRUCTIVE development reset operation.")
		fmt.Println("It will delete all existing demo societies, test users, units, gates, and")
		fmt.Println("members, and seed exactly:")
		fmt.Println("  - 1 Super Admin (+919967089101)")
		fmt.Println("  - 1 Platform Admin (+918422862001)")
		fmt.Println("  - 2 Societies (Shree Ram CHS [Mumbai], Greenview Residency [Pune])")
		fmt.Println("  - 37 Residential Units (25 in Mumbai, 12 in Pune)")
		fmt.Println("  - 3 Gates (2 in Mumbai, 1 in Pune)")
		fmt.Println("  - Committee appointments, shifts, and 3 stationed gatekeepers")
		fmt.Println()
		fmt.Println("To proceed, execute:")
		fmt.Println("  go run ./cmd/seed --confirm-dev-reset")
		fmt.Println("================================================================================")
		os.Exit(1)
	}

	// 3. Validate target database host/project
	if cfg.DatabaseURL == "" {
		log.Fatal("FATAL: DATABASE_URL is required")
	}

	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to parse DATABASE_URL: %v", err)
	}

	isMatchingTarget := strings.Contains(u.Hostname(), ExpectedProjectRef) ||
		(u.User != nil && strings.Contains(u.User.Username(), ExpectedProjectRef)) ||
		strings.Contains(cfg.DatabaseURL, ExpectedProjectRef)

	if !isMatchingTarget {
		log.Fatalf("FATAL: Target database does not match verified development project ref (%s). Aborting.", ExpectedProjectRef)
	}

	fmt.Printf("[PRE-FLIGHT] Verified Target: host=%s, env=%s, project_ref=%s\n", u.Hostname(), cfg.Env, ExpectedProjectRef)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 4. Begin single atomic transaction
	tx, err := db.Pool().Begin(ctx)
	if err != nil {
		log.Fatalf("FATAL: Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	fmt.Println("[RESET] Executing dependency-ordered teardown of application data...")

	teardownQueries := []string{
		// 1. Notifications (S15)
		"DELETE FROM notification_outbox",
		"DELETE FROM notification_deliveries",
		"DELETE FROM notifications",
		"DELETE FROM user_devices",

		// 2. Documents & Records (S13)
		"UPDATE society_documents SET current_version_id = NULL",
		"ALTER TABLE document_versions DISABLE TRIGGER trg_document_versions_immutable",
		"DELETE FROM document_versions",
		"ALTER TABLE document_versions ENABLE TRIGGER trg_document_versions_immutable",
		"DELETE FROM society_documents",
		"DELETE FROM document_categories",

		// 3. Maintenance Operations & Foundation (S11-S12)
		"DELETE FROM maintenance_receipts",
		"DELETE FROM maintenance_payment_allocations",
		"DELETE FROM maintenance_payments",
		"ALTER TABLE maintenance_bill_items DISABLE TRIGGER trg_bill_items_immutable",
		"DELETE FROM maintenance_bill_items",
		"ALTER TABLE maintenance_bill_items ENABLE TRIGGER trg_bill_items_immutable",
		"DELETE FROM maintenance_bills",
		"DELETE FROM maintenance_billing_periods",
		"DELETE FROM maintenance_charge_configs",
		"DELETE FROM maintenance_charge_categories",

		// 4. Communication & Announcements (S10)
		"DELETE FROM society_announcement_reads",
		"DELETE FROM society_announcements",

		// 5. Visitor Operations & Gate Events (S09)
		"DELETE FROM society_gate_events",
		"DELETE FROM society_visitor_passes",
		"DELETE FROM society_visit_approvals",
		"DELETE FROM society_visits",

		// 6. Gate Staffing (S08)
		"DELETE FROM society_gate_sessions",
		"DELETE FROM society_staffing_requirements",
		"DELETE FROM society_gate_shifts",
		"DELETE FROM society_gates",

		// 7. Designations & Governance (S07)
		"DELETE FROM society_committee_appointments",
		"DELETE FROM society_designations",

		// 8. Units, Wings, Buildings, Members, Roles (S05-S06)
		"DELETE FROM unit_relationships",
		"DELETE FROM society_units",
		"DELETE FROM society_wings",
		"DELETE FROM society_buildings",
		"DELETE FROM society_members",
		"DELETE FROM society_user_roles",

		// 9. Audit Logs (S02/S14 - clear before societies to avoid ON DELETE SET NULL update firing trigger)
		"ALTER TABLE platform_audit_logs DISABLE TRIGGER platform_audit_logs_immutability_trg",
		"TRUNCATE TABLE platform_audit_logs",
		"ALTER TABLE platform_audit_logs ENABLE TRIGGER platform_audit_logs_immutability_trg",

		// 10. Societies (S01)
		"DELETE FROM societies",

		// 11. Auth, Sessions, Platform Roles & Users (S00/S02)
		"DELETE FROM user_sessions",
		"DELETE FROM auth_otps",
		"DELETE FROM platform_roles",
		"DELETE FROM users",
	}

	for _, q := range teardownQueries {
		if _, err := tx.Exec(ctx, q); err != nil {
			log.Fatalf("FATAL: Teardown step failed on %q: %v", q, err)
		}
	}
	fmt.Println("[RESET] Teardown complete. All application tables cleared; canonical catalogues preserved.")

	// 5. Seed Platform Administrators
	fmt.Println("[SEED] Seeding Platform Administrators...")
	var superAdminID, platformAdminID uuid.UUID

	err = tx.QueryRow(ctx, `
		INSERT INTO users (phone, status)
		VALUES ($1, 'ACTIVE')
		RETURNING id
	`, SuperAdminPhone).Scan(&superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Super Admin user: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status, created_by)
		VALUES ($1, 'SUPER_ADMIN', 'ACTIVE', NULL)
	`, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to assign SUPER_ADMIN role: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO users (phone, status)
		VALUES ($1, 'ACTIVE')
		RETURNING id
	`, PlatformAdminPhone).Scan(&platformAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Platform Admin user: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO platform_roles (user_id, role, status, created_by)
		VALUES ($1, 'ADMIN', 'ACTIVE', $2)
	`, platformAdminID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to assign ADMIN role: %v", err)
	}

	// Transactional bootstrap audit logs
	superAuditDetails, _ := json.Marshal(map[string]interface{}{
		"phone": SuperAdminPhone,
		"role":  "SUPER_ADMIN",
		"seed":  true,
	})
	adminAuditDetails, _ := json.Marshal(map[string]interface{}{
		"phone":      PlatformAdminPhone,
		"role":       "ADMIN",
		"created_by": superAdminID.String(),
		"seed":       true,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details)
		VALUES ($1, 'SUPER_ADMIN_BOOTSTRAPPED', $1, $2::jsonb),
		       ($1, 'ADMIN_CREATED', $3, $4::jsonb)
	`, superAdminID, string(superAuditDetails), platformAdminID, string(adminAuditDetails))
	if err != nil {
		log.Fatalf("FATAL: Failed to insert admin bootstrap audit logs: %v", err)
	}

	fmt.Printf("[SEED] Super Admin (%s) & Platform Admin (%s) bootstrapped.\n", SuperAdminPhone, PlatformAdminPhone)

	// 6. Seed Society 1: Shree Ram Cooperative Housing Society (Mumbai)
	fmt.Println("[SEED] Seeding Society 1: Shree Ram Cooperative Housing Society (Mumbai)...")
	var soc1ID, bld1ID, wing1ID uuid.UUID
	soc1Code := "SOC-MUM-SRM101"

	err = tx.QueryRow(ctx, `
		INSERT INTO societies (
			society_code, name, registration_number,
			address_line1, locality, city, state, pincode,
			contact_phone, status, created_by, updated_by
		)
		VALUES ($1, 'Shree Ram Cooperative Housing Society', 'MUM/CHS/2026/001',
		        'Plot 42, Sector 17', 'Vashi', 'Mumbai', 'Maharashtra', '400703',
		        $2, 'ACTIVE', $3, $3)
		RETURNING id
	`, soc1Code, SuperAdminPhone, superAdminID).Scan(&soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Society 1: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, 'Building A', 'BLD-A', 5)
		RETURNING id
	`, soc1ID).Scan(&bld1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Building A: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_wings (society_id, building_id, name, wing_code, total_floors)
		VALUES ($1, $2, 'Wing A', 'WING-A', 5)
		RETURNING id
	`, soc1ID, bld1ID).Scan(&wing1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Wing A: %v", err)
	}

	// Exactly 25 units: A-101 .. A-505
	unitMapSoc1 := make(map[string]uuid.UUID)
	for floor := 1; floor <= 5; floor++ {
		for u := 1; u <= 5; u++ {
			unitNum := fmt.Sprintf("%d%02d", floor, u)
			unitIdent := fmt.Sprintf("A-%s", unitNum)
			var unitID uuid.UUID
			err = tx.QueryRow(ctx, `
				INSERT INTO society_units (
					society_id, building_id, wing_id, floor_number, unit_number, unit_identifier, unit_type, status
				)
				VALUES ($1, $2, $3, $4, $5, $6, 'RESIDENTIAL', 'ACTIVE')
				RETURNING id
			`, soc1ID, bld1ID, wing1ID, floor, unitNum, unitIdent).Scan(&unitID)
			if err != nil {
				log.Fatalf("FATAL: Failed to insert unit %s: %v", unitIdent, err)
			}
			unitMapSoc1[unitIdent] = unitID
		}
	}

	// 2 Gates for Society 1
	var gate1Soc1ID, gate2Soc1ID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO society_gates (
			society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description
		)
		VALUES ($1, 'Main Gate', 'GATE-01', 'VEHICLE_AND_PEDESTRIAN', 'ACTIVE', 'BIDIRECTIONAL', 'ACTIVE', 1, 'Primary entry and exit portal on Sector 17 road')
		RETURNING id
	`, soc1ID).Scan(&gate1Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Gate 1 for Society 1: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_gates (
			society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description
		)
		VALUES ($1, 'Service Gate', 'GATE-02', 'SERVICE', 'ACTIVE', 'BIDIRECTIONAL', 'ACTIVE', 1, 'Dedicated logistics, utility, and maintenance portal')
		RETURNING id
	`, soc1ID).Scan(&gate2Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Gate 2 for Society 1: %v", err)
	}

	// Staffing requirements
	_, err = tx.Exec(ctx, `
		INSERT INTO society_staffing_requirements (society_id, shifts_per_day, total_required_headcount, notes, updated_by)
		VALUES ($1, 3, 3, '2 guards day shift, 1 guard night shift across 2 operational gates', $2)
	`, soc1ID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert staffing requirements for Society 1: %v", err)
	}

	// Shifts for Society 1
	var shift1Soc1ID, shift2Soc1ID, shift3Soc1ID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, 'Morning Shift', '06:00:00', '14:00:00', true)
		RETURNING id
	`, soc1ID).Scan(&shift1Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Morning Shift: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, 'Afternoon Shift', '14:00:00', '22:00:00', true)
		RETURNING id
	`, soc1ID).Scan(&shift2Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Afternoon Shift: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, 'Night Shift', '22:00:00', '06:00:00', true)
		RETURNING id
	`, soc1ID).Scan(&shift3Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Night Shift: %v", err)
	}

	// 2 Gatekeeper Accounts for Society 1
	var guard1ID, guard2ID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ('919820010001', 'ACTIVE') RETURNING id`).Scan(&guard1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Guard 1: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by) VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)`, soc1ID, guard1ID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to assign Guard 1 role: %v", err)
	}

	err = tx.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ('919820010002', 'ACTIVE') RETURNING id`).Scan(&guard2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Guard 2: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by) VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)`, soc1ID, guard2ID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to assign Guard 2 role: %v", err)
	}

	// Station guards on active duty (Main Gate + Service Gate)
	_, err = tx.Exec(ctx, `
		INSERT INTO society_gate_sessions (society_id, gate_id, user_id, shift_id, started_at, status, handover_notes)
		VALUES ($1, $2, $3, $4, NOW() - INTERVAL '2 hours', 'ACTIVE', 'Morning perimeter check complete. All bollards functional.')
	`, soc1ID, gate1Soc1ID, guard1ID, shift1Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create duty session for Guard 1: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO society_gate_sessions (society_id, gate_id, user_id, shift_id, started_at, status, handover_notes)
		VALUES ($1, $2, $3, $4, NOW() - INTERVAL '1 hour', 'ACTIVE', 'Delivery bay operational. Service entrance clear.')
	`, soc1ID, gate2Soc1ID, guard2ID, shift1Soc1ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create duty session for Guard 2: %v", err)
	}

	// Default Designations for Society 1
	desigMapSoc1 := seedDesignations(ctx, tx, soc1ID)

	// Seed Members and Relationships for Society 1
	seedSociety1Households(ctx, tx, soc1ID, unitMapSoc1, desigMapSoc1)

	// 7. Seed Society 2: Greenview Residency (Pune)
	fmt.Println("[SEED] Seeding Society 2: Greenview Residency (Pune)...")
	var soc2ID, bld2ID, wing2ID uuid.UUID
	soc2Code := "SOC-PUN-GVR202"

	err = tx.QueryRow(ctx, `
		INSERT INTO societies (
			society_code, name, registration_number,
			address_line1, locality, city, state, pincode,
			contact_phone, status, created_by, updated_by
		)
		VALUES ($1, 'Greenview Residency', 'PUN/CHS/2026/002',
		        'Baner Road, Survey 88', 'Baner', 'Pune', 'Maharashtra', '411045',
		        $2, 'ACTIVE', $3, $3)
		RETURNING id
	`, soc2Code, SuperAdminPhone, superAdminID).Scan(&soc2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Society 2: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, 'Building 1', 'BLD-1', 3)
		RETURNING id
	`, soc2ID).Scan(&bld2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Building 1: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_wings (society_id, building_id, name, wing_code, total_floors)
		VALUES ($1, $2, 'Wing 1', 'WING-1', 3)
		RETURNING id
	`, soc2ID, bld2ID).Scan(&wing2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Wing 1: %v", err)
	}

	// Exactly 12 units: 101 .. 304
	unitMapSoc2 := make(map[string]uuid.UUID)
	for floor := 1; floor <= 3; floor++ {
		for u := 1; u <= 4; u++ {
			unitNum := fmt.Sprintf("%d%02d", floor, u)
			unitIdent := unitNum
			var unitID uuid.UUID
			err = tx.QueryRow(ctx, `
				INSERT INTO society_units (
					society_id, building_id, wing_id, floor_number, unit_number, unit_identifier, unit_type, status
				)
				VALUES ($1, $2, $3, $4, $5, $6, 'RESIDENTIAL', 'ACTIVE')
				RETURNING id
			`, soc2ID, bld2ID, wing2ID, floor, unitNum, unitIdent).Scan(&unitID)
			if err != nil {
				log.Fatalf("FATAL: Failed to insert unit %s: %v", unitIdent, err)
			}
			unitMapSoc2[unitIdent] = unitID
		}
	}

	// 1 Gate for Society 2
	var gate1Soc2ID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO society_gates (
			society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description
		)
		VALUES ($1, 'Main Gate', 'GATE-01', 'VEHICLE_AND_PEDESTRIAN', 'ACTIVE', 'BIDIRECTIONAL', 'ACTIVE', 1, 'Single access gate on Baner Road')
		RETURNING id
	`, soc2ID).Scan(&gate1Soc2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Gate 1 for Society 2: %v", err)
	}

	// Staffing requirements
	_, err = tx.Exec(ctx, `
		INSERT INTO society_staffing_requirements (society_id, shifts_per_day, total_required_headcount, notes, updated_by)
		VALUES ($1, 2, 2, '1 guard per shift across 2 operational shifts', $2)
	`, soc2ID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert staffing requirements for Society 2: %v", err)
	}

	// Shifts for Society 2
	var shift1Soc2ID, shift2Soc2ID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, 'General Shift', '08:00:00', '20:00:00', true)
		RETURNING id
	`, soc2ID).Scan(&shift1Soc2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert General Shift: %v", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO society_gate_shifts (society_id, name, start_time, end_time, is_active)
		VALUES ($1, 'Night Shift', '20:00:00', '08:00:00', true)
		RETURNING id
	`, soc2ID).Scan(&shift2Soc2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert Night Shift: %v", err)
	}

	// 1 Gatekeeper Account for Society 2
	var guard3ID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO users (phone, status) VALUES ('919820020001', 'ACTIVE') RETURNING id`).Scan(&guard3ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create Guard 3: %v", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO society_user_roles (society_id, user_id, role, status, assigned_by) VALUES ($1, $2, 'GATEKEEPER', 'ACTIVE', $3)`, soc2ID, guard3ID, superAdminID)
	if err != nil {
		log.Fatalf("FATAL: Failed to assign Guard 3 role: %v", err)
	}

	// Station guard on active duty (Main Gate)
	_, err = tx.Exec(ctx, `
		INSERT INTO society_gate_sessions (society_id, gate_id, user_id, shift_id, started_at, status, handover_notes)
		VALUES ($1, $2, $3, $4, NOW() - INTERVAL '30 minutes', 'ACTIVE', 'General shift started. Visitor barrier calibrated.')
	`, soc2ID, gate1Soc2ID, guard3ID, shift1Soc2ID)
	if err != nil {
		log.Fatalf("FATAL: Failed to create duty session for Guard 3: %v", err)
	}

	// Default Designations for Society 2
	desigMapSoc2 := seedDesignations(ctx, tx, soc2ID)

	// Seed Members and Relationships for Society 2
	seedSociety2Households(ctx, tx, soc2ID, unitMapSoc2, desigMapSoc2)

	// 8. Assert Database Invariants
	fmt.Println("[VALIDATION] Asserting database invariants...")
	assertCount(ctx, tx, "users", 5) // 2 Admins + 3 Guards
	assertCount(ctx, tx, "platform_roles", 2)
	assertCount(ctx, tx, "societies", 2)
	assertCount(ctx, tx, "society_buildings", 2)
	assertCount(ctx, tx, "society_wings", 2)
	assertCount(ctx, tx, "society_units", 37)
	assertCount(ctx, tx, "society_gates", 3)
	assertCount(ctx, tx, "society_gate_sessions", 3)
	assertCount(ctx, tx, "society_suspension_reasons", 8)

	// Verify Super Admin & Admin roles
	var saRole, paRole string
	_ = tx.QueryRow(ctx, `SELECT pr.role FROM platform_roles pr JOIN users u ON pr.user_id = u.id WHERE u.phone = $1`, SuperAdminPhone).Scan(&saRole)
	_ = tx.QueryRow(ctx, `SELECT pr.role FROM platform_roles pr JOIN users u ON pr.user_id = u.id WHERE u.phone = $1`, PlatformAdminPhone).Scan(&paRole)
	if saRole != "SUPER_ADMIN" || paRole != "ADMIN" {
		log.Fatalf("FATAL: Invariant violation: Expected SUPER_ADMIN and ADMIN, got %s and %s", saRole, paRole)
	}

	// Verify society unit counts
	var uCountSoc1, uCountSoc2 int
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM society_units WHERE society_id = $1`, soc1ID).Scan(&uCountSoc1)
	if err != nil {
		log.Fatalf("FATAL: Failed to query unit count for Society 1: %v", err)
	}
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM society_units WHERE society_id = $1`, soc2ID).Scan(&uCountSoc2)
	if err != nil {
		log.Fatalf("FATAL: Failed to query unit count for Society 2: %v", err)
	}
	if uCountSoc1 != 25 || uCountSoc2 != 12 {
		log.Fatalf("FATAL: Invariant violation: Expected 25 and 12 units, got %d and %d", uCountSoc1, uCountSoc2)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("FATAL: Transaction commit failed: %v", err)
	}

	fmt.Println("================================================================================")
	fmt.Println("             DEVELOPMENT DATABASE RESET & SEED SUCCESSFUL")
	fmt.Println("================================================================================")
	fmt.Printf("Platform Identities : 2 (Super Admin: +%s, Platform Admin: +%s)\n", SuperAdminPhone, PlatformAdminPhone)
	fmt.Printf("Societies           : 2 (Shree Ram CHS [25 units], Greenview Residency [12 units])\n")
	fmt.Printf("Total Units         : 37\n")
	fmt.Printf("Gates               : 3 (Shree Ram CHS: 2, Greenview: 1)\n")
	fmt.Printf("Active Guard Duty   : 3 active duty sessions (0 double-booked)\n")
	fmt.Printf("Reference Data      : 8 canonical suspension reasons preserved\n")
	fmt.Println("================================================================================")
}

func seedDesignations(ctx context.Context, tx pgx.Tx, societyID uuid.UUID) map[string]uuid.UUID {
	defaults := []struct {
		code           string
		name           string
		description    string
		allowsMulti    bool
		allowsMultiPos bool
	}{
		{
			code:           "CHAIRMAN",
			name:           "Chairman",
			description:    "Presiding officer and head of the managing committee",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           "SECRETARY",
			name:           "Secretary",
			description:    "Chief administrative officer responsible for executive management and records",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           "TREASURER",
			name:           "Treasurer",
			description:    "Financial officer responsible for accounts, collections, and disbursements",
			allowsMulti:    false,
			allowsMultiPos: true,
		},
		{
			code:           "COMMITTEE_MEMBER",
			name:           "Committee Member",
			description:    "Elected managing committee member representing society residents",
			allowsMulti:    true,
			allowsMultiPos: true,
		},
	}

	res := make(map[string]uuid.UUID)
	for _, d := range defaults {
		var dID uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO society_designations (
				society_id, code, name, description, allows_multiple_members, allows_multiple_positions_per_member, is_active
			)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			RETURNING id
		`, societyID, d.code, d.name, d.description, d.allowsMulti, d.allowsMultiPos).Scan(&dID)
		if err != nil {
			log.Fatalf("FATAL: Failed to insert designation %s: %v", d.code, err)
		}
		res[d.code] = dID
	}
	return res
}

type memberSpec struct {
	firstName string
	lastName  string
	phone     string
	email     string
}

type relSpec struct {
	relType  string
	share    *float64
	occupant bool
}

func seedSociety1Households(ctx context.Context, tx pgx.Tx, societyID uuid.UUID, unitMap map[string]uuid.UUID, desigMap map[string]uuid.UUID) {
	floatPtr := func(v float64) *float64 { return &v }

	// Seed Committee Leaders first
	// A-101: Chairman Rajesh Sharma
	mChairmanID := insertMember(ctx, tx, societyID, memberSpec{"Rajesh", "Sharma", "919811000101", "rajesh.sharma@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-101"], mChairmanID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	mSunitaID := insertMember(ctx, tx, societyID, memberSpec{"Sunita", "Sharma", "919811000102", "sunita.sharma@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-101"], mSunitaID, relSpec{"OWNER_FAMILY", nil, true})
	insertAppointment(ctx, tx, societyID, desigMap["CHAIRMAN"], mChairmanID)

	// A-102: Secretary Vikram Deshmukh
	mSecretaryID := insertMember(ctx, tx, societyID, memberSpec{"Vikram", "Deshmukh", "919811000103", "vikram.deshmukh@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-102"], mSecretaryID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	mNehaID := insertMember(ctx, tx, societyID, memberSpec{"Neha", "Deshmukh", "919811000104", "neha.deshmukh@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-102"], mNehaID, relSpec{"OWNER_FAMILY", nil, true})
	insertAppointment(ctx, tx, societyID, desigMap["SECRETARY"], mSecretaryID)

	// A-103: Treasurer Amit Kulkarni
	mTreasurerID := insertMember(ctx, tx, societyID, memberSpec{"Amit", "Kulkarni", "919811000105", "amit.kulkarni@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-103"], mTreasurerID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	mPoojaID := insertMember(ctx, tx, societyID, memberSpec{"Pooja", "Kulkarni", "919811000106", "pooja.kulkarni@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-103"], mPoojaID, relSpec{"OWNER_FAMILY", nil, true})
	insertAppointment(ctx, tx, societyID, desigMap["TREASURER"], mTreasurerID)

	// A-104: Committee Member Priya Nair
	mPriyaID := insertMember(ctx, tx, societyID, memberSpec{"Priya", "Nair", "919811000107", "priya.nair@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-104"], mPriyaID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertAppointment(ctx, tx, societyID, desigMap["COMMITTEE_MEMBER"], mPriyaID)

	// A-105: Committee Member Anand Joshi (Joint Owner with Meera Joshi)
	mAnandID := insertMember(ctx, tx, societyID, memberSpec{"Anand", "Joshi", "919811000108", "anand.joshi@example.com"})
	mMeeraID := insertMember(ctx, tx, societyID, memberSpec{"Meera", "Joshi", "919811000109", "meera.joshi@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-105"], mAnandID, relSpec{"PRIMARY_OWNER", floatPtr(50.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-105"], mMeeraID, relSpec{"SECONDARY_OWNER", floatPtr(50.0), true})
	insertAppointment(ctx, tx, societyID, desigMap["COMMITTEE_MEMBER"], mAnandID)

	// A-201: Sunil Mehta (Family Unit)
	mSunilID := insertMember(ctx, tx, societyID, memberSpec{"Sunil", "Mehta", "919811000110", "sunil.mehta@example.com"})
	mKavitaID := insertMember(ctx, tx, societyID, memberSpec{"Kavita", "Mehta", "919811000111", "kavita.mehta@example.com"})
	mRohanID := insertMember(ctx, tx, societyID, memberSpec{"Rohan", "Mehta", "919811000112", "rohan.mehta@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-201"], mSunilID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-201"], mKavitaID, relSpec{"OWNER_FAMILY", nil, true})
	insertRel(ctx, tx, societyID, unitMap["A-201"], mRohanID, relSpec{"OWNER_FAMILY", nil, true})

	// A-202: Tenant Unit (Absentee Owner Arvind Shah, Tenants Rahul & Sneha Gupta)
	mArvindID := insertMember(ctx, tx, societyID, memberSpec{"Arvind", "Shah", "919811000113", "arvind.shah@example.com"})
	mRahulGID := insertMember(ctx, tx, societyID, memberSpec{"Rahul", "Gupta", "919811000114", "rahul.gupta@example.com"})
	mSnehaGID := insertMember(ctx, tx, societyID, memberSpec{"Sneha", "Gupta", "919811000115", "sneha.gupta@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-202"], mArvindID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["A-202"], mRahulGID, relSpec{"PRIMARY_TENANT", nil, true})
	insertRel(ctx, tx, societyID, unitMap["A-202"], mSnehaGID, relSpec{"TENANT_FAMILY", nil, true})

	// A-203: Dilip Verma (Single Owner)
	mDilipID := insertMember(ctx, tx, societyID, memberSpec{"Dilip", "Verma", "919811000116", "dilip.verma@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-203"], mDilipID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// A-204: Nitin & Swati Gadre (Joint Owners 60/40)
	mNitinID := insertMember(ctx, tx, societyID, memberSpec{"Nitin", "Gadre", "919811000117", "nitin.gadre@example.com"})
	mSwatiID := insertMember(ctx, tx, societyID, memberSpec{"Swati", "Gadre", "919811000118", "swati.gadre@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-204"], mNitinID, relSpec{"PRIMARY_OWNER", floatPtr(60.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-204"], mSwatiID, relSpec{"SECONDARY_OWNER", floatPtr(40.0), true})

	// A-205: Vacant Unit (Intentionally no relationships)

	// A-301: Sanjay Rao & Ananya Rao
	mSanjayID := insertMember(ctx, tx, societyID, memberSpec{"Sanjay", "Rao", "919811000119", "sanjay.rao@example.com"})
	mAnanyaID := insertMember(ctx, tx, societyID, memberSpec{"Ananya", "Rao", "919811000120", "ananya.rao@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-301"], mSanjayID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-301"], mAnanyaID, relSpec{"OWNER_FAMILY", nil, true})

	// A-302: Tenant Unit (Absentee Owner Vivek Shastry, Tenants Alok Kumar & Manoj Singh)
	mVivekID := insertMember(ctx, tx, societyID, memberSpec{"Vivek", "Shastry", "919811000121", "vivek.shastry@example.com"})
	mAlokID := insertMember(ctx, tx, societyID, memberSpec{"Alok", "Kumar", "919811000122", "alok.kumar@example.com"})
	mManojID := insertMember(ctx, tx, societyID, memberSpec{"Manoj", "Singh", "919811000123", "manoj.singh@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-302"], mVivekID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["A-302"], mAlokID, relSpec{"PRIMARY_TENANT", nil, true})
	insertRel(ctx, tx, societyID, unitMap["A-302"], mManojID, relSpec{"SECONDARY_TENANT", nil, true})

	// A-303: Mahesh & Ritu Bhatt
	mMaheshID := insertMember(ctx, tx, societyID, memberSpec{"Mahesh", "Bhatt", "919811000124", "mahesh.bhatt@example.com"})
	mRituID := insertMember(ctx, tx, societyID, memberSpec{"Ritu", "Bhatt", "919811000125", "ritu.bhatt@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-303"], mMaheshID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-303"], mRituID, relSpec{"OWNER_FAMILY", nil, true})

	// A-304: Ashok Chavan & Authorized Occupant Deepak Chavan
	mAshokID := insertMember(ctx, tx, societyID, memberSpec{"Ashok", "Chavan", "919811000126", "ashok.chavan@example.com"})
	mDeepakID := insertMember(ctx, tx, societyID, memberSpec{"Deepak", "Chavan", "919811000127", "deepak.chavan@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-304"], mAshokID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-304"], mDeepakID, relSpec{"AUTHORIZED_OCCUPANT", nil, true})

	// A-305: Vacant Unit

	// A-401: Pradeep Sawant
	mPradeepID := insertMember(ctx, tx, societyID, memberSpec{"Pradeep", "Sawant", "919811000128", "pradeep.sawant@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-401"], mPradeepID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// A-402: Sandeep & Rashmi Tawde
	mSandeepID := insertMember(ctx, tx, societyID, memberSpec{"Sandeep", "Tawde", "919811000129", "sandeep.tawde@example.com"})
	mRashmiID := insertMember(ctx, tx, societyID, memberSpec{"Rashmi", "Tawde", "919811000130", "rashmi.tawde@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-402"], mSandeepID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-402"], mRashmiID, relSpec{"OWNER_FAMILY", nil, true})

	// A-403: Tenant Unit (Absentee Owner Bharat Gokhale, Tenant Varun Kapoor)
	mBharatID := insertMember(ctx, tx, societyID, memberSpec{"Bharat", "Gokhale", "919811000131", "bharat.gokhale@example.com"})
	mVarunID := insertMember(ctx, tx, societyID, memberSpec{"Varun", "Kapoor", "919811000132", "varun.kapoor@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-403"], mBharatID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["A-403"], mVarunID, relSpec{"PRIMARY_TENANT", nil, true})

	// A-404: Hemant & Archana Rane (Joint Owners 50/50)
	mHemantID := insertMember(ctx, tx, societyID, memberSpec{"Hemant", "Rane", "919811000133", "hemant.rane@example.com"})
	mArchanaID := insertMember(ctx, tx, societyID, memberSpec{"Archana", "Rane", "919811000134", "archana.rane@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-404"], mHemantID, relSpec{"PRIMARY_OWNER", floatPtr(50.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-404"], mArchanaID, relSpec{"SECONDARY_OWNER", floatPtr(50.0), true})

	// A-405: Vacant Unit

	// A-501: Chetan Merchant
	mChetanID := insertMember(ctx, tx, societyID, memberSpec{"Chetan", "Merchant", "919811000135", "chetan.merchant@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-501"], mChetanID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// A-502: Kishore & Geeta Vora
	mKishoreID := insertMember(ctx, tx, societyID, memberSpec{"Kishore", "Vora", "919811000136", "kishore.vora@example.com"})
	mGeetaID := insertMember(ctx, tx, societyID, memberSpec{"Geeta", "Vora", "919811000137", "geeta.vora@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-502"], mKishoreID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["A-502"], mGeetaID, relSpec{"OWNER_FAMILY", nil, true})

	// A-503: Tenant Unit (Absentee Owner Jayant Doshi, Tenants Tanya & Rohit Sen)
	mJayantID := insertMember(ctx, tx, societyID, memberSpec{"Jayant", "Doshi", "919811000138", "jayant.doshi@example.com"})
	mTanyaID := insertMember(ctx, tx, societyID, memberSpec{"Tanya", "Sen", "919811000139", "tanya.sen@example.com"})
	mRohitID := insertMember(ctx, tx, societyID, memberSpec{"Rohit", "Sen", "919811000140", "rohit.sen@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-503"], mJayantID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["A-503"], mTanyaID, relSpec{"PRIMARY_TENANT", nil, true})
	insertRel(ctx, tx, societyID, unitMap["A-503"], mRohitID, relSpec{"TENANT_FAMILY", nil, true})

	// A-504: Satish Pai
	mSatishID := insertMember(ctx, tx, societyID, memberSpec{"Satish", "Pai", "919811000141", "satish.pai@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-504"], mSatishID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// A-505: Nikhil Kamat
	mNikhilID := insertMember(ctx, tx, societyID, memberSpec{"Nikhil", "Kamat", "919811000142", "nikhil.kamat@example.com"})
	insertRel(ctx, tx, societyID, unitMap["A-505"], mNikhilID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
}

func seedSociety2Households(ctx context.Context, tx pgx.Tx, societyID uuid.UUID, unitMap map[string]uuid.UUID, desigMap map[string]uuid.UUID) {
	floatPtr := func(v float64) *float64 { return &v }

	// 101: Chairman Vijay Godbole
	mVijayID := insertMember(ctx, tx, societyID, memberSpec{"Vijay", "Godbole", "919822000101", "vijay.godbole@example.com"})
	mShrutiID := insertMember(ctx, tx, societyID, memberSpec{"Shruti", "Godbole", "919822000102", "shruti.godbole@example.com"})
	insertRel(ctx, tx, societyID, unitMap["101"], mVijayID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["101"], mShrutiID, relSpec{"OWNER_FAMILY", nil, true})
	insertAppointment(ctx, tx, societyID, desigMap["CHAIRMAN"], mVijayID)

	// 102: Secretary Milind Date
	mMilindID := insertMember(ctx, tx, societyID, memberSpec{"Milind", "Date", "919822000103", "milind.date@example.com"})
	insertRel(ctx, tx, societyID, unitMap["102"], mMilindID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertAppointment(ctx, tx, societyID, desigMap["SECRETARY"], mMilindID)

	// 103: Treasurer Sudhir Apte
	mSudhirID := insertMember(ctx, tx, societyID, memberSpec{"Sudhir", "Apte", "919822000104", "sudhir.apte@example.com"})
	mVandanaID := insertMember(ctx, tx, societyID, memberSpec{"Vandana", "Apte", "919822000105", "vandana.apte@example.com"})
	insertRel(ctx, tx, societyID, unitMap["103"], mSudhirID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["103"], mVandanaID, relSpec{"OWNER_FAMILY", nil, true})
	insertAppointment(ctx, tx, societyID, desigMap["TREASURER"], mSudhirID)

	// 104: Committee Member Rekha Chitnis
	mRekhaID := insertMember(ctx, tx, societyID, memberSpec{"Rekha", "Chitnis", "919822000106", "rekha.chitnis@example.com"})
	insertRel(ctx, tx, societyID, unitMap["104"], mRekhaID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertAppointment(ctx, tx, societyID, desigMap["COMMITTEE_MEMBER"], mRekhaID)

	// 201: Prasad & Smita Phadke (Joint Owners 50/50)
	mPrasadID := insertMember(ctx, tx, societyID, memberSpec{"Prasad", "Phadke", "919822000107", "prasad.phadke@example.com"})
	mSmitaID := insertMember(ctx, tx, societyID, memberSpec{"Smita", "Phadke", "919822000108", "smita.phadke@example.com"})
	insertRel(ctx, tx, societyID, unitMap["201"], mPrasadID, relSpec{"PRIMARY_OWNER", floatPtr(50.0), true})
	insertRel(ctx, tx, societyID, unitMap["201"], mSmitaID, relSpec{"SECONDARY_OWNER", floatPtr(50.0), true})

	// 202: Tenant Unit (Absentee Owner Ajit Kanitkar, Tenant Siddharth Roy)
	mAjitID := insertMember(ctx, tx, societyID, memberSpec{"Ajit", "Kanitkar", "919822000109", "ajit.kanitkar@example.com"})
	mSiddharthID := insertMember(ctx, tx, societyID, memberSpec{"Siddharth", "Roy", "919822000110", "siddharth.roy@example.com"})
	insertRel(ctx, tx, societyID, unitMap["202"], mAjitID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["202"], mSiddharthID, relSpec{"PRIMARY_TENANT", nil, true})

	// 203: Shrikant Oak
	mShrikantID := insertMember(ctx, tx, societyID, memberSpec{"Shrikant", "Oak", "919822000111", "shrikant.oak@example.com"})
	insertRel(ctx, tx, societyID, unitMap["203"], mShrikantID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// 204: Vacant Unit

	// 301: Vinod & Tanvi Agashe
	mVinodID := insertMember(ctx, tx, societyID, memberSpec{"Vinod", "Agashe", "919822000112", "vinod.agashe@example.com"})
	mTanviID := insertMember(ctx, tx, societyID, memberSpec{"Tanvi", "Agashe", "919822000113", "tanvi.agashe@example.com"})
	insertRel(ctx, tx, societyID, unitMap["301"], mVinodID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})
	insertRel(ctx, tx, societyID, unitMap["301"], mTanviID, relSpec{"OWNER_FAMILY", nil, true})

	// 302: Tenant Unit (Absentee Owner Makarand Bapat, Tenants Karan & Pooja Saxena)
	mMakarandID := insertMember(ctx, tx, societyID, memberSpec{"Makarand", "Bapat", "919822000114", "makarand.bapat@example.com"})
	mKaranID := insertMember(ctx, tx, societyID, memberSpec{"Karan", "Saxena", "919822000115", "karan.saxena@example.com"})
	mPoojaSID := insertMember(ctx, tx, societyID, memberSpec{"Pooja", "Saxena", "919822000116", "pooja.saxena@example.com"})
	insertRel(ctx, tx, societyID, unitMap["302"], mMakarandID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["302"], mKaranID, relSpec{"PRIMARY_TENANT", nil, true})
	insertRel(ctx, tx, societyID, unitMap["302"], mPoojaSID, relSpec{"TENANT_FAMILY", nil, true})

	// 303: Girish Kelkar
	mGirishID := insertMember(ctx, tx, societyID, memberSpec{"Girish", "Kelkar", "919822000117", "girish.kelkar@example.com"})
	insertRel(ctx, tx, societyID, unitMap["303"], mGirishID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), true})

	// 304: Tenant Unit (Absentee Owner Anant Lele, Tenant Meenal Jain)
	mAnantID := insertMember(ctx, tx, societyID, memberSpec{"Anant", "Lele", "919822000118", "anant.lele@example.com"})
	mMeenalID := insertMember(ctx, tx, societyID, memberSpec{"Meenal", "Jain", "919822000119", "meenal.jain@example.com"})
	insertRel(ctx, tx, societyID, unitMap["304"], mAnantID, relSpec{"PRIMARY_OWNER", floatPtr(100.0), false})
	insertRel(ctx, tx, societyID, unitMap["304"], mMeenalID, relSpec{"PRIMARY_TENANT", nil, true})
}

func insertMember(ctx context.Context, tx pgx.Tx, societyID uuid.UUID, s memberSpec) uuid.UUID {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO society_members (society_id, user_id, first_name, last_name, phone, email, status)
		VALUES ($1, NULL, $2, $3, $4, $5, 'ACTIVE')
		RETURNING id
	`, societyID, s.firstName, s.lastName, s.phone, s.email).Scan(&id)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert member %s %s (%s): %v", s.firstName, s.lastName, s.phone, err)
	}
	return id
}

func insertRel(ctx context.Context, tx pgx.Tx, societyID, unitID, memberID uuid.UUID, s relSpec) {
	_, err := tx.Exec(ctx, `
		INSERT INTO unit_relationships (
			society_id, unit_id, member_id, relationship_type, is_occupant, ownership_share, start_date, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_DATE - INTERVAL '180 days', 'ACTIVE')
	`, societyID, unitID, memberID, s.relType, s.occupant, s.share)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert relationship %s for member %s: %v", s.relType, memberID, err)
	}
}

func insertAppointment(ctx context.Context, tx pgx.Tx, societyID, designationID, memberID uuid.UUID) {
	_, err := tx.Exec(ctx, `
		INSERT INTO society_committee_appointments (
			society_id, designation_id, member_id, start_date, expected_end_date, status
		)
		VALUES ($1, $2, $3, '2026-01-01', '2027-12-31', 'ACTIVE')
	`, societyID, designationID, memberID)
	if err != nil {
		log.Fatalf("FATAL: Failed to insert committee appointment: %v", err)
	}
}

func assertCount(ctx context.Context, tx pgx.Tx, table string, expected int) {
	var count int
	err := tx.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	if err != nil {
		log.Fatalf("FATAL: Failed to query count of %s: %v", table, err)
	}
	if count != expected {
		log.Fatalf("FATAL: Invariant violated for %s: expected %d, got %d", table, expected, count)
	}
}
