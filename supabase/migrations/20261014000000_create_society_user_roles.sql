-- Migration: 20261014000000_create_society_user_roles.sql
-- Description: Create society_user_roles table for society administration (S04)

CREATE TABLE IF NOT EXISTS society_user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role TEXT NOT NULL DEFAULT 'SOC_ADMIN',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    assigned_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_user_roles_role_check CHECK (role IN ('SOC_ADMIN')),
    CONSTRAINT society_user_roles_status_check CHECK (status IN ('ACTIVE', 'REVOKED')),
    CONSTRAINT society_user_roles_revocation_check CHECK (
        (status = 'ACTIVE' AND revoked_at IS NULL AND revoked_by IS NULL) OR
        (status = 'REVOKED' AND revoked_at IS NOT NULL)
    )
);

-- Partial unique index: at most one ACTIVE assignment per (society_id, user_id, role)
-- Allows historical REVOKED records without conflict
CREATE UNIQUE INDEX IF NOT EXISTS society_user_roles_active_uq
    ON society_user_roles (society_id, user_id, role)
    WHERE status = 'ACTIVE';

-- Lookup index for user's active society assignments (login discovery and context)
CREATE INDEX IF NOT EXISTS society_user_roles_user_active_idx
    ON society_user_roles (user_id, status)
    WHERE status = 'ACTIVE';

-- Lookup index for society administration roster in platform management
CREATE INDEX IF NOT EXISTS society_user_roles_society_roster_idx
    ON society_user_roles (society_id, status, created_at DESC);

-- Composite index for fast authorization verification
CREATE INDEX IF NOT EXISTS society_user_roles_auth_lookup_idx
    ON society_user_roles (society_id, user_id, role, status);
