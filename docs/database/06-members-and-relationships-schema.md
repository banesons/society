# 06 — Members & Relationships Database Schema

## Overview

Sprint S06 introduces two primary database tables (`society_members` and `unit_relationships`) and strengthens multi-tenant referential integrity across the society schema.

---

## 1. Schema Definitions

### 1.1 `society_units` Enhancement
To enable composite foreign key protection, `society_units` adds a composite unique constraint:

```sql
-- Ensure composite uniqueness for composite foreign key references
ALTER TABLE society_units 
    ADD CONSTRAINT society_units_id_society_uq UNIQUE (id, society_id);
```

### 1.2 `society_members` Table
Stores resident and member profiles scoped to a housing society tenant.

```sql
CREATE TABLE IF NOT EXISTS society_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    user_id UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    first_name TEXT NOT NULL,
    last_name TEXT NULL,
    phone TEXT NOT NULL,
    email TEXT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'LEFT')),
    departure_reason_code TEXT NULL,
    departure_reason_details TEXT NULL,
    departed_at TIMESTAMPTZ NULL,
    emergency_contact_name TEXT NULL,
    emergency_contact_phone TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite uniqueness for composite foreign key references
    CONSTRAINT society_members_id_society_uq UNIQUE (id, society_id),

    -- Departure reason constraints
    CONSTRAINT chk_society_members_departure_reason CHECK (
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
    ),
    CONSTRAINT chk_society_members_departure_other_details CHECK (
        departure_reason_code != 'OTHER' OR (departure_reason_details IS NOT NULL AND TRIM(departure_reason_details) != '')
    ),
    CONSTRAINT chk_society_members_departure_details_len CHECK (
        departure_reason_details IS NULL OR length(departure_reason_details) <= 500
    )
);

-- Partial Unique Index: Prevent duplicate active members with same phone in same society
CREATE UNIQUE INDEX IF NOT EXISTS uq_society_members_active_phone 
    ON society_members (society_id, phone) 
    WHERE status = 'ACTIVE';

-- Partial Unique Index: Prevent linking same user to multiple person records in same society
CREATE UNIQUE INDEX IF NOT EXISTS uq_society_members_user_per_society 
    ON society_members (society_id, user_id) 
    WHERE user_id IS NOT NULL;

-- General Query Indexes
CREATE INDEX IF NOT EXISTS idx_society_members_society_status 
    ON society_members (society_id, status);

CREATE INDEX IF NOT EXISTS idx_society_members_phone 
    ON society_members (phone);

CREATE INDEX IF NOT EXISTS idx_society_members_user_id 
    ON society_members (user_id);
```

### 1.3 `unit_relationships` Table
Stores temporal ownership, tenancy, and occupancy associations between members and physical units.

```sql
CREATE TABLE IF NOT EXISTS unit_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    member_id UUID NOT NULL,
    relationship_type TEXT NOT NULL CHECK (
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
    is_occupant BOOLEAN NOT NULL DEFAULT true,
    ownership_share NUMERIC(5,2) NULL,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'ENDED')),
    notes TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Composite Foreign Keys: Enforce strict society consistency
    CONSTRAINT fk_unit_rel_society_unit 
        FOREIGN KEY (unit_id, society_id) 
        REFERENCES society_units(id, society_id) 
        ON DELETE RESTRICT,

    CONSTRAINT fk_unit_rel_society_member 
        FOREIGN KEY (member_id, society_id) 
        REFERENCES society_members(id, society_id) 
        ON DELETE RESTRICT,

    -- Temporal consistency
    CONSTRAINT chk_unit_rel_dates 
        CHECK (start_date <= COALESCE(end_date, start_date)),

    -- Status and end_date consistency
    CONSTRAINT chk_unit_rel_status_dates 
        CHECK (
            (status = 'ACTIVE' AND end_date IS NULL) OR 
            (status = 'ENDED' AND end_date IS NOT NULL)
        ),

    -- Ownership share validity: Only allowed for owners
    CONSTRAINT chk_unit_rel_share_valid 
        CHECK (
            (relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER') AND 
             (ownership_share IS NULL OR (ownership_share > 0 AND ownership_share <= 100)))
            OR
            (relationship_type NOT IN ('PRIMARY_OWNER', 'SECONDARY_OWNER') AND 
             ownership_share IS NULL)
        ),

    -- Occupancy validity: AUTHORIZED_OCCUPANT must be an occupant
    CONSTRAINT chk_unit_rel_occupant_valid 
        CHECK (
            relationship_type != 'AUTHORIZED_OCCUPANT' OR is_occupant = true
        )
);

-- Partial Unique Index: Prevent duplicate active identical relationships for same person in same unit
CREATE UNIQUE INDEX IF NOT EXISTS uq_unit_rel_active_duplicate 
    ON unit_relationships (unit_id, member_id, relationship_type) 
    WHERE status = 'ACTIVE';

-- General Query Indexes
CREATE INDEX IF NOT EXISTS idx_unit_relationships_unit_status 
    ON unit_relationships (unit_id, status);

CREATE INDEX IF NOT EXISTS idx_unit_relationships_member_status 
    ON unit_relationships (member_id, status);

CREATE INDEX IF NOT EXISTS idx_unit_relationships_society 
    ON unit_relationships (society_id);
```

---

## 2. Integrity and Concurrency Control Rules

1. **Composite Multi-Tenant Isolation**:
   - The composite foreign keys `(unit_id, society_id)` and `(member_id, society_id)` guarantee that a unit relationship cannot reference a unit and member from different societies.
2. **Referential Deletion Protection**:
   - `ON DELETE RESTRICT` on `users(id)`, `society_units`, and `society_members` ensures that historical audit trails, accounting records, and dispute logs cannot be orphaned.
3. **Pessimistic Unit Row Locking for Aggregate Shares**:
   - When modifying ownership shares on a unit:
     ```sql
     SELECT id FROM society_units WHERE id = $1 AND society_id = $2 FOR UPDATE;
     ```
   - Serializes concurrent transactions targeting the same physical unit, guaranteeing that aggregate percentage calculations cannot suffer race conditions. Total of all specified percentages is enforced $\le 100.00\%$. Unspecified percentages remain `NULL`.
4. **Departure Concurrency & Version Token**:
   - The two-phase departure workflow computes a SHA-256 hash over the member's active relationship IDs and timestamps during preview.
   - Confirmation checks this hash against the live database state under transaction, aborting with conflict (`409 PREVIEW_STALE`) if concurrent writes occurred.

---

## 3. Transactional Audit Log Events

Mutations execute in a database transaction synchronously inserting into `platform_audit_logs`:

| Event | Target Type | Target ID | Metadata Captured |
| :--- | :--- | :--- | :--- |
| `MEMBER_REGISTERED` | `MEMBER` | `society_members.id` | `first_name`, `phone_mask`, `society_id` |
| `MEMBER_UPDATED` | `MEMBER` | `society_members.id` | Changed fields, `society_id` |
| `MEMBER_STATUS_CHANGED` | `MEMBER` | `society_members.id` | `old_status`, `new_status`, `society_id` |
| `MEMBER_DEPARTED` | `MEMBER` | `society_members.id` | `reason`, `society_id`, affected unit IDs |
| `UNIT_RELATIONSHIP_CREATED` | `UNIT_RELATIONSHIP` | `unit_relationships.id` | `unit_id`, `member_id`, `relationship_type`, `is_occupant`, `ownership_share` |
| `UNIT_RELATIONSHIP_ENDED` | `UNIT_RELATIONSHIP` | `unit_relationships.id` | `unit_id`, `member_id`, `end_date`, `reason` |

---

## 4. References
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
