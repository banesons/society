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
- [ ] Universal authentication remains intact; zero synthetic auth accounts created for pre-registered members.
- [ ] Person identity correctly links to verified `users` when phone numbers match, failing closed if ambiguous.
- [ ] Clear distinction between unit ownership (`PRIMARY_OWNER`, `SECONDARY_OWNER`) and physical occupancy (`is_occupant`).
- [ ] Co-ownership and tenant occupancy legitimately coexist on the same unit.
- [ ] Transitioning `ACTIVE` $\to$ `INACTIVE` does not modify any unit relationships.
- [ ] Transitioning `ACTIVE` $\to$ `LEFT` uses two-phase preview and confirmation, preserving ownership title and ending occupancy.
- [ ] Returning members (`LEFT` $\to$ `ACTIVE`) never automatically reactivate historical relationships.
- [ ] Pessimistic unit row locking protects aggregate ownership shares ($\le 100\%$) across concurrent writers.
- [ ] Terminated relationships preserve complete historical records (`status = 'ENDED'`, `end_date` set).
- [ ] Units with zero active occupants derived as Vacant.
- [ ] Database composite foreign keys guarantee zero cross-society unit/member mismatch.
- [ ] Duplicate active memberships and duplicate active identical unit relationships prevented by database constraints.
- [ ] All mutations write transactional audit logs to `platform_audit_logs`.
- [ ] PII masking applied on list endpoints.
- [ ] Unit and integration tests pass for both Go backend and frontend.

## Related Documentation
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]
- [[../database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]
- [[../api/07-members-and-relationships-api|07 — Members & Relationships API Specification]]
- [[../security/06-members-and-relationships-security|06 — Members & Relationships Security Specification]]
- [[README|Master Sprint Index]]

## Status
- Architecture & Design: **APPROVED (Refinements Incorporated)**
- Database Migrations: **PENDING APPROVAL**
- Implementation: **PENDING APPROVAL**
