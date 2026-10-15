package main

import (
	"context"
	"fmt"
	"log"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Connect error: %v", err)
	}
	defer db.Close()

	conn := db.Pool()
	ctx := context.Background()

	fmt.Println("=== 1. PLATFORM VERIFICATION ===")
	var saCount, aCount, userCount, roleCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM platform_roles").Scan(&roleCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM platform_roles WHERE role = 'SUPER_ADMIN' AND status = 'ACTIVE'").Scan(&saCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM platform_roles WHERE role = 'ADMIN' AND status = 'ACTIVE'").Scan(&aCount)
	fmt.Printf("Users total: %d (2 Admins + 3 Guards)\n", userCount)
	fmt.Printf("Platform roles total: %d (SUPER_ADMIN: %d, ADMIN: %d)\n", roleCount, saCount, aCount)

	rows, _ := conn.Query(ctx, `SELECT u.phone, pr.role, pr.status FROM platform_roles pr JOIN users u ON pr.user_id = u.id`)
	for rows.Next() {
		var p, r, s string
		_ = rows.Scan(&p, &r, &s)
		fmt.Printf("  - Phone: %s -> %s (%s)\n", p, r, s)
	}
	rows.Close()

	fmt.Println("\n=== 2. SOCIETIES VERIFICATION ===")
	var socCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM societies WHERE status = 'ACTIVE'").Scan(&socCount)
	fmt.Printf("Active societies: %d\n", socCount)

	sRows, _ := conn.Query(ctx, "SELECT id, society_code, name, city, state, status FROM societies ORDER BY name")
	var soc1ID, soc2ID string
	for sRows.Next() {
		var id, code, name, city, state, status string
		_ = sRows.Scan(&id, &code, &name, &city, &state, &status)
		fmt.Printf("  - %s (%s) in %s, %s [%s] ID: %s\n", name, code, city, state, status, id)
		if city == "Mumbai" {
			soc1ID = id
		} else {
			soc2ID = id
		}
	}
	sRows.Close()

	fmt.Println("\n=== 3. STRUCTURE VERIFICATION ===")
	var bldCount, wingCount, unitCount, gateCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_buildings").Scan(&bldCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_wings").Scan(&wingCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_units").Scan(&unitCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_gates").Scan(&gateCount)
	fmt.Printf("Buildings: %d, Wings: %d, Units: %d, Gates: %d\n", bldCount, wingCount, unitCount, gateCount)

	var uSoc1, uSoc2, gSoc1, gSoc2 int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_units WHERE society_id = $1", soc1ID).Scan(&uSoc1)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_units WHERE society_id = $1", soc2ID).Scan(&uSoc2)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_gates WHERE society_id = $1", soc1ID).Scan(&gSoc1)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_gates WHERE society_id = $1", soc2ID).Scan(&gSoc2)
	fmt.Printf("Society 1 (Mumbai): %d units, %d gates\n", uSoc1, gSoc1)
	fmt.Printf("Society 2 (Pune): %d units, %d gates\n", uSoc2, gSoc2)

	// Duplicate unit identifiers check
	var dupUnits int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM (SELECT society_id, unit_identifier, COUNT(*) FROM society_units GROUP BY society_id, unit_identifier HAVING COUNT(*) > 1) t").Scan(&dupUnits)
	fmt.Printf("Duplicate unit identifiers within any society: %d\n", dupUnits)

	fmt.Println("\n=== 4. MEMBERS & RELATIONSHIPS VERIFICATION ===")
	var memCount, relCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_members").Scan(&memCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM unit_relationships").Scan(&relCount)
	fmt.Printf("Total society members: %d\n", memCount)
	fmt.Printf("Total unit relationships: %d\n", relCount)

	// Check non-canonical relationships
	var invalidRels int
	_ = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM unit_relationships 
		WHERE relationship_type NOT IN ('PRIMARY_OWNER', 'SECONDARY_OWNER', 'OWNER_FAMILY', 'PRIMARY_TENANT', 'SECONDARY_TENANT', 'TENANT_FAMILY', 'AUTHORIZED_OCCUPANT')
	`).Scan(&invalidRels)
	fmt.Printf("Non-canonical relationship types: %d\n", invalidRels)

	// Check cross-society relationships
	var crossSocRels int
	_ = conn.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM unit_relationships ur
		JOIN society_units u ON ur.unit_id = u.id
		JOIN society_members m ON ur.member_id = m.id
		WHERE ur.society_id != u.society_id OR ur.society_id != m.society_id
	`).Scan(&crossSocRels)
	fmt.Printf("Cross-society relationship leaks: %d\n", crossSocRels)

	// Check fake user IDs on pre-registered residents
	var fakeUsersOnMembers int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_members WHERE user_id IS NOT NULL").Scan(&fakeUsersOnMembers)
	fmt.Printf("Pre-registered members with assigned user_id: %d (expected 0)\n", fakeUsersOnMembers)

	// Check admin phones accidentally used as members
	var adminPhoneOverlap int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_members WHERE phone IN ('919967089101', '918422862001')").Scan(&adminPhoneOverlap)
	fmt.Printf("Admin phones duplicated as members: %d (expected 0)\n", adminPhoneOverlap)

	fmt.Println("\n=== 5. COMMITTEE APPOINTMENTS VERIFICATION ===")
	var apptCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_committee_appointments WHERE status = 'ACTIVE'").Scan(&apptCount)
	fmt.Printf("Active committee appointments: %d\n", apptCount)

	appRows, _ := conn.Query(ctx, `
		SELECT s.name, d.code, m.first_name || ' ' || m.last_name, a.start_date::text, a.expected_end_date::text
		FROM society_committee_appointments a
		JOIN societies s ON a.society_id = s.id
		JOIN society_designations d ON a.designation_id = d.id
		JOIN society_members m ON a.member_id = m.id
		ORDER BY s.name, d.code
	`)
	for appRows.Next() {
		var sName, dCode, mName, sDate, eDate string
		_ = appRows.Scan(&sName, &dCode, &mName, &sDate, &eDate)
		fmt.Printf("  - [%s] %-16s : %-22s (Term: %s to %s)\n", sName, dCode, mName, sDate, eDate)
	}
	appRows.Close()

	fmt.Println("\n=== 6. GATE FOUNDATION & STAFFING VERIFICATION ===")
	var shiftCount, gkCount, sessionCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_gate_shifts").Scan(&shiftCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_user_roles WHERE role = 'GATEKEEPER' AND status = 'ACTIVE'").Scan(&gkCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_gate_sessions WHERE status = 'ACTIVE'").Scan(&sessionCount)
	fmt.Printf("Gate shifts: %d\n", shiftCount)
	fmt.Printf("Assigned gatekeepers: %d\n", gkCount)
	fmt.Printf("Active duty sessions: %d\n", sessionCount)

	sessRows, _ := conn.Query(ctx, `
		SELECT s.name, g.name, u.phone, COALESCE(gs.name, 'No Shift'), ses.started_at::text, COALESCE(ses.handover_notes, '')
		FROM society_gate_sessions ses
		JOIN societies s ON ses.society_id = s.id
		JOIN society_gates g ON ses.gate_id = g.id
		JOIN users u ON ses.user_id = u.id
		LEFT JOIN society_gate_shifts gs ON ses.shift_id = gs.id
		ORDER BY s.name, g.name
	`)
	for sessRows.Next() {
		var sName, gName, phone, shiftName, startedAt, notes string
		_ = sessRows.Scan(&sName, &gName, &phone, &shiftName, &startedAt, &notes)
		fmt.Printf("  - [%s] %s (Guard: %s, Shift: %s) Notes: %q\n", sName, gName, phone, shiftName, notes)
	}
	sessRows.Close()

	// Check double-booking
	var doubleBooked int
	_ = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM (
			SELECT user_id, COUNT(*) FROM society_gate_sessions WHERE status = 'ACTIVE' GROUP BY user_id HAVING COUNT(*) > 1
		) t
	`).Scan(&doubleBooked)
	fmt.Printf("Double-booked guards: %d (expected 0)\n", doubleBooked)

	fmt.Println("\n=== 7. CANONICAL REFERENCE DATA & AUDIT VERIFICATION ===")
	var reasonCount, auditCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM society_suspension_reasons").Scan(&reasonCount)
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM platform_audit_logs").Scan(&auditCount)
	fmt.Printf("Preserved suspension reasons: %d (expected 8)\n", reasonCount)
	fmt.Printf("Platform audit log count: %d\n", auditCount)

	fmt.Println("\n=== 8. S09 VISITOR OPERATIONS SCHEMA VERIFICATION ===")
	tables := []string{"society_visits", "society_visitor_passes", "society_visit_approvals", "society_gate_events"}
	for _, tbl := range tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-25s: exists=%v\n", tbl, exists)
	}

	var uqGateExists bool
	_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'society_gates_id_society_uq')").Scan(&uqGateExists)
	fmt.Printf("Constraint society_gates_id_society_uq: exists=%v\n", uqGateExists)

	fmt.Println("\n=== 9. S10 SOCIETY COMMUNICATION SCHEMA VERIFICATION ===")
	s10Tables := []string{"society_announcements", "society_announcement_reads"}
	for _, tbl := range s10Tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-25s: exists=%v\n", tbl, exists)
	}

	s10Constraints := []string{
		"society_buildings_id_society_uq",
		"society_wings_id_society_uq",
		"chk_announcement_audience_targets",
		"chk_announcement_publication_consistency",
		"society_announcement_reads_uq",
	}
	for _, con := range s10Constraints {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = $1)", con).Scan(&exists)
		fmt.Printf("Constraint %-40s: exists=%v\n", con, exists)
	}

	fmt.Println("\n=== 10. S11 MAINTENANCE FOUNDATION SCHEMA VERIFICATION ===")
	s11Tables := []string{
		"maintenance_charge_categories",
		"maintenance_charge_configs",
		"maintenance_billing_periods",
		"maintenance_bills",
		"maintenance_bill_items",
	}
	for _, tbl := range s11Tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-32s: exists=%v\n", tbl, exists)
	}

	s11Constraints := []string{
		"uq_maintenance_charge_categories_name",
		"uq_maintenance_charge_categories_id_society",
		"fk_charge_configs_category",
		"uq_maintenance_charge_configs_id_society",
		"uq_billing_periods_dates",
		"uq_billing_periods_id_society",
		"fk_bills_unit",
		"fk_bills_period",
		"uq_maintenance_bills_unit_period",
		"uq_maintenance_bills_number",
		"uq_maintenance_bills_id_society",
		"fk_bill_items_bill",
	}
	for _, con := range s11Constraints {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = $1)", con).Scan(&exists)
		fmt.Printf("Constraint %-44s: exists=%v\n", con, exists)
	}

	fmt.Println("\n=== 11. S12 MAINTENANCE OPERATIONS & COLLECTIONS SCHEMA VERIFICATION ===")
	s12Tables := []string{
		"maintenance_payments",
		"maintenance_payment_allocations",
		"maintenance_receipts",
	}
	for _, tbl := range s12Tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-35s: exists=%v\n", tbl, exists)
	}

	s12Constraints := []string{
		"uq_maintenance_payments_ref",
		"uq_maintenance_payments_idem",
		"uq_maintenance_payments_id_society",
		"fk_alloc_payment",
		"fk_alloc_bill",
		"uq_maintenance_alloc_id_society",
		"fk_receipt_payment",
		"uq_maintenance_receipts_number",
		"uq_maintenance_receipts_payment",
		"uq_maintenance_receipts_id_society",
	}
	for _, con := range s12Constraints {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = $1)", con).Scan(&exists)
		fmt.Printf("Constraint %-44s: exists=%v\n", con, exists)
	}

	s12Sequences := []string{
		"maintenance_bill_seq",
		"maintenance_payment_seq",
		"maintenance_receipt_seq",
	}
	for _, seq := range s12Sequences {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_class WHERE relkind = 'S' AND relname = $1)", seq).Scan(&exists)
		fmt.Printf("Sequence %-42s: exists=%v\n", seq, exists)
	}

	fmt.Println("\n=== 12. S13 DOCUMENTS & RECORDS SCHEMA VERIFICATION ===")
	s13Tables := []string{
		"document_categories",
		"society_documents",
		"document_versions",
	}
	for _, tbl := range s13Tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-35s: exists=%v\n", tbl, exists)
	}

	fmt.Println("\n=== 13. S14 AUDIT & ADMINISTRATION SCHEMA VERIFICATION ===")
	var colExists bool
	_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'platform_audit_logs' AND column_name = 'society_id')").Scan(&colExists)
	fmt.Printf("Column platform_audit_logs.society_id      : exists=%v\n", colExists)

	s14Indexes := []string{
		"idx_platform_audit_logs_society_created",
		"idx_platform_audit_logs_details_society_created",
		"idx_platform_audit_logs_action_created",
		"idx_platform_audit_logs_actor_created",
	}
	for _, idx := range s14Indexes {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)", idx).Scan(&exists)
		fmt.Printf("Index %-44s: exists=%v\n", idx, exists)
	}

	var trgExists bool
	_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'platform_audit_logs_immutability_trg')").Scan(&trgExists)
	fmt.Printf("Trigger platform_audit_logs_immutability_trg: exists=%v\n", trgExists)

	fmt.Println("\n=== 14. S15 NOTIFICATIONS SCHEMA VERIFICATION ===")
	s15Tables := []string{
		"user_devices",
		"notifications",
		"notification_deliveries",
		"notification_outbox",
	}
	for _, tbl := range s15Tables {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", tbl).Scan(&exists)
		fmt.Printf("Table %-35s: exists=%v\n", tbl, exists)
	}

	s15Indexes := []string{
		"idx_user_devices_token",
		"idx_user_devices_user_active",
		"idx_notifications_recipient_created",
		"idx_notifications_recipient_unread",
		"idx_notifications_society",
		"idx_deliveries_pending",
		"idx_outbox_pending",
	}
	for _, idx := range s15Indexes {
		var exists bool
		_ = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)", idx).Scan(&exists)
		fmt.Printf("Index %-44s: exists=%v\n", idx, exists)
	}
}
