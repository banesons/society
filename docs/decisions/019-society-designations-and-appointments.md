# ADR 019: Society Designations, Committee Appointments, and Role Segregation

## Status
Accepted (S07 Foundation)

## Context
Sprint S04 established `society_user_roles` (`SOC_ADMIN`) for housing society system authorization. Sprint S06 established `society_members` and physical unit occupancy/ownership relationships (`unit_relationships`).

Sprint S07 requires modeling society governance roles and committee appointments (e.g., Chairman, Secretary, Treasurer, Committee Member):
1. **Designation vs. Authorization Grant Separation**: A designation (e.g., Chairman, Treasurer) is a civic, committee title. It is **not** an automated system authorization grant (`SOC_ADMIN`). A committee member only receives system administrator rights if an authorized administrator explicitly assigns them `SOC_ADMIN` in `society_user_roles`.
2. **Designation Catalogue Extensibility**: Different housing societies, apartment associations, and co-operatives have different bye-laws and committee structures. The system must provide sensible standard defaults while allowing societies to define custom designations.
3. **Cardinality Rules**:
   - Certain designations are singletons by nature or law (e.g., at most one active Chairman, Secretary, Treasurer per society).
   - Other designations accommodate multiple concurrent office-bearers (e.g., Committee Member, Floor Representative).
   - Cardinality must be enforced with concurrency protection.
4. **Member Reference Integrity**: Appointments must link strictly to valid `society_members` records within the same society. Pre-registered residents or linked users may be appointed without creating duplicate identity records.
5. **Historical Preservation & Departure Semantics**:
   - Appointments have defined effective terms (`start_date`, optional `expected_end_date`, and actual `end_date`).
   - Ending or revoking an appointment preserves the historical record with `status = 'ENDED'` or `'REVOKED'`. Records are **never** hard-deleted.
   - When a member departs the society (`society_members.status = 'LEFT'`), active appointments must be terminated automatically to prevent phantom committee members.
   - Inactive designations cannot be selected for new appointments, but existing and historical appointments remain readable.
6. **Multi-Tenant Isolation**: Database composite foreign keys must guarantee that designations and appointments cannot reference members or entities across different societies.

---

## Decision

### 1. Society-Scoped Designation Catalogue (`society_designations`)
We implement a configurable catalogue scoped to each society tenant:
- **Columns**:
  - `id`: UUID primary key.
  - `society_id`: Foreign key referencing `societies(id) ON DELETE RESTRICT`.
  - `code`: Stable uppercase identifier (e.g., `CHAIRMAN`, `SECRETARY`, `TREASURER`, `COMMITTEE_MEMBER`, `VICE_CHAIRMAN`).
  - `name`: Display name (e.g., "Chairman", "Hon. Secretary").
  - `description`: Optional text describing duties or qualifications.
  - `allows_multiple_members`: Boolean indicating whether multiple members may concurrently hold this designation.
  - `allows_multiple_positions_per_member`: Boolean indicating whether a member holding this designation can hold other positions simultaneously.
  - `is_active`: Boolean lifecycle toggle. When `false`, the designation is archived and hidden from new appointment selectors, but historical appointments remain intact.
- **Natural Uniqueness & Composite Integrity**:
  - `UNIQUE (society_id, code)` prevents duplicate designation codes in the same society.
  - `UNIQUE (id, society_id)` enables composite foreign key referencing.
- **Default Seed Catalogue**:
  When a society accesses designations for the first time or during formation, the catalogue seeds standard defaults:
  - `CHAIRMAN` (Singleton)
  - `SECRETARY` (Singleton)
  - `TREASURER` (Singleton)
  - `COMMITTEE_MEMBER` (Multi-member)

### 2. Committee Appointments (`society_committee_appointments`)
We track terms and designations held by society members:
- **Columns**:
  - `id`: UUID primary key.
  - `society_id`: Scoped tenant reference.
  - `designation_id`: Scoped designation reference.
  - `member_id`: Scoped member reference.
  - `start_date`: Date when term commences (`DATE NOT NULL`).
  - `expected_end_date`: Optional projected expiry date (`DATE NULL`).
  - `end_date`: Actual term conclusion date (`DATE NULL`).
  - `status`: Lifecycle state (`ACTIVE`, `ENDED`, `REVOKED`).
  - `notes`: Administrative notes or resolution reference.
- **Composite Foreign Key Multi-Tenancy**:
  ```sql
  FOREIGN KEY (designation_id, society_id) REFERENCES society_designations(id, society_id) ON DELETE RESTRICT
  FOREIGN KEY (member_id, society_id) REFERENCES society_members(id, society_id) ON DELETE RESTRICT
  ```
- **Constraints**:
  - `CHECK (start_date <= COALESCE(end_date, expected_end_date, start_date))`
  - `CHECK ((status = 'ACTIVE' AND end_date IS NULL) OR (status IN ('ENDED', 'REVOKED') AND end_date IS NOT NULL))`
  - Partial unique index: `(society_id, member_id, designation_id) WHERE status = 'ACTIVE'` prevents a member from holding the identical designation actively more than once concurrently.

### 3. Concurrency Protection & Cardinality Enforcement
- When creating an appointment for a designation where `allows_multiple_members = false`:
  1. Transaction acquires row lock on the designation record: `SELECT id, allows_multiple_members, is_active FROM society_designations WHERE id = $1 AND society_id = $2 FOR UPDATE`.
  2. Verifies that `is_active = true` (rejecting inactive designations with `400 DESIGNATION_INACTIVE`).
  3. Verifies that existing active appointments count for this designation is `0`. If $\ge 1$, rejects with `409 DESIGNATION_ALREADY_APPOINTED`.
  4. Verifies target member is `status = 'ACTIVE'` in `society_members` (rejecting inactive/departed members with `400 MEMBER_NOT_ACTIVE`).
  5. Inserts appointment and audit record transactionally.

### 4. Member Departure Interplay
- When a member departs (`POST .../departure/confirm` in S06):
  - In addition to terminating tenancy relationships and setting owner occupancy to `false`, the transaction automatically terminates all active committee appointments for that member:
    ```sql
    UPDATE society_committee_appointments
    SET status = 'ENDED',
        end_date = $endDate,
        notes = COALESCE(notes || ' | ', '') || 'Ended on member departure: ' || $reasonCode,
        updated_at = NOW()
    WHERE member_id = $memberID AND society_id = $societyID AND status = 'ACTIVE'
    ```
  - Emits `COMMITTEE_APPOINTMENT_ENDED` audit records for each terminated appointment.

### 5. Role & Authorization Boundaries
- **Platform Roles** (`platform_roles`): `SUPER_ADMIN` and `ADMIN`. Governs platform infrastructure and onboarding.
- **Society Authorization** (`society_user_roles`): `SOC_ADMIN`. Authorizes administrative API execution on society resources.
- **Society Designations** (`society_designations`): Committee titles. **No implicit API permissions are conferred by holding a designation.**
- Managing designations and appointments requires `SOC_ADMIN` access or platform `SUPER_ADMIN`.
- Unassigned platform `ADMIN` callers are barred (`403 SOCIETY_ACCESS_DENIED`).
- Suspended societies are barred from appointment and designation mutations (`403 SOCIETY_SUSPENDED`).

---

## Consequences
- Clean decoupled model separating governance titles from system access privileges.
- Flexible catalogue accommodating diverse society bye-law structures.
- Strict multi-tenant data integrity enforced by composite foreign keys.
- Guaranteed history preservation and departure safety.

## Related Documentation
- [[012-platform-role-separation|ADR 012: Separation of Platform Administration from Society Domain Roles]]
- [[016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
- [[018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]
- [[../database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]
- [[../api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API Specification]]
- [[../security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security Specification]]
- [[../sprints/S07-society-roles-designations|S07 — Society Roles & Designations Sprint]]
