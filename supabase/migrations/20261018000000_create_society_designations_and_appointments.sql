-- S07: Society Designations and Committee Appointments Schema
-- Models society-specific governance positions (Chairman, Secretary, Treasurer, etc.)
-- and committee appointment tenures with composite multi-tenant foreign keys,
-- lifecycle validation, and historical preservation.

-- 1. Society Designations Table (Governance Roles Catalogue)
CREATE TABLE IF NOT EXISTS society_designations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NULL,
    allows_multiple_members BOOLEAN NOT NULL DEFAULT FALSE,
    allows_multiple_positions_per_member BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_society_designations_code UNIQUE (society_id, code),
    CONSTRAINT uq_society_designations_tenant UNIQUE (id, society_id),
    CONSTRAINT chk_society_designations_code_len CHECK (code ~ '^[A-Z0-9_]{2,50}$'),
    CONSTRAINT chk_society_designations_name_len CHECK (length(trim(name)) >= 2 AND length(trim(name)) <= 100)
);

CREATE INDEX IF NOT EXISTS idx_society_designations_society_status 
    ON society_designations (society_id, is_active);

-- 2. Committee Appointments Table (Terms & Tenures)
CREATE TABLE IF NOT EXISTS society_committee_appointments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    designation_id UUID NOT NULL,
    member_id UUID NOT NULL,
    start_date DATE NOT NULL,
    expected_end_date DATE NULL,
    end_date DATE NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite Foreign Keys enforcing multi-tenant boundary
    CONSTRAINT fk_appointments_designation 
        FOREIGN KEY (designation_id, society_id) 
        REFERENCES society_designations(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_appointments_member 
        FOREIGN KEY (member_id, society_id) 
        REFERENCES society_members(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT chk_appointments_status 
        CHECK (status IN ('ACTIVE', 'ENDED', 'REVOKED')),

    CONSTRAINT chk_appointments_dates 
        CHECK (start_date <= COALESCE(end_date, expected_end_date, start_date)),

    CONSTRAINT chk_appointments_active_end_date 
        CHECK ((status = 'ACTIVE' AND end_date IS NULL) OR (status IN ('ENDED', 'REVOKED') AND end_date IS NOT NULL))
);

-- Partial Unique Index: Prevent a member from actively holding the identical designation more than once concurrently
CREATE UNIQUE INDEX IF NOT EXISTS uq_appointment_active_member_designation 
    ON society_committee_appointments (society_id, member_id, designation_id) 
    WHERE status = 'ACTIVE';

-- Query Indexes for committee roster and member appointments
CREATE INDEX IF NOT EXISTS idx_appointments_society_status 
    ON society_committee_appointments (society_id, status);

CREATE INDEX IF NOT EXISTS idx_appointments_member 
    ON society_committee_appointments (member_id, society_id);

CREATE INDEX IF NOT EXISTS idx_appointments_designation 
    ON society_committee_appointments (designation_id, society_id);
