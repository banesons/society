# 05 — Society Administration API Specification

## Overview

This document specifies the HTTP API endpoints introduced in Sprint S04 for **Society Administration**.

All endpoints conform to the unified response contract defined in [[../decisions/002-api-contract|ADR 002]] and require valid Bearer JWT authentication and session headers.

---

## 1. Assign Society Administrator

Assigns an existing user to a specific society as a `SOC_ADMIN`.

```http
POST /api/v1/platform/societies/{id}/admins
Authorization: Bearer <JWT>
X-Session-ID: <session_id>
Content-Type: application/json
```

### Authorization
- Required Platform Role: `SUPER_ADMIN`.
- Platform `ADMIN`, `SOC_ADMIN`, and ordinary users receive HTTP 403 `FORBIDDEN`.

### Request Payload
```json
{
  "phone": "+919820123456",
  "role": "SOC_ADMIN"
}
```
*(Alternatively, `user_id`: `"a90b4d1c-1549-410e-862d-98e3b1c60124"` can be provided directly).*

### Response (`201 Created`)
```json
{
  "success": true,
  "data": {
    "id": "c1f7b19a-4e63-4702-8d75-6e9f2b34a108",
    "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
    "user_id": "a90b4d1c-1549-410e-862d-98e3b1c60124",
    "user_phone_masked": "+91 9820xxxx56",
    "role": "SOC_ADMIN",
    "status": "ACTIVE",
    "assigned_by": "ca27348e-2dd1-4ae2-85cc-bad427b774ac",
    "created_at": "2026-10-12T10:00:00Z"
  },
  "meta": null
}
```

### Error Responses
- `400 BAD_REQUEST`: Malformed UUID or invalid phone syntax.
- `404 NOT_FOUND` (`SOCIETY_NOT_FOUND`): Target society does not exist.
- `404 NOT_FOUND` (`USER_NOT_FOUND`): No user registered with the specified phone/ID.
- `409 CONFLICT` (`ALREADY_ASSIGNED`): User already holds an active `SOC_ADMIN` assignment for this society.

---

## 2. List Society Administrators

Lists all administrators assigned to a target society.

```http
GET /api/v1/platform/societies/{id}/admins?status=ACTIVE&limit=20&offset=0
Authorization: Bearer <JWT>
X-Session-ID: <session_id>
```

### Authorization
- Required Platform Role: `SUPER_ADMIN`.

### Response (`200 OK`)
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "c1f7b19a-4e63-4702-8d75-6e9f2b34a108",
        "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
        "user_id": "a90b4d1c-1549-410e-862d-98e3b1c60124",
        "user_phone_masked": "+91 9820xxxx56",
        "role": "SOC_ADMIN",
        "status": "ACTIVE",
        "assigned_by": "ca27348e-2dd1-4ae2-85cc-bad427b774ac",
        "created_at": "2026-10-12T10:00:00Z"
      }
    ],
    "total": 1,
    "limit": 20,
    "offset": 0
  },
  "meta": null
}
```

---

## 3. Revoke Society Administrator Assignment

Revokes an existing administrator assignment.

```http
DELETE /api/v1/platform/societies/{id}/admins/{assignment_id}
Authorization: Bearer <JWT>
X-Session-ID: <session_id>
```

### Authorization
- Required Platform Role: `SUPER_ADMIN`.

### Response (`200 OK`)
```json
{
  "success": true,
  "data": {
    "id": "c1f7b19a-4e63-4702-8d75-6e9f2b34a108",
    "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
    "user_id": "a90b4d1c-1549-410e-862d-98e3b1c60124",
    "role": "SOC_ADMIN",
    "status": "REVOKED",
    "revoked_by": "ca27348e-2dd1-4ae2-85cc-bad427b774ac",
    "revoked_at": "2026-10-12T11:30:00Z"
  },
  "meta": null
}
```

### Error Responses
- `404 NOT_FOUND` (`ASSIGNMENT_NOT_FOUND`): Assignment does not exist for this society.
- `409 CONFLICT` (`ALREADY_REVOKED`): Assignment has already been revoked.

---

## 4. Current User Society Assignments (`/auth/me`)

Retrieves the authenticated user's profile, platform roles, and active society assignments.

```http
GET /api/v1/auth/me
Authorization: Bearer <JWT>
X-Session-ID: <session_id>
```

### Response (`200 OK`)
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "a90b4d1c-1549-410e-862d-98e3b1c60124",
      "phone": "+91 9820xxxx56",
      "status": "ACTIVE"
    },
    "platform_roles": [],
    "society_assignments": [
      {
        "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
        "society_code": "SOC-MUM-7K4P2X",
        "society_name": "Palm Grove Cooperative Housing Society",
        "role": "SOC_ADMIN",
        "status": "ACTIVE"
      }
    ]
  },
  "meta": null
}
```

---

## 5. Society Profile Endpoint (`SOC_ADMIN` Access)

Retrieves society details for an assigned administrator within their validated society context.

```http
GET /api/v1/societies/{id}/profile
Authorization: Bearer <JWT>
X-Session-ID: <session_id>
X-Society-ID: 7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1
```

### Authorization
- Handled by `RequireSocietyRole("SOC_ADMIN")` middleware.
- Validates caller holds an `ACTIVE` role in `society_user_roles` matching `{id}`.

### Response (`200 OK`)
```json
{
  "success": true,
  "data": {
    "id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
    "society_code": "SOC-MUM-7K4P2X",
    "name": "Palm Grove Cooperative Housing Society",
    "status": "ACTIVE",
    "city": "Mumbai",
    "state": "Maharashtra",
    "pincode": "400703",
    "contact_email": "secretary@palmgrove.in",
    "contact_phone": "+91 9820xxxx56"
  },
  "meta": null
}
```

### Error Responses
- `403 FORBIDDEN` (`SOCIETY_ACCESS_DENIED`): Authenticated user is not an active administrator of this society.
- `403 FORBIDDEN` (`SOCIETY_SUSPENDED`): Target society is on administrative hold.
- `403 FORBIDDEN` (`SOCIETY_PENDING`): Target society has not been activated.

---

## Standard Error Codes

| HTTP Status | Error Code | Description |
| :--- | :--- | :--- |
| `401` | `UNAUTHORIZED` | Invalid, expired, or missing JWT/session token |
| `403` | `FORBIDDEN` | Missing required platform `SUPER_ADMIN` authority |
| `403` | `SOCIETY_ACCESS_DENIED` | Caller lacks active `SOC_ADMIN` assignment for requested society |
| `403` | `SOCIETY_SUSPENDED` | Requested society is currently in `SUSPENDED` status |
| `403` | `SOCIETY_PENDING` | Requested society is currently in `PENDING` status |
| `404` | `SOCIETY_NOT_FOUND` | Target society does not exist |
| `404` | `USER_NOT_FOUND` | Specified user phone or ID does not exist |
| `404` | `ASSIGNMENT_NOT_FOUND` | Role assignment does not exist |
| `409` | `ALREADY_ASSIGNED` | User is already actively assigned as `SOC_ADMIN` for this society |
| `409` | `ALREADY_REVOKED` | Role assignment is already revoked |

---

## Related Documentation
- [[01-api-overview|01 — API Overview]]
- [[04-society-api|04 — Society Onboarding API]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../database/04-society-administration-schema|04 — Society Administration Database Schema]]
- [[../decisions/016-society-role-and-assignment-model|ADR 016: Society Role and Administrator Assignment Model]]
