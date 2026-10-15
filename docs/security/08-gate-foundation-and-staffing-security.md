# 08 — Gate Foundation and Staffing Security Specification

## 1. Security Architecture Overview
The Gate Foundation security model safeguards physical society perimeter points and duty session audit trails.

Core principles:
- **Role Isolation**: Explicit segregation between administrative access (`SOC_ADMIN`), guard terminal operations (`GATEKEEPER`), and civic committee governance (S07).
- **Tenant Containment**: All gate, shift, and duty operations strictly scoped to the caller's authorized society tenant.
- **Fail-Closed Session Locking**: Concurrency controls prevent multi-gate presence anomalies (a single guard simultaneously checked in at multiple gates).
- **Immutable Perimeter Auditing**: All guard assignments, revocations, and shift handovers produce tamper-evident audit records.

---

## 2. Authorization Matrix

| Endpoint | `SUPER_ADMIN` | `SOC_ADMIN` (Active) | `GATEKEEPER` (Active) | Member / Resident | Unassigned Admin |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `GET /societies/{id}/gates` | ✅ | ✅ | ✅ | ❌ 403 | ❌ 403 |
| `PATCH /societies/{id}/gates/{gateId}` | ✅ | ✅ | ❌ 403 | ❌ 403 | ❌ 403 |
| `GET /societies/{id}/gatekeepers` | ✅ | ✅ | ❌ 403 | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/gatekeepers` | ✅ | ✅ | ❌ 403 | ❌ 403 | ❌ 403 |
| `DELETE /societies/{id}/gatekeepers/{assignmentId}` | ✅ | ✅ | ❌ 403 | ❌ 403 | ❌ 403 |
| `GET /societies/{id}/shifts` | ✅ | ✅ | ✅ | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/shifts` | ✅ | ✅ | ❌ 403 | ❌ 403 | ❌ 403 |
| `GET /societies/{id}/duty-sessions` | ✅ | ✅ | ✅ | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/gates/{gateId}/sessions/check-in` | ✅ | ✅ | ✅ (Self) | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/gates/{gateId}/sessions/check-out` | ✅ | ✅ | ✅ (Own Active) | ❌ 403 | ❌ 403 |

---

## 3. Concurrency & Duty Session Guarantees

### 3.1 Anti-Double-Booking Constraint
- An active gate session asserts physical presence at a physical post.
- Enforced at the database level via a partial unique index:
  ```sql
  CREATE UNIQUE INDEX uq_gatekeeper_active_session 
      ON society_gate_sessions (society_id, user_id) 
      WHERE status = 'ACTIVE';
  ```
- Attempting to check in while another session is active returns HTTP 409 `SESSION_ALREADY_ACTIVE`.

### 3.2 Gatekeeper Revocation Cascades
- When a `GATEKEEPER` assignment is revoked by an administrator, the backend transactionally terminates any active duty sessions for that guard (`status = 'ENDED'`, `ended_at = NOW()`), ensuring revoked personnel cannot remain listed on active post duty.

---

## 4. Audit Logging
Every mutation generates an immutable record in `platform_audit_logs`:
- `GATE_UPDATED`: Logs changed attributes (status, directionality, headcount).
- `GATEKEEPER_ASSIGNED`: Logs target user ID and society ID.
- `GATEKEEPER_REVOKED`: Logs assignment ID and target user ID.
- `GATE_DUTY_STARTED`: Logs gate ID, guard user ID, and shift ID.
- `GATE_DUTY_ENDED`: Logs gate ID, guard user ID, and handover notes.

---

## 5. Related Documentation
- [[../decisions/020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations]]
- [[../architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]
- [[../api/09-gate-foundation-and-staffing-api|09 — Gate Foundation and Staffing API Specification]]
- [[../sprints/S08-gate-foundation|S08 — Gate Foundation Sprint]]
