-- Migration: 20261019000000_create_gate_foundation_and_staffing.sql
-- Description: S08 Gate Foundation, Gatekeeper Roles, Shifts, and Duty Sessions

-- 1. Extend society_user_roles to support GATEKEEPER role
ALTER TABLE society_user_roles DROP CONSTRAINT IF EXISTS society_user_roles_role_check;
ALTER TABLE society_user_roles ADD CONSTRAINT society_user_roles_role_check 
    CHECK (role IN ('SOC_ADMIN', 'GATEKEEPER'));

-- 2. Extend society_gates with operational parameters
ALTER TABLE society_gates 
    ADD COLUMN IF NOT EXISTS directionality TEXT NOT NULL DEFAULT 'BIDIRECTIONAL',
    ADD COLUMN IF NOT EXISTS operational_status TEXT NOT NULL DEFAULT 'ACTIVE';

ALTER TABLE society_gates DROP CONSTRAINT IF EXISTS society_gates_directionality_check;
ALTER TABLE society_gates ADD CONSTRAINT society_gates_directionality_check 
    CHECK (directionality IN ('BIDIRECTIONAL', 'INGRESS_ONLY', 'EGRESS_ONLY'));

ALTER TABLE society_gates DROP CONSTRAINT IF EXISTS society_gates_operational_status_check;
ALTER TABLE society_gates ADD CONSTRAINT society_gates_operational_status_check 
    CHECK (operational_status IN ('ACTIVE', 'INACTIVE', 'MAINTENANCE', 'EMERGENCY_OPEN'));

-- 3. Shift Template / Schedule table
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

-- 4. Active Gate Duty Sessions table
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
