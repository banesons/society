-- Migration: 20261021000000_add_unit_visitor_approval_preference.sql
-- Description: Add resident-controlled visitor approval policy to society_units, defaulting to FALSE (OFF).

ALTER TABLE society_units
    ADD COLUMN IF NOT EXISTS requires_visitor_approval BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN society_units.requires_visitor_approval IS 
    'Unit-level visitor approval policy controlled exclusively by authorized residents. Defaults to FALSE (direct entry permitted subject to gate check).';
