# ADR 028: Audit and Administration Architecture

## Status
ACCEPTED

## Date
2026-10-14

## Context
Cooperative Housing Societies require end-to-end transparency, regulatory compliance, and non-repudiation for administrative actions. CHS currently logs high-impact events into `platform_audit_logs`. However:
1. Platform administrators at `/admin` have only basic action-filtering without date ranges, actor search, target lookup, society filtering, or enriched event inspection.
2. Society administrators (`SOC_ADMIN`) have no scoped audit interface to view operational activity (member registrations, committee appointments, gate duty sessions, visitor approvals, announcements, maintenance billing/collections, and document lifecycles) for their assigned society.
3. Arbitrary JSONB event metadata poses a potential security risk if sensitive keys (e.g., credentials, tokens, session secrets, or full PII) were ever leaked into API responses.
4. Historical records (3,360+ rows) already exist in `platform_audit_logs`, with 3,200+ containing `society_id` embedded within their `details` JSONB document.
5. The audit ledger is protected by an active PostgreSQL trigger (`platform_audit_logs_immutability_trg`) that forbids `UPDATE` and `DELETE` operations.

## Decisions

### 1. Unified Ledger with Hybrid Scoping
- **Single Source of Truth**: Retain and extend the existing `platform_audit_logs` table rather than introducing a separate, duplicate society audit table.
- **Forward-Only Schema Extension**:
  - Add an optional, indexed `society_id UUID REFERENCES societies(id) ON DELETE SET NULL` column to `platform_audit_logs` for direct relational association on all new writes.
  - Retain compatibility with all 3,360+ historical records without updating them (which would violate the immutability trigger or require artificial backfills).
  - Queries for society-scoped events utilize a unified predicate:
    ```sql
    WHERE (society_id = $1 OR details->>'society_id' = $1::text)
    ```
  - Index both paths with B-tree indexes: `platform_audit_logs(society_id, created_at DESC)` and functional index `platform_audit_logs((details->>'society_id'), created_at DESC)`.

### 2. Role-Based Access and Tenant Isolation
- **Platform Audit Access**:
  - `SUPER_ADMIN`: Full visibility across all platform and society events. Can filter by society, actor, action, date range, or free-text query.
  - `ADMIN`: Follows existing documented platform-role policies; cannot access society-specific audit logs unless assigned.
- **Society Audit Access**:
  - `SOC_ADMIN`: Strictly scoped to the society for which they hold an active assignment in `society_user_roles`. Multi-tenant boundaries are enforced in the database query. Manipulating the society ID URL parameter returns `403 Forbidden` or `404 Not Found`.
  - `COMMITTEE`, `GATEKEEPER`, `RESIDENT`: Denied audit access by default (`403 Forbidden`).
  - Society administrators are restricted from viewing platform-internal actions (e.g., platform administrator provisioning or platform-wide role mutations).

### 3. Server-Side Recursive Metadata Redaction Engine
- Prior to serializing audit records in API responses, the backend runs a recursive redaction engine on the `details` JSONB document.
- **Redaction Rules**:
  - Case-insensitive key matching for sensitive terms: `token`, `password`, `secret`, `otp`, `jwt`, `private_key`, `api_key`, `apikey`, `authorization`, `cookie`, `credential`, `card`, `cvv`, `pan`, `account_number`, `signed_url`.
  - Values matching sensitive keys are replaced with `"[REDACTED]"`.
  - Phone numbers are formatted and masked via `auth.MaskPhone`.
  - Deep traversal handles nested JSON maps and arrays.
  - Bounded string sizes prevent memory denial-of-service from maliciously large historical payloads.

### 4. Deterministic Server-Side Pagination & Filtering
- Pagination uses deterministic ordering: `ORDER BY created_at DESC, id DESC`.
- Page size is bounded (`limit` default 50, maximum 200).
- Supported server-side filters:
  - `action`: Exact action filter or canonical category.
  - `actor_id`: UUID of the performing administrator.
  - `target_user_id`: UUID of the affected user.
  - `start_date` and `end_date`: ISO/RFC3339 timestamps filtered via PostgreSQL `TIMESTAMPTZ`.
  - `search`: Case-insensitive substring search matching action code, details text, actor phone, or client IP.

### 5. Immutability & Trust Boundaries
- Application code provides **zero update or delete endpoints** for audit logs.
- The PostgreSQL trigger `platform_audit_logs_immutability_trg` rejects any application-level `UPDATE` or `DELETE` with `P0001: platform_audit_logs is append-only`.
- Acknowledged trust boundary: Database superusers with direct SQL console access could alter triggers or data. For application-tier actors, the ledger is strictly append-only.

## Consequences

### Positive
- Zero data loss or modification of existing 3,360+ audit records.
- Standardized, enriched event contract across both platform and society tiers.
- Defense-in-depth through server-side recursive metadata redaction.
- Sub-millisecond indexed query performance for society and platform filters.
- Seamless UI experience matching the Swiss editorial design language.

### Negative / Trade-offs
- Society-scoped queries must check both `society_id` and `details->>'society_id'` to seamlessly incorporate historical records. Dual indexing eliminates the query performance cost.

## Related Documentation
- [[../architecture/19-audit-and-administration|Architecture 19: Audit & Administration]]
- [[../database/14-audit-and-administration-schema|Database Schema 14: Audit & Administration Schema]]
- [[../api/15-audit-and-administration-api|API 15: Audit & Administration API Contracts]]
- [[../security/14-audit-and-administration-security|Security 14: Audit & Administration Security]]
- [[../sprints/S14-audit-and-administration|Sprint S14 Tracker]]
