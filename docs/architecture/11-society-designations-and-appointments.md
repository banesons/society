# 11 — Society Designations and Appointments Architecture

## 1. Architectural Overview

The Society Designations and Appointments subsystem models the managing committee, governing body, and administrative office-bearers of a housing society.

It builds on top of:
- Multi-tenant boundary: [[../decisions/015-society-lifecycle-and-tenancy|ADR 015]] and [[07-society-onboarding|07 — Society Onboarding]]
- Administrative authorization: [[../decisions/016-society-role-and-assignment-model|ADR 016]] and [[08-society-administration|08 — Society Administration]]
- Member and resident directory: [[../decisions/018-member-and-relationship-model|ADR 018]] and [[10-members-and-relationships|10 — Members & Relationships]]

```text
Platform Layer:
  platform_roles (SUPER_ADMIN, ADMIN)
        ↓
Society Governance Boundary:
  society_user_roles (SOC_ADMIN) ───[Authorizes]───→ Manages Catalogue & Appointments
        ↓
Society Community & Committee:
  society_members (Resident Profiles)
        ↓ (Appointed)
  society_committee_appointments (Terms & History)
        ↑ (Classified by)
  society_designations (Catalogue: Chairman, Secretary, Treasurer, Committee Member...)
```

---

## 2. Key Architectural Tenets

### 2.1 Role Segregation vs. Designation
- **Platform Roles** (`SUPER_ADMIN`, `ADMIN`): Operational infrastructure authority.
- **Society Roles** (`SOC_ADMIN`): System authorization within a specific society tenant.
- **Society Designations** (`CHAIRMAN`, `SECRETARY`, `TREASURER`, etc.): Civic committee titles.
- **Rule**: A designation is **never** an authorization grant. Holding the title of "Chairman" does not grant system administrator permissions unless the member is also explicitly assigned `SOC_ADMIN` in `society_user_roles`.

### 2.2 Reusable Identity without Synthetic Accounts
- Committee appointments reference `society_members` exclusively.
- Society administrators can appoint any `ACTIVE` society member (whether they are an owner, tenant, or occupant, subject to society bye-laws).
- Zero synthetic authentication accounts are generated. If the member logs in via verified phone OTP, their identity links seamlessly through existing S06 mechanisms.

### 2.3 Catalogue Flexibility & Sensible Defaults
- Every society possesses its own designation catalogue (`society_designations`).
- Pre-seeded canonical positions include:
  - `CHAIRMAN` (Singleton, allows_multiple_members: false)
  - `SECRETARY` (Singleton, allows_multiple_members: false)
  - `TREASURER` (Singleton, allows_multiple_members: false)
  - `COMMITTEE_MEMBER` (Multi-member, allows_multiple_members: true)
- Societies may configure additional positions (e.g., `VICE_CHAIRMAN`, `JOINT_TREASURER`, `GARDEN_SECRETARY`).
- Designations can be deactivated (`is_active = false`), which hides them from new appointment workflows while fully preserving historical tenures.

### 2.4 Terms, Preservation, and Concurrency
- Committee members serve terms (`start_date`, optional `expected_end_date`, and `end_date`).
- Appointments are **never** hard-deleted. Ending or revoking an appointment updates `status = 'ENDED'` or `'REVOKED'` with termination metadata.
- **Pessimistic Concurrency Protocol**:
  1. `society_designations` row is locked `FOR UPDATE` inside the appointment creation transaction.
  2. `society_members` row is locked `FOR UPDATE` to serialize concurrent requests involving the same member.
  3. Active appointment count checks and date overlap validation occur **after** acquiring row locks.
  4. The appointment row insert and corresponding `platform_audit_logs` record are committed atomically in the same transaction.
- **Appointment Date Overlap Policy**:
  - `end_date IS NULL` on `status = 'ACTIVE'` represents an ongoing, open-ended appointment term (enforced by DB check constraint `chk_appointments_active_end_date`).
  - Historical appointments with `status IN ('ENDED', 'REVOKED')` span `[start_date, end_date]`.
  - For singleton designations (`allows_multiple_members = false`), a candidate appointment period `[start_date, coalesce(expected_end_date, open_ended)]` is rejected with `ErrConflictingDateRange` (`409 CONFLICTING_DATE_RANGE`) if it overlaps any historical appointment's `[start_date, end_date]`. Subsequent non-overlapping terms (`start_date > historical.end_date`) succeed.
  - For the same member and same designation, a candidate appointment cannot overlap the member's historical terms for that designation.
- **Multiple Designations Per Member Policy**:
  - `allows_multiple_members`: Controls position-level cardinality (can multiple people hold this role simultaneously?).
  - `allows_multiple_positions_per_member`: Controls member-level exclusivity. If `false`, the appointed member must not hold any other active appointment in the society. If `true`, the member may hold other positions, provided none of those other positions disallow multiple holdings (`allows_multiple_positions_per_member = false`).
  - Concurrent violations are prevented by locking `society_members FOR UPDATE`.

### 2.5 Lifecycle Interplay & Member Departure
- **`ACTIVE` $\to$ `INACTIVE` Member Transition**: Member is placed on administrative hold; does not automatically terminate committee term.
- **`ACTIVE` $\to$ `LEFT` Member Transition**: When member departure is confirmed, any active committee appointments for that member are automatically terminated (`status = 'ENDED'`, `end_date = departure_date`), ensuring departed residents cannot linger on the active committee.

---

## 3. Data Flow

```text
Administrator (SOC_ADMIN or SUPER_ADMIN)
   │
   ├─► POST /api/v1/societies/{id}/designations
   │     Acquires society lock → validates code/name uniqueness → creates designation
   │
   ├─► POST /api/v1/societies/{id}/appointments
   │     Acquires designation lock → verifies is_active = true
   │     Verifies singleton cardinality (count active == 0)
   │     Verifies member status == ACTIVE
   │     Transactionally inserts appointment + platform_audit_logs
   │
   ├─► GET /api/v1/societies/{id}/committee/overview
   │     Fetches executive office-bearers (Chairman, Secretary, Treasurer) and active appointments
   │
   └─► POST /api/v1/societies/{id}/appointments/{appointmentId}/end
         Sets status = 'ENDED'/'REVOKED', end_date = date, notes = reason
         Transactionally updates appointment + platform_audit_logs
```

---

## 4. Related Documentation
- [[../decisions/019-society-designations-and-appointments|ADR 019: Society Designations, Committee Appointments, and Role Segregation]]
- [[../database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]
- [[../api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API Specification]]
- [[../security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security Specification]]
- [[../sprints/S07-society-roles-designations|S07 — Society Roles & Designations Sprint]]
