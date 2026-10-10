# ADR 015: Society Lifecycle, Tenancy Boundary, and Multi-Tenant Isolation Strategy

## Status
Accepted (Sprint S03)

## Context
Sprint S03 introduces Housing Societies as first-class entities in the CHS platform. Societies form the core multi-tenant boundary for all future operational resources (buildings, units, members, designations, gate logs, maintenance accounts, notices).

We must decide:
1. What lifecycle states a society transitions through on the platform and what rules govern these transitions.
2. How the tenant boundary is defined and how multi-tenant isolation will be enforced between different societies.
3. Whether multi-tenant isolation should be implemented via PostgreSQL Row Level Security (RLS) or via application-level Go middleware and parameterized queries.

## Decision

### 1. Society Lifecycle State Machine
A society's platform lifecycle is governed by three explicit, deterministic states:
- `PENDING`: Initial state upon registration. The society profile and configuration are under review. It is not active on the platform. Society members and gate operations cannot function.
- `ACTIVE`: The society is operational. Platform features, member access, and operations are permitted.
- `SUSPENDED`: The society is temporarily locked (due to administrative hold, dispute, or non-payment). All society-scoped operations and member logins for this society are blocked. Platform `SUPER_ADMIN` can inspect the profile and audit logs.

#### Permitted Transitions:
- `PENDING` → `ACTIVE`: Explicit activation by `SUPER_ADMIN`.
- `ACTIVE` → `SUSPENDED`: Explicit suspension by `SUPER_ADMIN`. Requires an operational reason recorded in the immutable audit log.
- `SUSPENDED` → `ACTIVE`: Explicit reactivation by `SUPER_ADMIN`. Requires an operational reason recorded in the immutable audit log.

#### Disallowed Transitions:
- `PENDING` → `SUSPENDED`: A society must be activated before it can be suspended.
- Self-transition (`ACTIVE` → `ACTIVE`, `SUSPENDED` → `SUSPENDED`): Rejected as no-op or error.
- Society deletion: In S03, societies cannot be deleted to preserve relational integrity and immutable audit history.

### 2. Multi-Tenant Boundary Definition & Society Code Convention
- Every society has a permanent, immutable UUID primary key (`id`) and a human-readable unique identifier (`society_code`).
- **Society Code Format**: `SOC-{CITY_CODE}-{RANDOM_SUFFIX}`
  - Generated exclusively on the Go backend using a deterministic 3-character uppercase city code and a 6-character cryptographically secure uppercase alphanumeric suffix.
  - Examples:
    - `SOC-MUM-7K4P2X` (Mumbai society)
    - `SOC-PUN-9D3R8A` (Pune society)
  - **Immutability**: Once created, `society_code` is strictly immutable in application logic and enforced via PostgreSQL trigger `societies_code_immutability_trg`.
- In S04+, every society-owned resource (e.g. `buildings`, `units`, `society_members`, `maintenance_bills`, `gate_logs`) MUST contain an explicit foreign key `society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT`.
- The multi-tenant boundary is strictly rooted at the `societies` record. No resource may belong to multiple societies directly.

### 3. Application-Level Isolation vs PostgreSQL RLS
We deliberately decide **against** implementing PostgreSQL Row Level Security (RLS) for multi-tenancy in CHS, choosing instead **strict Go application-level tenant isolation**.

**Rationale**:
1. **Single Connection Role**: The Go backend connects to PostgreSQL using a shared connection pool (`pgxpool`) under a single application service role with `pgx.QueryExecModeSimpleProtocol`.
2. **Session Leak Vulnerability with Pooling**: PostgreSQL RLS with connection pooling requires setting session-level config variables (e.g., `SET LOCAL app.current_society_id = ...`) inside every checked-out connection transaction. If a connection is returned to the pool without clean reset, or if a non-transactional query runs, severe tenant cross-talk leaks can occur.
3. **Sole Authority in Go**: Go is already the established authority for authentication, session verification, rate limiting, and RBAC in CHS. Tenant verification naturally belongs in Go middleware.
4. **Tenant Context Derivation**: Go authorization middleware will extract and verify the caller's assigned society memberships from the database, store the verified `society_id` in `context.Context`, and ensure that repository queries explicitly bind `WHERE society_id = $1`.
5. **No Client-Supplied Society Privilege**: A client supplying a `society_id` in headers or URL parameters NEVER grants access; the Go middleware verifies caller entitlement before executing any query.

## Consequences
- Predictable lifecycle states managed exclusively by `SUPER_ADMIN`.
- Clean separation between platform onboarding and future tenant domain data.
- Avoids complex, error-prone database RLS configuration while maintaining complete tenant isolation through tested Go middleware.
- Foreign keys with `ON DELETE RESTRICT` prevent accidental cascading loss of society data.

## Related Documentation
- [[../architecture/07-society-onboarding|07 — Society Onboarding Architecture]]
- [[../database/03-society-schema|03 — Society Database Schema]]
- [[../api/04-society-api|04 — Society API Specification]]
- [[../security/03-society-security|03 — Society Security Specification]]
- [[012-platform-role-separation|ADR 012: Platform Role Separation]]
