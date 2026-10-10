# 03 — Society Security Specification

## Overview

Housing societies represent the fundamental tenant boundary for the CHS platform. The security of society onboarding directly affects multi-tenant integrity, privilege isolation, and auditability.

---

## 1. Threat Analysis & Mitigations

| Threat Vector | Severity | Vulnerability Description | Mitigation Strategy |
| :--- | :--- | :--- | :--- |
| **Unauthorized Creation** | High | An ordinary resident or platform `ADMIN` attempts to register a society | Enforce Go `RequireSuperAdmin` middleware. Reject all requests lacking active `SUPER_ADMIN` platform role with 403 `FORBIDDEN`. |
| **Privilege Escalation** | High | Registering a society attempts to grant the creator local society admin or extra platform roles | S03 records `created_by` for provenance only. No roles or tenant permissions are automatically granted upon society creation. |
| **Cross-Tenant Data Leakage** | Critical | Future endpoints allow an authenticated user from Society A to view or mutate Society B data | 1. Go application-level tenant isolation.<br>2. Tenant context derived solely from validated server-side relationships.<br>3. Repositories explicitly bind `WHERE society_id = $1`. |
| **Duplicate Society Creation** | Medium | Accidental or malicious duplication of existing societies | 1. Unique uppercase B-tree index on `society_code`.<br>2. Bounded collision retry logic in Go backend. |
| **Invalid Lifecycle Bypass** | High | Bypassing `PENDING` state directly to `ACTIVE` or suspending unverified societies | Deterministic state machine in `SocietyService`. Reject invalid transitions with 400 `INVALID_STATUS_TRANSITION`. |
| **Audit Log Omission** | High | Society mutations occur without corresponding audit records | All society creation and status mutations execute within a PostgreSQL database transaction (`tx.Begin`). Mutation and audit write commit together; audit failure rolls back the mutation. |
| **Audit Log Tampering** | Critical | An actor attempts to alter or erase society creation/suspension records | Database trigger `platform_audit_logs_immutability_trg` blocks `UPDATE` and `DELETE` on `platform_audit_logs`. |
| **ID Enumeration & Scraping** | Medium | Attackers sequentially enumerate society identifiers | Societies use non-sequential UUID primary keys (`gen_random_uuid()`). Public APIs require authentication and `SUPER_ADMIN` authorization. |
| **PII Exposure** | Medium | Contact phone numbers or emails exposed unnecessarily | Contact phone numbers are masked using canonical phone masking (`+91 9820xxxx56`) on list endpoints. Only authorized Super Admin detail views receive full administrative contact information. |
| **Mass Assignment** | Medium | Client payload attempts to override `id`, `status`, `created_by`, or timestamps during profile edit | Go handlers strictly bind to dedicated DTOs (`UpdateSocietyRequest`) containing only explicitly allowed profile fields. `status` can only be changed via dedicated lifecycle endpoints (`/activate`, `/suspend`). |
| **Concurrent Mutation Race** | Low | Simultaneous status changes cause inconsistent audit entries | Status mutations execute with row-level locks (`SELECT ... FOR UPDATE`) or transactional conditional updates. |
| **Society Code Manipulation** | High | Client attempts to control, spoof, or modify generated society codes | Backend derives code deterministically in `SOC-{CITY_CODE}-{RANDOM_SUFFIX}` format with cryptographic random suffix; client-supplied codes are ignored; database trigger `societies_code_immutability_trg` blocks `UPDATE` on `society_code`. |
| **Suspension Reason Forgery & Origin Spoofing** | High | Client submits arbitrary free-text or unapproved reason identifier, or spoofs AUTOMATED origin | Backend resolves `reason_id` or `reason_code` against active database catalogue records where `allows_manual = true`; validates mandatory details for `OTHER_REVIEW_REQUIRED`; backend unconditionally sets `suspension_origin = "MANUAL"` for API callers and ignores any client-supplied origin or actor ID. |

---

## 2. Authorization Rules & Role Boundary

1. **`SUPER_ADMIN` (Platform Governance)**:
   - Permitted: Register societies, inspect all societies, update profiles, activate, suspend, reactivate.
2. **`ADMIN` (Platform Operator)**:
   - Denied: By default, platform `ADMIN` cannot register or alter societies in S03. Receives 403 `FORBIDDEN`.
3. **`SOC_ADMIN` & Future Society Roles**:
   - Denied: Society-level roles have zero platform-level governance authority.
4. **Ordinary Residents**:
   - Denied: Receives 403 `FORBIDDEN`.

---

## 3. Fail-Closed Guarantees

In alignment with S01 and S02 architectural principles:
- **Audit Failure**: If `INSERT INTO platform_audit_logs` fails for any reason during society creation, activation, or suspension, the transaction rolls back immediately and returns HTTP 500 `INTERNAL_ERROR`.
- **Database Error**: If society lookup or validation cannot be completed, the operation fails closed.

---

## Related Documentation
- [[01-authentication-security|01 — Authentication Security]]
- [[02-platform-administration-security|02 — Platform Administration Security]]
- [[../architecture/07-society-onboarding|07 — Society Onboarding Architecture]]
- [[../database/03-society-schema|03 — Society Database Schema]]
- [[../api/04-society-api|04 — Society API Specification]]
- [[../decisions/015-society-lifecycle-and-tenancy|ADR 015: Society Lifecycle and Tenancy Boundary]]
