# S14 — Audit & Administration

## Objective
Deliver a unified, immutable, and performant audit and administrative activity experience across CHS. Extend the existing `platform_audit_logs` append-only ledger without introducing duplicate tables or secondary event systems. Deliver global audit visibility for `SUPER_ADMIN` in `/admin` and tenant-isolated audit visibility for `SOC_ADMIN` in `/society/{id}` with server-side pagination, structured search, multi-field filtering, sensitive metadata redaction, and strict role boundaries.

---

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| **AC-S14-01** | Existing audit architecture inspected and documented | PASS | Completed in [[docs/decisions/028-audit-and-administration|ADR 028]] and [[docs/architecture/19-audit-and-administration|Architecture 19]]; discovered 3,360 baseline records, append-only trigger, and details JSONB schema. |
| **AC-S14-02** | No duplicate audit system introduced without documented justification | PASS | Single ledger `platform_audit_logs` extended via forward-only nullable column `society_id` and dual indexes in migration `20261027000000_create_audit_and_administration_schema.sql`. |
| **AC-S14-03** | Platform audit list works with bounded server-side pagination | PASS | Verified in `TestAuditAndAdministrationSuite/AC-S14-03_&_04` and live API runner `s14_verify` (total=3,383, page=1, limit=5, pages=677). |
| **AC-S14-04** | Platform audit search and filters work | PASS | Verified in `TestAuditAndAdministrationSuite` and live API runner across action (`SOCIETY_CREATED`), society ID, date bounds, and free-text search. |
| **AC-S14-05** | Audit event details are readable and safely serialized | PASS | Verified in `GetPlatformAuditLogDetailHandler`, `TestAuditAndAdministrationSuite/AC-S14-05`, and `s14_verify` with human-readable action labels and actor display references. |
| **AC-S14-06** | Society-scoped audit history is available to authorized SOC_ADMIN users | PASS | Verified in `ListSocietyAuditLogsHandler`, `TestAuditAndAdministrationSuite/AC-S14-06_&_07` and `AC-S14-06b` (server-side checks strictly reject revoked assignments, non-admin roles, and suspended societies with 403 Forbidden; client header mismatch rejected), and `s14_verify`. |
| **AC-S14-07** | Cross-society audit access is denied | PASS | Verified in `TestAuditAndAdministrationSuite/AC-S14-06_&_07` (403 Forbidden on foreign society list, 404 Not Found on foreign event detail). |
| **AC-S14-08** | Platform role separation remains intact | PASS | Verified in `TestAuditAndAdministrationSuite/AC-S14-08` (platform `ADMIN` denied 403 Forbidden; unauthenticated denied 401 Unauthorized). |
| **AC-S14-09** | Sensitive metadata is redacted | PASS | Verified in unit tests `TestSanitizeAuditDetails_Redaction`, `TestSanitizeAuditDetails_EdgeCases` (testing case variations, unexpected keys, arbitrary types, oversized metadata, and phone masking), and integration test `TestAuditAndAdministrationSuite/AC-S14-09`. |
| **AC-S14-10** | Historical audit records remain compatible | PASS | Verified in `TestListAuditLogsEnriched_Integration` and `s14_verify` querying baseline 3,360 historical rows via dual index evaluation without data rewrite. |
| **AC-S14-11** | Existing audit immutability protections are verified | PASS | Verified in `TestAuditAndAdministrationSuite/AC-S14-11` and `TestPlatformDatabaseIntegration` (PostgreSQL trigger `platform_audit_logs_immutability_trg` rejects UPDATE and DELETE with P0001). |
| **AC-S14-12** | Audit transaction guarantees are documented and tested | PASS | Verified in `TestPlatformDatabaseIntegration/Audit_failure_rolls_back_administrative_mutation` confirming atomic transaction rollback. |
| **AC-S14-13** | Existing module audit coverage has been inventoried | PASS | Complete 59-action catalog documented in [[docs/architecture/19-audit-and-administration|Architecture 19]] covering S02 through S13 modules. |
| **AC-S14-14** | Any newly added events follow established action and transaction conventions | PASS | Verified transactional writes in `platform_audit_logs` populating `action`, `actor_id`, `society_id`, and structured `details`. |
| **AC-S14-15** | Frontend filters, pagination, details, and error states work | PASS | Implemented in `audit-viewer.svelte` (platform) and `society-audit-console.svelte` (society); verified via component and unit tests. |
| **AC-S14-16** | Society switching does not leak cached audit records | PASS | Verified in `society-audit-console.svelte` reactive `$effect` watching `societyId` to clear state and re-fetch for new tenant. |
| **AC-S14-17** | Backend quality gates pass | PASS | `gofmt`, `go vet ./...`, `go test -count=1 ./internal/...` (all packages passed), and `go build ./...` passed. |
| **AC-S14-18** | Frontend quality gates pass | PASS | `npm run check` (0 errors), `npm run lint` (0 errors/warnings), `npm test` (141/141 passed), and `npm run build` passed. |
| **AC-S14-19** | Database invariant verification passes with existing data preserved | PASS | Verified via `cmd/verify` (296 units, 163 members, 8 reasons, 0 rows deleted; count reconciliation: 3,360 baseline + 23 S14 suite + 221 full package suite tests = 3,604; all records immutably preserved). |
| **AC-S14-20** | All relevant documentation is updated | PASS | ADR 028, Architecture 19, Database Schema 14, API Contracts 15, Security 14, and README updated. |
| **AC-S14-21** | Browser acceptance passes, or its limitation is explicitly documented | BLOCKED (CDP Protocol Environment) / PASS (Live API Runner) | Automated browser execution blocked by Chrome CDP protocol error (`Browser.setDownloadBehavior unsupported`). Live HTTP API test suite `s14_verify` verified all live endpoints end-to-end. |

---

## Related Documentation
- [[../decisions/028-audit-and-administration|ADR 028: Audit and Administration Architecture]]
- [[../architecture/19-audit-and-administration|Architecture 19: Audit and Administration]]
- [[../database/14-audit-and-administration-schema|Database Schema 14: Audit and Administration Database Schema]]
- [[../api/15-audit-and-administration-api|API 15: Audit and Administration API Contracts]]
- [[../security/14-audit-and-administration-security|Security 14: Audit and Administration Security]]

---

## Status
COMPLETED (Verified)
