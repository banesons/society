# S08 — Gate Foundation

## Objective
Establish the physical and operational foundation for gate security, including operational gate configuration, gatekeeper accounts (`GATEKEEPER` role in `society_user_roles`), shift scheduling, and active duty session management, without duplicating identity systems or conflating civic committee designations with operational perimeter staffing.

## Scope

### Backend
- Extend `society_user_roles.role` check constraint to include `'GATEKEEPER'`.
- Extend `society_gates` with `directionality` (`BIDIRECTIONAL`, `INGRESS_ONLY`, `EGRESS_ONLY`) and `operational_status` (`ACTIVE`, `INACTIVE`, `MAINTENANCE`, `EMERGENCY_OPEN`).
- Gate operational configuration updates (`PATCH /api/v1/societies/{id}/gates/{gateId}`).
- Gatekeeper roster management: assign user as `GATEKEEPER`, list gatekeepers, and revoke assignments with automatic active duty session closure.
- Shift definitions table (`society_gate_shifts`) and APIs (`GET / POST`).
- Active duty sessions table (`society_gate_sessions`) with check-in and check-out APIs and anti-double-booking concurrency constraint (`uq_gatekeeper_active_session`).
- Multi-tenant isolation and society lifecycle validation (`403 SOCIETY_SUSPENDED`, `403 SOCIETY_PENDING`).
- Transactional audit logging for all mutations in `platform_audit_logs`.

### Frontend
- Gate & Security Management section in Society Console (`/society/[id]`).
- Three operational views adhering to CHS Editorial Swiss UI:
  1. **Gate Configuration**: List gates with directionality and operational status badges, modal to edit gate settings.
  2. **Gatekeeper Staffing**: Table of assigned gatekeepers, modal to assign new gatekeeper by phone, revoke action.
  3. **Duty Sessions & Terminal Base**: Roster of active gate duty sessions, live on-duty badges, check-in and check-out dialogs.

## Acceptance Criteria
- [x] No parallel identity store created; gatekeepers authenticate via core phone OTP and hold `GATEKEEPER` role in `society_user_roles`.
- [x] Role segregation enforced: `GATEKEEPER` access cannot perform `SOC_ADMIN` actions.
- [x] Existing `society_gates` extended with `directionality` and `operational_status` without breaking S05 structure models.
- [x] Admins can update gate attributes (`PATCH /api/v1/societies/{id}/gates/{gateId}`).
- [x] Gatekeeper assignments supported by normalized phone number (`POST /api/v1/societies/{id}/gatekeepers`).
- [x] Revoking a gatekeeper terminates any active duty sessions for that guard.
- [x] Configurable shifts supported (`society_gate_shifts`).
- [x] Guards can check in to an active gate session (`society_gate_sessions`).
- [x] A guard cannot hold multiple active sessions concurrently (`409 SESSION_ALREADY_ACTIVE`).
- [x] Inactive gates reject duty check-in (`400 GATE_INACTIVE`).
- [x] Check-out records `ended_at` and handover notes.
- [x] Multi-tenant isolation enforced via `society_id` scoping.
- [x] Suspended society restrictions enforced (`403 SOCIETY_SUSPENDED`).
- [x] Transactional audit logs written to `platform_audit_logs` for all operations.
- [x] UI strictly follows CHS Editorial Swiss design system (Hedvig fonts, off-white canvas, hairline borders, jet-black buttons, airy tables).
- [x] All quality gates pass (`gofmt`, `go vet`, `go test`, `go build`, `npm run check`, `npm run lint`, `npm test`, `npm run build`).

## Related Documentation
- [[../decisions/020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations]]
- [[../architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]
- [[../api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API Specification]]
- [[../security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security Specification]]
- [[README|Master Sprint Index]]

## Status
- Architecture & Design: **COMPLETE & APPROVED**
- Database Migrations: **COMPLETE & APPLIED**
- Backend Implementation: **COMPLETE & PASSING**
- Frontend Implementation: **COMPLETE & PASSING**
- Quality Gates & Verification: **COMPLETE & VERIFIED**
- Sprint State: **CLOSED & VERIFIED**

