# ADR 011: Strict Two-Device Concurrent Session Limit

## Status
Accepted

## Context
Housing society management applications manage sensitive physical access (gates, visitor approvals, and financial accounts). Allowing unlimited concurrent sessions increases exposure to credential sharing and stale active logins across untracked devices.

## Decision
1. **Hard Limit**: A user is permitted a maximum of **2 active concurrent devices/sessions** (`revoked_at IS NULL AND expires_at > NOW()`).
2. **Third-Device Login Policy**:
   - If a user has 2 active sessions and attempts to verify an OTP on a 3rd device, the backend **rejects** the session creation with error code `SESSION_LIMIT_REACHED`.
   - The system does NOT silently delete or evict an existing session.
   - The user must explicitly log out from an existing device or trigger a global session reset (`logout-all`) before registering a new active device.

## Consequences
- Prevents account sharing among non-residents or unauthorized third parties.
- Enforces explicit user control over active devices.
- Foundation established for future device management UI.

## Related Documentation
- [[010-session-model|ADR 010: Session Model]]
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
