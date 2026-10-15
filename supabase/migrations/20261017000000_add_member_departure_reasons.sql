-- Migration: Add structured departure reason columns and constraints to society_members
-- S06: Members & Relationships

ALTER TABLE society_members
    ADD COLUMN IF NOT EXISTS departure_reason_code TEXT NULL,
    ADD COLUMN IF NOT EXISTS departure_reason_details TEXT NULL,
    ADD COLUMN IF NOT EXISTS departed_at TIMESTAMPTZ NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_society_members_departure_reason'
    ) THEN
        ALTER TABLE society_members
            ADD CONSTRAINT chk_society_members_departure_reason CHECK (
                departure_reason_code IS NULL OR departure_reason_code IN (
                    'TENANT_MOVED_OUT',
                    'LEASE_EXPIRED',
                    'OWNERSHIP_TRANSFERRED',
                    'MEMBER_RELOCATED',
                    'FAMILY_MEMBER_DEPARTED',
                    'MEMBERSHIP_TERMINATED',
                    'DECEASED',
                    'OTHER'
                )
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_society_members_departure_other_details'
    ) THEN
        ALTER TABLE society_members
            ADD CONSTRAINT chk_society_members_departure_other_details CHECK (
                departure_reason_code != 'OTHER' OR (departure_reason_details IS NOT NULL AND TRIM(departure_reason_details) != '')
            );
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_society_members_departure_details_len'
    ) THEN
        ALTER TABLE society_members
            ADD CONSTRAINT chk_society_members_departure_details_len CHECK (
                departure_reason_details IS NULL OR length(departure_reason_details) <= 500
            );
    END IF;
END $$;
