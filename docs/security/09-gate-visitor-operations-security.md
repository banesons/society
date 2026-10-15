# 09 — Gate Visitor Operations Security

## Overview
This document specifies the security boundaries, threat model, authorization controls, PII protection, and integrity guarantees for Sprint S09 Gate Visitor Operations.

## Security Controls

### 1. Zero Trust Cross-Society Tenant Isolation
All visitor queries, mutations, passes, and gate events enforce multi-tenant isolation through two defense layers:
- **Application Level**: Handlers extract and validate `societyID` via authentication context (`middleware.GetSocietyID(ctx)`). Database queries unconditionally append `WHERE society_id = $X`.
- **Database Level**: Composite foreign keys mathematically enforce that `unit_id`, `resident_id`, `entry_gate_id`, `exit_gate_id`, and `visit_id` belong to the exact same `society_id`. An attacker cannot reference an entity belonging to Society B inside Society A.

### 2. Authorization Hierarchy & Least Privilege
- **SUPER_ADMIN**: Platform governance with emergency operational visibility.
- **ADMIN**: Platform administration. Denied implicit society data access (returns 403 Forbidden).
- **SOC_ADMIN**: Administrative oversight over their assigned society's visitors, passes, approvals, and logs. Cannot override resident-owned unit approval preferences.
- **GATEKEEPER**: Operational authority over walk-in registration, pass lookup, check-in, and check-out. Strictly barred from approving or rejecting visits (returns 403). Cannot alter unit approval preferences. Must hold an active duty session (`society_gate_sessions`) at the designated gate to record check-in/out.
- **Resident Occupant**: Authorized only for visits directed to units they actively occupy or own (`unit_relationships`). May configure their own unit's approval policy, register expected visitors, approve/reject pending visits for their flat, and issue passes. Cannot approve/reject visits for other units (returns 403 `OCCUPANT_UNAUTHORIZED`). Cannot check-in visitors at gates.

### 3. Concurrency & Replay Attacks
- **Double Entry / Race Conditions**: Handlers use PostgreSQL row locks (`SELECT status FROM society_visits WHERE id = $1 FOR UPDATE`) within transactions. If two guards click Check-In simultaneously, the first transitions the record to `CHECKED_IN`, and the second encounters status != `APPROVED` and is rejected with `409 Conflict`.
- **Pass Token Replay**: Pass verification consumes single-use passes by setting status to `CONSUMED`. Replaying the same pass code or bearer token fails validation.
- **Bearer Token Security**: High entropy tokens (`crypto/rand`) formatted as `chs_pass_<random_hex>`. The database persists only `SHA256(token)`. Leaked database dumps do not disclose usable bearer tokens.

### 4. PII Minimization and URL Privacy
- No sensitive PII in URL parameters. All identifiers are UUIDs.
- Visitor phone numbers are masked by default in responses (e.g., `+919820xxxx23`) unless explicitly authorized.
- No government identity card numbers, biometric data, or visitor photographs are stored.
- Raw bearer tokens are never logged in application logs or audit tables.

### 5. Audit Logging Fail-Closed Guarantee
All operational actions (`VISIT_REGISTERED`, `VISIT_APPROVED`, `VISIT_REJECTED`, `VISIT_CHECKED_IN`, `VISIT_CHECKED_OUT`, `PASS_ISSUED`, `PASS_REVOKED`) write an entry to `platform_audit_logs` in the identical PostgreSQL transaction. If audit log insertion fails, the transaction is aborted with `ROLLBACK`.

## Threat Matrix & Mitigations

| Threat Vector | Severity | Mitigation Strategy |
| :--- | :--- | :--- |
| Forged Resident Approval | High | Resident identity validated against active `unit_relationships` and `society_members.user_id`. Gatekeepers cannot submit approval transitions. |
| Replayed Visitor Pass | High | Passes transition to `CONSUMED` upon first check-in. Time windows enforced (`valid_until < NOW()`). |
| Gate Direction Bypass | Medium | Ingress through `EGRESS_ONLY` gates and egress through `INGRESS_ONLY` gates are blocked at domain and DB check levels. |
| Cross-Society Leaks | High | Composite foreign keys and society-scoped middleware queries. 404 returned for foreign IDs. |
| Check-in at Inactive Gate | Medium | Gate `operational_status` validated; only `ACTIVE` and `EMERGENCY_OPEN` allow traffic. |
| Spoofed Client Timestamps | Low | All event timestamps (`check_in_at`, `check_out_at`, `created_at`) use server-authoritative `NOW()`. |

## Related Documents
- [[08-gate-foundation-and-staffing-security]]
- [[../architecture/14-gate-visitor-operations]]
- [[../decisions/023-visitor-lifecycle-and-gate-operations]]
- [[../database/09-gate-visitor-operations-schema]]
- [[../api/10-gate-visitor-operations-api]]
