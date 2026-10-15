# ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations

## Context
Sprint S05 established physical society structures, including units, physical gates (`society_gates`), and aggregate headcount requirements (`society_staffing_requirements`).
Sprint S06 established resident member identities and unit relationships (`society_members`, `society_unit_members`).
Sprint S07 established civic governance titles and managing committee appointments (`society_designations`, `society_committee_appointments`).

Sprint S08 introduces the operational gate security foundation. Physical entry and exit points must be configured for day-to-day operations, staffed by authorized security personnel (`GATEKEEPER`), and tracked via active duty sessions before visitor access workflows (S09) can be introduced.

Key architectural questions arise:
1. How should gatekeeper identity and system access be modeled without creating a disconnected, parallel credential store?
2. How do civic committee roles (e.g. "Security In-Charge" from S07) relate to operational security personnel?
3. How should physical gates be extended with operational modes and directionality without breaking existing S05 structure models?
4. How should gate duty sessions and staffing accountability be enforced atomically and securely across tenant boundaries?

---

## Decisions

### 1. Unified Phone-Based Identity with `GATEKEEPER` Role
- **No parallel identity store**: Gatekeepers are authenticated users in the core `users` table via standard phone OTP.
- **Tenant Authorization**: Operational gate security privileges are granted via `society_user_roles` with `role = 'GATEKEEPER'`.
- The database constraint on `society_user_roles.role` is extended from `('SOC_ADMIN')` to `('SOC_ADMIN', 'GATEKEEPER')`.
- Assigning `GATEKEEPER` confers operational gate permissions only (viewing society gates, checking into duty sessions, and in S09 logging visitors). It confers zero administrative privileges (`SOC_ADMIN`) or platform privileges (`ADMIN`, `SUPER_ADMIN`).

### 2. Separation of Civic Governance vs. Operational Staffing
- **Civic Governance (S07)**: Society members holding committee appointments (e.g., `CHAIRMAN`, `SECRETARY`, or a custom `SECURITY_CONVENOR` designation) govern security policy, contract security agencies, and oversee audits. They are tracked in `society_committee_appointments`.
- **Operational Staffing (S08)**: Gatekeepers are stationed personnel on duty. They hold `role = 'GATEKEEPER'` in `society_user_roles`.
- A user may conceivably hold both (e.g. a resident volunteer acting as a relief gatekeeper), but the data models remain strictly decoupled.

### 3. Operational Gate Extension
- The existing `society_gates` table is augmented with operational parameters:
  - `directionality`: `BIDIRECTIONAL` (default), `INGRESS_ONLY`, `EGRESS_ONLY`.
  - `operational_status`: `ACTIVE`, `INACTIVE`, `MAINTENANCE`, `EMERGENCY_OPEN`.
- Existing columns (`name`, `gate_code`, `gate_type`, `required_staff_per_shift`, `description`) remain preserved, ensuring 100% backward compatibility with S05 formation models.
- Admins can update gate attributes (`PATCH /api/v1/societies/{id}/gates/{gateId}`).

### 4. Shift Schedules and Gate Duty Sessions
- **Configurable Shifts (`society_gate_shifts`)**: Societies define their daily shifts (e.g., Morning `06:00-14:00`, Afternoon `14:00-22:00`, Night `22:00-06:00`).
- **Active Duty Sessions (`society_gate_sessions`)**:
  - When a gatekeeper reports to a gate, they initiate a duty session (`POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-in`).
  - At shift end or handover, the session is completed (`POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-out`).
  - Strict concurrency: a gatekeeper cannot be checked into multiple active sessions across different gates simultaneously.
  - Duty sessions provide real-time visibility into who is currently stationed at each gate and produce immutable audit records for physical perimeter accountability.

### 5. Multi-Tenant Isolation & Lifecycle Enforcement
- All gate, shift, gatekeeper, and session queries enforce composite foreign keys and tenant isolation (`society_id`).
- Societies in `PENDING` or `SUSPENDED` status reject gatekeeper operations with standard HTTP 403 status codes (`SOCIETY_PENDING`, `SOCIETY_SUSPENDED`).
- All configuration changes, roster modifications, and check-in/out transitions write transactional records to `platform_audit_logs`.

---

## Consequences

### Positive
- Zero duplication of identity infrastructure or authentication credentials.
- Clear separation between civic committee governance (S07), residential tenancy (S06), and operational perimeter staffing (S08).
- Complete backward compatibility with S05 gate structures.
- Foundation in place for visitor logging and verification in S09.

### Negative / Trade-offs
- Gatekeepers require a phone number capable of receiving OTP or test-mode verification for initial sign-in.
- Society admins must configure both gate definitions and gatekeeper role assignments.

---

## Status
**ACCEPTED & APPROVED** (Sprint S08)

## Related Documentation
- [[../architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]
- [[../api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API Specification]]
- [[../security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security Specification]]
- [[../sprints/S08-gate-foundation|S08 — Gate Foundation Sprint]]
