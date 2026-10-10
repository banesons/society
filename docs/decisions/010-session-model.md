# ADR 010: Stateful Session Model with Refresh Token Rotation

## Status
Accepted

## Context
Stateless access tokens cannot be revoked immediately if a device is compromised, lost, or logged out. A pure stateless JWT model prevents immediate revocation, whereas querying the database on every micro-request introduces latency bottlenecks.

## Decision
1. **Hybrid Authentication Model**:
   - Short-lived stateless **Access Token** (JWT, 15-minute validity) passed via standard `Authorization: Bearer <token>` header.
   - Long-lived stateful **Refresh Token** (cryptographically random 32-byte opaque string, 30-day validity) stored server-side in `user_sessions`.
2. **Server-Side Session Storage**:
   - Refresh tokens are NEVER stored in plaintext in PostgreSQL; only a SHA-256 hash (`refresh_token_hash`) is retained.
   - The `user_sessions` table records:
     - `id UUID PRIMARY KEY`
     - `user_id UUID REFERENCES users(id)`
     - `refresh_token_hash TEXT NOT NULL`
     - `user_agent TEXT`
     - `client_ip TEXT`
     - `expires_at TIMESTAMPTZ NOT NULL`
     - `revoked_at TIMESTAMPTZ NULL`
     - `created_at TIMESTAMPTZ`
     - `last_active_at TIMESTAMPTZ`
3. **Refresh Token Rotation & Reuse Detection**:
   - Every call to `POST /api/v1/auth/refresh` invalidates the old refresh token and issues a new pair of access and refresh tokens.
   - If an already-revoked refresh token is presented, the system detects a potential token replay attack, immediately revokes the entire session, and returns `REFRESH_TOKEN_REUSED`.
4. **Logout Semantics**:
   - `POST /api/v1/auth/logout`: Revokes only the active session (`revoked_at = NOW()`).
   - `POST /api/v1/auth/logout-all`: Revokes all active sessions for that user.

## Consequences
- Immediate revocation capability without hitting the database for normal authenticated API calls.
- High resilience against token theft through automatic rotation and replay invalidation.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../database/01-authentication-schema|Authentication Schema]]
- [[../security/01-authentication-security|Authentication Security]]
- [[011-two-device-limit|ADR 011: Two-Device Limit]]
