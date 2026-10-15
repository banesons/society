# 15 — Audit and Administration API Contracts

## 1. Overview
Sprint S14 defines unified, server-paginated, and metadata-sanitized audit access for:
1. **Platform Administrators**: Global visibility into platform and multi-society events via `/api/v1/platform/audit-logs`.
2. **Society Administrators**: Tenant-isolated visibility into society-scoped events via `/api/v1/societies/{id}/audit-logs`.

All endpoints require JWT authentication. Endpoints are strictly read-only; no endpoints exist to alter or delete audit history.

---

## 2. Platform Audit Endpoints

### 2.1 List Platform Audit Logs
- **Method**: `GET`
- **Path**: `/api/v1/platform/audit-logs`
- **Access**: `SUPER_ADMIN` (or explicitly authorized platform admins)
- **Query Parameters**:
  - `page`: integer (default: `1`, min: `1`)
  - `limit`: integer (default: `20`, max: `100`)
  - `search`: string (free text search over action, actor name/phone, society name/code)
  - `action`: string (exact action code filter)
  - `actor_id`: UUID (filter by actor user ID)
  - `target_user_id`: UUID (filter by target user ID)
  - `society_id`: UUID (filter by society context)
  - `start_date`: ISO 8601 string / RFC3339 timestamp (inclusive lower bound)
  - `end_date`: ISO 8601 string / RFC3339 timestamp (inclusive upper bound)
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
        "actor_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
        "actor_name": "Super Administrator",
        "actor_phone": "+91 98*** ***10",
        "action": "SOCIETY_REGISTERED",
        "action_label": "Society Registered",
        "target_user_id": null,
        "target_display": null,
        "society_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "society_name": "Emerald Heights CHS",
        "society_code": "EHCHS",
        "details": {
          "society_name": "Emerald Heights CHS",
          "city": "Mumbai",
          "registration_number": "MH/MUM/2026/012"
        },
        "ip_address": "127.0.0.1",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2026-10-14T09:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total_items": 3360,
      "total_pages": 168
    }
  }
}
```

### 2.2 Get Platform Audit Log Detail
- **Method**: `GET`
- **Path**: `/api/v1/platform/audit-logs/{event_id}`
- **Access**: `SUPER_ADMIN`
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "item": {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "actor_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "actor_name": "Super Administrator",
      "actor_phone": "+91 98*** ***10",
      "action": "SOCIETY_REGISTERED",
      "action_label": "Society Registered",
      "target_user_id": null,
      "target_display": null,
      "society_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
      "society_name": "Emerald Heights CHS",
      "society_code": "EHCHS",
      "details": {
        "society_name": "Emerald Heights CHS",
        "city": "Mumbai"
      },
      "ip_address": "127.0.0.1",
      "user_agent": "Mozilla/5.0 ...",
      "created_at": "2026-10-14T09:30:00Z"
    }
  }
}
```

---

## 3. Society-Scoped Audit Endpoints

### 3.1 List Society Audit Logs
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/audit-logs`
- **Access**: `SOC_ADMIN` (assigned to `{id}`), `SUPER_ADMIN`.
- **Rejected Roles**: `ADMIN` (403), `COMMITTEE` (403), `GATEKEEPER` (403), `RESIDENT` (403).
- **Query Parameters**:
  - `page`: integer (default: `1`, min: `1`)
  - `limit`: integer (default: `20`, max: `100`)
  - `search`: string (free text search over action, actor name)
  - `action`: string (exact action code)
  - `actor_id`: UUID (filter by actor user ID)
  - `target_user_id`: UUID (filter by target user ID)
  - `start_date`: ISO 8601 string / RFC3339 timestamp
  - `end_date`: ISO 8601 string / RFC3339 timestamp
- **Tenant Enforcement**:
  - Filter parameter `society_id` cannot be overridden; the path parameter `{id}` is strictly enforced by the query layer (`WHERE (society_id = $1 OR details->>'society_id' = $1::text)`).
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "e5f6a7b8-9012-cdef-1234-567890123456",
        "actor_id": "f6a7b8c9-0123-def1-2345-678901234567",
        "actor_name": "Rajesh Sharma",
        "actor_phone": "+91 98*** ***22",
        "action": "DOCUMENT_CREATED",
        "action_label": "Document Created",
        "target_user_id": null,
        "target_display": "DOC-EHCHS-202610-10001",
        "society_id": "c3d4e5f6-a7b8-9012-cdef-123456789012",
        "society_name": "Emerald Heights CHS",
        "society_code": "EHCHS",
        "details": {
          "title": "AGM Minutes 2026",
          "document_number": "DOC-EHCHS-202610-10001",
          "classification": "SOCIETY_RESIDENTS"
        },
        "ip_address": "192.168.1.10",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2026-10-14T08:15:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total_items": 125,
      "total_pages": 7
    }
  }
}
```

### 3.2 Get Society Audit Log Detail
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/audit-logs/{event_id}`
- **Access**: `SOC_ADMIN` (assigned to `{id}`), `SUPER_ADMIN`
- **Tenant Enforcement**:
  - If the event exists but does not belong to `{id}`, the API returns `404 Not Found` (preventing ID enumeration across tenants).
- **Response** (`200 OK`): Single audit log item matching the envelope structure.

---

## 4. Error Responses

| Status Code | Code | Condition |
| :--- | :--- | :--- |
| `400 Bad Request` | `BAD_REQUEST` | Malformed date range, invalid pagination limits, or bad UUIDs |
| `401 Unauthorized` | `UNAUTHORIZED` | Missing or invalid JWT session |
| `403 Forbidden` | `FORBIDDEN` | Insufficient role or unauthorized society assignment |
| `404 Not Found` | `NOT_FOUND` | Audit event does not exist or belongs to another society |
| `500 Internal Server Error` | `INTERNAL_ERROR` | Database or unhandled error |
