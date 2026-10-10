# 04 — Society Administration Database Schema

## Overview

Sprint S04 introduces the society-level administrator assignment schema. This schema manages the mapping between users and specific housing society tenants as `SOC_ADMIN`, providing multi-tenant isolation, many-to-many cardinality, immediate revocation, and auditability.

---

## 1. `society_user_roles` Table Specification

The `society_user_roles` table stores tenant-scoped role assignments.

```sql
CREATE TABLE society_user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role TEXT NOT NULL DEFAULT 'SOC_ADMIN',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    assigned_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Column Dictionary

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | `PRIMARY KEY DEFAULT gen_random_uuid()` | Unique assignment identifier |
| `society_id` | `UUID` | `NOT NULL REFERENCES societies(id) ON DELETE RESTRICT` | Target housing society tenant |
| `user_id` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Assigned user |
| `role` | `TEXT` | `NOT NULL DEFAULT 'SOC_ADMIN'` | Scoped role identifier (`SOC_ADMIN`) |
| `status` | `TEXT` | `NOT NULL DEFAULT 'ACTIVE'` | Assignment lifecycle: `ACTIVE`, `REVOKED` |
| `assigned_by` | `UUID` | `NOT NULL REFERENCES users(id) ON DELETE RESTRICT` | Super Admin who created the assignment |
| `revoked_by` | `UUID` | `NULL REFERENCES users(id) ON DELETE SET NULL` | Super Admin who revoked the assignment |
| `revoked_at` | `TIMESTAMPTZ` | `NULL` | Timestamp when assignment was revoked |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Record creation timestamp |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | Record modification timestamp |

---

## 2. Table Constraints & Validation Rules

### Check Constraints
```sql
-- Allowed role values in S04
ALTER TABLE society_user_roles
    ADD CONSTRAINT society_user_roles_role_check
    CHECK (role IN ('SOC_ADMIN'));

-- Allowed assignment statuses
ALTER TABLE society_user_roles
    ADD CONSTRAINT society_user_roles_status_check
    CHECK (status IN ('ACTIVE', 'REVOKED'));

-- Revocation consistency check
ALTER TABLE society_user_roles
    ADD CONSTRAINT society_user_roles_revocation_check
    CHECK (
        (status = 'ACTIVE' AND revoked_at IS NULL AND revoked_by IS NULL) OR
        (status = 'REVOKED' AND revoked_at IS NOT NULL)
    );
```

### Unique Constraints
```sql
-- Strictly prevent duplicate concurrent ACTIVE assignments for the same role
-- Allows historical audit trail of previous REVOKED assignments
CREATE UNIQUE INDEX society_user_roles_active_uq
    ON society_user_roles (society_id, user_id, role)
    WHERE status = 'ACTIVE';
```

---

## 3. Query Optimization Indexes

```sql
-- 1. Fast lookup of active society assignments for authenticated user (login / context switch)
CREATE INDEX society_user_roles_user_active_idx
    ON society_user_roles (user_id, status)
    WHERE status = 'ACTIVE';

-- 2. Fast retrieval of society administrator roster in Super Admin console
CREATE INDEX society_user_roles_society_roster_idx
    ON society_user_roles (society_id, status, created_at DESC);

-- 3. Immediate server-side authorization check (user_id, society_id, role)
CREATE INDEX society_user_roles_auth_lookup_idx
    ON society_user_roles (society_id, user_id, role, status);
```

---

## 4. Referential Integrity & Foreign Key Behavior

- **`society_id`**: `REFERENCES societies(id) ON DELETE RESTRICT`. A society cannot be dropped while assignments reference it.
- **`user_id`**: `REFERENCES users(id) ON DELETE RESTRICT`. An assigned user cannot be deleted from `users`.
- **`assigned_by`**: `REFERENCES users(id) ON DELETE RESTRICT`. Preserves administrative provenance.
- **`revoked_by`**: `REFERENCES users(id) ON DELETE SET NULL`. If an administrator who revoked access is deleted, the revocation timestamp remains intact.

---

## 5. Audit Trail Alignment

Assignment operations emit transactional records into `platform_audit_logs`:
- **Actions**:
  - `SOCIETY_ADMIN_ASSIGNED`: Emitted when an administrator is assigned to a society.
  - `SOCIETY_ADMIN_REVOKED`: Emitted when an administrator assignment is revoked.
- **Details JSON Structure**:
  ```json
  {
    "assignment_id": "c1f7b19a-4e63-4702-8d75-6e9f2b34a108",
    "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
    "society_code": "SOC-MUM-7K4P2X",
    "user_id": "a90b4d1c-1549-410e-862d-98e3b1c60124",
    "user_phone_mask": "+91 9820xxxx56",
    "role": "SOC_ADMIN",
    "previous_status": "NONE",
    "new_status": "ACTIVE"
  }
  ```

---

## 6. Migration Sequence

1. `20261004000000_create_auth_schema.sql` (provides `users`, `user_sessions`)
2. `20261010000000_create_platform_admin_schema.sql` (provides `platform_roles`, `platform_audit_logs`)
3. `20261011000000_create_societies_schema.sql` (provides `societies`)
4. `20261012000000_create_society_suspension_reasons.sql` (provides `society_suspension_reasons`)
5. `20261013000000_add_id_to_society_suspension_reasons.sql` (adds surrogate identity ID)
6. `20261014000000_create_society_user_roles.sql` (provides `society_user_roles` in S04)

---

## Related Documentation
- [[01-authentication-schema|01 — Authentication Database Schema]]
- [[02-platform-administration-schema|02 — Platform Administration Database Schema]]
- [[03-society-schema|03 — Society Database Schema]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
