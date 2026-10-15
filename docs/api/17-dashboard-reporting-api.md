# API 17: Dashboard & Reporting API Contracts

## 1. Overview
Dashboard and reporting endpoints provide read-only aggregated operational, financial, and security intelligence. All calculations enforce tenant isolation, data minimization, integer monetary representations (paise), and Indian Standard Time (`Asia/Kolkata`, UTC+05:30) date boundaries.

Endpoints:
- `GET /api/v1/platform/dashboard/summary` (Platform Admin overview)
- `GET /api/v1/societies/{id}/dashboard/summary` (Society operations & finances)
- `GET /api/v1/societies/{id}/gate/dashboard` (Gate shift operations & visitor flow)

---

## 2. Endpoints

### 2.1 Platform Dashboard Summary
Retrieves platform-wide ecosystem metrics across societies, accounts, transaction velocity, and audit activity.

- **Method**: `GET`
- **Path**: `/api/v1/platform/dashboard/summary`
- **Authentication**: Bearer JWT (`Authorization: Bearer <token>`)
- **Authorization**: `SUPER_ADMIN` or `ADMIN`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "societies": {
      "total": 12,
      "active": 10,
      "pending": 1,
      "suspended": 1
    },
    "new_societies_30d": 3,
    "total_registered_units": 1420,
    "members": {
      "total_active_accounts": 980
    },
    "financial_velocity_30d": {
      "amount_paise": 48500000,
      "formatted_inr": "₹4,85,000.00"
    },
    "platform_audit_events_24h": 348
  }
}
```
- **Errors**:
  - `401 UNAUTHORIZED`: Missing or invalid token.
  - `403 FORBIDDEN`: Insufficient platform administrative privileges.

---

### 2.2 Society Dashboard Summary
Retrieves society-specific operational, occupancy, gate, financial, and announcement summaries.

- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/dashboard/summary`
- **Authentication**: Bearer JWT (`Authorization: Bearer <token>`)
- **Optional Header**: `X-Society-ID: <uuid>` (Must match canonical path `{id}` if provided)
- **Authorization**: `SOC_ADMIN` (or platform `SUPER_ADMIN`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "society_id": "893c528f-7ee3-463d-82d2-8d76a5ebca57",
    "occupancy": {
      "total_units": 120,
      "occupied_units": 108,
      "vacant_units": 12,
      "occupancy_rate_percent": 90.0,
      "owner_occupied_units": 82,
      "tenant_occupied_units": 26
    },
    "members": {
      "total_active_members": 240
    },
    "gate": {
      "visitors_on_premises": 4,
      "today_entries": 32,
      "today_exits": 28,
      "pending_approvals": 1
    },
    "finance": {
      "current_period_id": "b3e34346-6bf4-4f04-9457-3a1ec4d5743b",
      "current_period_name": "October 2026",
      "billed_paise": 60000000,
      "collected_paise": 51000000,
      "collection_efficiency_percent": 85.0,
      "overdue_arrears_paise": 9000000,
      "payments_collected_mtd_paise": 51000000
    },
    "communication": {
      "active_announcements_count": 2
    }
  }
}
```
- **Errors**:
  - `401 UNAUTHORIZED`: Missing or invalid token.
  - `403 SOCIETY_ACCESS_DENIED`: User does not hold administrative access to target society.
  - `404 SOCIETY_NOT_FOUND`: Target society does not exist.

---

### 2.3 Gate Shift Operations Summary
Retrieves real-time shift metrics for security and gate operations.

- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/gate/dashboard`
- **Authentication**: Bearer JWT (`Authorization: Bearer <token>`)
- **Optional Header**: `X-Society-ID: <uuid>` (Must match canonical path `{id}` if provided)
- **Authorization**: `SOC_ADMIN` or `GATEKEEPER` (or platform `SUPER_ADMIN`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "society_id": "893c528f-7ee3-463d-82d2-8d76a5ebca57",
    "visitors_on_premises": 4,
    "today_entries": 32,
    "today_exits": 28,
    "pending_approvals": 1,
    "active_guards_on_duty": 2
  }
}
```
- **Errors**:
  - `401 UNAUTHORIZED`: Missing or invalid token.
  - `403 SOCIETY_ACCESS_DENIED`: User does not hold `SOC_ADMIN` or `GATEKEEPER` role for target society.
  - `404 SOCIETY_NOT_FOUND`: Target society does not exist.

---

## 3. Security & Invariant Rules
1. **Tenant Isolation**: Every society query strictly binds `WHERE society_id = $1`. Cross-society data access is strictly forbidden.
2. **Data Minimization**: Dashboards return statistical aggregates and numeric scorecards. No resident names, phone numbers, flat identifiers, or visitor PII are included in dashboard summaries.
3. **Monetary Integrity**: All financial fields are returned as integer paise (`amount_paise`), preventing floating-point rounding divergence.
4. **Read-Only Non-Mutating Execution**: Endpoints execute purely read-only `SELECT` queries with no table modifications.
