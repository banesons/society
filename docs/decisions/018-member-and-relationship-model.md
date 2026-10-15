# ADR 018: Society Member, Unit Relationship, and Identity Linking Model

## Status
Accepted (S06 Final Design Gate)

## Context
Sprint S05 established physical society structure (buildings, wings, units, and gates). Sprint S06 models the people associated with a housing society and their relationships to physical units.

Key architectural and business requirements:
1. **Ownership vs. Occupancy Separation**: Ownership (declared title interest in the society registry) and physical occupancy (residing in the unit) are independent dimensions. An owner who moves away remains the owner. A tenant occupies the unit without holding title.
2. **Co-Ownership Without Artificial Limits**: Housing units may have multiple joint owners (e.g., spouses, family heirs). Multiple co-owners (joint `PRIMARY_OWNER` or `SECONDARY_OWNER`) are permitted.
3. **Identity vs. Society Residency**: Universal authentication (`users`, phone OTP verified) must remain clean. Pre-registration of residents by society admins must not create synthetic authentication accounts.
4. **Controlled Member Departure Workflow**:
   - Transitioning `ACTIVE` $\to$ `INACTIVE` is a pure administrative hold and modifies **zero** unit relationships.
   - Transitioning `ACTIVE` $\to$ `LEFT` requires a two-phase preview and confirmation workflow. Ownership title is **never** silently cancelled; resident owners become absentee owners (`is_occupant = false`).
   - Stale previews are rejected with concurrency tokens.
   - Returning members never automatically reactivate historical relationships.
5. **Concurrency-Safe Identity Linking**:
   - Phone normalization to canonical E.164.
   - Duplicate active phone numbers blocked by partial unique index `uq_society_members_active_phone`.
   - Ambiguous candidate matches in the same society fail closed (no arbitrary linking).
   - Linking executes under row lock (`FOR UPDATE`) and prevents a user from linking to multiple members in the same society.
6. **Concurrency-Safe Ownership-Share Enforcement**:
   - Concurrent updates to unit shares are serialized using pessimistic unit row locking (`SELECT ... FOR UPDATE`).
   - Aggregate sum of specified percentages cannot exceed 100.00%.
   - Unspecified percentages are permitted as `NULL`.
7. **Database-Enforced Multi-Tenant Integrity**:
   - Cross-society relationship leakage is barred by composite foreign keys:
     - `(unit_id, society_id) REFERENCES society_units(id, society_id)`
     - `(member_id, society_id) REFERENCES society_members(id, society_id)`
8. **Declared vs. Legal Ownership**: Society registry ownership is declared for administrative purposes and does not constitute a legal property deed or title guarantee.

---

## Decision

### 1. Dedicated Member Entity (`society_members`)
- **Tenant Scoping**: Scoped directly to `society_id`.
- **User Link (`user_id`)**: Nullable foreign key referencing `users(id) ON DELETE RESTRICT`.
  - Admin pre-registration creates a `society_members` row with `user_id = NULL`. Zero synthetic auth accounts are created.
  - When an individual verifies via phone OTP, the system links `society_members.user_id = users.id`.
  - **Ambiguity Guard**: If multiple candidate member records match the same phone within a society, automatic linking fails closed; administrator review is required.
  - **One Member Per User Per Society**: Enforced by partial unique index `(society_id, user_id) WHERE user_id IS NOT NULL`.
- **Active Phone Uniqueness**: Enforced by `(society_id, phone) WHERE status = 'ACTIVE'`.
- **Retention**: `ON DELETE RESTRICT` on `users(id)` preserves historical resident identity references.

### 2. Explicit Unit Relationships (`unit_relationships`)
- **Relationship Types**:
  - `PRIMARY_OWNER`: Registered primary/joint owner.
  - `SECONDARY_OWNER`: Co-owner.
  - `OWNER_FAMILY`: Family member of an owner.
  - `PRIMARY_TENANT`: Primary leaseholder.
  - `SECONDARY_TENANT`: Co-tenant.
  - `TENANT_FAMILY`: Family member of a tenant.
  - `AUTHORIZED_OCCUPANT`: Authorized long-term resident.
- **Valid Combinations & Constraints**:
  - `is_occupant`:
    - `PRIMARY_OWNER`, `SECONDARY_OWNER`, `OWNER_FAMILY`, `PRIMARY_TENANT`, `SECONDARY_TENANT`, `TENANT_FAMILY` may have `is_occupant = true` OR `false`.
    - `AUTHORIZED_OCCUPANT` must have `is_occupant = true`.
  - `ownership_share`:
    - Allowed ONLY for `PRIMARY_OWNER` and `SECONDARY_OWNER` (`NUMERIC(5,2)` between `0.01` and `100.00`, or `NULL` if unrecorded).
    - Must be `NULL` for all non-owner relationship types (`OWNER_FAMILY`, tenants, occupants).
    - Enforced by database `CHECK` constraint.
  - Multiple `PRIMARY_OWNER` and `SECONDARY_OWNER` records can coexist.
- **Status & Dates Consistency**:
  - `CHECK (start_date <= COALESCE(end_date, start_date))`.
  - `CHECK ((status = 'ACTIVE' AND end_date IS NULL) OR (status = 'ENDED' AND end_date IS NOT NULL))`.

### 3. Concurrency Protection & Serialized Ownership Shares
- To protect aggregate shares across concurrent writers:
  1. Transaction acquires row lock: `SELECT id FROM society_units WHERE id = $unitID FOR UPDATE`.
  2. Repository recalculates `sum(ownership_share)` for all existing active owners.
  3. Rejects with `400 INVALID_OWNERSHIP_SHARE` if `new_share + existing_sum > 100.00%`.
  4. Inserts/updates `unit_relationships` and audit log within the locked transaction.
- Ownership percentages can remain unspecified (`NULL`). Unspecified shares represent unrecorded/undivided title fractions and coexist alongside specified percentages so long as the sum of known percentages $\le 100.00\%$.

### 4. Controlled Member Departure Workflow
- **`ACTIVE` $\to$ `INACTIVE`**: Modifies **zero** unit relationships.
- **`ACTIVE` $\to$ `LEFT`**:
  - Requires explicit two-phase execution:
    1. `POST .../departure/preview` returns manifest with version hash of active relationships.
    2. `POST .../departure/confirm` validates version hash. Rejects with `409 PREVIEW_STALE` if relationships changed between preview and confirmation.
  - Preserves ownership records (`PRIMARY_OWNER`, `SECONDARY_OWNER`). If resident, flips `is_occupant` to `false`.
  - Transitions confirmed tenancies and family occupancy records to `status = 'ENDED'`, `end_date = confirmed_date`.
  - **Structured Departure Reasons**: Requires a canonical `reason_code` (`TENANT_MOVED_OUT`, `LEASE_EXPIRED`, `OWNERSHIP_TRANSFERRED`, `MEMBER_RELOCATED`, `FAMILY_MEMBER_DEPARTED`, `MEMBERSHIP_TERMINATED`, `DECEASED`, `OTHER`). When `OTHER` is selected, `reason_details` is mandatory and limited to 500 characters. Reason codes and details are durably persisted in `society_members` and recorded in transactional audit logs.
  - Executes atomically with audit logs in a single transaction.
- **Returning Member Semantics**: If a departed member returns (`LEFT` $\to$ `ACTIVE`), historical ended relationships are **never** reactivated automatically. New tenancy or occupancy requires an explicit new relationship with a fresh `start_date`.

### 5. Database Engine Multi-Tenant Consistency
To make cross-society relationship leakage physically impossible:
- `society_units` includes `UNIQUE (id, society_id)`.
- `society_members` includes `UNIQUE (id, society_id)`.
- `unit_relationships` defines composite foreign keys:
  ```sql
  FOREIGN KEY (unit_id, society_id) REFERENCES society_units(id, society_id) ON DELETE RESTRICT
  FOREIGN KEY (member_id, society_id) REFERENCES society_members(id, society_id) ON DELETE RESTRICT
  ```

---

## Consequences

### Positive
- Strict separation of ownership and residency preserves property rights through moves.
- Pessimistic unit locking eliminates race conditions on aggregate ownership shares.
- Two-phase departure with version hashing prevents stale or accidental relationship termination.
- Database composite foreign keys eliminate cross-society data corruption.
- Ambiguity guard prevents incorrect account merges on phone collisions.
- No synthetic authentication credentials poison the security model.

### Trade-offs
- Pessimistic unit locking requires serializing writes to the same physical unit.
- Two-phase departure workflow adds a preview roundtrip before member departure.

---

## References
- [[012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
- [[017-society-structure-and-formation|ADR 017: Society Structure, Formation Workflow, and Unit Generation Model]]
- [[../architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]
- [[../database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
