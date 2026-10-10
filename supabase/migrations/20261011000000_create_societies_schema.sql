-- Migration: 20261011000000_create_societies_schema.sql
-- Description: Housing societies multi-tenant core schema, constraints, immutability trigger and indexes

-- 1. Societies Table
CREATE TABLE IF NOT EXISTS societies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_code TEXT NOT NULL,
    name TEXT NOT NULL,
    registration_number TEXT,
    address_line1 TEXT NOT NULL,
    address_line2 TEXT,
    locality TEXT NOT NULL,
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    pincode VARCHAR(6) NOT NULL,
    contact_email TEXT,
    contact_phone TEXT,
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT societies_status_check CHECK (status IN ('PENDING', 'ACTIVE', 'SUSPENDED')),
    CONSTRAINT societies_code_format_check CHECK (society_code ~ '^SOC-[A-Z0-9]{3}-[A-Z0-9]{6}$'),
    CONSTRAINT societies_pincode_format_check CHECK (pincode ~ '^[1-9][0-9]{5}$'),
    CONSTRAINT societies_name_length_check CHECK (length(trim(name)) >= 3 AND length(name) <= 255)
);

-- 2. Indexes
CREATE UNIQUE INDEX IF NOT EXISTS societies_code_unique_idx ON societies(upper(society_code));
CREATE INDEX IF NOT EXISTS societies_status_created_idx ON societies(status, created_at DESC);
CREATE INDEX IF NOT EXISTS societies_location_idx ON societies(state, city);
CREATE INDEX IF NOT EXISTS societies_name_lower_idx ON societies(lower(name));

-- 3. Code Immutability Trigger
CREATE OR REPLACE FUNCTION prevent_society_code_update()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.society_code IS DISTINCT FROM OLD.society_code THEN
        RAISE EXCEPTION 'society_code is immutable and cannot be updated';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS societies_code_immutability_trg ON societies;

CREATE TRIGGER societies_code_immutability_trg
BEFORE UPDATE ON societies
FOR EACH ROW
EXECUTE FUNCTION prevent_society_code_update();
