# 14 — Audit and Administration Database Schema

## 1. Overview
Sprint S14 extends the foundational audit infrastructure established in [[docs/decisions/014-platform-audit-trail|ADR 014]] and [[docs/database/02-platform-administration-schema|02 — Platform Administration Schema]] without introducing duplicate audit tables or systems.

All administrative actions across the platform (Platform Admin, Society Admin, Gate Operations, Society Communication, Maintenance Operations, Documents & Records) write append-only event logs to the unified `platform_audit_logs` ledger.

---

## 2. Immutability & Trust Boundaries

The `platform_audit_logs` table is protected by a strict PostgreSQL database trigger:
```sql
CREATE OR REPLACE FUNCTION trg_enforce_platform_audit_logs_immutability()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'platform_audit_logs is append-only: updates and deletes are prohibited';
END;
$$ LANGUAGE plpgsql;
```
Any `UPDATE` or `DELETE` statement executed by application connections or standard roles is rejected with error code `P0001`.

### Data Preservation & Non-Mutation Rule
- Existing historical rows (3,360 baseline rows) are immutable.
- Historical rows cannot and MUST NOT be rewritten or modified via `UPDATE`.
- 3,206 existing rows contain `"society_id"` within the JSONB `details` column (`details->>'society_id'`).
- Forward-only writes can populate an explicit first-class `society_id` column for fast relational integrity, while historical and transition queries evaluate both the column and the JSONB property.

---

## 3. Schema Changes (Forward-Only Migration)

The migration `20261027000000_create_audit_and_administration_schema.sql` performs the following safe, non-destructive alterations:

### 3.1 Column Addition
```sql
ALTER TABLE platform_audit_logs
ADD COLUMN IF NOT EXISTS society_id UUID REFERENCES societies(id) ON DELETE SET NULL;
```
- Type: `UUID`
- Nullable: `YES` (allows platform-level events such as `SUPER_ADMIN_BOOTSTRAPPED`, `ADMIN_CREATED`, and `ADMIN_DISABLED` to remain null).
- Foreign Key: `REFERENCES societies(id) ON DELETE SET NULL`.

### 3.2 Dual Performance Indexes
To guarantee bounded sub-millisecond query execution across large audit volumes without scanning unbounded JSONB:
```sql
-- 1. Index on first-class society_id and timestamp
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_society_created
ON platform_audit_logs (society_id, created_at DESC);

-- 2. Functional index on historical details->>'society_id' and timestamp
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_details_society_created
ON platform_audit_logs ((details->>'society_id'), created_at DESC)
WHERE details->>'society_id' IS NOT NULL;

-- 3. Composite index for action + timestamp queries
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_action_created
ON platform_audit_logs (action, created_at DESC);
```

---

## 4. Unified Audit Table Structure

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Immutable audit event identifier |
| `actor_id` | UUID | NOT NULL, REFERENCES `users(id)` | User who initiated the action |
| `action` | TEXT | NOT NULL | Standardized action code |
| `target_user_id` | UUID | NULL, REFERENCES `users(id)` | Target user (if applicable) |
| `society_id` | UUID | NULL, REFERENCES `societies(id)` | Associated society scope |
| `details` | JSONB | NOT NULL DEFAULT '{}'::jsonb | Redacted structured metadata |
| `ip_address` | TEXT | NULL | Client IP address at event time |
| `user_agent` | TEXT | NULL | Client user agent at event time |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Timestamp in UTC |

---

## 5. Society Query Scoping Pattern

To ensure complete historical compatibility and zero-loss queries across 100% of historical and new events, society-scoped queries use:
```sql
WHERE (
    platform_audit_logs.society_id = $1
    OR
    platform_audit_logs.details->>'society_id' = $1::text
)
```
PostgreSQL's query planner uses Bitmap Or on `idx_platform_audit_logs_society_created` and `idx_platform_audit_logs_details_society_created`, executing in bounded time.

---

## 6. Migration Verification Checklist

- [x] Schema alter is forward-only (`ADD COLUMN IF NOT EXISTS`).
- [x] Zero updates or deletes executed on existing rows.
- [x] Existing 3,360 audit rows remain fully readable.
- [x] Foreign keys allow nulls for platform-wide events.
- [x] Immutability trigger remains active.
