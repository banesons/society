-- S10: Society Communication Schema Migration
-- Introduces society_announcements, society_announcement_reads, and composite target constraints.

-- 1. Ensure composite uniqueness on society_buildings and society_wings for multi-tenant composite foreign keys
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'society_buildings_id_society_uq'
    ) THEN
        ALTER TABLE society_buildings 
            ADD CONSTRAINT society_buildings_id_society_uq UNIQUE (id, society_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'society_wings_id_society_uq'
    ) THEN
        ALTER TABLE society_wings 
            ADD CONSTRAINT society_wings_id_society_uq UNIQUE (id, society_id);
    END IF;
END $$;

-- 2. Announcements Table
CREATE TABLE IF NOT EXISTS society_announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL CHECK (char_length(trim(title)) > 0),
    body TEXT NOT NULL CHECK (char_length(trim(body)) > 0),
    category VARCHAR(50) NOT NULL DEFAULT 'GENERAL' CHECK (
        category IN ('GENERAL', 'MAINTENANCE', 'SECURITY', 'EVENT', 'EMERGENCY', 'ADMINISTRATIVE')
    ),
    priority VARCHAR(20) NOT NULL DEFAULT 'NORMAL' CHECK (
        priority IN ('NORMAL', 'URGENT', 'CRITICAL')
    ),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (
        status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')
    ),
    audience_type VARCHAR(20) NOT NULL DEFAULT 'SOCIETY' CHECK (
        audience_type IN ('SOCIETY', 'BUILDING', 'WING', 'UNIT')
    ),
    target_building_id UUID NULL,
    target_wing_id UUID NULL,
    target_unit_id UUID NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_by UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ NULL,
    archived_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite foreign keys ensuring target belongs strictly to the same society
    CONSTRAINT fk_announcements_building FOREIGN KEY (target_building_id, society_id)
        REFERENCES society_buildings(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_announcements_wing FOREIGN KEY (target_wing_id, society_id)
        REFERENCES society_wings(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_announcements_unit FOREIGN KEY (target_unit_id, society_id)
        REFERENCES society_units(id, society_id) ON DELETE CASCADE,

    -- Audience target consistency check
    CONSTRAINT chk_announcement_audience_targets CHECK (
        (audience_type = 'SOCIETY' AND target_building_id IS NULL AND target_wing_id IS NULL AND target_unit_id IS NULL) OR
        (audience_type = 'BUILDING' AND target_building_id IS NOT NULL AND target_wing_id IS NULL AND target_unit_id IS NULL) OR
        (audience_type = 'WING' AND target_wing_id IS NOT NULL AND target_building_id IS NULL AND target_unit_id IS NULL) OR
        (audience_type = 'UNIT' AND target_unit_id IS NOT NULL AND target_building_id IS NULL AND target_wing_id IS NULL)
    ),

    -- Lifecycle publication timestamp consistency
    CONSTRAINT chk_announcement_publication_consistency CHECK (
        (status = 'DRAFT' AND published_at IS NULL AND published_by IS NULL) OR
        (status IN ('PUBLISHED', 'ARCHIVED') AND published_at IS NOT NULL AND published_by IS NOT NULL)
    )
);

-- Query and sorting indexes
CREATE INDEX IF NOT EXISTS idx_announcements_society_status_pub 
    ON society_announcements (society_id, status, published_at DESC);

CREATE INDEX IF NOT EXISTS idx_announcements_society_created 
    ON society_announcements (society_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_announcements_targets 
    ON society_announcements (society_id, audience_type, target_building_id, target_wing_id, target_unit_id);

-- 3. Announcement Read Receipts Table
CREATE TABLE IF NOT EXISTS society_announcement_reads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
    announcement_id UUID NOT NULL REFERENCES society_announcements(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Prevent duplicate read records
    CONSTRAINT society_announcement_reads_uq UNIQUE (announcement_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_announcement_reads_lookup 
    ON society_announcement_reads (society_id, user_id, announcement_id);
