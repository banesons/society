# 01 — Authentication Security & Abuse Prevention

## Threat Vectors & Countermeasures

| Threat | Impact | Countermeasure |
| :--- | :--- | :--- |
| **SMS Toll Fraud / Bombing** | Financial cost & user harassment | - 60-second cooldown per phone<br>- Max 5 requests/hour per phone<br>- Max 10 requests/hour per IP |
| **Brute Force OTP Guessing** | Account compromise | - Cryptographic 6-digit space (1,000,000 possibilities)<br>- Hard limit of 3 attempts per OTP<br>- Instant OTP invalidation on 3rd failure |
| **Plaintext Credential Theft** | Database compromise exposure | - OTPs hashed with SHA-256 before storage<br>- Refresh tokens stored only as SHA-256 hashes |
| **Token Theft / Replay** | Session hijacking | - 15-minute access token lifespan<br>- Refresh token rotation on every use<br>- Automatic session revocation on reused refresh token |
| **Account Sharing / Credential Leak** | Unauthorized access | - Strict 2-device limit per user account with automatic FIFO session eviction<br>- Immediate refresh token invalidation for evicted sessions<br>- Global `logout-all` revocation capability |
| **PII Exposure** | Privacy violation / scrapers | - API endpoints mask phone numbers (`+91 9967xxxx01`)<br>- Full phone numbers never exposed to frontend |

---

## Session Revocation & Eviction Boundaries

1. **Immediate Revocation Across All Endpoints (ADR 010)**:
   - When an existing session is evicted via FIFO replacement, logged out (`/auth/logout`), or invalidated globally (`/auth/logout-all`), its database record has `revoked_at` set to `NOW()`.
   - Every protected API request performs an authoritative primary-key index lookup against `user_sessions` to verify that `revoked_at IS NULL` and `expires_at > NOW()`.
   - Any request carrying an access JWT for a revoked session is immediately terminated with HTTP 401 `SESSION_REVOKED`. The previous 15-minute validity window for revoked sessions is completely closed.
2. **Database Lookup Cost & Caching Rationale**:
   - Query: `SELECT user_id, expires_at, revoked_at FROM user_sessions WHERE id = $1`
   - Cost: $O(1)$ single-row B-tree primary key index scan ($<1$ms latency).
   - Unbounded/stale caches (e.g. in-memory token maps) were deliberately avoided to eliminate cache invalidation delays and multi-instance synchronization holes. Immediate revocation is guaranteed across all cluster nodes.
3. **Fail-Closed Security**:
   - If PostgreSQL session validation cannot be completed due to network or pool errors, the request is aborted with HTTP 500 `INTERNAL_ERROR`. The system never fails open.
4. **Platform Administrative Shield**:
   - Platform routes (`/api/v1/platform/*`) additionally verify active administrative status in `platform_roles` (`RequireSuperAdmin`), providing defense-in-depth against suspended administrators.

---

## Rate Limiting Rules

1. **Phone Cooldown**: 60 seconds between OTP requests for the same normalized number.
2. **Phone Rate Limit**: Maximum 5 OTP requests per normalized phone per hour.
3. **IP Rate Limit**: Maximum 10 OTP requests per source IP per hour.
4. **Client IP Resolution**: Trust client IP only from trusted reverse proxy configurations (`X-Forwarded-For` or `X-Real-IP` behind verified ingress).

---

## Token Handling & Storage

1. **Access Tokens**:
   - Short lifespan (15 minutes).
   - Stored in frontend memory state.
   - Passed in standard `Authorization: Bearer <token>` header.
2. **Refresh Tokens**:
   - Stored in secure, HTTP-only, `SameSite=Lax`, `Secure` cookies (or memory with secure storage).
   - Rotated upon every refresh call.
3. **Signing Keys**:
   - Ed25519 private key is held strictly by the backend environment (`JWT_PRIVATE_KEY`).
   - Never committed to Git, logged, or included in client payloads.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../decisions/010-session-model|ADR 010: Session Model]]
- [[../decisions/011-two-device-limit|ADR 011: Two-Device Limit]]
