# S07 — Society Roles & Designations

## Objective
Support society-level committee designations (Secretary, Treasurer, Chairman, etc.) and committee appointment management on top of the completed S00–S06 foundation without conflating civic designations with global platform roles or system authorization grants.

## Scope

### Backend
- Society-scoped designation catalogue (`society_designations`) with stable identifiers, display names, cardinality flags (`allows_multiple_members`), and active/inactive lifecycle states.
- Default designation seed catalogue (`CHAIRMAN`, `SECRETARY`, `TREASURER`, `COMMITTEE_MEMBER`).
- Committee appointment registry (`society_committee_appointments`) linking `society_members` to designations with effective terms (`start_date`, optional `expected_end_date`, actual `end_date`), statuses (`ACTIVE`, `ENDED`, `REVOKED`), and notes.
- Composite foreign key constraints enforcing multi-tenant isolation.
- Pessimistic locking and validation preventing concurrent conflicting appointments on singleton designations.
- Automatic termination of active committee appointments upon member departure (`LEFT`).
- Transactional audit logging for designation and appointment lifecycle actions.

### Frontend
- Committee & Designations section in Society Console (`/society/[id]`).
- Current Committee Roster view displaying active appointments, terms, and office-bearers.
- Designation Catalogue management (list, create custom, toggle active/inactive).
- Appointment creation modal linking active members to active designations with date validation.
- End/Revoke appointment dialog with date and reason tracking.
- Historical appointments viewer.

## Acceptance Criteria
- [x] Platform roles (`SUPER_ADMIN`, `ADMIN`), society authorization (`SOC_ADMIN`), and committee designations (`Chairman`, etc.) remain strictly separated.
- [x] Designation assignment confers zero implicit system authorization grants.
- [x] Appointment records reference existing `society_members` records; zero synthetic auth accounts or duplicate member identities created.
- [x] Society-scoped designation catalogue supports stable codes, display names, descriptions, and active/inactive states.
- [x] Sensible defaults (`CHAIRMAN`, `SECRETARY`, `TREASURER`, `COMMITTEE_MEMBER`) available for every society.
- [x] Inactive designations barred from new appointments while historical records remain readable.
- [x] Singleton designations (`allows_multiple_members = false`) reject concurrent active appointments.
- [x] Multi-member designations (`allows_multiple_members = true`) permit multiple simultaneous holders.
- [x] A member cannot hold the identical designation actively more than once concurrently.
- [x] Date validation enforces `start_date <= end_date`.
- [x] Appointment date overlaps: Historical terms on singleton designations and member-level conflicts strictly rejected (`409 CONFLICTING_DATE_RANGE`).
- [x] Multiple designations per member rule strictly enforced concurrently (`409 MULTIPLE_DESIGNATIONS_DISALLOWED`).
- [x] Singleton concurrency: pessimistic locks on designation and member serialize appointment creation; concurrent conflict integration test passes.
- [x] Only `ACTIVE` members can be appointed; `INACTIVE` and `LEFT` members rejected.
- [x] Member departure (`LEFT`) automatically ends active committee appointments.
- [x] Historical appointment records preserved on end/revoke (`status = 'ENDED'` or `'REVOKED'`, never hard-deleted).
- [x] Cross-society access rejected via composite foreign keys and tenant-checking middleware.
- [x] Suspended society restrictions respected (`403 SOCIETY_SUSPENDED`).
- [x] Unauthenticated requests return 401; unassigned platform `ADMIN` returns 403.
- [x] Transactional audit logs written to `platform_audit_logs` for all mutations.

## Related Documentation
- [[../decisions/019-society-designations-and-appointments|ADR 019: Society Designations, Committee Appointments, and Role Segregation]]
- [[../architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]
- [[../database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]
- [[../api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API Specification]]
- [[../security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security Specification]]
- [[README|Master Sprint Index]]

## Status
- Architecture & Design: **COMPLETE & APPROVED**
- Database Migrations: **COMPLETE & APPLIED**
- Backend Implementation: **COMPLETE & VERIFIED**
- Frontend Implementation: **COMPLETE & VERIFIED**
- Quality Gates & Verification: **ALL GATES PASSED**
- Targeted Closure Review: **COMPLETE & VERIFIED (4/4 AREAS PASS)**
- Sprint State: **CLOSED & VERIFIED**

