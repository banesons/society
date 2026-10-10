# 07 — Society Onboarding Architecture

## Overview

Sprint S03 establishes housing societies as first-class multi-tenant entities within the CHS platform. It delivers the platform-level architecture to register, inspect, activate, suspend, and maintain societies.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        SvelteKit Frontend                              │
│         (/admin ──► Societies Tab: Roster, Registration, Status)       │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                            Bearer JWT (S01)
                            GET  /api/v1/platform/societies
                            POST /api/v1/platform/societies
                            POST /api/v1/platform/societies/{id}/activate
                            POST /api/v1/platform/societies/{id}/suspend
                            POST /api/v1/platform/societies/{id}/reactivate
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                          Go Backend API                                │
│                                                                        │
│   [Authenticate Middleware]       ──► Validates Ed25519 JWT & Active   │
│                                       Session in PostgreSQL (S01)      │
│                                                                        │
│   [RequireSuperAdmin Middleware]  ──► Enforces active SUPER_ADMIN      │
│                                       platform role (S02)              │
│                                                                        │
│   [Society Service]               ──► Validates inputs & uniqueness,   │
│                                       enforces lifecycle transitions   │
│                                                                        │
│   [Audit Service]                 ──► Records SOCIETY_* actions into   │
│                                       platform_audit_logs transaction  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ SQL (pgxpool Simple Protocol)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                       Supabase PostgreSQL                              │
│                                                                        │
│   users (S01)       platform_roles (S02)     platform_audit_logs (S02) │
│                               ▲                                        │
│                               │ created_by                             │
│                       societies (S03)                                  │
│                       • id (UUID, PK)                                  │
│                       • society_code (UNIQUE)                          │
│                       • name, address, city, state, pincode            │
│                       • status (PENDING, ACTIVE, SUSPENDED)            │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 1. Domain Separation & Boundaries

CHS strictly separates concerns into four hierarchical tiers:

1. **Platform Authentication (S01)**: Mobile phone OTP, Ed25519 access JWTs, server-side sessions in `user_sessions`, 2-device concurrency, fail-closed validation.
2. **Platform Governance (S02 & S03)**:
   - Platform roles (`SUPER_ADMIN`, `ADMIN`) in `platform_roles`.
   - Global multi-tenant onboarding in `societies`.
   - `SUPER_ADMIN` holds platform-wide authority to register and manage all societies.
   - Platform `ADMIN` has no default society-creation or society-management privileges in S03.
3. **Society Administration (S04+)**:
   - Explicit assignment of `ADMIN` or `SOC_ADMIN` to specific societies (`1..N` mapping).
   - Society-scoped permissions.
4. **Society Domain & Unit Structure (S05–S07)**:
   - Buildings, wings, floors, units/flats.
   - Resident relationships (Primary Owner, Secondary Owner, Tenant, Family).
   - Designations (Secretary, Treasurer, Chairman).

---

## 2. Society Code Convention & Generation Architecture

Every society receives a unique, immutable, human-readable society code generated exclusively by the Go backend during registration:

```text
SOC-{CITY_CODE}-{RANDOM_SUFFIX}
```

- **Prefix (`SOC`)**: Society identifier prefix.
- **City Code (`{CITY_CODE}`)**: 3-character deterministic uppercase code derived from normalized city name (e.g. `MUM` for Mumbai, `PUN` for Pune, `BLR` for Bangalore, `DEL` for Delhi, with a deterministic alphanumeric fallback for unmapped cities).
- **Random Suffix (`{RANDOM_SUFFIX}`)**: Exactly 6 uppercase alphanumeric characters (`23456789ABCDEFGHJKLMNPQRSTUVWXYZ`) generated using `crypto/rand`.

### Documented Examples:
- `SOC-MUM-7K4P2X` (Mumbai society)
- `SOC-PUN-9D3R8A` (Pune society)

### Architecture Rules:
1. **Server Authority**: The frontend never generates, selects, or overrides society codes. Requests attempting to supply `society_code` are ignored or rejected.
2. **PostgreSQL Uniqueness**: Uniqueness is enforced at the database level with a unique index. The backend incorporates bounded collision retry logic (up to 5 attempts).
3. **Immutability**: Once created, `society_code` cannot be altered by profile updates or city changes. Enforced in application code and via database trigger `societies_code_immutability_trg`.
4. **Internal Primary Key**: The UUID primary key (`id`) remains the authoritative identifier for foreign keys, internal relationships, and database joins.

---

## 3. Society Lifecycle State Machine

A society's platform lifecycle status controls whether tenant operations and member logins can execute for that society.

```text
           ┌──────────────────────┐
           │     Registration     │
           └──────────┬───────────┘
                      │
                      ▼
               ┌─────────────┐
               │   PENDING   │ (Profile registered; onboarding review)
               └──────┬──────┘
                      │
                      │ POST /activate (SUPER_ADMIN)
                      ▼
     ┌─────────► ┌────────────┐
     │           │   ACTIVE   │ (Fully operational; tenant access enabled)
     │           └──────┬─────┘
     │                  │
     │ POST /activate   │ POST /suspend (SUPER_ADMIN, Reason required)
     │ (SUPER_ADMIN)    ▼
     │           ┌────────────┐
     └───────────┤ SUSPENDED  │ (Administrative hold; tenant operations locked)
                 └────────────┘
```

### Transition Invariants
- **Registration**: Always initializes society with `status = 'PENDING'`.
- **Activation**:
  - `PENDING` → `ACTIVE`: Society profile approved; platform services enabled.
  - `SUSPENDED` → `ACTIVE`: Reactivates an administrative hold.
- **Suspension**:
  - `ACTIVE` → `SUSPENDED`: Temporarily locks society operations. Mandatory machine-readable `reason_code` must be selected from `society_suspension_reasons` (with mandatory `reason_details` if `OTHER_REVIEW_REQUIRED`) and logged with snapshot in `platform_audit_logs`.
- **Invalid Transitions**:
  - `PENDING` → `SUSPENDED`: Denied with `INVALID_STATUS_TRANSITION`.
  - Self-transitions (`ACTIVE` → `ACTIVE`, `SUSPENDED` → `SUSPENDED`): Denied with `ALREADY_IN_STATUS`.

---

## 3. Multi-Tenant Isolation Architecture

### Tenant Root Boundary
The `societies` table is the sole tenant boundary root. Every future society-owned resource (buildings, flats, members, gate entries, maintenance charges) will reference `societies(id)` via:
```sql
society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT
```

### Application-Level Scoping in Go
As documented in [[../decisions/015-society-lifecycle-and-tenancy|ADR 015]]:
1. CHS uses **Go application-level tenant isolation**, not PostgreSQL RLS.
2. In S03, endpoints are strictly platform-level (`/api/v1/platform/societies/*`) and require `SUPER_ADMIN`.
3. In S04+, society-level endpoints (`/api/v1/societies/{society_id}/*`) will verify that the authenticated caller holds an active relationship to `{society_id}` before executing queries.
4. Repositories will explicitly bind `WHERE society_id = $1` on all scoped queries, preventing cross-tenant leakage.
5. Client-supplied IDs in headers or request bodies never bypass authorization.

---

## 4. Frontend Integration Architecture

The S03 frontend cleanly extends the `/admin` Super Admin console without modifying the existing navigation framework or authentication store.

```text
/admin (Super Admin Console)
├── Tab 1: Administrators (S02 Roster & Creation)
├── Tab 2: Societies (S03 Roster, Registration, Filters, Status Controls)
├── Tab 3: Audit Trail (S02/S03 Append-Only Ledger)
└── Tab 4: Account & Sessions (S01/S02 Identity & Concurrency)
```

### Components Structure
- `SocietyRoster.svelte`: Paginated table of societies showing Code, Name, Location (City, State), Status Badge, Created Timestamp, and Action buttons (View Details, Activate, Suspend).
- `RegisterSocietyDialog.svelte`: Modal form to register a new society with input validation (Indian PIN code, unique society code, address fields).
- `SocietyDetailsDialog.svelte`: Inspects comprehensive society profile, registration number, contact info, timestamps, and creator ID.
- `SocietyStatusDialog.svelte`: Confirmation dialog for activation/suspension with mandatory reason capture.

---

## 5. Audit Logging Architecture

All society lifecycle actions and mutations are recorded in `platform_audit_logs` transactionally.

| Action | Trigger | Details Captured |
| :--- | :--- | :--- |
| `SOCIETY_CREATED` | Society registered | `society_id`, `society_code`, `name`, `city`, `state`, `pincode` |
| `SOCIETY_ACTIVATED` | Society activated from PENDING or SUSPENDED | `society_id`, `society_code`, `previous_status`, `new_status: ACTIVE`, `reason` |
| `SOCIETY_SUSPENDED` | Society suspended from ACTIVE | `society_id`, `society_code`, `previous_status: ACTIVE`, `new_status: SUSPENDED`, `reason_code`, `reason_label_snapshot`, `suspension_origin`, `reason_details` |
| `SOCIETY_PROFILE_UPDATED` | Allowed profile fields edited | `society_id`, `society_code`, updated field list |

If writing the audit log record fails, the entire transaction rolls back, ensuring that no un-audited society mutation can persist.

---

## 6. Suspension Reason Catalogue & Automation Boundary

Suspension reasons are decoupled from free-text strings into a managed database table `society_suspension_reasons`.

### Reason Structure
- `id`: Compact database surrogate identifier (`SMALLINT GENERATED BY DEFAULT AS IDENTITY`).
- `code`: Stable, immutable uppercase snake_case business identifier (e.g. `PAYMENT_OVERDUE`, `LEGAL_DISPUTE`).
- `label`: UI display title.
- `category`: Functional domain classification (`PAYMENT`, `LEGAL`, `COMPLIANCE`, `SECURITY`, `ADMINISTRATIVE`).
- `allows_manual`: Boolean flag indicating manual accessibility in Super Admin UI.
- `allows_automated`: Boolean flag designating readiness for backend automated workflows.
- `is_active`: Operational availability toggle.

API clients submit the compact numeric `reason_id` (or stable `reason_code`), while business rules and audit logs retain both `reason_id` and authoritative `reason_code`. Future automation workflows will map canonical codes (e.g. `PAYMENT_OVERDUE`) to catalogue records and trigger suspensions with `suspension_origin = "AUTOMATED"`.

### Manual vs Automated Origin Policy
- Manual suspensions initiated by administrators record `suspension_origin = "MANUAL"` and authenticated Super Admin `actor_id`.
- Clients are strictly barred from supplying or forging `suspension_origin` or `actor_id`.
- Future automated background processors (e.g. overdue billing checks) will call domain suspension services directly within the backend using `suspension_origin = "AUTOMATED"`.

---

## Related Documentation
- [[05-authentication|05 — Authentication Architecture]]
- [[06-platform-administration|06 — Platform Administration Architecture]]
- [[../database/03-society-schema|03 — Society Database Schema]]
- [[../api/04-society-api|04 — Society API Specification]]
- [[../security/03-society-security|03 — Society Security Specification]]
- [[../decisions/015-society-lifecycle-and-tenancy|ADR 015: Society Lifecycle and Tenancy Boundary]]
