# ADR 023: Visitor Lifecycle and Gate Operations

## Status
Accepted

## Context
Sprint S08 established the Gate Foundation, introducing physical gate entities, operational directionality (`BIDIRECTIONAL`, `INGRESS_ONLY`, `EGRESS_ONLY`), operational status (`ACTIVE`, `INACTIVE`, `MAINTENANCE`, `EMERGENCY_OPEN`), gatekeeper staff assignments (`society_user_roles` with `role = 'GATEKEEPER'`), shift templates, and active duty sessions (`society_gate_sessions`).

However, S08 deliberately omitted visitor operations. A residential Cooperative Housing Society (CHS) requires a robust, secure, real-time visitor-management system to govern perimeter access. Physical entry of non-residents (guests, deliveries, maintenance technicians, service personnel, cabs) introduces significant physical and security risks.

Key operational constraints:
1. Visitors are not CHS platform users: they must not be registered in the global `users` table, nor forced to install an application or undergo OTP onboarding.
2. The Go backend is the sole authentication and authorization authority; Supabase Auth is bypassed.
3. Every operation must strictly preserve tenant isolation across societies.
4. Gate operations must integrate with active gatekeeper duty sessions.
5. Ingress and egress must be strictly distinguished and recorded with server-authoritative timestamps.
6. Resident approvals must prevent unauthorized check-in while supporting fast, verifiable entry for both expected visitors and walk-ins.
7. Visitor passes must be time-bound, cryptographically verifiable, and replay-resistant.

## Decisions

### 1. Dedicated Normalized Visitor Schema
We establish four society-scoped entities:
- `society_visits`: The primary visit lifecycle entity capturing destination unit, intended resident, visitor identity (name, phone), vehicle number, visit type (`EXPECTED`, `WALK_IN`), and current status (`PENDING_APPROVAL`, `APPROVED`, `REJECTED`, `CHECKED_IN`, `CHECKED_OUT`, `CANCELLED`, `EXPIRED`).
- `society_visitor_passes`: Cryptographically verifiable bearer tokens (`chs_pass_<random_hex>`) stored exclusively as SHA-256 hashes (`token_hash`) alongside short alphanumeric pass codes for manual gate lookup.
- `society_visit_approvals`: Append-only record of approval decisions (`APPROVED` / `REJECTED`) tied to authorized resident users or society admins.
- `society_gate_events`: Append-only perimeter movement ledger recording every physical `ENTRY` and `EXIT` event, linked to the active duty session and gatekeeper.

### 2. Strict Visit State Machine
The visit lifecycle transitions are strictly governed:
- `EXPECTED`: Created in `APPROVED` (if created by unit resident/admin) or `PENDING_APPROVAL`.
- `WALK_IN`: Created in `PENDING_APPROVAL` (resident approval required) or `APPROVED` (instant approval policy/delivery bypass).
- `PENDING_APPROVAL` -> `APPROVED` (by authorized resident or SOC_ADMIN).
- `PENDING_APPROVAL` -> `REJECTED` (by resident or SOC_ADMIN).
- `APPROVED` -> `CHECKED_IN` (recorded at eligible gate by stationed gatekeeper with active duty session).
- `CHECKED_IN` -> `CHECKED_OUT` (recorded at eligible exit gate).
- `PENDING_APPROVAL` / `APPROVED` -> `CANCELLED` (by creator, resident, or SOC_ADMIN before entry).
- `PENDING_APPROVAL` / `APPROVED` -> `EXPIRED` (when arrival window lapses).

Terminal states: `REJECTED`, `CHECKED_OUT`, `CANCELLED`, `EXPIRED`.
Check-in is forbidden for any visit not in `APPROVED` state.
Check-out is forbidden for any visit not in `CHECKED_IN` state.

### 3. Concurrency Control and Duplicate Prevention
Concurrent check-in attempts for the same visit are serialized using `SELECT ... FOR UPDATE` database row locks. The transition updates the status atomically from `APPROVED` to `CHECKED_IN`. Subsequent requests encounter status != `APPROVED` and fail immediately with `409 Conflict`.
A visitor cannot be checked in twice, checked out twice, or checked out before check-in.

### 4. Gate Directionality and Operational Status
Entry is strictly blocked if:
- Target gate `status != 'ACTIVE'` or `operational_status != 'ACTIVE'` (or `EMERGENCY_OPEN`).
- Gate `directionality == 'EGRESS_ONLY'`.
- Gatekeeper does not hold an active duty session (`society_gate_sessions`) at the target gate (or society policy requires duty session).
Exit is similarly blocked if gate `directionality == 'INGRESS_ONLY'`.

### 5. Cryptographic Pass Model
Pass tokens are generated using 32 bytes of cryptographic randomness (`crypto/rand`). Only the SHA-256 hash is persisted in `society_visitor_passes.token_hash`. Verification matches the hash of the presented bearer token against active passes. Expired, revoked, or consumed passes are rejected immediately.

### 6. Transactional Audit Integration
All operational mutations (visit registration, approval decision, pass issuance, check-in, checkout, cancellation) insert into `platform_audit_logs` within the same database transaction. If audit logging fails, the business transaction rolls back entirely (fail-closed).

### 7. Resident-Controlled Approval Policy (Default: OFF)
- Approval requirement is a persistent, unit-level preference stored on `society_units.requires_visitor_approval`.
- **Default Value**: `FALSE` (OFF). Direct entry is permitted subject to gate check-in and valid visit registration.
- **Authoritative Ownership**: Only active occupants or owners of the destination unit (`unit_relationships`) can view and update their unit's approval policy.
- **Gatekeeper & Admin Restrictions**: Gatekeepers recording arrivals cannot control, toggle, or override the unit's approval policy. Society administrators cannot impersonate residents or override the resident's policy preference.
- **Backend Resolution**: On walk-in registration, the backend server resolves `requires_visitor_approval` directly from `society_units` for the destination unit. Any client-supplied override is discarded.
- **Visit State Mapping**: If `requires_visitor_approval = true`, walk-in visits initialize in `PENDING_APPROVAL`. If `false`, visits initialize in `APPROVED`.

### 8. Strict Gatekeeper Duty Session Enforcement
Operational gate movements (`check-in` and `check-out`) performed by users holding the `GATEKEEPER` role require an active duty session (`society_gate_sessions`) at the designated gate. Requests without an active session at that gate fail with `403 DUTY_SESSION_REQUIRED`. Platform administrators retain audit-logged override capabilities.

## Consequences
- **Positive**: Strict data integrity, deterministic lifecycle, replay prevention, comprehensive audit trails, zero duplicate entries under concurrency.
- **Positive**: Complete tenant isolation via composite foreign keys `(entity_id, society_id)`.
- **Positive**: No unnecessary PII collection or global user pollution.
- **Positive**: Residents retain total privacy sovereignty over their flat's visitor screening policy without gatekeeper tampering.
- **Negative**: Gatekeepers must be checked into an active duty session before operating the live gate terminal.

## Related Documents
- [[../architecture/12-gate-foundation-and-staffing]]
- [[../architecture/14-gate-visitor-operations]]
- [[../database/09-gate-visitor-operations-schema]]
- [[../api/10-gate-visitor-operations-api]]
- [[../security/09-gate-visitor-operations-security]]
- [[../sprints/S09-gate-operations]]
