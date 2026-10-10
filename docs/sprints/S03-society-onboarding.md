# S03 — Society Onboarding

## Objective
Establish housing societies as first-class multi-tenant entities within the CHS platform, providing the platform-level governance architecture and workflow for a Super Admin to register, inspect, activate, suspend, and maintain societies.

The principal business outcome is:
> **"A Super Admin can register a housing society, maintain its platform-level profile, and control whether that society is active on the CHS platform."**

---

## 1. Scope Definition

### A. Mandatory S03 Features
- **Database Schema**:
  - `societies` table with UUID primary key, unique uppercase `society_code`, profile attributes (name, registration number, address, locality, city, state, pincode, contact info), lifecycle status, and timestamps.
  - Check constraints for Indian 6-digit PIN code and society code format.
  - B-tree and partial indexes for status, location, code, and registration number.
  - Foreign key constraints with `ON DELETE RESTRICT` for referential safety.
- **Backend Domain & Services**:
  - Society registration service with normalization and duplicate detection.
  - Deterministic lifecycle state machine (`PENDING`, `ACTIVE`, `SUSPENDED`).
  - Allowed profile modification service.
  - Status transition controls with mandatory operational reason capture for suspension.
  - Transactional audit log recording for all society mutations (`platform_audit_logs`).
  - Authorization middleware enforcement via `RequireSuperAdmin`.
- **Platform APIs**:
  - `GET /api/v1/platform/societies` (Paginated list, status filter, name/code search)
  - `POST /api/v1/platform/societies` (Registration into `PENDING` state)
  - `GET /api/v1/platform/societies/{id}` (Full profile details)
  - `PATCH /api/v1/platform/societies/{id}` (Update allowed profile fields)
  - `POST /api/v1/platform/societies/{id}/activate` (Activate from `PENDING` or `SUSPENDED`)
  - `POST /api/v1/platform/societies/{id}/suspend` (Suspend from `ACTIVE` with mandatory reason)
- **Frontend Super Admin Console**:
  - Integrated "Societies" tab within `/admin`.
  - Society roster table displaying code, name, city/state, status badge, created date, and action triggers.
  - Search input and status filter dropdown (`PENDING`, `ACTIVE`, `SUSPENDED`).
  - "Register Society" modal form with client validation.
  - "Society Details" inspection view.
  - Status transition modal (Activate / Suspend) with reason capture.
  - Comprehensive loading, empty, and error feedback states.

### B. Explicitly Deferred Features (Out of Scope for S03)
- End-user public society self-registration wizard (reserved for future public self-service workflows).
- Society Admin (`SOC_ADMIN`) assignments and mappings (deferred to S04).
- Buildings, wings, floors, and units/flats (deferred to S05).
- Resident and member relationships (owners, tenants, family) (deferred to S06).
- Committee designations (Secretary, Treasurer, Chairman) (deferred to S07).
- Gate operations and visitor management (deferred to S08 & S09).
- Maintenance charges and billing (deferred to S11 & S12).
- Restaurant POS (deferred to S19+).

---

## 2. Dependencies
- [[S01-identity-authentication]]: Phone OTP, Ed25519 access JWTs, server-side sessions.
- [[S02-platform-administration]]: `platform_roles` (`SUPER_ADMIN` check), `platform_audit_logs` immutable ledger.

---

## 3. Acceptance Criteria & Test Strategy

### Authorization Acceptance
- Unauthenticated requests to `/api/v1/platform/societies/*` return 401 `UNAUTHORIZED`.
- Active `SUPER_ADMIN` can register, inspect, update, activate, and suspend societies.
- Platform `ADMIN` requests return 403 `FORBIDDEN`.
- Ordinary resident requests return 403 `FORBIDDEN`.

### Lifecycle Acceptance
- New societies are created with `status = 'PENDING'`.
- Duplicate `society_code` is rejected with 409 `SOCIETY_CODE_EXISTS`.
- Invalid PIN code or society code syntax is rejected with 400 `VALIDATION_ERROR`.
- `PENDING` → `ACTIVE` transition succeeds via `/activate`.
- `ACTIVE` → `SUSPENDED` transition succeeds via `/suspend` when valid `reason_code` is provided (and details if required); fails with 400 `SUSPENDED_REASON_REQUIRED` when missing, 400 `INVALID_SUSPENSION_REASON` when unknown or inactive, or 400 `SUSPENSION_DETAILS_REQUIRED` when `OTHER_REVIEW_REQUIRED` lacks details.
- `PENDING` → `SUSPENDED` is rejected with 400 `INVALID_STATUS_TRANSITION`.
- `SUSPENDED` → `ACTIVE` transition succeeds via `/reactivate` (or `/activate`).

### Audit Acceptance
- Every society creation, profile update, activation, and suspension generates an immutable record in `platform_audit_logs`.
- If audit record insertion fails, the database transaction rolls back and returns 500 `INTERNAL_ERROR`.
- `UPDATE` or `DELETE` on society audit logs is blocked by database trigger.

### Regression Acceptance
- S01 authentication test suite passes 100%.
- S02 platform administration test suite passes 100%.
- Frontend type checking (`svelte-check`), linting, and unit tests pass 100%.

---

## 4. Proposed Implementation Sequence (Phase 2)

1. **Database Phase**:
   - Write migration `20261011000000_create_societies_schema.sql`.
   - Apply to Supabase database.
   - Verify table constraints, foreign keys, and indexes.
2. **Backend Domain & Repository Phase**:
   - Create `backend/internal/society` package with domain models and `Repository`.
   - Implement transactional registration, profile updates, status transitions, and queries.
   - Write unit and database integration tests.
3. **Backend API & Middleware Phase**:
   - Implement HTTP handlers and DTO validation in `backend/internal/server`.
   - Mount routes in `backend/internal/server/routes.go` behind `RequireSuperAdmin`.
   - Write API endpoint integration tests.
4. **Frontend Phase**:
   - Add society API methods to `frontend/src/lib/api/platform.ts` (or `society.ts`).
   - Implement `SocietyRoster.svelte`, `RegisterSocietyDialog.svelte`, `SocietyDetailsDialog.svelte`, and `SocietyStatusDialog.svelte`.
   - Mount "Societies" tab in `/admin/+page.svelte`.
   - Write frontend unit tests for society client and routing.
5. **E2E & Validation Phase**:
   - Run complete backend and frontend test suites.
   - Verify live browser registration, activation, suspension, and audit logging.

---

### Society Code Convention
- **Format**: `SOC-{CITY_CODE}-{RANDOM_SUFFIX}`
  - `SOC-MUM-7K4P2X` (Mumbai society)
  - `SOC-PUN-9D3R8A` (Pune society)
- Backend generated only; collision retry bounded; database uniqueness enforced; immutability trigger active.

---

## 5. Implementation & Verification Summary

### Database Migration
- File: `supabase/migrations/20261011000000_create_societies_schema.sql`
- Status: Applied to Supabase PostgreSQL (`glpjpqpbhdcmgmwlnkay`).
- Schema: Table `societies`, check constraints, case-insensitive unique code index, immutability trigger `societies_code_immutability_trg`, query indexes.

### Backend Implementation
- Package: `backend/internal/society` (`code.go`, `models.go`, `repository.go`)
- Handlers: `backend/internal/server/society_handlers.go`, mounted in `routes.go` behind `RequireSuperAdmin`.
- Tests: `code_test.go`, `repository_test.go`, `society_test.go` (11 subtests). All passed 100%.

### Frontend Implementation
- API Client: `frontend/src/lib/api/society.ts`
- Super Admin Console: "Societies" tab in `/admin/+page.svelte`
- Components: `society-roster.svelte`, `register-society-dialog.svelte`, `society-details-dialog.svelte`, `edit-society-dialog.svelte`, `society-status-dialog.svelte`
- Quality Gates: `svelte-check` 0 errors, 0 warnings; `eslint`/`prettier` clean; `vitest` 42 passed (100%); `vite build` succeeded.
- E2E Browser Test: Complete flow verified via automated browser testing (login, registration with generated `SOC-MUM-` code, activation, suspension with mandatory reason, reactivation, audit log ledger check).

---

## 6. Related Documentation
- [[README|Master Sprint Index]]
- [[../README|CHS Documentation Index]]
- [[../architecture/07-society-onboarding|07 — Society Onboarding Architecture]]
- [[../database/03-society-schema|03 — Society Database Schema]]
- [[../api/04-society-api|04 — Society Onboarding API]]
- [[../security/03-society-security|03 — Society Security Specification]]
- [[../decisions/015-society-lifecycle-and-tenancy|ADR 015: Society Lifecycle and Tenancy Boundary]]

---

## Status
- Architecture, Requirements & Documentation Phase: **COMPLETE**
- Implementation & Verification Phase: **COMPLETE**
