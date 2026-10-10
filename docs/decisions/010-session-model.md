# ADR 010: Stateful Session Model with Server-Side Active Session Hardening

## Status
Accepted (Amended to Server-Side Active Session Enforcement)

## Context
Purely stateless JWT access tokens cannot be revoked immediately if a device is compromised, lost, logged out, or evicted by FIFO policy. Relying solely on token expiration (15 minutes) creates a window during which a revoked device can continue accessing protected APIs.

Housing society operations govern physical gate access, resident visitor approvals, and financial transactions. Permitting revoked sessions to persist until access token expiry presents an unacceptable security exposure.

## Decision
1. **Asymmetric Access Tokens with Embedded Session Identity**:
   - Short-lived Ed25519 JWT (15-minute validity) passed via standard `Authorization: Bearer <token>` header.
   - JWT claims embed `session_id` alongside the user subject (`sub`).
2. **Server-Side Active Session Verification**:
   - The authentication middleware (`middleware.Authenticate`) performs two-tier validation on every protected API request:
     1. Cryptographic validation of Ed25519 signature, algorithm, and claims expiration.
     2. Authoritative database lookup against `user_sessions`: verifies that the session exists, matches the authenticated `user_id`, is not expired (`expires_at > NOW()`), and has not been revoked (`revoked_at IS NULL`).
   - If the session was revoked (by FIFO replacement, explicit logout, logout-all, or administrator deactivation), the request is rejected immediately with HTTP 401 `SESSION_REVOKED`.
3. **Fail-Closed Principle**:
   - If session verification cannot be completed due to database errors or unavailable validator, the request fails closed immediately with HTTP 500 `INTERNAL_ERROR`. Unverifiable sessions are never admitted.
4. **No Unbounded or Stale Caching**:
   - Session lookups execute directly against PostgreSQL using the primary key B-tree index on `user_sessions(id)`.
   - Execution cost is an indexed $O(1)$ single-row lookup (typically $<1$ms), avoiding stale cache windows while ensuring immediate, zero-latency revocation across the platform.
5. **Stateful Refresh Tokens**:
   - Stored server-side only as SHA-256 hashes (`refresh_token_hash`).
   - Rotated on every call to `POST /api/v1/auth/refresh`. Revoked or reused tokens trigger immediate termination.

## Consequences
- Guaranteed immediate revocation: FIFO eviction, logout, and logout-all terminate API access instantly across all endpoints.
- Predictable performance overhead: Single indexed query per authenticated request, well within acceptable latency budgets for low-latency Go HTTP services.
- Eliminates stale cache invalidation bugs and distributed cache synchronization complexity.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../database/01-authentication-schema|Authentication Schema]]
- [[../security/01-authentication-security|Authentication Security]]
- [[011-two-device-limit|ADR 011: Two-Device Limit with Automatic FIFO Replacement]]
