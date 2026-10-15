# S09 — Gate Operations

## Objective
Implement a complete, secure, operational visitor-management system integrated with the existing S08 gate foundation. A society must be able to manage expected visitors, walk-in visitors, resident approvals, visitor passes, gate entry, gate exit, visitor status and lifecycle, gatekeeper operations, visitor history, and audit records.

## Scope

### Backend
- Domain entities: `society_visits`, `society_visitor_passes`, `society_visit_approvals`, `society_gate_events`.
- Strict visit state machine: `PENDING_APPROVAL`, `APPROVED`, `REJECTED`, `CHECKED_IN`, `CHECKED_OUT`, `CANCELLED`, `EXPIRED`.
- Concurrency-safe entry/exit with PostgreSQL row-level locks preventing duplicate check-in.
- Gatekeeper duty session validation: movements authorized only by guards on active duty at the designated gate.
- Gate directionality enforcement: `INGRESS_ONLY`, `EGRESS_ONLY`, `BIDIRECTIONAL`.
- Pass generator with SHA-256 hashed bearer tokens and short pass codes.
- Transactional audit logging in `platform_audit_logs` (fail-closed on failure).
- Versioned endpoints under `/api/v1/societies/{id}/visitors/...`.

### Frontend
- Gate Console tab expansion: Live Gate Terminal, Expected Arrivals, Awaiting Approval, Visitor History.
- Walk-in and Expected visitor registration dialogs.
- Resident approval queue and quick approve/reject workflows.
- Live check-in and check-out actions with real-time feedback.
- Pass display modal with pass code and valid time-window.
- Responsive, accessible Editorial Swiss design system components.
- Complete state cleanup and cache isolation on society switching.

## Non-Goals
- Maintenance management or billing.
- Biometric hardware integrations.
- Social networking or messaging forums.
- Parallel authentication system for visitors.
- Separate visitor standalone application.

## Final Acceptance Criteria Matrix (S09 Closure)

| ID | Criterion | Status |
| :--- | :--- | :--- |
| AC-S09-F01 | Existing registered members can authenticate and access only authorized societies | VERIFIED |
| AC-S09-F02 | Require Resident Approval is OFF by default | VERIFIED |
| AC-S09-F03 | Gatekeepers cannot control or override resident-owned approval preferences | VERIFIED |
| AC-S09-F04 | Society administrators cannot impersonate residents or override resident preferences | VERIFIED |
| AC-S09-F05 | Authorized residents can manage their own units' approval preferences | VERIFIED |
| AC-S09-F06 | Residents can approve or reject visits only for units they are authorized to manage | VERIFIED |
| AC-S09-F07 | Backend enforces effective approval policy regardless of frontend state / request tampering | VERIFIED |
| AC-S09-F08 | Pass consumption and check-in are transactionally consistent | VERIFIED |
| AC-S09-F09 | Concurrent entry requests cannot create duplicate entries | VERIFIED |
| AC-S09-F10 | Gate operations enforce documented assignment, gate, and duty-session policy | VERIFIED |
| AC-S09-F11 | All applicable mutations retain transactional audit logging | VERIFIED |
| AC-S09-F12 | Backend and frontend quality gates pass | VERIFIED |
| AC-S09-F13 | Actual browser acceptance tests pass or unverified scenarios explicitly recorded | VERIFIED |
| AC-S09-F14 | Development seed baseline is preserved | VERIFIED |
| AC-S09-F15 | Documentation and acceptance evidence match implementation | VERIFIED |

## Related Documentation
- [[../decisions/023-visitor-lifecycle-and-gate-operations|ADR 023: Visitor Lifecycle and Gate Operations]]
- [[../architecture/14-gate-visitor-operations|Architecture 14: Gate Visitor Operations]]
- [[../database/09-gate-visitor-operations-schema|Database Schema 09: Gate Visitor Operations Schema]]
- [[../api/10-gate-visitor-operations-api|API 10: Gate Visitor Operations API]]
- [[../security/09-gate-visitor-operations-security|Security 09: Gate Visitor Operations Security]]
- [[S08-gate-foundation|S08 Gate Foundation]]

## Status
COMPLETED
