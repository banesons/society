# ADR 013: Idempotent Super Admin Bootstrapping Mechanism

## Status
Proposed (Sprint S02)

## Context
At launch, no administrative users exist in the system. We need a deterministic, secure, and idempotent mechanism to establish the initial `SUPER_ADMIN` identity (`+919967089101`) without:
- Exposing a public registration or escalation endpoint.
- Requiring manual raw SQL database surgery in production.
- Allowing unauthorized users or the frontend to trigger role elevation.
- Hardcoding sensitive identity data directly in compiled business logic or client bundles.

## Decision
1. **Configuration-Driven Bootstrap Target**:
   - The initial Super Admin phone number is supplied exclusively via server-side configuration: `SUPER_ADMIN_PHONE` (default: `+919967089101`).
   - Normalization uses the existing canonical E.164 utility (`NormalizePhone` from `internal/auth/phone.go`) to prevent format fragmentation.
   - It is never exposed in client bundles or public APIs.
2. **Idempotent Database Bootstrap Service**:
   - On backend service startup, the backend inspects the database:
     1. Normalizes `SUPER_ADMIN_PHONE` using `NormalizePhone`.
     2. Ensures a user record exists in `users` (pre-seeding `status = 'ACTIVE'` if not present).
     3. Checks `platform_roles` for the user:
        - If an existing role assignment is present, **never silently overwrite it**.
        - If no platform role exists, inserts `SUPER_ADMIN` with `status = 'ACTIVE'`.
     4. Transactionally appends an immutable audit record (`SUPER_ADMIN_BOOTSTRAPPED`).
3. **Mandatory OTP Authentication**:
   - Bootstrapping establishes authorization records in the database; it does NOT bypass authentication or issue tokens.
   - The bootstrapped Super Admin must still complete standard S01 phone OTP authentication to obtain an Ed25519 access token.
4. **Zero Request-Parameter Escalation & No UI Elevation**:
   - The caller's role is determined exclusively by server-side database lookup of the authenticated user ID (`claims.Subject`).
   - Request parameters or headers requesting role elevation are ignored and rejected.
   - The UI provides zero capability to create or promote another user to `SUPER_ADMIN`.

## Consequences
- The system is instantly operable after migration and restart.
- Safe to run on every startup: if the Super Admin already exists, the operation is a no-op.
- Zero risk of race conditions or public escalation endpoints.

## Related Documentation
- [[012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
