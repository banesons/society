# ADR 003: Semantic HTTP Methods and Security Model

## Status
Accepted

## Context
Some legacy or misguided architectures force all API operations into `POST` requests under the mistaken belief that `POST` is inherently more secure or prevents browser URL leakage. This breaks RESTful conventions, disables HTTP caching, complicates intermediary tooling, and provides false security guarantees.

## Decision
CHS will strictly use semantic HTTP methods:
- `GET`: Safe, idempotent retrieval of resources. Must never modify state.
- `POST`: Creation of resources or execution of domain actions.
- `PUT` / `PATCH`: Idempotent full replacement or partial update of resources.
- `DELETE`: Removal or soft deletion of resources.

Security in CHS is NOT derived from using `POST` for every call. Real security is enforced via:
- HTTPS transport encryption.
- Robust authentication and JWT validation (S01+).
- Strict role and tenancy-based authorization checks at the handler and domain layers.
- Input validation and sanitization.
- Rate limiting and size-bounded request bodies.
- Data minimization and PII masking.
- Secure HTTP headers (CSP, HSTS, X-Content-Type-Options).

## Consequences
- Clean, semantic API routes following standard REST paradigms.
- Frontend and caching proxies can reliably leverage standard HTTP caching for safe `GET` operations.

## Related Documentation
- [[../api/01-api-overview|API Overview]]
- [[../architecture/02-backend|Backend Architecture]]
