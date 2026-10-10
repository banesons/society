# S04 — Society Administration

## Objective

Establish society-level tenant administration by mapping designated users to specific housing societies as `SOC_ADMIN` with strictly scoped administrative privileges, multi-tenant isolation, immediate revocation semantics, and auditability.

---

## 1. Architectural Foundation & Decisions

- **Role Segregation**: Society roles (`society_user_roles`) are completely decoupled from platform roles (`platform_roles`), adhering to [[../decisions/012-platform-role-separation|ADR 012]] and [[../decisions/016-society-role-and-assignment-model|ADR 016]].
- **Many-to-Many Mapping**: Normalized `society_user_roles` table supports multi-society administration (`1..N`) without embedding a `society_id` column on the `users` table.
- **Immediate Revocation**: Server-side validation on every society-scoped request ensures instant revocation without waiting for access JWT token expiry.
- **Fail-Closed Society Context**: A client-supplied `society_id` (via URL parameter or `X-Society-ID` header) is strictly an assertion of user intent. Access is granted only after server-side validation against active assignments and active society lifecycle status.

---

## 2. Scope

### Backend Scope
1. **Schema & Migration**:
   - Create `society_user_roles` table with check constraints, foreign keys (`ON DELETE RESTRICT`), and partial unique index on `(society_id, user_id, role) WHERE status = 'ACTIVE'`.
2. **Platform Management Endpoints (`SUPER_ADMIN` only)**:
   - `POST /api/v1/platform/societies/{id}/admins`: Assign user as `SOC_ADMIN`.
   - `GET /api/v1/platform/societies/{id}/admins`: List assigned administrators with pagination and masked phone numbers.
   - `DELETE /api/v1/platform/societies/{id}/admins/{assignment_id}`: Revoke administrator assignment.
3. **Session & Profile Enrichment**:
   - Update `GET /api/v1/auth/me` to include active `society_assignments` array.
4. **Society Scoped Middleware & Endpoints**:
   - Implement `RequireSocietyRole(role string)` middleware:
     - Validates target society is `ACTIVE` (rejects `SUSPENDED` with 403 `SOCIETY_SUSPENDED`, `PENDING` with 403 `SOCIETY_PENDING`).
     - Validates caller holds an active assignment in `society_user_roles` (rejects unassigned with 403 `SOCIETY_ACCESS_DENIED`).
   - Implement `GET /api/v1/societies/{id}/profile` as the foundational society-scoped endpoint.
5. **Transactional Audit Logging**:
   - Emit `SOCIETY_ADMIN_ASSIGNED` and `SOCIETY_ADMIN_REVOKED` records into `platform_audit_logs`.

### Frontend Scope
1. **Super Admin Society Admin Management (`/admin`)**:
   - Administrator roster inside society details or dedicated sub-tab.
   - Assignment modal (phone search, confirmation, role selection).
   - Revocation confirmation dialog.
2. **Society Switcher & Context Provider**:
   - Society switcher dropdown in navigation header for users with multiple society assignments.
   - Active society store tracking current working society context.
3. **Society Console Shell (`/society/[id]` or `/app`)**:
   - Baseline society administration dashboard shell displaying active society badge, code, and profile.

### Out of Scope for S04
- Resident memberships, owners, tenants, family members (deferred to S06).
- Buildings, wings, floors, and units/flats (deferred to S05).
- Society designations (Secretary, Treasurer, Chairman) (deferred to S07).
- Gate operations and visitor management (deferred to S08/S09).
- Maintenance, billing, notices, documents (deferred to S10–S13).

---

## 3. Acceptance Criteria

| ID | Category | Requirement | Verification Method |
| :--- | :--- | :--- | :--- |
| **AC-01** | Authorization | Only `SUPER_ADMIN` can assign or revoke society administrators | Automated test verifying 401/403 for unauthenticated, resident, and platform `ADMIN` callers |
| **AC-02** | Explicit Assignment | A user must have an active assignment record before accessing that society | Attempt to call society-scoped endpoint without assignment returns 403 `SOCIETY_ACCESS_DENIED` |
| **AC-03** | Role Segregation | Platform roles and society roles remain strictly segregated | Assigning `SOC_ADMIN` does not modify `platform_roles` table |
| **AC-04** | Least Privilege | Platform `ADMIN` does not automatically receive society permissions | Platform `ADMIN` request to `/societies/{id}/*` rejected with 403 unless explicitly assigned |
| **AC-05** | Multi-Society Support | Single user can administer multiple societies simultaneously | Seed user with 2 active society assignments; both accessible sequentially |
| **AC-06** | Audit Trail | Assignment creation and revocation emit transactional audit logs | Query `platform_audit_logs` for `SOCIETY_ADMIN_ASSIGNED` and `SOCIETY_ADMIN_REVOKED` |
| **AC-07** | Immediate Revocation | Revoking an assignment denies subsequent requests immediately | Revoke assignment; verify subsequent request with existing JWT returns 403 |
| **AC-08** | Client Context Forgery | Client-supplied `society_id` is validated server-side fail-closed | Call with forged `X-Society-ID` returns 403 `SOCIETY_ACCESS_DENIED` |
| **AC-09** | Platform Isolation | Society administrators cannot access platform governance | `SOC_ADMIN` calling `/platform/*` returns 403 `FORBIDDEN` |
| **AC-10** | Society Suspension | Suspending a society blocks `SOC_ADMIN` operational requests immediately | Suspend society; verify `SOC_ADMIN` call returns 403 `SOCIETY_SUSPENDED` |
| **AC-11** | Suspension Decoupling | Suspending a society does not revoke administrator assignments | Reactivate society; verify `SOC_ADMIN` access restored without reassignment |
| **AC-12** | Society Switcher UI | Multi-society admin can alternate active working society in UI | Frontend component test verifying context change upon dropdown selection |
| **AC-13** | Pre-assignment | Super Admin can assign admins to `PENDING` societies | Verify assignment succeeds; verify operational access blocked until activation |

---

## 4. Implementation Phases & Test Strategy

### Phase 1: Database Migration
- Add migration `20261014000000_create_society_user_roles.sql`.
- Verify unique constraints, indexes, and foreign keys.

### Phase 2: Go Backend Service & Platform Endpoints
- Implement `internal/societyadmin` domain repository, models, and errors.
- Implement handlers: `AssignSocietyAdminHandler`, `ListSocietyAdminsHandler`, `RevokeSocietyAdminHandler`.
- Mount routes under `/api/v1/platform/societies/{id}/admins` behind `RequireSuperAdmin`.
- Unit & integration tests for all assignment flows, conflict detection, and audit verification.

### Phase 3: Society Scoped Middleware & Routing
- Implement `RequireSocietyRole("SOC_ADMIN")` middleware.
- Mount society-scoped routes under `/api/v1/societies/{id}/*`.
- Unit tests for context derivation, multi-society switching, suspension blocking, and immediate revocation.

### Phase 4: SvelteKit Frontend
- Update `societyApi` with admin assignment methods.
- Implement Society Admin roster in `/admin` console.
- Implement Society Switcher component and society context store.
- Implement society administration console shell.
- Component and unit tests in Vitest.

---

## 5. Related Documentation

- [[../README|Master Documentation Index]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../database/04-society-administration-schema|04 — Society Administration Database Schema]]
- [[../api/05-society-administration-api|05 — Society Administration API Specification]]
- [[../security/04-society-administration-security|04 — Society Administration Security Specification]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]

---

## Status
- Architecture, Requirements & Documentation Phase: **COMPLETE**
- Implementation Phase: **COMPLETE**
- Verification: All Acceptance Criteria AC-01 through AC-13 verified passing.
