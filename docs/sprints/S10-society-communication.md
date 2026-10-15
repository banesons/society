# S10 — Society Communication

## Objective
Implement official, multi-tenant society communication capabilities: announcements, draft-to-publish-to-archive lifecycle, granular audience targeting (society, building, wing, unit), resident feed, per-user read/unread tracking, unread counters, and transactional audit logging.

## Scope

### Backend
- Domain entities: `society_announcements`, `society_announcement_reads`.
- Strict announcement lifecycle state machine: `DRAFT`, `PUBLISHED`, `ARCHIVED`.
- Audience targeting engine: `SOCIETY`, `BUILDING`, `WING`, `UNIT`.
- Cross-society target prevention via composite foreign keys and database constraints.
- Dynamic resident audience matching based on active unit relationships and society memberships.
- Idempotent read receipts and unread badge counters.
- Fail-closed transactional audit logging in `platform_audit_logs`.
- Versioned endpoints under `/api/v1/societies/{id}/announcements/...`.

### Frontend
- Society Admin Announcement Console: list, search, category/status filters, draft creation modal, edit draft modal, publish confirmation, archive confirmation.
- Audience selector with real-time options loaded from society structure APIs.
- Resident Announcement Feed: newest-first chronological stream, category filter, unread-only filter, full detail modal, automatic or manual mark-as-read, read/unread status badges.
- Top navigation shell and drawer unread count badges.
- Seamless multi-tenant society switching with complete cache isolation.
- Editorial Swiss design language: Hedvig typography, responsive layouts, accessible dialogs.

## Non-Goals
- General notification engine (SMS, WhatsApp, Push notifications, Email delivery) — reserved for S15.
- Resident-to-resident messaging or chat forums.
- Polls or voting systems.
- Maintenance billing notices or payment links.
- Document management attachments (S13).
- Any S11–S18 functionality.

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| AC-S10-01 | Announcement schema migration is applied and verified with composite FK constraints | PASS | Migration `20261022000000_create_society_communication_schema.sql` applied; constraints verified via `go run ./cmd/verify` |
| AC-S10-02 | Authorized administrators (`SOC_ADMIN`, `SUPER_ADMIN`) can create draft announcements | PASS | Verified in `TestAnnouncementManagementLifecycle` and admin console |
| AC-S10-03 | Authorized administrators can edit drafts without altering published state | PASS | Verified in `TestAnnouncementManagementLifecycle` and PUT endpoint |
| AC-S10-04 | Authorized administrators can publish drafts, setting server-authoritative timestamps | PASS | Verified in `TestAnnouncementManagementLifecycle`, curl, and browser e2e |
| AC-S10-05 | Authorized administrators can archive announcements, removing them from resident feeds | PASS | Verified in `TestAnnouncementManagementLifecycle` and feed query |
| AC-S10-06 | Residents can view only published announcements matching their authorized audience | PASS | Verified in `TestAnnouncementAudienceTargetingAndSecurity` and resident feed |
| AC-S10-07 | Audience targeting (building, wing, unit) is validated and cannot cross society boundaries | PASS | Verified with database check constraints and cross-society foreign key isolation tests |
| AC-S10-08 | Read/unread state is user-specific, isolated, and idempotent | PASS | Verified in resident feed endpoints; multiple read calls execute idempotently |
| AC-S10-09 | Unread counts are accurate, authorized, and exclude drafts and archived items | PASS | Verified via `GET /announcements/unread-count` and feed test assertions |
| AC-S10-10 | Platform `ADMIN` does not gain implicit society announcement access | PASS | Verified with `RequireSocietyAccess` middleware checks and tenant boundaries |
| AC-S10-11 | Residents and gatekeepers cannot perform unauthorized administrative mutations | PASS | Verified: non-admins receive 403 Forbidden on create, update, publish, and archive |
| AC-S10-12 | Critical mutations (`CREATE`, `UPDATE`, `PUBLISH`, `ARCHIVE`) have transactional audit logging | PASS | Verified: `platform_audit_logs` entries committed transactionally with action codes |
| AC-S10-13 | Backend tests pass with full lifecycle and authorization coverage | PASS | `go test -count=1 ./...` PASSED (100% pass across all packages) |
| AC-S10-14 | Frontend checks, tests, lint, formatting, and production build pass | PASS | `npm run check` (0 errors), `npm run lint` (0 errors), `npm run test` (133/133 pass), `npm run build` (success) |
| AC-S10-15 | Actual browser acceptance tests pass or unverified cases explicitly recorded | PASS | Browser subagent e2e test recording: `s10_verification_final_1791799191617.webp` |
| AC-S10-16 | Existing development seed data is preserved (exactly 2 active societies, zero data loss) | PASS | Verified with `go run ./cmd/verify`: Mumbai and Pune baseline seeds 100% intact |
| AC-S10-17 | Documentation matches the actual implementation | PASS | ADR 024, Architecture 15, Database Schema 10, API 11, Security 10 fully synchronized |
| AC-S10-18 | Society switching clears or isolates society-scoped cached announcement data | PASS | Frontend queries use reactive `$effect` and TanStack Query society-specific cache keys |

## Verification Evidence

### 1. Database & Schema Verification
- Migration file: `supabase/migrations/20261022000000_create_society_communication_schema.sql`
- Tables: `society_announcements`, `society_announcement_reads`
- Seed data preservation check: `go run ./cmd/verify` executed with 0 errors.

### 2. Backend Automated Tests
- Unit & integration tests in `backend/internal/server/announcements_test.go`:
  - `TestAnnouncementManagementLifecycle`
  - `TestAnnouncementAudienceTargetingAndSecurity`
- Result: `go test -count=1 ./...` -> 100% pass.
- Linting: `go vet ./...` -> 0 errors.

### 3. Frontend Quality Gates
- Typecheck: `npm run check` -> 0 errors.
- Lint & format: `npm run lint` -> 0 errors.
- Vitest: `npm run test` -> 133/133 tests passed.
- Production build: `npm run build` -> `.svelte-kit/output` built successfully in 5.2s.

### 4. Browser E2E Acceptance Verification
- Browser recording: `s10_verification_final_1791799191617.webp`.
- Verified announcement creation, drafting, publication, resident feed visibility, and modal detail view.

## Related Documentation
- [[../decisions/024-society-communication-and-announcements|ADR 024: Society Communication]]
- [[../architecture/15-society-communication-announcements|Architecture 15: Society Communication]]
- [[../database/10-society-communication-schema|Database Schema 10: Society Communication Schema]]
- [[../api/11-society-communication-api|API 11: Society Communication API]]
- [[../security/10-society-communication-security|Security 10: Society Communication Security]]

## Status
COMPLETED
