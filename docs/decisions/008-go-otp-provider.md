# ADR 008: Go Backend as OTP Authority and Pluggable SMS Provider

## Status
Accepted

## Context
Indian telecommunications regulations mandate strict Telecom Regulatory Authority of India (TRAI) Distributed Ledger Technology (DLT) compliance for commercial and transactional SMS. Relying on Supabase Auth's phone provider (Twilio / MessageBird) introduces high per-SMS costs (~₹3–4/SMS vs ₹0.15–0.20/SMS domestically), latency issues, and severe friction matching approved DLT Entity and Template IDs.

## Decision
1. **Go as OTP Authority**:
   - The Go backend generates, hashes, stores, and verifies all one-time passwords (OTPs).
   - Supabase Auth is NOT used for OTP generation, delivery, or verification.
2. **OTP Properties**:
   - 6-digit numeric OTP generated via cryptographically secure randomness (`crypto/rand`).
   - 5-minute validity window, strictly single-use.
   - Stored in the database only as a salted SHA-256 hash in the `auth_otps` table (never in plaintext).
   - Maximum 3 verification attempts per OTP before invalidation.
3. **Pluggable SMS Provider**:
   - The Go OTP service depends on an interface:
     ```go
     type SMSProvider interface {
         SendOTP(ctx context.Context, phone string, otp string) error
     }
     ```
   - In `development` and `test` environments, `MockSMSProvider` logs the OTP to structured logs or accepts predefined test OTPs (`123456`) for configured test numbers (`+919967089101`, `+918422862001`).
   - In `production`, a pluggable DLT-compliant Indian gateway (e.g. MSG91 or Fast2SMS) will be instantiated via configuration without altering core OTP or business logic.

## Consequences
- Fast delivery (3–8 seconds) via Indian telco interconnects.
- Low per-SMS operating costs.
- Zero SMS cost during local development and testing.
- Full auditability and rate-limiting control inside the Go backend.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
- [[../api/02-authentication-api|Authentication API]]
