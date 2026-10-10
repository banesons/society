# 02 — Platform Administration Database Schema

## Overview

Sprint S02 adds two tables to Supabase PostgreSQL:
1. `platform_roles`: Stores platform-level administrative assignments and their active status.
2. `platform_audit_logs`: Append-only immutable log of all administrative actions.

```text
┌───────────────────────┐
│      users (S01)      │
├───────────────────────┤
│ id (PK, UUID)         │
│ phone (UNIQUE, TEXT)  │
│ status (TEXT)         │
└───────────┬───────────┘
            │ 1
            │
            ├─────────────────────────────────────────┐
            │ 1..N                                    │ 1..N
┌───────────▼───────────┐                 ┌───────────▼───────────┐
│    platform_roles     │                 │  platform_audit_logs  │
├───────────────────────┤                 ├───────────────────────┤
│ id (PK, UUID)         │                 │ id (PK, UUID)         │
│ user_id (FK, UUID)    │                 │ actor_id (FK, UUID)   │
│ role (TEXT)           │                 │ action (TEXT)         │
│ status (TEXT)         │                 │ target_user_id (UUID) │
│ created_by (FK, UUID) │                 │ details (JSONB)       │
│ created_at            │                 │ client_ip (TEXT)      │
│ updated_at            │                 │ user_agent (TEXT)     │
└───────────────────────┘                 │ created_at            │
                                          └───────────────────────┘
```

---

## 1. `platform_roles` Table

Stores administrative assignments at the global platform tier.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique role assignment ID |
| `user_id` | `UUID` | `NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE` | Associated platform user (1 role per user in S02) |
| `role` | `TEXT` | `NOT NULL CHECK (role IN ('SUPER_ADMIN', 'ADMIN'))` | Assigned platform privilege |
| `status` | `TEXT` | `NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED'))` | Operational state of the role |
| `created_by` | `UUID` | `REFERENCES users(id) ON DELETE SET NULL` | Super Admin who granted this role |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Status change timestamp |

**Constraints & Indexes**:
- `CONSTRAINT platform_roles_user_id_key UNIQUE (user_id)`: Enforces strictly one platform role per user in S02.
- `CREATE INDEX platform_roles_user_id_idx ON platform_roles(user_id);`
- `CREATE INDEX platform_roles_active_idx ON platform_roles(user_id, role, status);`

---

## 2. `platform_audit_logs` Table

Append-only ledger capturing all platform management events.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Audit log entry ID |
| `actor_id` | `UUID` | `NOT NULL REFERENCES users(id)` | Administrator performing the action |
| `action` | `TEXT` | `NOT NULL` | Event identifier (e.g. `ADMIN_CREATED`, `ADMIN_DISABLED`) |
| `target_user_id` | `UUID` | `REFERENCES users(id)` | Target user subject to the action (nullable) |
| `details` | `JSONB` | `NOT NULL DEFAULT '{}'` | Structured payload (old/new status, metadata) |
| `client_ip` | `TEXT` | `NULL` | Source IP from trusted server context |
| `user_agent` | `TEXT` | `NULL` | Client User-Agent header |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Immutable timestamp |

**Indexes**:
- `CREATE INDEX platform_audit_logs_actor_idx ON platform_audit_logs(actor_id, created_at DESC);`
- `CREATE INDEX platform_audit_logs_target_idx ON platform_audit_logs(target_user_id, created_at DESC);`
- `CREATE INDEX platform_audit_logs_action_idx ON platform_audit_logs(action, created_at DESC);`

---

## Immutability Policy
Database security policies and an active PostgreSQL trigger prevent any `UPDATE` or `DELETE` statements on `platform_audit_logs`:
```sql
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'platform_audit_logs is append-only: updates and deletes are forbidden';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER platform_audit_logs_immutability_trg
BEFORE UPDATE OR DELETE ON platform_audit_logs
FOR EACH ROW
EXECUTE FUNCTION prevent_audit_log_modification();
```

---

## Related Documentation
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../decisions/012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../decisions/014-platform-audit-trail|ADR 014: Platform Audit Trail]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
