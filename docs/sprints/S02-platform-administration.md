# S02 — Platform Administration

## Objective
Establish platform-level administrative identity, roles (`SUPER_ADMIN` and `ADMIN`), administrative account lifecycle management, server-side authorization enforcement, and an immutable audit trail without modifying S01 authentication foundation.

## Scope
### Backend
- Idempotent bootstrapping of initial `SUPER_ADMIN` (`+919967089101`) via backend environment configuration.
- Platform role data model (`SUPER_ADMIN`, `ADMIN`) with operational status (`ACTIVE`, `DISABLED`).
- Authorization middleware: `RequireSuperAdmin` and `RequirePlatformRole` enforcing role and active status in Go.
- Administrative lifecycle APIs:
  - `GET /api/v1/platform/admins` (List admins)
  - `POST /api/v1/platform/admins` (Create `ADMIN` account by phone)
  - `PATCH /api/v1/platform/admins/{id}/status` (Enable/Disable `ADMIN` with immediate session revocation)
  - `GET /api/v1/platform/audit-logs` (List append-only audit trail)
- Updated `/api/v1/auth/me` to expose authenticated caller's active platform roles.
- Immutable append-only audit ledger (`platform_audit_logs`).

### Frontend
- Super Admin platform console view (`/admin`).
- Administrative team roster displaying masked phones, roles, and status tags.
- Admin creation modal with phone normalization and validation.
- Status toggle controls (Suspend / Reactivate) with immediate UI update.
- Audit history viewer showing recent administrative events.
- Client-side navigation guards for `/admin` based on `platform_roles` from `/api/v1/auth/me`.

## Dependencies
- [[S01-identity-authentication]]

## Deliverables
- Database migration `create_platform_admin_schema.sql` (`platform_roles`, `platform_audit_logs`).
- Go authorization middleware and platform admin HTTP handlers.
- SvelteKit application architecture refactored:
  - `/` — Public landing page with product introduction and Sign In action.
  - `/login` — Dedicated OTP authentication entry point with role-based routing.
  - `/admin` — Protected Super Admin management console (roster, audit ledger, session management).
  - `/developer` — Isolated diagnostics and API test console (restricted outside dev).
  - `/unauthorized` — Clear access-denied state with switch-account action.
  - Global session termination (`LogoutAllDialog` via `POST /api/v1/auth/logout-all`).
- Backend and frontend test suites covering role checks, admin creation, status toggles, session revocations, and routing rules.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../README|CHS Documentation Index]]
- [[../architecture/03-frontend|03 — Frontend Architecture]]
- [[../architecture/05-authentication|05 — Authentication Architecture]]
- [[../architecture/06-platform-administration|06 — Platform Administration Architecture]]
- [[../api/03-platform-administration-api|03 — Platform Administration API]]
- [[../database/02-platform-administration-schema|02 — Platform Administration Database Schema]]
- [[../security/02-platform-administration-security|02 — Platform Administration Security]]
- [[../decisions/012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../decisions/013-super-admin-bootstrap|ADR 013: Super Admin Bootstrap]]
- [[../decisions/014-platform-audit-trail|ADR 014: Platform Audit Trail]]

## Status
- Database Phase: COMPLETE (Migration applied, repository integration verified)
- Backend API Phase: COMPLETE (Endpoints implemented, authorization middleware verified, 100% backend test suite passing)
- Frontend Phase: COMPLETE (Route structure refactored, Super Admin console, /login entry point, /developer diagnostics, /unauthorized state, 100% frontend tests passing, clean build)
- End-to-End Validation Phase: COMPLETE (Browser automated E2E verified landing page, login, role redirection, Super Admin console tabs, audit ledger, and session revocation)
- Overall Sprint Status: COMPLETE (Ready for S02 sign-off)
