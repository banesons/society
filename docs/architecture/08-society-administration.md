# 08 — Society Administration Architecture

## Overview

Sprint S04 introduces **Society Administration**, establishing tenant-level governance within individual housing societies. It provides the mechanism for `SUPER_ADMIN` to map users as `SOC_ADMIN` to specific societies, enforces strict multi-tenant isolation, supports multi-society administration, and guarantees immediate revocation semantics.

---

## 1. System Topology & Request Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        Super Admin Governance                          │
│                                                                        │
│   POST /api/v1/platform/societies/{id}/admins                          │
│   DELETE /api/v1/platform/societies/{id}/admins/{assignment_id}        │
│   GET /api/v1/platform/societies/{id}/admins                           │
│                     │                                                  │
│                     ▼ [RequireSuperAdmin Middleware]                   │
│         ┌───────────────────────┐                                      │
│         │ Platform Admin Engine │                                      │
│         └───────────┬───────────┘                                      │
└─────────────────────┼──────────────────────────────────────────────────┘
                      │ Mutates & Logs Transactionally
                      ▼
┌────────────────────────────────────────────────────────────────────────┐
│               PostgreSQL Database (Supabase Cloud)                     │
│                                                                        │
│   ┌─────────────────────┐       ┌──────────────────────┐               │
│   │      societies      │       │        users         │               │
│   └──────────┬──────────┘       └──────────┬───────────┘               │
│              │ 1..N                        │ 1..N                      │
│              ▼                             ▼                           │
│   ┌────────────────────────────────────────────────────┐               │
│   │                society_user_roles                  │               │
│   │  (society_id, user_id, role='SOC_ADMIN', status)   │               │
│   └────────────────────────────────────────────────────┘               │
│                               ▲                                        │
│                               │ Validates on Every Request             │
└───────────────────────────────┼────────────────────────────────────────┘
                                │
┌───────────────────────────────┴────────────────────────────────────────┐
│                    Society Admin Operational Traffic                   │
│                                                                        │
│   GET /api/v1/societies/{id}/profile                                   │
│   X-Society-ID: <society_id>                                           │
│                     │                                                  │
│                     ▼ [Authenticate Middleware (JWT Valid)]            │
│                     │                                                  │
│                     ▼ [RequireSocietyRole("SOC_ADMIN")]                │
│         ┌───────────────────────────────────────┐                      │
│         │  1. Check society status == 'ACTIVE'  │                      │
│         │  2. Check society_user_roles ACTIVE   │                      │
│         │  3. Bind verified society_id to ctx   │                      │
│         └───────────────────────────────────────┘                      │
│                     │                                                  │
│                     ▼                                                  │
│         ┌────────────────────────┐                                     │
│         │ Society Scoped Handler │                                     │
│         └────────────────────────┘                                     │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Role Boundary & Privilege Separation

In alignment with [[../decisions/012-platform-role-separation|ADR 012]] and [[../decisions/016-society-role-and-assignment-model|ADR 016]]:

| Capability | `SUPER_ADMIN` | Platform `ADMIN` | `SOC_ADMIN` | Ordinary User |
| :--- | :---: | :---: | :---: | :---: |
| Platform Infrastructure & Audits | Yes | Read Audits | No | No |
| Create/Manage Platform Admins | Yes | No | No | No |
| Onboard/Activate/Suspend Societies | Yes | No | No | No |
| Assign/Revoke `SOC_ADMIN` Roles | **Yes** | **No** | **No** | **No** |
| Access Assigned Society Operations | Optional (Gov) | No | **Yes** | No |
| Cross-Society Access | Global | None | **Assigned Only** | Assigned Only |
| Platform Role Alteration | Yes | No | **No** | No |

### Guarantees
1. **Zero Automatic Elevation**: An active platform `ADMIN` receives zero implicit access to any society tenant.
2. **Platform Role Invariance**: Assigning a user as a `SOC_ADMIN` leaves `platform_roles` untouched.
3. **Tenant Containment**: A `SOC_ADMIN` has zero access to platform management endpoints (`/api/v1/platform/*`).

---

## 3. Assignment Lifecycle & Concurrency Model

```text
        Assign User to Society
         (SUPER_ADMIN Only)
                 │
                 ▼
          ┌─────────────┐
          │   ACTIVE    │  ◄── Allowed operational access if society is ACTIVE
          └──────┬──────┘
                 │
                 │ Revoke Assignment (SUPER_ADMIN)
                 ▼
          ┌─────────────┐
          │   REVOKED   │  ◄── Instantaneous denial; immutable audit record
          └─────────────┘
```

### Lifecycle Invariants
1. **Idempotent / Conflict-Free Assignment**:
   - Evaluated under a partial unique index `(society_id, user_id, role) WHERE status = 'ACTIVE'`.
   - If an active assignment already exists for `(society_id, user_id, 'SOC_ADMIN')`, the assignment request is rejected with HTTP 409 `ALREADY_ASSIGNED`.
   - If a previous assignment for that pair was `REVOKED`, a new active assignment record is created with a new UUID and timestamp.
2. **Immediate Revocation**:
   - Revocation marks `status = 'REVOKED'`, `revoked_by = <actor_id>`, `revoked_at = NOW()`.
   - Subsequent calls from that user for that society are immediately denied.

---

## 4. Multi-Society Administration & Active Context Selection

A user may administer multiple independent housing societies (`1..N`).

### Flow
1. **Discovery**:
   - When the user logs in, `GET /api/v1/auth/me` returns their active assignments:
     ```json
     {
       "user": { ... },
       "platform_roles": [],
       "society_assignments": [
         {
           "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
           "society_code": "SOC-MUM-7K4P2X",
           "society_name": "Palm Grove CHS",
           "role": "SOC_ADMIN"
         },
         {
           "society_id": "9d12a884-3889-4bc2-84cf-e5e6834ff89a",
           "society_code": "SOC-PUN-9D3R8A",
           "society_name": "Silver Oak Residency",
           "role": "SOC_ADMIN"
         }
       ]
     }
     ```
2. **Context Selection**:
   - In the frontend, the user selects their active working society via a top-bar **Society Switcher**.
   - The selected society ID is stored in the frontend society context store.
3. **Request Dispatch**:
   - Frontend transmits the target society either in URL path `/api/v1/societies/{society_id}/...` or header `X-Society-ID: {society_id}`.
4. **Server Validation**:
   - The server middleware extracts the requested `society_id`.
   - It performs a database verification:
     ```sql
     SELECT sur.role, s.status
     FROM society_user_roles sur
     JOIN societies s ON s.id = sur.society_id
     WHERE sur.society_id = $1
       AND sur.user_id = $2
       AND sur.status = 'ACTIVE';
     ```
   - If no active assignment exists: HTTP 403 `SOCIETY_ACCESS_DENIED`.
   - If society is suspended: HTTP 403 `SOCIETY_SUSPENDED`.
   - If society is pending: HTTP 403 `SOCIETY_PENDING`.
   - If valid: sets `verified_society_id` and `society_role` in `r.Context()`.

---

## 5. Interaction with Society Lifecycle States

The interaction between society tenant states and administrator assignments is governed by strict rules:

| Society State | Can SUPER_ADMIN Assign/Revoke Admins? | Can SOC_ADMIN Access Operational APIs? |
| :--- | :---: | :---: |
| **`PENDING`** | **Yes** (pre-configuration during onboarding) | **No** (rejected with 403 `SOCIETY_PENDING`) |
| **`ACTIVE`** | **Yes** | **Yes** |
| **`SUSPENDED`** | **Yes** (governance/handover during hold) | **No** (rejected with 403 `SOCIETY_SUSPENDED`) |

### Suspension vs Revocation Decoupling
- **Suspending a Society**: An operational lock applied to the entire tenant. Does **not** mutate or revoke `society_user_roles`. When the society is reactivated, the administrators automatically regain access.
- **Revoking an Assignment**: A specific action targeting a single user's privilege for a society.

---

## 6. Audit Trail & Transaction Guarantees

Every assignment and revocation runs inside a database transaction:
1. Validate society existence and user existence.
2. Validate caller holds active `SUPER_ADMIN` role.
3. Apply `society_user_roles` insert or update.
4. Insert into `platform_audit_logs`.
5. Commit transaction.

If audit insertion fails, the entire transaction rolls back fail-closed.

---

## 7. Frontend Architecture

### Super Admin Roster (`/admin` Console)
- Within Society Details dialog or dedicated Societies tab:
  - Administrator sub-section showing assigned admins.
  - "Assign Administrator" modal: phone search, user confirmation, assignment.
  - "Revoke Administrator" action with confirmation dialog.

### Society Admin Console Shell (`/society` or `/app`)
- Navigation header with **Society Switcher** dropdown.
- Dedicated society context indicator showing society code, name, and admin status badge.
- Society profile inspection view for assigned administrators.

---

## Related Documentation
- [[07-society-onboarding|07 — Society Onboarding Architecture]]
- [[../database/04-society-administration-schema|04 — Society Administration Database Schema]]
- [[../api/05-society-administration-api|05 — Society Administration API Specification]]
- [[../security/04-society-administration-security|04 — Society Administration Security Specification]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
