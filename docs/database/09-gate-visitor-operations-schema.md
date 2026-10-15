# 09 — Gate Visitor Operations Database Schema

## Overview
This document specifies the database tables, constraints, indexes, and relationship integrity rules for Sprint S09 Gate Operations.

All visitor tables are strictly scoped to `societies(id)` and utilize composite unique and foreign key constraints to prevent cross-tenant association.

## Schema Specifications

### 1. Composite Uniqueness Pre-requisite on `society_gates`
To support composite foreign keys from visitor gate events to gates:
```sql
ALTER TABLE society_gates 
    ADD CONSTRAINT society_gates_id_society_uq UNIQUE (id, society_id);
```

### 1b. Resident-Controlled Visitor Approval Policy on `society_units`
To support persistent, unit-level approval preferences controlled exclusively by authorized residents:
```sql
ALTER TABLE society_units
    ADD COLUMN IF NOT EXISTS requires_visitor_approval BOOLEAN NOT NULL DEFAULT FALSE;
```
- **Default**: `FALSE` (Direct entry allowed after gate registration).
- **Owner**: Managed exclusively by authenticated occupants/owners of the unit.
- **Integrity**: Non-null constraint guarantees deterministic resolution for all existing and newly created units.

### 2. Table: `society_visits`
Primary operational entity representing a visitor's request or physical visit.

```sql
CREATE TABLE IF NOT EXISTS society_visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    resident_id UUID NULL,
    visitor_name TEXT NOT NULL,
    visitor_phone TEXT NOT NULL,
    purpose TEXT NOT NULL,
    visit_type TEXT NOT NULL DEFAULT 'EXPECTED',
    status TEXT NOT NULL DEFAULT 'PENDING_APPROVAL',
    vehicle_number TEXT NULL,
    expected_arrival TIMESTAMPTZ NULL,
    valid_until TIMESTAMPTZ NULL,
    check_in_at TIMESTAMPTZ NULL,
    check_out_at TIMESTAMPTZ NULL,
    entry_gate_id UUID NULL,
    exit_gate_id UUID NULL,
    rejection_reason TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite FKs for tenant isolation
    CONSTRAINT fk_visits_society_unit 
        FOREIGN KEY (unit_id, society_id) 
        REFERENCES society_units(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_visits_society_member 
        FOREIGN KEY (resident_id, society_id) 
        REFERENCES society_members(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_visits_entry_gate 
        FOREIGN KEY (entry_gate_id, society_id) 
        REFERENCES society_gates(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_visits_exit_gate 
        FOREIGN KEY (exit_gate_id, society_id) 
        REFERENCES society_gates(id, society_id) 
        ON DELETE RESTRICT,

    -- Tenant isolation support for child tables
    CONSTRAINT society_visits_id_society_uq UNIQUE (id, society_id),

    -- Enumeration constraints
    CONSTRAINT chk_visits_type CHECK (visit_type IN ('EXPECTED', 'WALK_IN')),
    CONSTRAINT chk_visits_status CHECK (
        status IN (
            'PENDING_APPROVAL', 
            'APPROVED', 
            'REJECTED', 
            'CHECKED_IN', 
            'CHECKED_OUT', 
            'CANCELLED', 
            'EXPIRED'
        )
    ),

    -- Lifecycle state consistency checks
    CONSTRAINT chk_visits_check_in_out CHECK (
        (status = 'CHECKED_IN' AND check_in_at IS NOT NULL AND entry_gate_id IS NOT NULL AND check_out_at IS NULL) OR
        (status = 'CHECKED_OUT' AND check_in_at IS NOT NULL AND check_out_at IS NOT NULL AND check_out_at >= check_in_at AND entry_gate_id IS NOT NULL AND exit_gate_id IS NOT NULL) OR
        (status NOT IN ('CHECKED_IN', 'CHECKED_OUT'))
    )
);
```

#### Indexes on `society_visits`
```sql
CREATE INDEX IF NOT EXISTS idx_visits_society_status 
    ON society_visits (society_id, status);

CREATE INDEX IF NOT EXISTS idx_visits_society_unit 
    ON society_visits (society_id, unit_id);

CREATE INDEX IF NOT EXISTS idx_visits_active_inside 
    ON society_visits (society_id, check_in_at) 
    WHERE status = 'CHECKED_IN';

CREATE INDEX IF NOT EXISTS idx_visits_history 
    ON society_visits (society_id, created_at DESC);
```

### 3. Table: `society_visitor_passes`
Manages bearer tokens and manual pass codes for visitor access.

```sql
CREATE TABLE IF NOT EXISTS society_visitor_passes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    visit_id UUID NOT NULL,
    pass_code TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    revoked_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_passes_visit 
        FOREIGN KEY (visit_id, society_id) 
        REFERENCES society_visits(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT chk_passes_status CHECK (status IN ('ACTIVE', 'CONSUMED', 'REVOKED', 'EXPIRED')),
    CONSTRAINT chk_passes_validity CHECK (valid_until >= valid_from),
    CONSTRAINT society_visitor_passes_code_uq UNIQUE (society_id, pass_code),
    CONSTRAINT society_visitor_passes_hash_uq UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_visitor_passes_lookup 
    ON society_visitor_passes (society_id, pass_code, status);
```

### 4. Table: `society_visit_approvals`
Audit ledger of resident and administrative approval decisions.

```sql
CREATE TABLE IF NOT EXISTS society_visit_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    visit_id UUID NOT NULL,
    decision TEXT NOT NULL,
    decided_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_approvals_visit 
        FOREIGN KEY (visit_id, society_id) 
        REFERENCES society_visits(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT chk_approvals_decision CHECK (decision IN ('APPROVED', 'REJECTED'))
);

CREATE INDEX IF NOT EXISTS idx_visit_approvals_visit 
    ON society_visit_approvals (society_id, visit_id, created_at DESC);
```

### 5. Table: `society_gate_events`
Append-only log of physical entry and exit movements across perimeter gates.

```sql
CREATE TABLE IF NOT EXISTS society_gate_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    visit_id UUID NOT NULL,
    gate_id UUID NOT NULL,
    session_id UUID NULL REFERENCES society_gate_sessions(id) ON DELETE SET NULL,
    gatekeeper_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_type TEXT NOT NULL,
    direction TEXT NOT NULL,
    vehicle_number TEXT NULL,
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_gate_events_visit 
        FOREIGN KEY (visit_id, society_id) 
        REFERENCES society_visits(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_gate_events_gate 
        FOREIGN KEY (gate_id, society_id) 
        REFERENCES society_gates(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT chk_gate_events_type CHECK (event_type IN ('ENTRY', 'EXIT')),
    CONSTRAINT chk_gate_events_direction CHECK (direction IN ('INGRESS', 'EGRESS'))
);

CREATE INDEX IF NOT EXISTS idx_gate_events_society_visit 
    ON society_gate_events (society_id, visit_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_gate_events_gate_history 
    ON society_gate_events (society_id, gate_id, created_at DESC);
```

## Related Documents
- [[08-gate-foundation-and-staffing-schema]]
- [[../architecture/14-gate-visitor-operations]]
- [[../decisions/023-visitor-lifecycle-and-gate-operations]]
- [[../api/10-gate-visitor-operations-api]]
