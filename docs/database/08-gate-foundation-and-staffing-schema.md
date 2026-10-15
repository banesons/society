# 08 — Gate Foundation and Staffing Database Schema

## 1. Overview
Sprint S08 extends the existing gate and staffing infrastructure with operational configurations, shift scheduling, and guard duty sessions.

Key database modifications:
1. Extension of `society_user_roles.role` check constraint to include `'GATEKEEPER'`.
2. Extension of `society_gates` with `directionality` and `operational_status`.
3. New table `society_gate_shifts` for daily shift scheduling.
4. New table `society_gate_sessions` for guard check-ins and active duty tracking.

---

## 2. Table Modifications

### 2.1 Extension of `society_user_roles`
The existing role check constraint is extended:
```sql
ALTER TABLE society_user_roles DROP CONSTRAINT IF EXISTS society_user_roles_role_check;
ALTER TABLE society_user_roles ADD CONSTRAINT society_user_roles_role_check 
    CHECK (role IN ('SOC_ADMIN', 'GATEKEEPER'));
```

### 2.2 Extension of `society_gates`
Add operational parameters while preserving existing S05 columns:
```sql
ALTER TABLE society_gates 
    ADD COLUMN IF NOT EXISTS directionality TEXT NOT NULL DEFAULT 'BIDIRECTIONAL',
    ADD COLUMN IF NOT EXISTS operational_status TEXT NOT NULL DEFAULT 'ACTIVE';

ALTER TABLE society_gates DROP CONSTRAINT IF EXISTS society_gates_directionality_check;
ALTER TABLE society_gates ADD CONSTRAINT society_gates_directionality_check 
    CHECK (directionality IN ('BIDIRECTIONAL', 'INGRESS_ONLY', 'EGRESS_ONLY'));

ALTER TABLE society_gates DROP CONSTRAINT IF EXISTS society_gates_operational_status_check;
ALTER TABLE society_gates ADD CONSTRAINT society_gates_operational_status_check 
    CHECK (operational_status IN ('ACTIVE', 'INACTIVE', 'MAINTENANCE', 'EMERGENCY_OPEN'));
```

---

## 3. New Table: `society_gate_shifts`

Stores society-configured recurring shifts (e.g., Morning, Evening, Night).

```sql
CREATE TABLE IF NOT EXISTS society_gate_shifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_gate_shifts_name_uq UNIQUE (society_id, name),
    CONSTRAINT society_gate_shifts_society_tenant UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_gate_shifts_society_active 
    ON society_gate_shifts (society_id, is_active);
```

### Columns
| Column | Type | Nullable | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | No | Primary key surrogate identifier. |
| `society_id` | `UUID` | No | Tenant scoping foreign key to `societies(id)`. |
| `name` | `TEXT` | No | Shift display name (e.g. `Morning`, `Evening`, `Night`). |
| `start_time` | `TIME` | No | Shift start time (`HH:MM:SS`). |
| `end_time` | `TIME` | No | Shift end time (`HH:MM:SS`). |
| `is_active` | `BOOLEAN` | No | Whether the shift is active for scheduling. |
| `created_at` | `TIMESTAMPTZ` | No | Creation timestamp. |
| `updated_at` | `TIMESTAMPTZ` | No | Last update timestamp. |

---

## 4. New Table: `society_gate_sessions`

Tracks physical guard presence and active duty at gates.

```sql
CREATE TABLE IF NOT EXISTS society_gate_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    gate_id UUID NOT NULL REFERENCES society_gates(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    shift_id UUID NULL REFERENCES society_gate_shifts(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    handover_notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_gate_sessions_status CHECK (status IN ('ACTIVE', 'ENDED')),
    CONSTRAINT chk_gate_sessions_ended_at CHECK (
        (status = 'ACTIVE' AND ended_at IS NULL) OR 
        (status = 'ENDED' AND ended_at IS NOT NULL AND ended_at >= started_at)
    )
);

-- Partial index preventing a gatekeeper from holding multiple ACTIVE duty sessions concurrently
CREATE UNIQUE INDEX IF NOT EXISTS uq_gatekeeper_active_session 
    ON society_gate_sessions (society_id, user_id) 
    WHERE status = 'ACTIVE';

-- Lookup indexes for operational queries
CREATE INDEX IF NOT EXISTS idx_gate_sessions_gate_status 
    ON society_gate_sessions (society_id, gate_id, status);

CREATE INDEX IF NOT EXISTS idx_gate_sessions_history 
    ON society_gate_sessions (society_id, started_at DESC);
```

### Columns
| Column | Type | Nullable | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | No | Primary key surrogate identifier. |
| `society_id` | `UUID` | No | Tenant scoping foreign key to `societies(id)`. |
| `gate_id` | `UUID` | No | Stationed physical gate foreign key. |
| `user_id` | `UUID` | No | Authenticated guard user foreign key. |
| `shift_id` | `UUID` | Yes | Associated shift definition if assigned. |
| `started_at` | `TIMESTAMPTZ` | No | Shift check-in timestamp. |
| `ended_at` | `TIMESTAMPTZ` | Yes | Shift check-out / handover timestamp. |
| `status` | `TEXT` | No | `ACTIVE` (on duty) or `ENDED` (handed over). |
| `handover_notes` | `TEXT` | Yes | Shift logs, incident remarks, or handover notes. |
| `created_at` | `TIMESTAMPTZ` | No | Creation timestamp. |
| `updated_at` | `TIMESTAMPTZ` | No | Modification timestamp. |

---

## 5. Related Documentation
- [[../decisions/020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations]]
- [[../architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]
- [[../api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API Specification]]
- [[../security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security Specification]]
