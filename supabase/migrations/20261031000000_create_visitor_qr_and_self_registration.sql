-- Migration: 20261031000000_create_visitor_qr_and_self_registration.sql
-- Description: Gate QR Configuration, Public Visitor Self-Registration, Photo & Pass Reference Support

-- 1. Create society_gate_qrs table for administrator-managed QR codes
CREATE TABLE IF NOT EXISTS society_gate_qrs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    gate_id UUID NULL,
    token TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'REVOKED')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_gate_qrs_society FOREIGN KEY (society_id) REFERENCES societies(id) ON DELETE RESTRICT,
    CONSTRAINT fk_gate_qrs_gate FOREIGN KEY (gate_id, society_id) REFERENCES society_gates(id, society_id) ON DELETE CASCADE,
    CONSTRAINT society_gate_qrs_id_society_uq UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_society_gate_qrs_token ON society_gate_qrs (token);
CREATE INDEX IF NOT EXISTS idx_society_gate_qrs_society ON society_gate_qrs (society_id, status);
CREATE INDEX IF NOT EXISTS idx_society_gate_qrs_gate ON society_gate_qrs (gate_id) WHERE gate_id IS NOT NULL;

-- 2. Modify society_visits for public self-registration support
-- Make created_by nullable so unauthenticated public registrations can be inserted safely
ALTER TABLE society_visits
    ALTER COLUMN created_by DROP NOT NULL;

-- Add photo, origin, pass reference, and QR linkage columns
ALTER TABLE society_visits
    ADD COLUMN IF NOT EXISTS visitor_photo_url TEXT NULL,
    ADD COLUMN IF NOT EXISTS coming_from TEXT NULL,
    ADD COLUMN IF NOT EXISTS pass_reference TEXT NULL,
    ADD COLUMN IF NOT EXISTS qr_id UUID NULL;

-- Add index on pass_reference for fast lookup by gatekeeper and visitor
CREATE INDEX IF NOT EXISTS idx_visits_pass_reference ON society_visits (society_id, pass_reference)
    WHERE pass_reference IS NOT NULL;

-- Add foreign key constraint for qr_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_visits_qr_id'
    ) THEN
        ALTER TABLE society_visits
            ADD CONSTRAINT fk_visits_qr_id
            FOREIGN KEY (qr_id, society_id)
            REFERENCES society_gate_qrs(id, society_id)
            ON DELETE SET NULL;
    END IF;
END $$;

-- 3. Update check constraints to support QR_SELF_REGISTER and REGISTERED_PENDING_GATE_VERIFICATION
ALTER TABLE society_visits DROP CONSTRAINT IF EXISTS chk_visits_type;
ALTER TABLE society_visits ADD CONSTRAINT chk_visits_type CHECK (
    visit_type IN ('EXPECTED', 'WALK_IN', 'QR_SELF_REGISTER')
);

ALTER TABLE society_visits DROP CONSTRAINT IF EXISTS chk_visits_status;
ALTER TABLE society_visits ADD CONSTRAINT chk_visits_status CHECK (
    status IN (
        'PENDING_APPROVAL', 
        'REGISTERED_PENDING_GATE_VERIFICATION',
        'APPROVED', 
        'REJECTED', 
        'CHECKED_IN', 
        'CHECKED_OUT', 
        'CANCELLED', 
        'EXPIRED'
    )
);
