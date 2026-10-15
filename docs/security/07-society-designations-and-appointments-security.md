# 07 — Society Designations and Appointments Security Specification

## 1. Security Architecture & Threat Model

### 1.1 Authorization & Tenancy Boundaries
- **Endpoint Protection**: Every designation and appointment endpoint is guarded by `RequireSocietyAccess("SOC_ADMIN")`.
- **Tenant Isolation**:
  - All database queries filter strictly by `society_id`.
  - Composite foreign keys in PostgreSQL guarantee physical referential integrity across tenants:
    - `(designation_id, society_id) REFERENCES society_designations(id, society_id)`
    - `(member_id, society_id) REFERENCES society_members(id, society_id)`
  - A client cannot specify a `member_id` from Society A with an appointment in Society B.
- **Elevation of Privilege Mitigation**:
  - Holding a designation (e.g. Chairman, Treasurer) confers **zero** system authorization grants.
  - A member with designation `CHAIRMAN` cannot invoke `/api/v1/societies/{id}/*` admin endpoints unless they hold an explicit `SOC_ADMIN` row in `society_user_roles`.
- **Suspension Enforcement**:
  - When a society is in `SUSPENDED` status, `RequireSocietyAccess` blocks non-SUPER_ADMIN callers with HTTP `403 SOCIETY_SUSPENDED`.

---

## 2. Transactional Audit Trail

Every designation and appointment mutation produces a synchronous audit log in `platform_audit_logs`:

| Action | Target User / Entity | Details Payload |
| :--- | :--- | :--- |
| `DESIGNATION_CREATED` | NULL | `designation_id`, `society_id`, `code`, `name`, `allows_multiple_members` |
| `DESIGNATION_UPDATED` | NULL | `designation_id`, `society_id`, `name`, `allows_multiple_members` |
| `DESIGNATION_STATUS_CHANGED` | NULL | `designation_id`, `society_id`, `is_active` |
| `COMMITTEE_APPOINTMENT_CREATED` | NULL | `appointment_id`, `society_id`, `designation_id`, `designation_code`, `member_id`, `start_date` |
| `COMMITTEE_APPOINTMENT_ENDED` | NULL | `appointment_id`, `society_id`, `designation_id`, `member_id`, `end_date`, `notes` |
| `COMMITTEE_APPOINTMENT_REVOKED` | NULL | `appointment_id`, `society_id`, `designation_id`, `member_id`, `end_date`, `notes` |

If the audit log insert fails, the enclosing database transaction aborts and rolls back completely.

---

## 3. Related Documentation
- [[../decisions/019-society-designations-and-appointments|ADR 019: Society Designations, Committee Appointments, and Role Segregation]]
- [[../architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]
- [[../database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]
- [[../api/08-society-designations-and-appointments-api|08 — Society Designations and Appointments API Specification]]
