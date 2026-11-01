-- Migration: 20261030000000_create_visitor_identities_schema.sql
-- Description: Unified Visitor Identities, Visitor-to-User Linking, and Cross-Gate Visit Relationships

-- 1. Create society_visitor_identities table
CREATE TABLE IF NOT EXISTS society_visitor_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    phone TEXT NOT NULL,
    full_name TEXT NOT NULL,
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    linked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_visitor_identities_society_phone_uq UNIQUE (society_id, phone),
    CONSTRAINT society_visitor_identities_id_society_uq UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_visitor_identities_society_phone
    ON society_visitor_identities (society_id, phone);

CREATE INDEX IF NOT EXISTS idx_visitor_identities_user_id
    ON society_visitor_identities (user_id)
    WHERE user_id IS NOT NULL;

-- 2. Add visitor_identity_id column to society_visits
ALTER TABLE society_visits
    ADD COLUMN IF NOT EXISTS visitor_identity_id UUID NULL;

-- 3. Backfill visitor identities from existing society_visits
INSERT INTO society_visitor_identities (society_id, phone, full_name, user_id, linked_at, created_at, updated_at)
SELECT DISTINCT ON (v.society_id, v.visitor_phone)
    v.society_id,
    v.visitor_phone,
    v.visitor_name,
    u.id,
    CASE WHEN u.id IS NOT NULL THEN NOW() ELSE NULL END,
    MIN(v.created_at) OVER (PARTITION BY v.society_id, v.visitor_phone),
    NOW()
FROM society_visits v
LEFT JOIN users u ON u.phone = v.visitor_phone
ON CONFLICT (society_id, phone) DO UPDATE
SET user_id = EXCLUDED.user_id,
    linked_at = COALESCE(society_visitor_identities.linked_at, EXCLUDED.linked_at),
    updated_at = NOW()
WHERE society_visitor_identities.user_id IS NULL AND EXCLUDED.user_id IS NOT NULL;

-- 4. Update society_visits.visitor_identity_id with resolved visitor identity IDs
UPDATE society_visits v
SET visitor_identity_id = vi.id
FROM society_visitor_identities vi
WHERE v.society_id = vi.society_id AND v.visitor_phone = vi.phone
  AND v.visitor_identity_id IS NULL;

-- 5. Composite Foreign Key Constraint on society_visits
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_visits_visitor_identity'
    ) THEN
        ALTER TABLE society_visits
            ADD CONSTRAINT fk_visits_visitor_identity
            FOREIGN KEY (visitor_identity_id, society_id)
            REFERENCES society_visitor_identities(id, society_id)
            ON DELETE RESTRICT;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_visits_visitor_identity
    ON society_visits (society_id, visitor_identity_id);
