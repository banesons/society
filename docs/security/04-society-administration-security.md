# 04 — Society Administration Security Specification

## Overview

This specification defines the security architecture, threat model, authorization controls, and fail-closed guarantees for **Society Administration** (Sprint S04).

---

## 1. Threat Analysis & Mitigations

| Threat Vector | Severity | Vulnerability Description | Mitigation Strategy |
| :--- | :--- | :--- | :--- |
| **Unauthorized Assignment** | High | An ordinary resident or platform `ADMIN` attempts to assign themselves or others as `SOC_ADMIN` | Enforce `RequireSuperAdmin` middleware on `/api/v1/platform/societies/{id}/admins/*`. Only authenticated users holding an active `SUPER_ADMIN` platform role can assign or revoke roles. |
| **Client-Supplied Society Forgery** | Critical | An attacker sends an arbitrary `X-Society-ID` header or path parameter to access another society's resources | Go authorization middleware (`RequireSocietyRole`) treats client headers as assertions of intent, **never** authorization grants. The server independently queries `society_user_roles` for the caller's active relationship with the target society. |
| **Stale JWT Revocation Window** | High | A revoked society administrator continues making calls until their 15-minute access JWT expires | In-database active assignment validation executed on every society-scoped request. When an assignment is revoked (`status = 'REVOKED'`), authorization terminates immediately on the very next request. |
| **Cross-Society Access Leakage** | Critical | An administrator assigned to Society A attempts to read or mutate resources in Society B | Repositories explicitly bind verified `WHERE society_id = $1` derived exclusively from server-side validated context, preventing horizontal privilege escalation. |
| **Bypassing Society Suspension** | High | A society administrator attempts to perform operational actions while their society is `SUSPENDED` | Middleware explicitly validates `societies.status = 'ACTIVE'`. If the society is suspended, calls are rejected with 403 `SOCIETY_SUSPENDED`, even for legitimate administrators. |
| **Platform Privilege Escalation** | High | Assigning a user as `SOC_ADMIN` accidentally grants access to platform administration | `platform_roles` and `society_user_roles` are physically distinct tables. `SOC_ADMIN` confers zero authority on `/api/v1/platform/*`. |
| **Audit Log Tampering or Omission** | Critical | Assignment or revocation occurs without an immutable audit record | Transactional write (`tx.Begin`) binds mutation and `platform_audit_logs` insert together. Failure of audit logging aborts the mutation fail-closed. |
| **PII Exposure in Admin Rosters** | Medium | Phone numbers of society administrators exposed in public or unauthenticated endpoints | Administrator rosters require `SUPER_ADMIN` authentication. Phone numbers are canonically masked (`+91 9820xxxx56`) in responses. |

---

## 2. Authorization Rules & Role Boundary

```text
Request Path:
  /api/v1/platform/*      ──► Checked by RequireSuperAdmin()
                              (Validates platform_roles.role == 'SUPER_ADMIN')

  /api/v1/societies/{id}/* ──► Checked by RequireSocietyRole("SOC_ADMIN")
                              (Validates society_user_roles.role == 'SOC_ADMIN'
                               AND society_user_roles.status == 'ACTIVE'
                               AND societies.status == 'ACTIVE')
```

### Authorization Rules Matrix
1. **`SUPER_ADMIN`**:
   - May manage society assignments (`POST/GET/DELETE /platform/societies/{id}/admins`).
   - May inspect society administration rosters.
2. **Platform `ADMIN`**:
   - Denied access to `/platform/societies/{id}/admins`.
   - Zero implicit access to society-level operations unless explicitly assigned as a `SOC_ADMIN` in a separate `society_user_roles` record.
3. **`SOC_ADMIN`**:
   - Allowed access strictly to assigned societies where `status == 'ACTIVE'`.
   - Denied access to all `/api/v1/platform/*` endpoints.
   - Denied access to any unassigned society.
4. **Ordinary Residents**:
   - Denied administrative access across all levels.

---

## 3. Fail-Closed Guarantees

1. **Context Missing**: If neither URL path nor `X-Society-ID` provides a valid UUID, the request fails with 400 `BAD_REQUEST`.
2. **Database Failure**: If the database query checking assignment validity errors or times out, the middleware terminates the request immediately with 500 `INTERNAL_ERROR`.
3. **Suspended Society Protection**: If a society transitions to `SUSPENDED`, all operational requests fail immediately with 403 `SOCIETY_SUSPENDED`.
4. **Transactional Audit Logging**: If `INSERT INTO platform_audit_logs` fails, transaction rollback prevents assignment or revocation from persisting.

---

## Related Documentation
- [[01-authentication-security|01 — Authentication Security]]
- [[02-platform-administration-security|02 — Platform Administration Security]]
- [[03-society-security|03 — Society Security Specification]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
