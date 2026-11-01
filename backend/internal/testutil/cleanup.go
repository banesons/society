package testutil

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CanonicalSociety1Code = "SOC-MUM-SRM101"
	CanonicalSociety2Code = "SOC-PUN-GVR202"

	SuperAdminPhone    = "919967089101"
	PlatformAdminPhone = "918422862001"
	Guard1Phone        = "919820010001"
	Guard2Phone        = "919820010002"
	Guard3Phone        = "919820020001"
)

// CleanExtraSocieties deletes any non-canonical societies and test users created during
// integration test executions, restoring the database strictly to the 2 canonical seed societies.
func CleanExtraSocieties(ctx context.Context, pool *pgxpool.Pool) error {
	// 1. Identify canonical society IDs to retain
	var keepIDs []string
	keepRows, err := pool.Query(ctx, "SELECT id FROM societies WHERE society_code IN ($1, $2)", CanonicalSociety1Code, CanonicalSociety2Code)
	if err != nil {
		return fmt.Errorf("failed to query canonical societies: %w", err)
	}
	defer keepRows.Close()

	for keepRows.Next() {
		var id string
		if err := keepRows.Scan(&id); err != nil {
			return fmt.Errorf("failed to scan society ID: %w", err)
		}
		keepIDs = append(keepIDs, id)
	}

	if len(keepIDs) == 0 {
		return fmt.Errorf("canonical societies (%s, %s) not found in database", CanonicalSociety1Code, CanonicalSociety2Code)
	}

	// 2. Check if extra societies exist
	var extraCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM societies WHERE id != ALL($1)", keepIDs).Scan(&extraCount)
	if err != nil {
		return fmt.Errorf("failed to count extra societies: %w", err)
	}

	var extraUsersCount int
	canonicalPhones := []string{SuperAdminPhone, PlatformAdminPhone, Guard1Phone, Guard2Phone, Guard3Phone}
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE phone != ALL($1)", canonicalPhones).Scan(&extraUsersCount)

	if extraCount == 0 && extraUsersCount == 0 {
		return nil
	}

	// 3. Cascade delete extra societies' dependent records in a transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Disable immutability triggers
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bills DISABLE TRIGGER trg_bills_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bill_items DISABLE TRIGGER trg_bill_items_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE document_versions DISABLE TRIGGER trg_document_versions_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE platform_audit_logs DISABLE TRIGGER platform_audit_logs_immutability_trg")

	childTables := []string{
		"maintenance_bill_pdf_artifacts",
		"maintenance_receipts",
		"maintenance_payment_allocations",
		"maintenance_payments",
		"maintenance_bill_items",
		"maintenance_batch_units",
		"maintenance_billing_schedule_executions",
		"maintenance_billing_schedules",
		"maintenance_bills",
		"maintenance_billing_batches",
		"maintenance_unit_billing_overrides",
		"maintenance_template_fields",
		"maintenance_bill_template_versions",
		"maintenance_bill_templates",
		"maintenance_charge_heads",
		"maintenance_formula_versions",
		"maintenance_formula_definitions",
		"notifications",
		"document_versions",
		"society_documents",
		"document_categories",
		"maintenance_billing_periods",
		"maintenance_charge_configs",
		"maintenance_charge_categories",
		"society_announcement_reads",
		"society_announcements",
		"society_gate_events",
		"society_visitor_passes",
		"society_visit_approvals",
		"society_visits",
		"society_visitor_identities",
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
		_, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE society_id != ALL($1)", tbl), keepIDs)
		if err != nil {
			return fmt.Errorf("failed deleting from %s: %w", tbl, err)
		}
	}

	// Nullify society_id on platform_audit_logs for extra societies
	_, _ = tx.Exec(ctx, "UPDATE platform_audit_logs SET society_id = NULL WHERE society_id != ALL($1)", keepIDs)

	// Delete from societies
	if _, err := tx.Exec(ctx, "DELETE FROM societies WHERE id != ALL($1)", keepIDs); err != nil {
		return fmt.Errorf("failed deleting from societies: %w", err)
	}

	// Delete non-canonical platform roles created during test runs
	if _, err := tx.Exec(ctx, `
		DELETE FROM platform_roles 
		WHERE user_id IN (SELECT id FROM users WHERE phone != ALL($1))
	`, canonicalPhones); err != nil {
		return fmt.Errorf("failed deleting non-canonical platform roles: %w", err)
	}

	// Delete platform audit logs involving non-canonical test users
	if _, err := tx.Exec(ctx, `
		DELETE FROM platform_audit_logs 
		WHERE actor_id IN (SELECT id FROM users WHERE phone != ALL($1))
		   OR target_user_id IN (SELECT id FROM users WHERE phone != ALL($1))
	`, canonicalPhones); err != nil {
		return fmt.Errorf("failed deleting audit logs for non-canonical test users: %w", err)
	}

	// Only delete users who are NOT referenced by canonical society members
	if _, err := tx.Exec(ctx, `
		DELETE FROM user_sessions 
		WHERE user_id NOT IN (
			SELECT id FROM users WHERE phone = ANY($1)
			UNION
			SELECT user_id FROM society_members WHERE user_id IS NOT NULL
			UNION
			SELECT user_id FROM society_user_roles WHERE user_id IS NOT NULL
		)
	`, canonicalPhones); err != nil {
		return fmt.Errorf("failed deleting non-canonical sessions: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM users 
		WHERE phone != ALL($1)
		  AND id NOT IN (SELECT user_id FROM society_members WHERE user_id IS NOT NULL)
		  AND id NOT IN (SELECT user_id FROM society_user_roles WHERE user_id IS NOT NULL)
	`, canonicalPhones); err != nil {
		return fmt.Errorf("failed deleting extra test users: %w", err)
	}

	// Re-enable triggers
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bills ENABLE TRIGGER trg_bills_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE maintenance_bill_items ENABLE TRIGGER trg_bill_items_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE document_versions ENABLE TRIGGER trg_document_versions_immutable")
	_, _ = tx.Exec(ctx, "ALTER TABLE platform_audit_logs ENABLE TRIGGER platform_audit_logs_immutability_trg")

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit cleanup transaction: %w", err)
	}

	return nil
}
