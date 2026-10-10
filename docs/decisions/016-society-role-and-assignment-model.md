# ADR 016: Society Role and Administrator Assignment Model

## Status
Accepted (Implemented in S04)

## Context
Sprint S03 established Housing Societies as the multi-tenant boundary for the CHS platform, introducing lifecycle states (`PENDING`, `ACTIVE`, `SUSPENDED`) and immutable canonical codes (`SOC-{CITY}-{SUFFIX}`).

Sprint S04 requires establishing society-level governance:
1. `SUPER_ADMIN` must be able to assign, inspect, and revoke administrators (`SOC_ADMIN`) for specific housing societies.
2. A user may administer multiple societies (`1..N`), and a society may have multiple administrators (`1..N`).
3. Platform-wide administration (`platform_roles`) and society-specific administration must remain completely segregated per [[012-platform-role-separation|ADR 012]].
4. Immediate revocation semantics must be guaranteed without waiting for JWT token expiry.
5. Client-supplied society context must never confer unverified authority.

## Decision

### 1. Dedicated Assignment Table (`society_user_roles`)
We reject storing a single `society_id` on the `users` table or overloading `platform_roles`. Instead, we implement a normalized relation table: `society_user_roles`.

- **Cardinality**: Many-to-Many (`users` ↔ `societies`).
- **Primary Key**: Surrogate UUID (`id`).
- **Natural Uniqueness**: Enforced via partial unique index `(society_id, user_id, role) WHERE status = 'ACTIVE'`. This allows historical records of revoked assignments while strictly preventing duplicate concurrent active assignments.
- **Foreign Keys**: Both `society_id` and `user_id` specify `ON DELETE RESTRICT` to preserve referential integrity and audit provenance.

### 2. Strict Privilege and Role Separation
In accordance with [[012-platform-role-separation|ADR 012]]:
- **Platform Roles (`platform_roles`)**: `SUPER_ADMIN` and `ADMIN`. Governs platform infrastructure, onboarding, and platform audit logs.
- **Society Roles (`society_user_roles`)**: `SOC_ADMIN` (and future domain roles like `CHAIRMAN`, `SECRETARY`, `TREASURER` in S07). Governs resources strictly inside a specific society tenant.
- **Cross-Role Isolation**:
  - `SUPER_ADMIN` has platform authority to assign and revoke `SOC_ADMIN` assignments.
  - Platform `ADMIN` receives **zero** implicit society access and cannot assign `SOC_ADMIN` roles.
  - `SOC_ADMIN` receives **zero** platform administration privileges and cannot access `/api/v1/platform/*`.
  - Assigning a user as `SOC_ADMIN` does not alter their `platform_roles` record. If a platform `ADMIN` is also assigned as a `SOC_ADMIN` to a society, their permissions remain compartmentalized by endpoint prefix and middleware.

### 3. Active Society Context & Tenant Switching
For users administering multiple societies:
1. Upon login, `GET /api/v1/auth/me` returns the list of active assignments (`society_id`, `society_code`, `society_name`, `role`).
2. The client application tracks the active society selection and provides an explicit society switcher UI.
3. For society-scoped API calls, the client communicates the intended context via the `X-Society-ID: <uuid>` header or route parameter `/api/v1/societies/{society_id}/*`.
4. **Server Authority**: The client-provided `society_id` represents an assertion of user intent, **never** an authorization grant. The Go backend middleware (`RequireSocietyRole("SOC_ADMIN")`) unconditionally validates that the authenticated caller has an `ACTIVE` assignment for that exact `society_id`.

### 4. Immediate Revocation Semantics
Revoking an assignment sets `status = 'REVOKED'`, `revoked_by = <actor_id>`, and `revoked_at = NOW()`.
- The authorization middleware performs a database-backed check on every society-scoped request (mirroring the fail-closed active session lookup pattern established in S01).
- Revocation takes effect **instantaneously** on the very next HTTP request without waiting for the caller's 15-minute access JWT to expire.
- Revoking an assignment does not invalidate the user's base identity or sessions in other assigned societies.

### 5. Interaction with Society Lifecycle States
- **`PENDING` Societies**: `SUPER_ADMIN` may pre-assign `SOC_ADMIN` during onboarding review so that society administrators are configured prior to launch. However, `SOC_ADMIN` operations on society resources are barred until the society transitions to `ACTIVE`.
- **`SUSPENDED` Societies**:
  - `SUPER_ADMIN` retains the ability to list, assign, and revoke administrators on suspended societies.
  - `SOC_ADMIN` operational requests for a suspended society are blocked with HTTP 403 `SOCIETY_SUSPENDED`.
  - Society suspension does **not** revoke administrator assignments. When the society is reactivated by `SUPER_ADMIN`, administrative privileges resume automatically without requiring re-assignment.

### 6. Transactional Audit Trail
Every assignment, reassignment, and revocation executes in a database transaction that synchronously inserts an immutable record into `platform_audit_logs`:
- Actions: `SOCIETY_ADMIN_ASSIGNED`, `SOCIETY_ADMIN_REVOKED`.
- Details capture: `assignment_id`, `society_id`, `society_code`, `user_id`, `phone_mask`, `role`, and optional notes.
- If the audit insert fails, the transaction rolls back, guaranteeing zero un-audited assignment mutations.

## Consequences
- Clean many-to-many architecture supporting multi-society management.
- Multi-tenant boundary firmly rooted at `society_id` with fail-closed Go middleware verification.
- Immediate revocation guarantees with zero token-expiry delay.
- Clear separation between tenant lifecycle suspension and user assignment revocation.

## Related Documentation
- [[012-platform-role-separation|ADR 012: Separation of Platform Administration from Society Domain Roles]]
- [[015-society-lifecycle-and-tenancy|ADR 015: Society Lifecycle and Tenancy Boundary]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../database/04-society-administration-schema|04 — Society Administration Database Schema]]
- [[../api/05-society-administration-api|05 — Society Administration API Specification]]
- [[../security/04-society-administration-security|04 — Society Administration Security Specification]]
