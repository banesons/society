# S06 — Members & Relationships

## Objective
Establish unit-level occupancy and ownership relationships, modeling Owners (Primary, Secondary, Family), Tenants (Primary, Secondary, Family), and Authorized Occupants with decoupled ownership/occupancy dimensions, two-phase departure workflows, pessimistic unit locking for co-ownership concurrency, composite foreign key integrity, and PII masking.

## Scope

### Backend
- Composite unique constraint on `society_units(id, society_id)`.
- `society_members` schema for tenant-scoped resident profiles with `UNIQUE (id, society_id)`, `uq_society_members_active_phone`, and `uq_society_members_user_per_society`.
- `unit_relationships` schema with composite foreign keys, CHECK constraints on dates, status, ownership share, and occupancy.
- Safe identity linking logic upon verified OTP login with ambiguity guard and row locking (`FOR UPDATE`).
- Member registration, search, detail view, updating, and two-phase preview/confirm departure APIs.
- Unit relationship creation, co-ownership share tracking with pessimistic unit locking, listing, and termination APIs.
- PII masking on directory and search responses (`+91 9967xxxx01`, `ra****@example.com`).
- Transactional audit logging for all member and relationship operations.
- Society-scoped authorization via `RequireSocietyAccess("SOC_ADMIN", ...)`.

### Frontend
- Member Directory section in Society Console with search, pagination, and status filters.
- Register Member modal with contact details and phone format validation.
- Member Profile drawer/page showing active and historical unit relationships.
- Deactivate Member confirmation modal featuring the two-phase departure preview (demonstrating that ownership is preserved while occupancy is terminated).
- Unit Occupancy drawer/view showing co-owners, tenants, and family occupants.
- Add Unit Relationship dialog supporting relationship types, ownership share, and occupancy flag.
- End relationship dialog with date selection and history preservation.

## Acceptance Criteria
- [x] Universal authentication remains intact; zero synthetic auth accounts created for pre-registered members.
- [x] Person identity correctly links to verified `users` when phone numbers match, failing closed if ambiguous.
- [x] Clear distinction between unit ownership (`PRIMARY_OWNER`, `SECONDARY_OWNER`) and physical occupancy (`is_occupant`).
- [x] Co-ownership and tenant occupancy legitimately coexist on the same unit.
- [x] Transitioning `ACTIVE` $\to$ `INACTIVE` does not modify any unit relationships.
- [x] Transitioning `ACTIVE` $\to$ `LEFT` uses two-phase preview and confirmation, preserving ownership title and ending occupancy.
- [x] Returning members (`LEFT` $\to$ `ACTIVE`) never automatically reactivate historical relationships.
- [x] Pessimistic unit row locking protects aggregate ownership shares ($\le 100\%$) across concurrent writers.
- [x] Terminated relationships preserve complete historical records (`status = 'ENDED'`, `end_date` set).
- [x] Units with zero active occupants derived as Vacant.
- [x] Database composite foreign keys guarantee zero cross-society unit/member mismatch.
- [x] Duplicate active memberships and duplicate active identical unit relationships prevented by database constraints.
- [x] All mutations write transactional audit logs to `platform_audit_logs`.
- [x] Pure integer phone standard (`91XXXXXXXXXX`) stored and transmitted without `+` or whitespace across BE & FE.
- [x] Input fields render non-editable `{+}` visual adornment prefix; full unmasked numbers visible in rosters and profile views.
- [x] Non-production `/api/v1/auth/otp/send` returns randomly generated 6-digit `otp` in response payload for dev verification.
- [x] Controlled departure reason selection (`TENANT_MOVED_OUT`, `LEASE_EXPIRED`, `OWNERSHIP_TRANSFERRED`, `MEMBER_RELOCATED`, `FAMILY_MEMBER_DEPARTED`, `MEMBERSHIP_TERMINATED`, `DECEASED`, `OTHER`) with mandatory explanation on `OTHER` and DB constraints.
- [x] Departure reason code and details persisted in `society_members` and recorded in transactional audit logs.

## Related Documentation
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]
- [[../database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
- [[../security/06-members-and-relationships-security|06 — Members & Relationships Security Specification]]
- [[README|Master Sprint Index]]

## Status
- Architecture & Design: **COMPLETE & APPROVED**
- Database Migrations: **COMPLETE & APPLIED**
- Backend Implementation: **COMPLETE**
- Frontend Implementation: **COMPLETE**
- Quality Gates & Verification: **COMPLETE (ALL PASSING)**
- Sprint State: **CLOSED & VERIFIED**

## Targeted Closure Review Summary
- **Database & Model Integrity**: Composite foreign keys `(unit_id, society_id)` and `(member_id, society_id)` strictly enforce multi-tenant isolation. Uniqueness constraints `uq_society_members_active_phone`, `uq_society_members_user_per_society`, and `uq_unit_rel_active_duplicate` prevent duplicates.
- **Identity Linking & Ambiguity Guard**: Verified OTP login triggers `SafeLinkUserMembers`, locking candidate rows and failing closed if multiple candidates exist in the same society.
- **Ownership Concurrency & Validation**: Pessimistic unit locking (`SELECT ... FOR UPDATE`) prevents race conditions on total ownership share ($\le 100\%$). Unspecified percentages co-exist safely.
- **Controlled Departure & Version Hash**: Two-phase departure computes and verifies cryptographic version token `version_hash`, returning `409 PREVIEW_STALE` on concurrent modifications. Resident owners transition to absentee owners (`is_occupant = false`), preserving title. Tenancies transition to `ENDED` with preserved dates.
- **Defect Remediation**: Frontend relationship types (`CO_OWNER`, `FAMILY_MEMBER`, `OTHER_OCCUPANT`) reconciled with canonical DB CHECK constraint and backend types (`PRIMARY_OWNER`, `SECONDARY_OWNER`, `OWNER_FAMILY`, `PRIMARY_TENANT`, `SECONDARY_TENANT`, `TENANT_FAMILY`, `AUTHORIZED_OCCUPANT`). Mandatory occupancy constraint for `AUTHORIZED_OCCUPANT` enforced in UI.
- **Quality Gates**: All Go tests, vet, formatting, build, SvelteKit check, lint, unit tests, and production build passing.

