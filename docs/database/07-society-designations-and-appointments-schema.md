# 07 — Society Designations and Appointments Database Schema

## 1. Overview
Sprint S07 introduces two core tables to support society-level committee designations and member appointments:
1. `society_designations`: Scoped catalogue of governance positions.
2. `society_committee_appointments`: Historical and active terms of office held by society members.

---

## 2. Table: `society_designations`

Stores the titles and cardinality rules of committee roles within a society.

```sql
CREATE TABLE society_designations (
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

CREATE INDEX idx_society_designations_society_status 
    ON society_designations (society_id, is_active);
```

### Columns
| Column | Type | Nullable | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | No | Primary key surrogate identifier. |
| `society_id` | `UUID` | No | Tenant scoping foreign key to `societies(id)`. |
| `code` | `TEXT` | No | Stable canonical code (`CHAIRMAN`, `SECRETARY`, `TREASURER`, etc.). |
| `name` | `TEXT` | No | Friendly human-readable designation title. |
| `description` | `TEXT` | Yes | Description of committee responsibilities. |
| `allows_multiple_members` | `BOOLEAN` | No | If `FALSE`, only 1 active appointment allowed concurrently. |
| `allows_multiple_positions_per_member` | `BOOLEAN` | No | If `TRUE`, member can simultaneously hold other designations. |
| `is_active` | `BOOLEAN` | No | Catalogue lifecycle flag. Inactive positions cannot accept new appointments. |
| `created_at` | `TIMESTAMPTZ` | No | Timestamp of creation. |
| `updated_at` | `TIMESTAMPTZ` | No | Timestamp of last modification. |

---

## 3. Table: `society_committee_appointments`

Tracks terms and appointments of society members to designated positions.

```sql
CREATE TABLE society_committee_appointments (
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

    -- Composite Foreign Keys enforcing multi-tenant isolation
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

-- Partial index preventing duplicate active appointment for the identical designation and member
CREATE UNIQUE INDEX uq_appointment_active_member_designation 
    ON society_committee_appointments (society_id, member_id, designation_id) 
    WHERE status = 'ACTIVE';

-- Lookup indexes for committee views
CREATE INDEX idx_appointments_society_status 
    ON society_committee_appointments (society_id, status);

CREATE INDEX idx_appointments_member 
    ON society_committee_appointments (member_id, society_id);

CREATE INDEX idx_appointments_designation 
    ON society_committee_appointments (designation_id, society_id);
```

### Columns
| Column | Type | Nullable | Description |
| :--- | :--- | :--- | :--- |
| `id` | `UUID` | No | Primary key surrogate identifier. |
| `society_id` | `UUID` | No | Tenant scoping foreign key to `societies(id)`. |
| `designation_id` | `UUID` | No | Foreign key referencing `society_designations`. |
| `member_id` | `UUID` | No | Foreign key referencing `society_members`. |
| `start_date` | `DATE` | No | Date when tenure commences. |
| `expected_end_date` | `DATE` | Yes | Projected expiry of term (e.g. AGM date). |
| `end_date` | `DATE` | Yes | Actual term completion or revocation date. |
| `status` | `TEXT` | No | `ACTIVE`, `ENDED`, `REVOKED`. |
| `notes` | `TEXT` | Yes | Administrative or AGM resolution notes. |
| `created_at` | `TIMESTAMPTZ` | No | Timestamp of creation. |
| `updated_at` | `TIMESTAMPTZ` | No | Timestamp of last modification. |

---

## 4. Default Seed Catalogue

Upon first initialization or query in an active society, the system seeds 4 canonical designations if no designations exist:
1. `CHAIRMAN`: "Chairman", singleton (`allows_multiple_members = false`).
2. `SECRETARY`: "Secretary", singleton (`allows_multiple_members = false`).
3. `TREASURER`: "Treasurer", singleton (`allows_multiple_members = false`).
4. `COMMITTEE_MEMBER`: "Committee Member", multi-member (`allows_multiple_members = true`).

---

## 5. Related Documentation
- [[../decisions/019-society-designations-and-appointments|ADR 019: Society Designations, Committee Appointments, and Role Segregation]]
- [[../architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]
- [[../api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API Specification]]
- [[../security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security Specification]]
