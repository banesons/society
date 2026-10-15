# S13 — Documents & Records

## Objective
Establish a secure, auditable, and categorized document management lifecycle for societies: private object storage via Supabase Storage, multi-tenant RBAC (`SOCIETY_RESIDENTS`, `COMMITTEE`, `SOC_ADMIN_ONLY`), magic-byte file signature validation, append-only versioning, streaming download/preview, and archival with audit trail.

---

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| **AC-S13-01** | Authorized SOC_ADMIN can access Documents & Records section | PASS | Verified in browser acceptance session (`s13_complete_acceptance_1791879304944.webp`, `admin_document_console_1791878502153.png`); drawer navigation `#documents` routes to DocumentConsole. |
| **AC-S13-02** | Authorized SOC_ADMIN can upload a valid document | PASS | Verified in browser UI (`category_check_1791878683705.png`) creating `DOC-SOC-202610-10006` and integration test `TestDocuments_CompleteLifecycle_HappyPath`. |
| **AC-S13-03** | Unsupported file types are rejected | PASS | Verified in unit test `TestValidateFileContent` and integration test `TestDocuments_ValidationRejections` (rejects executables, HTML, text disguised as PDF). |
| **AC-S13-04** | Oversized files are rejected | PASS | Verified via 10MB file limit in `ValidateFileContent`, 12MB total request body ceiling in `http.MaxBytesReader`, and integration test `TestDocumentUploadLimits_OversizedAndMalformed`. |
| **AC-S13-05** | Document metadata is saved and displayed correctly | PASS | Verified title, document number, category badge, classification, version, size, uploader rendered in UI table and resident cards. |
| **AC-S13-06** | Documents appear in the correct category and society | PASS | Verified scoped by `society_id` and `category_id`; seeded categories present across societies. |
| **AC-S13-07** | An authorized user can download the current document version | PASS | Verified streaming download with `Content-Disposition` and `X-Content-Type-Options: nosniff` in `TestDocuments_DownloadStreams` and browser client blob trigger. |
| **AC-S13-08** | An authorized user can upload a new version | PASS | Verified version modal (`modal_after_typing_1791879362460.png`), sequential `version_number` increment, and integration test `TestDocuments_VersionUpload`. |
| **AC-S13-09** | The previous version remains available according to policy | PASS | Verified version history list in UI modal, and immutable database trigger `trg_prevent_document_version_mutation` preventing UPDATE/DELETE of version records. |
| **AC-S13-10** | A failed upload does not incorrectly change current version | PASS | Verified atomic PostgreSQL transaction in `UploadDocumentVersion` rolls back if storage upload or metadata insertion fails. |
| **AC-S13-11** | Residents can browse resident-visible documents | PASS | Verified `ResidentDocumentsView` (`resident-documents` section) displays active `SOCIETY_RESIDENTS` documents with category filters and search. |
| **AC-S13-12** | Residents cannot access committee-only documents | PASS | Verified in integration test `TestDocuments_RBAC_AccessDenial` (resident receives 403 on download and query filters them out). |
| **AC-S13-13** | Residents cannot access SOC_ADMIN-only documents | PASS | Verified in integration test `TestDocuments_RBAC_AccessDenial` (admin confidential records restricted to SOC_ADMIN). |
| **AC-S13-14** | Cross-society document and version access is rejected | PASS | Verified multi-tenant IDOR protection in `TestDocuments_IDOR_CrossSocietyDenial` (foreign society UUID returns 404 Not Found). |
| **AC-S13-15** | Direct storage object key access without auth is rejected | PASS | Supabase Storage bucket `society-documents` configured `public = false`; files accessible only through authenticated backend streaming. |
| **AC-S13-16** | Archived documents follow documented access policy | PASS | Verified archived documents hidden from resident feed; visible only in admin console when `status = 'ARCHIVED'` filter is selected. |
| **AC-S13-17** | Archive and restore actions work for authorized administrators | PASS | Verified mandatory archival reason prompt in modal and restore endpoint in `TestDocuments_ArchivalAndRestore` and browser tests. |
| **AC-S13-18** | Document mutations create expected audit events | PASS | Verified `CREATE`, `VERSION_UPLOAD`, `ARCHIVE`, `RESTORE` write structured JSON events to `platform_audit_logs`. |
| **AC-S13-19** | Loading, empty, success, and error states work correctly | PASS | Verified empty states, loading spinners, alert banners, and error banners in both Admin and Resident interfaces. |
| **AC-S13-20** | Existing society navigation and responsive layouts remain intact | PASS | Verified drawer navigation, header, formation metrics, member rosters, and gate consoles remain fully functional with 0 regressions. |
| **AC-S13-21** | Two-tier file size limits enforced (10MB selection / 5MB stored) | PASS | Verified in `TestValidateFileContent`, `TestValidateFileContent_BoundaryLimits`, `TestDocumentUploadLimits_OversizedAndMalformed`, and client form validation. |
| **AC-S13-22** | Safe type-aware optimization (PNG lossless, other formats preserved) | PASS | Verified in `TestOptimizeFileContent` (lossless `png.BestCompression`; original bytes preserved for PDF, JPEG, WebP, DOCX, XLSX). |
| **AC-S13-23** | Concurrency-safe version limit (max 20 versions per document) | PASS | Verified in `TestDocumentVersionLimitEnforcement` and `TestDocumentVersion_ConcurrencyAndIntegrity` via row lock `SELECT ... FOR UPDATE` and trigger `trg_enforce_document_version_limit`. |
| **AC-S13-24** | Storage metrics endpoint (`GET /metrics`) | PASS | Verified in `TestDocumentsAndRecordsLifecycle/Document_Storage_Metrics`; provides metadata-derived expected storage bytes. |
| **AC-S13-25** | Version 1 and Version 2 sequential upload, preservation, and retrieval | PASS (Live API) / BLOCKED (Automated Browser Tool) | Verified via live API runner (`s13_verify`) proving v1 SHA-256 (`ac07d0ddfd6a...`), v2 upload and SHA-256 (`fdbcb2aa9ce1...`), and v1 re-download hash preservation. Automated browser tool execution blocked by Chrome CDP protocol context management limitation (`Browser.setDownloadBehavior unsupported`). |

---

## Related Documentation
- [[../decisions/027-documents-and-records|ADR 027: Documents and Records Architecture]]
- [[../architecture/18-documents-and-records|Architecture 18: Documents and Records Architecture]]
- [[../database/13-documents-and-records-schema|Database Schema 13: Documents and Records Database Schema]]
- [[../api/14-documents-and-records-api|API 14: Documents and Records API Contracts]]
- [[../security/13-documents-and-records-security|Security 13: Documents and Records Security]]

---

## Status
COMPLETED (Hardened & Verified)


