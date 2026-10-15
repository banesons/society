-- S06: Members and Unit Relationships Schema
-- Establishes society_members and unit_relationships with composite foreign keys,
-- concurrency-safe uniqueness constraints, and lifecycle validation rules.

-- 1. Ensure composite uniqueness on society_units for composite foreign key references
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'society_units_id_society_uq'
    ) THEN
        ALTER TABLE society_units 
            ADD CONSTRAINT society_units_id_society_uq UNIQUE (id, society_id);
    END IF;
END $$;

-- 2. Members Table (Person Registry Scoped to Society)
CREATE TABLE IF NOT EXISTS society_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    user_id UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    first_name TEXT NOT NULL,
    last_name TEXT NULL,
    phone TEXT NOT NULL,
    email TEXT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    emergency_contact_name TEXT NULL,
    emergency_contact_phone TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_members_status_check CHECK (status IN ('ACTIVE', 'INACTIVE', 'LEFT')),
    CONSTRAINT society_members_id_society_uq UNIQUE (id, society_id)
);

-- Partial Unique Index: Prevent duplicate active members with same phone in same society
CREATE UNIQUE INDEX IF NOT EXISTS uq_society_members_active_phone 
    ON society_members (society_id, phone) 
    WHERE status = 'ACTIVE';

-- Partial Unique Index: Prevent linking same user to multiple member records in same society
CREATE UNIQUE INDEX IF NOT EXISTS uq_society_members_user_per_society 
    ON society_members (society_id, user_id) 
    WHERE user_id IS NOT NULL;

-- Query Indexes for Members
CREATE INDEX IF NOT EXISTS idx_society_members_society_status 
    ON society_members (society_id, status);

CREATE INDEX IF NOT EXISTS idx_society_members_phone 
    ON society_members (phone);

CREATE INDEX IF NOT EXISTS idx_society_members_user_id 
    ON society_members (user_id);


-- 3. Unit Relationships Table (Temporal Ownership and Occupancy Associations)
CREATE TABLE IF NOT EXISTS unit_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    member_id UUID NOT NULL,
    relationship_type TEXT NOT NULL,
    is_occupant BOOLEAN NOT NULL DEFAULT true,
    ownership_share NUMERIC(5,2) NULL,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite Foreign Keys: Mathematically enforce strict society consistency
    CONSTRAINT fk_unit_rel_society_unit 
        FOREIGN KEY (unit_id, society_id) 
        REFERENCES society_units(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_unit_rel_society_member 
        FOREIGN KEY (member_id, society_id) 
        REFERENCES society_members(id, society_id) 
        ON DELETE RESTRICT,

    -- Relationship Type Check
    CONSTRAINT chk_unit_rel_type CHECK (
        relationship_type IN (
            'PRIMARY_OWNER',
            'SECONDARY_OWNER',
            'OWNER_FAMILY',
            'PRIMARY_TENANT',
            'SECONDARY_TENANT',
            'TENANT_FAMILY',
            'AUTHORIZED_OCCUPANT'
        )
    ),

    -- Status Check
    CONSTRAINT chk_unit_rel_status CHECK (
        status IN ('ACTIVE', 'ENDED')
    ),

    -- Temporal consistency
    CONSTRAINT chk_unit_rel_dates CHECK (
        start_date <= COALESCE(end_date, start_date)
    ),

    -- Status and end_date consistency
    CONSTRAINT chk_unit_rel_status_dates CHECK (
        (status = 'ACTIVE' AND end_date IS NULL) OR 
        (status = 'ENDED' AND end_date IS NOT NULL)
    ),

    -- Ownership share validity: Only owners may have share, between 0.01 and 100.00
    CONSTRAINT chk_unit_rel_share_valid CHECK (
        (relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER') AND 
         (ownership_share IS NULL OR (ownership_share > 0 AND ownership_share <= 100)))
        OR
        (relationship_type NOT IN ('PRIMARY_OWNER', 'SECONDARY_OWNER') AND 
         ownership_share IS NULL)
    ),

    -- Occupancy validity: AUTHORIZED_OCCUPANT must be an occupant
    CONSTRAINT chk_unit_rel_occupant_valid CHECK (
        relationship_type != 'AUTHORIZED_OCCUPANT' OR is_occupant = true
    )
);

-- Partial Unique Index: Prevent duplicate active identical relationships for same person in same unit
CREATE UNIQUE INDEX IF NOT EXISTS uq_unit_rel_active_duplicate 
    ON unit_relationships (unit_id, member_id, relationship_type) 
    WHERE status = 'ACTIVE';

-- Query Indexes for Relationships
CREATE INDEX IF NOT EXISTS idx_unit_relationships_unit_status 
    ON unit_relationships (unit_id, status);

CREATE INDEX IF NOT EXISTS idx_unit_relationships_member_status 
    ON unit_relationships (member_id, status);

CREATE INDEX IF NOT EXISTS idx_unit_relationships_society 
    ON unit_relationships (society_id);
