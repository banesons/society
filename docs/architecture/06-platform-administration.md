# 06 — Platform Administration Architecture

## Overview

Sprint S02 establishes platform-level governance on top of the S01 identity and authentication layer. It introduces administrative roles (`SUPER_ADMIN` and `ADMIN`), administrative account lifecycle management, server-side authorization enforcement, and an immutable audit trail.

```text
┌─────────────────────────────────────────────────────────────┐
│                    SvelteKit Frontend                       │
│    (Super Admin Dashboard ──► Admin Management & Audits)    │
└──────────────┬───────────────────────────────▲──────────────┘
               │                               │
       Bearer JWT (Sub, SessionID)             │ Admin List / Status
       POST /platform/admins                   │ Audit History
       PATCH /platform/admins/{id}/status      │
               │                               │
               ▼                               │
┌──────────────────────────────────────────────┴──────────────┐
│                      Go Backend                             │
│                                                             │
│   [Authenticate Middleware]  ──► Validates S01 Ed25519 JWT  │
│                                                             │
│   [RequirePlatformRole]      ──► Validates Active Role:     │
│                                  • SUPER_ADMIN              │
│                                  • ADMIN                    │
│                                                             │
│   [Platform Admin Service]   ──► JIT User Provisioning,     │
│                                  Role Lifecycle & Status    │
│                                                             │
│   [Audit Service]            ──► Immutable Audit Logger     │
└──────────────┬──────────────────────────────────────────────┘
               │ SQL Queries (pgxpool)
               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Supabase PostgreSQL Cloud                   │
│   users (S01) │ platform_roles (S02) │ platform_audit (S02) │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Components

### 1. Platform Roles Service (`internal/platform/service.go`)
- **Super Admin Bootstrap**: On startup, idempotently uses existing `NormalizePhone` on `SUPER_ADMIN_PHONE`. If no platform role exists, assigns `SUPER_ADMIN`. Never silently overwrites an existing role.
- **Create Admin**: Only callable by an active `SUPER_ADMIN`.
  - Normalizes target mobile number using canonical `NormalizePhone`.
  - Provisions or finds user record in `users`.
  - Inserts role record (`role = 'ADMIN'`, `status = 'ACTIVE'`, `created_by = actor.ID`).
  - Transactionally records audit log entry (`ADMIN_CREATED`).
- **Update Admin Status**: Only callable by an active `SUPER_ADMIN`.
  - Toggles status between `ACTIVE` and `DISABLED`.
  - When disabled:
    - Sets `platform_roles.status = 'DISABLED'`.
    - Revokes all active sessions in `user_sessions` for the target admin (`revoked_at = NOW()`), immediately terminating current logins.
    - Preserves `users.status` as `'ACTIVE'` (identity remains valid for normal resident access).
  - Transactionally records audit log entry (`ADMIN_DISABLED` / `ADMIN_REACTIVATED`). Entire transaction rolls back if audit log fails.
- **List Admins**: Paginated query returning administrators with canonical masked phone, role, status, creation timestamp, and creator identity.
- **Get Current Platform Roles**: Helper to fetch active platform roles for `/auth/me` and authorization checks.

### 2. Authorization Middleware (`internal/middleware/platform_auth.go`)
- Chains onto `middleware.Authenticate` from S01.
- `RequireSuperAdmin()`:
  - Extracts `user_id` from request context.
  - Checks database/cache for an active `SUPER_ADMIN` entry in `platform_roles` where `status = 'ACTIVE'`.
  - Rejects with `403 Forbidden` (`FORBIDDEN`) if condition is not met.
- `RequirePlatformRole(roles ...string)`:
  - Flexible guard accepting any of the specified platform roles.
  - Prevents disabled administrators (`status = 'DISABLED'`) from executing administrative operations.
  - Request parameters or client headers cannot elevate roles.

### 3. Immediate Session Termination on Admin Deactivation
When a Super Admin disables an Admin, the following occurs within a single transaction:
1. `platform_roles.status` is set to `'DISABLED'`.
2. All records in `user_sessions` where `user_id = target_user_id AND revoked_at IS NULL` are set to `revoked_at = NOW()`.
3. `users.status` remains `'ACTIVE'`.
4. An immutable audit record is inserted into `platform_audit_logs`.
5. Even if the disabled admin possesses a valid 15-minute access token, the authorization middleware queries database status on administrative routes, blocking requests immediately.

### 4. Immutable Audit Logger (`internal/platform/audit.go`)
- Captures all administrative events into `platform_audit_logs`.
- Prevents tampering: no update or delete routes exist.

---

## Future Society-Assignment Integration (S03+)

Platform `ADMIN` accounts are designed to be mapped to one or more housing societies:
```text
Platform ADMIN (S02) ──► society_admin_mappings (S04) ──► Society (S03)
```
In S02, no society tables or columns are introduced. The `ADMIN` role is established as a clean platform identity ready for multi-tenant assignment in subsequent sprints.

---

## Related Documentation
- [[../decisions/012-platform-role-separation|ADR 012: Platform Role Separation]]
- [[../decisions/013-super-admin-bootstrap|ADR 013: Super Admin Bootstrap]]
- [[../decisions/014-platform-audit-trail|ADR 014: Platform Audit Trail]]
- [[../api/03-platform-administration-api|Platform Administration API Specification]]
- [[../database/02-platform-administration-schema|Platform Administration Schema]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
