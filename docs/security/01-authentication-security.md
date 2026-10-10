# 01 — Authentication Security & Abuse Prevention

## Threat Vectors & Countermeasures

| Threat | Impact | Countermeasure |
| :--- | :--- | :--- |
| **SMS Toll Fraud / Bombing** | Financial cost & user harassment | - 60-second cooldown per phone<br>- Max 5 requests/hour per phone<br>- Max 10 requests/hour per IP |
| **Brute Force OTP Guessing** | Account compromise | - Cryptographic 6-digit space (1,000,000 possibilities)<br>- Hard limit of 3 attempts per OTP<br>- Instant OTP invalidation on 3rd failure |
| **Plaintext Credential Theft** | Database compromise exposure | - OTPs hashed with SHA-256 before storage<br>- Refresh tokens stored only as SHA-256 hashes |
| **Token Theft / Replay** | Session hijacking | - 15-minute access token lifespan<br>- Refresh token rotation on every use<br>- Automatic session revocation on reused refresh token |
| **Account Sharing / Credential Leak** | Unauthorized access | - Strict 2-device limit per user account<br>- Global `logout-all` revocation capability |
| **PII Exposure** | Privacy violation / scrapers | - API endpoints mask phone numbers (`+91 9967xxxx01`)<br>- Full phone numbers never exposed to frontend |

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
