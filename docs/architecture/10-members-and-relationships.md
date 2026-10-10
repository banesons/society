# 10 — Members & Relationships Architecture

## 1. Overview

Sprint S06 implements the domain model, database integrity constraints, and administrative workflows for people associated with a housing society and their physical relationships to units.

```text
┌────────────────────────────────────────────────────────┐
│                   Universal Identity                   │
│                      users (S01)                       │
│        (phone: verified via OTP, id: universal UUID)    │
└───────────────────────────┬────────────────────────────┘
                            │ 0..1 (verified OTP link)
                            │
┌───────────────────────────▼────────────────────────────┐
│                    Housing Society                     │
│                    societies (S03)                     │
└─────────────┬────────────────────────────┬─────────────┘
              │ 1..N                       │ 1..N
              │                            │
┌─────────────▼─────────────┐ ┌────────────▼─────────────┐
│      society_members      │ │      society_units       │
│  (first_name, last_name,  │ │ (flat number, floor,     │
│   phone, email, status)   │ │  building, wing)         │
│  UNIQUE (id, society_id)  │ │  UNIQUE (id, society_id) │
└─────────────┬─────────────┘ └────────────┬─────────────┘
              │                            │
              └──────────────┬─────────────┘
                             │ Composite FKs (unit_id, society_id)
                             │ & (member_id, society_id)
              ┌──────────────▼─────────────┐
              │     unit_relationships     │
              │  - relationship_type       │
              │  - is_occupant (boolean)   │
              │  - ownership_share (%)     │
              │  - start_date / end_date   │
              │  - status (ACTIVE / ENDED) │
              └────────────────────────────┘
```

---

## 2. Person & Identity Linking Concurrency

### 2.1 The Identity Separation Principle
- **`users` Table (Global)**: Represents verified platform accounts. Each record has completed phone OTP verification.
- **`society_members` Table (Tenant)**: Represents resident profiles within a specific society.
- **Pre-Registration**: When a society administrator adds a resident, `user_id` is initialized to `NULL`. **Zero synthetic accounts, fake passwords, or mock sessions are created.**

### 2.2 Canonical Phone Normalization & Duplicate Prevention
- All phone inputs are sanitized into canonical E.164 format (`+91XXXXXXXXXX`).
- White space, hyphens, and parentheses are stripped.
- Two administrators concurrently registering the same phone number in the same society are safely serialized:
  - Database constraint `uq_society_members_active_phone` raises PostgreSQL error `23505` (unique_violation) on the second write.
  - The repository catches `23505` and returns typed `ErrMemberAlreadyExists`, mapping cleanly to HTTP `409 MEMBER_ALREADY_EXISTS`.

### 2.3 Safe Linking Rules Upon Verification
When a user logs in via OTP with verified phone $P$:
1. Search for records in `society_members` with canonical phone $P$ and `user_id IS NULL`.
2. **Ambiguity Guard**:
   - If exactly **one** matching `society_members` record exists in a society: acquire row lock (`SELECT ... FOR UPDATE`) and update `user_id = users.id`.
   - If **multiple** candidate records match $P$ within the same society: **fail closed**. Do **not** link arbitrarily or merge person profiles. Log warning `AMBIGUOUS_MEMBER_LINK_SKIPPED` and leave `user_id = NULL`. Require administrator resolution.
3. **One User Per Society Constraint**: A user can link to at most **one** member record within a given society (`uq_society_members_user_per_society`).
4. **Interaction with OTP Transaction**: Identity linking runs as a post-verification reconciliation step outside the transient OTP table lock, ensuring that OTP expiration and session creation remain performant and uncoupled from tenant queries.

### 2.4 Referential Retention Policy
- `society_members.user_id` references `users(id) ON DELETE RESTRICT`. Historical tenancy and resident records must not be orphaned or destroyed if an authentication account lifecycle changes.

---

## 3. Ownership vs Occupancy & Concurrency Semantics

### 3.1 Independence of Ownership and Occupancy
Ownership represents declared property title in the society registry. Occupancy represents physical residency. They are completely decoupled:

| Relationship Type | Allowed `is_occupant` | Allowed `ownership_share` | Description |
| :--- | :---: | :---: | :--- |
| `PRIMARY_OWNER` | `true` or `false` | `0.01` – `100.00` or `NULL` | Primary or joint titleholder |
| `SECONDARY_OWNER` | `true` or `false` | `0.01` – `100.00` or `NULL` | Co-owner |
| `OWNER_FAMILY` | `true` or `false` | Must be `NULL` | Family member of owner |
| `PRIMARY_TENANT` | `true` or `false` | Must be `NULL` | Primary leaseholder |
| `SECONDARY_TENANT` | `true` or `false` | Must be `NULL` | Co-tenant |
| `TENANT_FAMILY` | `true` or `false` | Must be `NULL` | Family member of tenant |
| `AUTHORIZED_OCCUPANT` | Must be `true` | Must be `NULL` | Authorized long-term occupant |

### 3.2 Co-Ownership & Pessimistic Unit Row Locking
- Multiple `PRIMARY_OWNER` records are explicitly supported (e.g., joint spouses).
- To prevent race conditions where concurrent writes cause total shares to exceed 100%:
  1. Transaction starts (`tx`).
  2. Acquires row lock on unit: `SELECT id FROM society_units WHERE id = $unitID AND society_id = $societyID FOR UPDATE;`.
  3. Computes active share total: `SELECT coalesce(sum(ownership_share), 0) FROM unit_relationships WHERE unit_id = $unitID AND status = 'ACTIVE' AND relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER') AND id != $currentRelID;`.
  4. If new share $S \neq \text{NULL}$:
     - If `existing_sum + S > 100.00`: rollback transaction and return `400 INVALID_OWNERSHIP_SHARE`.
  5. Inserts or updates `unit_relationships` row and emits audit log.
  6. `COMMIT`.
- **Policy on Unspecified Ownership Shares**:
  - Ownership share remains **optional** (`ownership_share IS NULL`).
  - Co-owners with unspecified shares legitimately coexist with co-owners with specified shares. The sum of *all known/specified* percentages for a unit must never exceed 100.00%. Unspecified shares represent unrecorded/undivided title fractions.
- **Legal Disclaimer**: Registered `OWNER` relationships reflect declared registry status within the housing society and do not constitute legal title deeds.

### 3.3 Derived Vacancy
A unit is derived as **Vacant** if and only if it has **zero** active `unit_relationships` where `is_occupant = true`.

---

## 4. Membership Lifecycle & Controlled Departure Workflow

### 4.1 Membership States
1. **`ACTIVE`**: Current resident/member. Can hold active ownership and occupancy relationships.
2. **`INACTIVE`**: Temporarily on hold (e.g., administrative dispute, pending document submission). Modifies **zero** unit relationships. Reversible to `ACTIVE`.
3. **`LEFT`**: Former resident who has vacated the society. Requires the two-phase departure workflow.

### 4.2 Two-Phase Controlled Departure Workflow
Marking a member as `LEFT` must **never** silently terminate ownership or blindly cancel unrelated relationships:

1. **Phase 1: Preview (`POST .../departure/preview`)**:
   - Computes active relationships for the member.
   - Categorizes relationships:
     - **Preserved Ownerships**: `PRIMARY_OWNER` / `SECONDARY_OWNER` remain active; if resident, `is_occupant` is slated to transition to `false` (resident $\to$ absentee landlord).
     - **Targeted Terminations**: `PRIMARY_TENANT`, `SECONDARY_TENANT`, `OWNER_FAMILY`, `TENANT_FAMILY`, `AUTHORIZED_OCCUPANT` slated for termination (`status = 'ENDED'`).
   - Generates a **version hash** (`version_hash`) reflecting the state of those relationships.
2. **Phase 2: Confirmation (`POST .../departure/confirm`)**:
   - Request body includes: `confirmed_end_date`, `expected_version_hash`.
   - **Stale Preview Check**: Compares `expected_version_hash` against live state. If any relationship on that member was modified or created between preview and confirmation $\to$ reject with `409 PREVIEW_STALE`.
   - **Atomic Execution**: Inside a single transaction:
     - Updates member status to `LEFT`.
     - Preserves ownership records and updates `is_occupant = false` if resident.
     - Transitions targeted tenancy/occupancy records to `status = 'ENDED'`, `end_date = confirmed_end_date`.
     - Inserts `MEMBER_DEPARTED` and `UNIT_RELATIONSHIP_ENDED` audit events synchronously.
3. **Returning Member Semantics**:
   - If a departed member returns (`LEFT` $\to$ `ACTIVE`), their historical ended relationships are **never** reactivated automatically.
   - Any new tenancy or residency requires creating a brand new `unit_relationships` record with a fresh `start_date`.

---

## 5. Database Multi-Tenant Consistency

Cross-society relationship leakage is barred by composite foreign keys:
- `society_units` enforces `UNIQUE (id, society_id)`.
- `society_members` enforces `UNIQUE (id, society_id)`.
- `unit_relationships` enforces:
  - `(unit_id, society_id) REFERENCES society_units(id, society_id)`
  - `(member_id, society_id) REFERENCES society_members(id, society_id)`

This mathematically prevents associating a unit in Society A with a member in Society B.

---

## 6. References
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
- [[../security/06-members-and-relationships-security|06 — Members & Relationships Security Specification]]
- [[../sprints/S06-members-relationships|S06 — Members & Relationships Tracker]]
