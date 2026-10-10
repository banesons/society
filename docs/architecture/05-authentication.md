# 05 — Authentication Architecture

## Architectural Overview

CHS implements a self-sovereign authentication engine within the Go backend, utilizing Supabase Cloud strictly as its relational PostgreSQL persistence layer.

```text
┌─────────────────────────────────────────────────────────────┐
│                    SvelteKit Frontend                       │
│    (Phone Entry ──► OTP Verification ──► Auth State)        │
└──────────────┬───────────────────────────────▲──────────────┘
               │                               │
       POST /auth/otp/send                     │ Access JWT (15m)
       POST /auth/otp/verify                   │ Refresh Token (30d)
               │                               │
               ▼                               │
┌──────────────────────────────────────────────┴──────────────┐
│                      Go Backend                             │
│                                                             │
│   [Phone Normalizer] ──► E.164 (+91XXXXXXXXXX)              │
│                                                             │
│   [Rate Limiter] ──────► 60s cooldown, 5/hr phone, 10/hr IP │
│                                                             │
│   [OTP Engine] ────────► crypto/rand 6-digit, 5m expiry     │
│                                                             │
│   [SMS Dispatcher] ────► SMSProvider (Mock / DLT Gateway)   │
│                                                             │
│   [User Lifecycle] ────► Just-In-Time User Provisioning     │
│                                                             │
│   [Session Manager] ───► Max 2 Devices, Rotation, Revoke    │
│                                                             │
│   [Token Issuer] ──────► Ed25519 Signed Access JWTs         │
└──────────────┬──────────────────────────────────────────────┘
               │ SQL Queries (pgxpool)
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Supabase PostgreSQL Cloud                   │
│         users  │  auth_otps  │  user_sessions               │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Phone Normalizer (`internal/auth/phone.go`)
- Strips non-digit formatting characters.
- Parses and converts 10-digit Indian numbers into standard E.164 (`+91XXXXXXXXXX`).
- Validates against `^\+91[6-9]\d{9}$`. Rejects malformed numbers before OTP generation.

### 2. OTP Engine (`internal/auth/otp.go`)
- Generates 6-digit numeric codes using `crypto/rand`.
- Hashes codes with SHA-256 and records them in `auth_otps`.
- Validates attempts (max 3), expiry (5 minutes), and single-use consumption.
- Supports configured test numbers (`DEV_TEST_PHONES`) with a fixed test code (`123456`) in non-production environments.

### 3. SMS Provider Abstraction (`internal/sms/provider.go`)
- Interface decoupling OTP logic from delivery vendors:
  ```go
  type Provider interface {
      SendOTP(ctx context.Context, phone string, otp string) error
  }
  ```
- `MockProvider`: Used for development and automated tests. Logs OTPs to structured logs without incurring SMS costs.
- `ProductionProvider`: Implements TRAI DLT headers, PE IDs, and template-based SMS dispatch.

### 4. Just-In-Time (JIT) User Lifecycle
- Frictionless onboarding: no separate registration flow.
- If the normalized phone does not exist in `users` upon successful OTP verification, a new record is provisioned with `status = 'ACTIVE'`.
- If the user exists, their account is loaded and their login timestamp updated.

### 5. Session & Device Management (`internal/auth/session.go`)
- Enforces a strict limit of **maximum 2 concurrent active sessions** per user.
- A 3rd login attempt returns `SESSION_LIMIT_REACHED` requiring the user to log out an existing device.
- Refresh tokens are rotated on each `/auth/refresh` invocation. Reused or revoked tokens trigger immediate session termination.

### 6. Token Authority (`internal/auth/token.go`)
- Asymmetric signing with **Ed25519**.
- Short-lived Access Tokens (15 minutes).
- Opaque random Refresh Tokens (30 days) stored as SHA-256 hashes in `user_sessions`.

## Authentication Middleware (`internal/middleware/auth.go`)
- Extracts `Authorization: Bearer <token>` from HTTP requests.
- Verifies Ed25519 signature, algorithm, and claims expiration.
- Injects `user_id` and `session_id` into `r.Context()`.
- Unauthenticated or expired requests receive standard 401 error envelopes.

## Related Documentation
- [[../decisions/007-phone-identity|ADR 007: Phone Identity]]
- [[../decisions/008-go-otp-provider|ADR 008: Go OTP Provider]]
- [[../decisions/009-go-jwt-authority|ADR 009: Go JWT Authority]]
- [[../decisions/010-session-model|ADR 010: Session Model]]
- [[../decisions/011-two-device-limit|ADR 011: Two-Device Limit]]
- [[../api/02-authentication-api|Authentication API]]
- [[../database/01-authentication-schema|Authentication Schema]]
- [[../security/01-authentication-security|Authentication Security]]
