# 14 — Audit and Administration Security

## 1. Threat Model & Security Perimeter

Sprint S14 governs platform and society administrative visibility, access controls, and audit integrity. The security model addresses the following core threats:

1. **Horizontal Privilege Escalation (Cross-Society Tenant Leakage)**:
   A society administrator of Society A attempts to view audit logs, member activities, or administrative operations of Society B by tampering with route parameters or query filters.
2. **Vertical Privilege Escalation**:
   Platform `ADMIN`, society `COMMITTEE` members, `GATEKEEPER` personnel, or general residents attempt to view sensitive audit trails containing administrative discussions, member sanctioning, or financial adjustments.
3. **Audit Log Tampering / Repudiation**:
   A rogue administrative user attempts to update, falsify, or delete audit entries to cover illicit actions or bypass accountability.
4. **Credential & Sensitive Data Leakage via Metadata**:
   Audit entries capturing request bodies or operational context could inadvertently log passwords, JWT tokens, OTPs, auth headers, private file URLs, or raw PII.
5. **Cross-Tenant ID Enumeration**:
   Querying audit detail endpoints with arbitrary UUIDs could allow attackers to infer the existence of events across other societies if `403` vs `404` is improperly handled.

---

## 2. Defensive Controls & Mitigation Strategy

### 2.1 Multi-Tenant Isolation & Society Scoping
- **Strict Tenant Queries**: All society-scoped queries enforce tenant boundaries at the database layer:
  ```sql
  WHERE (society_id = $1 OR details->>'society_id' = $1::text)
  ```
  Client-supplied `society_id` parameters are ignored; the server binds the authorized URL society ID validated by the session.
- **Uniform 404 on Foreign Events**: When querying a single event detail (`/api/v1/societies/{id}/audit-logs/{event_id}`), if the event exists in the database but does not belong to `{id}`, the API returns `404 Not Found` rather than `403 Forbidden`, eliminating tenant cross-enumeration vectors.

### 2.2 Role & Permission Matrix
| Role | Platform Audit (`/platform/audit-logs`) | Society Audit (`/societies/{id}/audit-logs`) |
| :--- | :--- | :--- |
| **`SUPER_ADMIN`** | **Permitted** (Full global audit visibility) | **Permitted** (Can inspect any society) |
| **`SOC_ADMIN`** | **Deny (403)** | **Permitted** (Assigned active societies ONLY) |
| **`ADMIN`** (Platform) | **Deny (403)** | **Deny (403)** |
| **`COMMITTEE`** | **Deny (403)** | **Deny (403)** |
| **`GATEKEEPER`** | **Deny (403)** | **Deny (403)** |
| **`RESIDENT`** | **Deny (403)** | **Deny (403)** |
| **Unauthenticated** | **Deny (401)** | **Deny (401)** |

> [!IMPORTANT]
> In accordance with [[docs/decisions/012-platform-role-separation|ADR 012]], platform `ADMIN` role is strictly separated and does NOT inherit automatic society-level audit access.

---

### 2.3 Cryptographic Immutability & Database Trigger Protection
- The underlying `platform_audit_logs` table has a native PostgreSQL row-level trigger that intercepts all `UPDATE` and `DELETE` commands:
  ```sql
  CREATE TRIGGER platform_audit_logs_immutability_trg
  BEFORE UPDATE OR DELETE ON platform_audit_logs
  FOR EACH ROW EXECUTE FUNCTION trg_enforce_platform_audit_logs_immutability();
  ```
- Any mutation attempt immediately raises an uncatchable exception: `P0001: platform_audit_logs is append-only: updates and deletes are prohibited`.
- The Go backend provides zero mutation or deletion routes for audit logs.

---

### 2.4 Recursive Metadata Redaction Engine
Before any audit event metadata is returned to API callers (platform or society), it passes through a recursive redaction engine in Go:

#### Denied Keys (Case-Insensitive Exact and Substring Match)
- `password`, `passwd`, `secret`, `token`, `access_token`, `refresh_token`, `otp`, `pin`, `auth_header`, `authorization`, `service_role_key`, `session_secret`, `signed_url`, `private_key`.

#### Redaction Rules
1. Any JSON object key matching a sensitive pattern has its value replaced with `"[REDACTED]"`.
2. Phone numbers in actor, target, or details fields are masked:
   - e.g., `+919876543210` -> `+91 98*** ***10`.
3. Nested JSON objects and JSON arrays are traversed recursively without altering or mutating database state.
4. Response serialization explicitly strips unselected internal columns.
5. Errors in JSON decoding or redaction default to returning an empty object `{}` rather than leaking raw unredacted bytes.

---

### 2.5 Audit Transactional Guarantees
- For all critical state changes (Society creation, Member departures, Committee appointments, Payment status transitions, Document version creation), the audit log write is executed within the same database transaction (`*sql.Tx`) as the business mutation.
- If the audit insert fails (e.g. invalid foreign key or database constraint), the transaction rolls back completely, preventing silent undocumented state mutations.
