# ADR 021: Development Database Reset and Controlled Demo Data Seeding

## Status
Accepted (S08 Closure / Pre-S09 Preparation)

## Context
During sprints S00 through S08, iterative integration testing and automated end-to-end verification generated over 180 transient societies, 80+ synthetic users, and 900+ audit log entries. To enable realistic manual acceptance testing and prepare clean, predictable state for S09 Gate Operations, a pristine, internally consistent dataset is required.

However, arbitrary truncation or schema drops risk destroying:
1. Canonical reference catalogues (`society_suspension_reasons`).
2. Migration state and database constraints.
3. Production safety guards.

## Decision
1. **Explicit Development-Only Reset Command**:
   - Provide a standalone CLI tool in `backend/cmd/seed/main.go`.
   - Never embed destructive reset logic into API server startup (`cmd/api`).
   - Refuse execution if `ENV=production`.
   - Require explicit confirmation flag `--confirm-dev-reset`.
   - Validate target database host/project against known development reference (`glpjpqpbhdcmgmwlnkay`).
2. **Transactional Dependency-Ordered Reset**:
   - Execute all deletions inside a single PostgreSQL transaction (`pgx.Tx`).
   - Do NOT drop or recreate tables, triggers, or views.
   - Do NOT disable constraints globally.
   - Respect foreign-key deletion order:
     1. `society_committee_appointments`
     2. `unit_relationships`
     3. `society_gate_sessions`
     4. `society_gate_shifts`
     5. `society_gates`
     6. `society_staffing_requirements`
     7. `society_units`
     8. `society_wings`
     9. `society_buildings`
     10. `society_designations`
     11. `society_members`
     12. `society_user_roles`
     13. `societies`
     14. `platform_audit_logs` (truncated within transaction)
     15. `user_sessions`
     16. `auth_otps`
     17. `platform_roles`
     18. `users`
   - Preserve canonical lookup catalogues (`society_suspension_reasons`).
3. **Exact Demo Dataset Target**:
   - **Platform Identities**:
     - Super Admin: `+919967089101` (`SUPER_ADMIN`, `ACTIVE`)
     - Platform Admin: `+918422862001` (`ADMIN`, `ACTIVE`, `created_by` = Super Admin)
   - **Societies**:
     - Society 1: *Shree Ram Cooperative Housing Society* (Mumbai, 25 units, 2 gates: Main Gate, Service Gate, 3 shifts, 2 stationed guards).
     - Society 2: *Greenview Residency* (Pune, 12 units, 1 gate: Main Gate, 2 shifts, 1 stationed guard).
   - **Households**:
     - 37 total units with realistic synthetic occupants (~70% owner-occupied, ~20% tenant-occupied, ~10% vacant).
     - Pure canonical relationship types (`PRIMARY_OWNER`, `SECONDARY_OWNER`, `OWNER_FAMILY`, `PRIMARY_TENANT`, `SECONDARY_TENANT`, `TENANT_FAMILY`, `AUTHORIZED_OCCUPANT`).
     - Pre-registered residents use `user_id = NULL`.
   - **Committee Designations & Appointments**:
     - Canonical governance roles (`CHAIRMAN`, `SECRETARY`, `TREASURER`, `COMMITTEE_MEMBER`).
     - Singleton appointments per society with zero conflicting overlapping terms.
   - **Gate Foundation**:
     - Gates configured with directionality (`BIDIRECTIONAL`) and operational status (`ACTIVE`).
     - Dedicated authenticated guard users in `users` with `GATEKEEPER` role in `society_user_roles`.
     - Active on-duty sessions strictly satisfying the single active session per guard constraint (`uq_gatekeeper_active_session`).
     - Zero visitor logs (deferred strictly to S09).
4. **Idempotency & Verification**:
   - The reset and seed sequence is strictly repeatable and asserts database invariants upon completion.

## Consequences
- Clean, realistic development environment matching S00–S08 architecture.
- Absolute preservation of database schema, constraints, triggers, and migrations.
- Clear separation between platform admin credentials and society operational records.
- Ready foundation for S09 Gate Operations.

## Related Documentation
- [[../architecture/13-development-seed-data|13 — Development Seed Data Architecture]]
- [[../sprints/S08-gate-foundation|S08 — Gate Foundation]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Schema]]
