package main

import (
	"context"
	"fmt"
	"os"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("Connect error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	conn := db.Pool()

	soc1Code := "SOC-MUM-SRM101"
	soc2Code := "SOC-PUN-GVR202"

	// 1. Identify canonical society IDs to retain
	var keepIDs []string
	keepRows, err := conn.Query(ctx, "SELECT id, society_code, name FROM societies WHERE society_code IN ($1, $2)", soc1Code, soc2Code)
	if err != nil {
		fmt.Printf("Query keepIDs error: %v\n", err)
		return
	}
	for keepRows.Next() {
		var id, code, name string
		_ = keepRows.Scan(&id, &code, &name)
		keepIDs = append(keepIDs, id)
		fmt.Printf("[RETAINING] %s | %s (%s)\n", id, name, code)
	}
	keepRows.Close()

	if len(keepIDs) == 0 {
		fmt.Println("Warning: Canonical societies not found in DB. Run `go run ./cmd/seed --confirm-dev-reset` to seed them.")
		return
	}

	// 2. List extra societies to be deleted
	fmt.Println("\n=== EXTRA SOCIETIES TO BE DELETED ===")
	rows, err := conn.Query(ctx, `
		SELECT id, society_code, name, status, created_at 
		FROM societies 
		WHERE id != ALL($1)
		ORDER BY created_at
	`, keepIDs)
	if err != nil {
		fmt.Printf("Query extra societies error: %v\n", err)
		return
	}

	var extraIDs []string
	for rows.Next() {
		var id, code, name, status, createdAt string
		_ = rows.Scan(&id, &code, &name, &status, &createdAt)
		extraIDs = append(extraIDs, id)
		fmt.Printf("  - %s | %s | %s [%s] (%s)\n", id, code, name, status, createdAt)
	}
	rows.Close()

	fmt.Printf("\nTotal extra societies found: %d\n", len(extraIDs))
	if len(extraIDs) == 0 {
		fmt.Println("No extra societies to delete. Exactly canonical societies exist.")
		return
	}

	// 3. Cascade delete extra societies' dependent records
	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Printf("Begin tx error: %v\n", err)
		return
	}
	defer tx.Rollback(ctx)

	// Disable triggers that block deletions
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bill_items DISABLE TRIGGER trg_bill_items_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE document_versions DISABLE TRIGGER trg_document_versions_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE platform_audit_logs DISABLE TRIGGER platform_audit_logs_immutability_trg")

	childTables := []string{
		"notifications",
		"document_versions",
		"society_documents",
		"document_categories",
		"maintenance_receipts",
		"maintenance_payment_allocations",
		"maintenance_payments",
		"maintenance_bill_items",
		"maintenance_bills",
		"maintenance_billing_periods",
		"maintenance_charge_configs",
		"maintenance_charge_categories",
		"society_announcement_reads",
		"society_announcements",
		"society_gate_events",
		"society_visitor_passes",
		"society_visit_approvals",
		"society_visits",
		"society_gate_sessions",
		"society_gate_shifts",
		"society_staffing_requirements",
		"society_gates",
		"society_committee_appointments",
		"society_designations",
		"unit_relationships",
		"society_units",
		"society_wings",
		"society_buildings",
		"society_members",
		"society_user_roles",
	}

	// Unlink current versions on society_documents
	_, _ = tx.Exec(ctx, "UPDATE society_documents SET current_version_id = NULL WHERE society_id != ALL($1)", keepIDs)

	for _, tbl := range childTables {
		tag, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE society_id != ALL($1)", tbl), keepIDs)
		if err != nil {
			fmt.Printf("Error deleting from %s: %v\n", tbl, err)
			return
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("Deleted %d rows from %s\n", tag.RowsAffected(), tbl)
		}
	}

	// Nullify society_id on platform_audit_logs for extra societies
	_, _ = tx.Exec(ctx, "UPDATE platform_audit_logs SET society_id = NULL WHERE society_id != ALL($1)", keepIDs)

	// Delete from societies
	delTag, err := tx.Exec(ctx, "DELETE FROM societies WHERE id != ALL($1)", keepIDs)
	if err != nil {
		fmt.Printf("Error deleting from societies: %v\n", err)
		return
	}
	fmt.Printf("Deleted %d extra societies from societies table.\n", delTag.RowsAffected())

	// Re-enable triggers
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bill_items ENABLE TRIGGER trg_bill_items_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE document_versions ENABLE TRIGGER trg_document_versions_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE platform_audit_logs ENABLE TRIGGER platform_audit_logs_immutability_trg")

	if err := tx.Commit(ctx); err != nil {
		fmt.Printf("Commit error: %v\n", err)
		return
	}

	// Verify remaining societies
	var remainingCount int
	_ = conn.QueryRow(ctx, "SELECT COUNT(*) FROM societies").Scan(&remainingCount)
	fmt.Printf("\n=== VERIFICATION: REMAINING SOCIETIES: %d ===\n", remainingCount)
	remRows, _ := conn.Query(ctx, "SELECT id, society_code, name, status FROM societies ORDER BY name")
	for remRows.Next() {
		var id, code, name, status string
		_ = remRows.Scan(&id, &code, &name, &status)
		fmt.Printf("  - %s: %s (%s) [%s]\n", id, name, code, status)
	}
	remRows.Close()
}
