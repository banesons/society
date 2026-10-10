# ADR 012: Separation of Platform Administration from Society Domain Roles

## Status
Proposed (Sprint S02)

## Context
Cooperative Housing Society (CHS) encompasses multiple independent societies, each with its own internal governance hierarchy (e.g., Society Admins, Secretaries, Treasurers, Committee Members, Owners, Tenants, and Gatekeepers). 

A common architectural anti-pattern is combining global system privileges into the same role model or database columns as local tenant/society roles. Doing so creates security leakage, complicates multi-tenant isolation, and prevents platform operators from managing cross-society assignments cleanly.

## Decision
1. **Explicit Role Separation & Cardinality**:
   - Platform administration privileges are strictly isolated from society-level roles.
   - Platform roles exist in a dedicated table `platform_roles` referencing `users(id)`.
   - **One Platform Role per User**: A user may hold at most one platform role in S02 (`UNIQUE (user_id)`).
   - Allowed platform roles in S02 are strictly:
     - `SUPER_ADMIN`: Full platform authority. Can bootstrap system, manage platform `ADMIN` accounts, and inspect global audit logs.
     - `ADMIN`: Operational administrator. In future sprints (S03/S04), can be mapped to manage one or more societies (`1..N`).
2. **Zero Society Roles in S02**:
   - Society-level roles (`SOC_ADMIN`, `GATEKEEPER`, etc.) and unit-level roles (`PRIMARY_OWNER`, `TENANT`) are NOT implemented in S02.
   - The platform role schema is designed so that a user who is a platform `ADMIN` can later hold society relationships without conflicting constraints.
3. **Admin Deactivation & Identity Preservation**:
   - Administrative access has its own status lifecycle (`ACTIVE`, `DISABLED`) in `platform_roles`.
   - When an `ADMIN` account is disabled:
     - `platform_roles.status` is set to `'DISABLED'`.
     - All active sessions for the user in `user_sessions` are immediately revoked (`revoked_at = NOW()`).
     - The base user account in `users.status` remains `'ACTIVE'`, preserving their identity for ordinary society/resident participation.
     - Disabling an administrator does not delete their identity or historical audit actions.

## Consequences
- Clean boundary between platform operators and housing society residents.
- Principle of Least Privilege: Platform `ADMIN` accounts cannot escalate themselves to `SUPER_ADMIN` or create other admins.
- No schema drift when society management (S03+) is introduced.

## Related Documentation
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../database/02-platform-administration-schema|Platform Administration Schema]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
- [[013-super-admin-bootstrap|ADR 013: Super Admin Bootstrap]]
