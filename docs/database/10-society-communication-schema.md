# 10 — Society Communication Database Schema

## Overview
Sprint S10 introduces two tables: `society_announcements` and `society_announcement_reads`, and adds composite uniqueness constraints on `society_buildings` and `society_wings` to guarantee multi-tenant referential integrity.

---

## 1. Schema Definitions

### 1.1 Structural Uniqueness Enhancements
To allow composite foreign keys matching `(target_building_id, society_id)` and `(target_wing_id, society_id)`, composite unique constraints are established:

```sql
ALTER TABLE society_buildings 
    ADD CONSTRAINT society_buildings_id_society_uq UNIQUE (id, society_id);

ALTER TABLE society_wings 
    ADD CONSTRAINT society_wings_id_society_uq UNIQUE (id, society_id);
```

### 1.2 `society_announcements` Table
Stores notices and announcements across their draft, published, and archived lifecycle.

```sql
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
```

### 1.3 `society_announcement_reads` Table
Tracks read receipts per authenticated user and announcement.

```sql
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
```

---

## 2. Related Documentation
- [[../decisions/024-society-communication-and-announcements|ADR 024: Society Communication]]
- [[../architecture/15-society-communication-announcements|Architecture 15: Society Communication]]
- [[../api/11-society-communication-api|API 11: Society Communication API]]
- [[../security/10-society-communication-security|Security 10: Society Communication Security]]
- [[../sprints/S10-society-communication|S10 Sprint Specification]]
