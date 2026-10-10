# ADR 011: Two-Device Concurrent Session Limit with Automatic FIFO Replacement

## Status
Accepted (Amended to Automatic FIFO Replacement)

## Context
Housing society management applications manage sensitive physical access (gates, visitor approvals, and financial accounts). Allowing unlimited concurrent sessions increases exposure to credential sharing and stale active logins across untracked devices.

Initially, a 3rd login attempt was rejected with `SESSION_LIMIT_REACHED`, requiring manual device logout. In practice, residents on a new or replaced device frequently lost access or suffered high friction if an old browser session remained unrevoked.

## Decision
1. **Hard Limit**: A user is permitted a maximum of **2 active concurrent devices/sessions** (`revoked_at IS NULL AND expires_at > NOW()`).
2. **Automatic FIFO Session Replacement**:
   - When a user with 2 active sessions verifies an OTP, the backend automatically revokes the **oldest active session** and provisions the new session in a single database transaction.
   - Users are not blocked by manual logout barriers or confirmation modals.
3. **Deterministic Ordering Rule**:
   - FIFO eviction orders sessions by authoritative creation timestamp `created_at ASC`.
   - Secondary tie-breaker: `id ASC` (UUID comparison) ensures deterministic eviction if timestamps match.
4. **Concurrency & Locking Protocol**:
   - Session operations acquire a transaction-scoped row lock on the user record: `SELECT id FROM users WHERE id = $1 FOR UPDATE`.
   - Active session rows are selected with `FOR UPDATE`.
   - If `len(activeSessions) >= 2`, the oldest `len(activeSessions) - 2 + 1` sessions are revoked (`UPDATE user_sessions SET revoked_at = NOW() WHERE id = $1`).
   - The new session is inserted and the transaction committed atomically. Transaction failures roll back any revocations.
5. **Revocation Semantics**:
   - Refresh tokens for FIFO-revoked sessions are rejected immediately with `SESSION_REVOKED`.
   - Existing 15-minute access JWTs expire at their natural boundary.
   - Response payload reports `session_replaced: true` to inform the frontend.

## Consequences
- Preserves the strict 2-device account constraint while eliminating resident login lockouts.
- Concurrency-safe against race conditions during simultaneous logins.
- Immediate refresh revocation prevents stale devices from maintaining perpetual access.

## Related Documentation
- [[010-session-model|ADR 010: Session Model]]
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../api/02-authentication-api|Authentication API]]
- [[../security/01-authentication-security|Authentication Security]]
