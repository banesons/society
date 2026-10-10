# S01 — Identity & Authentication

## Objective
Establish universal user identity, mobile-based OTP authentication, JWT issuing and validation, session tracking, and authenticated frontend state.

## Scope
### Backend
- Phone identity normalization (E.164) and PII masking.
- Go-managed OTP generation (`crypto/rand`, 6 digits, 5 mins), hashing, and verification.
- Pluggable SMS provider abstraction (`MockSMSProvider` for dev/test, DLT gateway interface for prod).
- Configured dev/test numbers (`+919967089101`, `+918422862001`) with deterministic dev OTP (`123456`).
- Just-in-Time (JIT) user auto-provisioning upon first successful verification.
- Ed25519 asymmetric access JWT issuance (15-min lifespan).
- Stateful sessions in `user_sessions` with 30-day opaque refresh tokens and rotation.
- Strict 2-device concurrent session limit per user account.
- Current device logout and `logout-all` revocation endpoints.
- Authentication middleware enforcing `Authorization: Bearer <token>` and injecting request context.
- Rate limiting: 60s resend cooldown, 5 OTP/hr per phone, 10 OTP/hr per IP.

### Frontend
- Phone number entry and OTP verification UI with resend cooldown countdown timer.
- Authenticated user state management and API client Bearer token integration.
- Protected route foundation and unauthenticated redirect handling.
- User profile banner showing masked identity and active session controls (Logout / Logout All).

## Dependencies
- [[S00-foundation]]

## Deliverables
- Working end-to-end mobile OTP login flow.
- Verified Ed25519 JWT and session rotation in Go backend.
- Authenticated state and route protection in SvelteKit frontend.
- Zero business tables or role authorization logic.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../README|CHS Documentation Index]]
- [[../architecture/05-authentication|05 — Authentication Architecture]]
- [[../api/02-authentication-api|02 — Authentication API]]
- [[../database/01-authentication-schema|01 — Authentication Database Schema]]
- [[../security/01-authentication-security|01 — Authentication Security]]
- [[../decisions/007-phone-identity|ADR 007: Phone Identity]]
- [[../decisions/008-go-otp-provider|ADR 008: Go OTP Provider]]
- [[../decisions/009-go-jwt-authority|ADR 009: Go JWT Authority]]
- [[../decisions/010-session-model|ADR 010: Session Model]]
- [[../decisions/011-two-device-limit|ADR 011: Two-Device Limit]]

## Status
Complete
