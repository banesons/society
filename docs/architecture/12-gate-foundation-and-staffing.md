# 12 — Gate Foundation and Staffing Architecture

## 1. Architectural Overview

The Gate Foundation subsystem establishes perimeter access management and security guard accountability for housing societies.

It bridges:
- **Physical Layout (S05)**: Buildings, wings, units, and entry/exit gates (`society_gates`).
- **Identity & Authentication (S01/S04)**: Unified phone OTP authentication (`users`) and scoped tenant roles (`society_user_roles`).
- **Civic Governance (S07)**: Managing committee designations and oversight (`society_committee_appointments`).
- **Visitor Operations (S09 - Future)**: Visitor entry logging, digital passes, and resident approvals.

```text
Platform Layer:
  platform_roles (SUPER_ADMIN, ADMIN)
        ↓
Society Governance Boundary:
  society_user_roles (SOC_ADMIN) ───[Configures]───→ Gates, Shifts & Roster
        │
        ├─► society_user_roles (GATEKEEPER) ───[Checks In]───→ Active Duty Session
        │                                                           │
        │                                                           ▼
Physical Perimeter:                                            society_gate_sessions
  society_gates (Main Gate, Service Gate, Rear Gate...)      (Stationed Guard On Duty)
```

---

## 2. Key Architectural Tenets

### 2.1 Unified Identity Without Synthetic Logins
- Guards and gatekeepers authenticate via standard mobile OTP through the core `auth` service.
- Operational permissions are scoped per society using `role = 'GATEKEEPER'` in `society_user_roles`.
- Zero separate gatekeeper credential vaults or shared device passwords.

### 2.2 Role Segregation & Principle of Least Privilege
- `SOC_ADMIN`: Can configure gates, create/edit shift definitions, assign/revoke `GATEKEEPER` personnel, and view staffing rosters and duty audit trails.
- `GATEKEEPER`: Can query gate lists for their assigned society, view active shift status, and check in / check out of duty sessions. Cannot modify society configuration, member rosters, or platform settings.
- `MEMBER`: Cannot access gate management or gatekeeper terminal operations.
- `SUPER_ADMIN`: Possesses platform override for society inspection and administrative support.

### 2.3 Operational Gate Lifecycle & Directionality
- Physical gates maintain two orthogonal states:
  - `status`: Physical availability (`ACTIVE`, `INACTIVE`).
  - `operational_status`: Dynamic operational posture (`ACTIVE`, `INACTIVE`, `MAINTENANCE`, `EMERGENCY_OPEN`).
- Directionality governs vehicular/pedestrian routing:
  - `BIDIRECTIONAL`: Both entry and exit supported.
  - `INGRESS_ONLY`: Dedicated entrance point (e.g., Gate 1 Inflow).
  - `EGRESS_ONLY`: Dedicated exit point (e.g., Gate 2 Outflow).

### 2.4 Staffing Accountability & Duty Sessions
- Societies define operational shifts (`society_gate_shifts`) with planned start and end times.
- Before logging visitor movements (S09), a gatekeeper must initiate an active duty session (`society_gate_sessions`) at a specific gate.
- Concurrency Guard: A gatekeeper cannot hold active duty sessions across multiple gates simultaneously. Starting a new session automatically closes or rejects conflicting concurrent sessions for that guard.
- Handover and checkout cleanly close the session with an ending timestamp and handover notes.

### 2.5 Multi-Tenant Isolation & Lifecycle Enforcement
- Multi-tenancy enforced by `society_id` scoping on all queries and composite foreign key constraints.
- Societies in `PENDING` or `SUSPENDED` states reject gate operational actions with HTTP 403 `SOCIETY_PENDING` / `SOCIETY_SUSPENDED`.
- All gate configuration edits, gatekeeper assignments/revocations, and session check-ins/check-outs write atomic records to `platform_audit_logs`.

---

## 3. Data Flow

```text
1. Society Admin Roster Management:
   SOC_ADMIN ──► POST /api/v1/societies/{id}/gatekeepers
                 (Assigns user phone as GATEKEEPER in society_user_roles + writes audit log)

2. Gate Operational Configuration:
   SOC_ADMIN ──► PATCH /api/v1/societies/{id}/gates/{gateId}
                 (Updates name, directionality, operational status, staff requirement + writes audit log)

3. Shift Schedule Management:
   SOC_ADMIN ──► POST /api/v1/societies/{id}/shifts
                 (Defines shift name, start time, end time)

4. Gatekeeper Shift Duty:
   GATEKEEPER ──► POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-in
                  (Locks guard session record, verifies gate is ACTIVE, creates active session)
   GATEKEEPER ──► POST /api/v1/societies/{id}/gates/{gateId}/sessions/check-out
                  (Sets ended_at, status = 'ENDED', records handover notes)
```

---

## 4. Related Documentation
- [[../decisions/020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]
- [[../api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API Specification]]
- [[../security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security Specification]]
- [[../sprints/S08-gate-foundation|S08 — Gate Foundation Sprint]]
