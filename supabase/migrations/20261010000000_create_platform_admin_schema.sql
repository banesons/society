-- Migration: 20261010000000_create_platform_admin_schema.sql
-- Description: Platform administration roles and append-only audit trail schema

-- 1. Platform Roles Table (Strictly 1 platform role per user in S02)
CREATE TABLE IF NOT EXISTS platform_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('SUPER_ADMIN', 'ADMIN')),
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'DISABLED')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS platform_roles_user_id_idx ON platform_roles(user_id);
CREATE INDEX IF NOT EXISTS platform_roles_active_idx ON platform_roles(user_id, role, status);

-- 2. Platform Audit Logs Table (Immutable Append-Only Ledger)
CREATE TABLE IF NOT EXISTS platform_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    target_user_id UUID REFERENCES users(id),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    client_ip TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS platform_audit_logs_actor_idx ON platform_audit_logs(actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS platform_audit_logs_target_idx ON platform_audit_logs(target_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS platform_audit_logs_action_idx ON platform_audit_logs(action, created_at DESC);

-- 3. Database-Level Immutability Enforcement for Platform Audit Logs
CREATE OR REPLACE FUNCTION prevent_audit_log_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'platform_audit_logs is append-only: updates and deletes are forbidden';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS platform_audit_logs_immutability_trg ON platform_audit_logs;

CREATE TRIGGER platform_audit_logs_immutability_trg
BEFORE UPDATE OR DELETE ON platform_audit_logs
FOR EACH ROW
EXECUTE FUNCTION prevent_audit_log_modification();
