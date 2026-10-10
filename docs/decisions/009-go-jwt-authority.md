# ADR 009: Go Backend as JWT Authority with Ed25519 Signing

## Status
Accepted

## Context
Decoupling application authentication from external auth daemons improves latency, avoids vendor lock-in, and eliminates runtime dependency on Supabase GoTrue. We need a modern, secure, and fast cryptographic signing algorithm for access tokens.

## Decision
1. **Go as JWT Authority**:
   - The Go backend issues, signs, and validates all JSON Web Tokens (JWTs).
   - Supabase Auth does NOT issue or manage CHS access tokens.
2. **Ed25519 Asymmetric Signing**:
   - Access tokens are signed using the **Ed25519** (Edwards-curve Digital Signature Algorithm).
   - Ed25519 offers high verification speed, resistance to side-channel attacks, and compact 32-byte public/private keys.
   - The Ed25519 private key is held strictly by the backend in memory and loaded from environment configuration (`JWT_PRIVATE_KEY`). It is NEVER exposed to the frontend, Git, logs, or client APIs.
3. **Access Token Lifespan & Claims**:
   - Lifespan is strictly short-lived: **15 minutes**.
   - Standard claims include:
     - `sub`: User UUID
     - `session_id`: UUID of the associated server-side session
     - `iat`: Issued-at timestamp
     - `exp`: Expiration timestamp (15 minutes from issue)
   - Unnecessary PII is excluded from the token payload.

## Consequences
- Sub-millisecond JWT verification in Go middleware without database lookups on every request.
- No public key distribution needed in early sprints; asymmetric design permits future services to verify tokens without access to the signing private key.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
- [[../api/02-authentication-api|Authentication API]]
