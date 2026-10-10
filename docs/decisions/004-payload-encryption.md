# ADR 004: Exclusion of Application-Level Payload Encryption

## Status
Accepted

## Context
Early architectural notes discussed wrapping API JSON payloads in an encrypted envelope (`{ "payload": "<encrypted-data>" }`). Application-level payload encryption introduces severe complexity: key distribution, in-browser key security, lack of replay prevention without complex nonce tracking, broken standard browser devtools debugging, and CPU serialization overhead.

## Decision
CHS will **NOT** implement application-level payload encryption.
- We will NOT use custom AES API envelopes, frontend AES keys, or backend symmetric key decryption middleware.
- Transport security relies strictly on **TLS/HTTPS** with modern cipher suites.
- Application-level data protection relies on:
  1. Strict authentication and authorization.
  2. PII masking on API outputs (e.g. `9967xxxx1`).
  3. API data minimization (fetching only fields required for the active screen).
  4. Server-side session validation.

## Consequences
- Clean, readable JSON payloads in development and production debugging.
- No client-side key storage vulnerabilities in LocalStorage or cookies.
- Standard HTTP tooling, proxies, and observability systems function seamlessly.

## Related Documentation
- [[../api/01-api-overview|API Overview]]
- [[../architecture/01-system-overview|System Overview]]
- [[../architecture/02-backend|Backend Architecture]]
