# 03 — Platform Administration API Specification

## Overview

All platform administration endpoints are versioned under `/api/v1/platform`.
All endpoints require a valid Bearer JWT from S01 and an active `SUPER_ADMIN` role.

---

### 1. List Platform Administrators
Retrieves all platform administrators (`SUPER_ADMIN` and `ADMIN`).

- **Method**: `GET`
- **Path**: `/api/v1/platform/admins`
- **Authentication**: `Bearer <access_token>`
- **Authorization**: `SUPER_ADMIN`
- **Query Parameters**:
  - `limit` (optional, default `20`, max `100`)
  - `offset` (optional, default `0`)
  - `role` (optional: `SUPER_ADMIN`, `ADMIN`)
  - `status` (optional: `ACTIVE`, `DISABLED`)
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
          "user_id": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
          "phone": "+91 9967xxxx01",
          "role": "SUPER_ADMIN",
          "status": "ACTIVE",
          "created_by": null,
          "created_at": "2026-10-10T12:00:00Z",
          "updated_at": "2026-10-10T12:00:00Z"
        },
        {
          "id": "e39ac10b-44bb-4122-a123-0e02b2c3d999",
          "user_id": "d1a8b4e2-7e2b-4d39-99e4-06f8c43b95c2",
          "phone": "+91 9820xxxx56",
          "role": "ADMIN",
          "status": "ACTIVE",
          "created_by": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
          "created_at": "2026-10-10T14:30:00Z",
          "updated_at": "2026-10-10T14:30:00Z"
        }
      ],
      "total": 2,
      "limit": 20,
      "offset": 0
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.
  - `FORBIDDEN`: User does not hold an active `SUPER_ADMIN` role.

---

### 2. Create Platform Administrator
Provisions a new platform administrator with the `ADMIN` role.

- **Method**: `POST`
- **Path**: `/api/v1/platform/admins`
- **Authentication**: `Bearer <access_token>`
- **Authorization**: `SUPER_ADMIN`
- **Request Body**:
  ```json
  {
    "phone": "9820123456"
  }
  ```
- **Validation Rules**:
  - `phone`: Required, valid Indian mobile number. Normalized to E.164 (`+91XXXXXXXXXX`).
  - Creation of additional `SUPER_ADMIN` accounts via this API is strictly forbidden (`ROLE_NOT_PERMITTED`).
- **Success Response (`201 Created`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "e39ac10b-44bb-4122-a123-0e02b2c3d999",
      "user_id": "d1a8b4e2-7e2b-4d39-99e4-06f8c43b95c2",
      "phone": "+91 9820xxxx56",
      "role": "ADMIN",
      "status": "ACTIVE",
      "created_at": "2026-10-10T14:30:00Z"
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `INVALID_PHONE`: Malformed phone number.
  - `ADMIN_ALREADY_EXISTS`: User already holds an active platform role.
  - `FORBIDDEN`: Caller is not a Super Admin.

---

### 3. Update Administrator Status
Enables, disables, or reactivates a platform administrator.

- **Method**: `PATCH`
- **Path**: `/api/v1/platform/admins/{id}/status`
- **Authentication**: `Bearer <access_token>`
- **Authorization**: `SUPER_ADMIN`
- **Request Body**:
  ```json
  {
    "status": "DISABLED",
    "reason": "Temporary suspension"
  }
  ```
- **Validation Rules**:
  - `status`: Must be `'ACTIVE'` or `'DISABLED'`.
  - Disabling oneself (`actor_id == target_user_id`) or disabling the initial bootstrap `SUPER_ADMIN` is rejected with `CANNOT_DISABLE_SELF`.
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "e39ac10b-44bb-4122-a123-0e02b2c3d999",
      "user_id": "d1a8b4e2-7e2b-4d39-99e4-06f8c43b95c2",
      "role": "ADMIN",
      "status": "DISABLED",
      "updated_at": "2026-10-10T15:00:00Z"
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `ADMIN_NOT_FOUND`: Specified admin ID does not exist.
  - `INVALID_STATUS`: Status must be `ACTIVE` or `DISABLED`.
  - `CANNOT_DISABLE_SELF`: A Super Admin cannot disable their own account.
  - `FORBIDDEN`: Caller is not a Super Admin.

---

### 4. List Platform Audit Logs
Retrieves immutable administrative audit logs.

- **Method**: `GET`
- **Path**: `/api/v1/platform/audit-logs`
- **Authentication**: `Bearer <access_token>`
- **Authorization**: `SUPER_ADMIN`
- **Query Parameters**:
  - `limit` (optional, default `50`, max `200`)
  - `offset` (optional, default `0`)
  - `action` (optional filter: `ADMIN_CREATED`, `ADMIN_DISABLED`, `ADMIN_REACTIVATED`)
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "items": [
        {
          "id": "a11ac10b-11aa-4222-a999-0e02b2c3d111",
          "actor_id": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
          "action": "ADMIN_DISABLED",
          "target_user_id": "d1a8b4e2-7e2b-4d39-99e4-06f8c43b95c2",
          "details": {
            "previous_status": "ACTIVE",
            "new_status": "DISABLED",
            "reason": "Temporary suspension"
          },
          "client_ip": "127.0.0.1",
          "created_at": "2026-10-10T15:00:00Z"
        }
      ],
      "total": 1,
      "limit": 50,
      "offset": 0
    },
    "meta": null
  }
  ```

---

### 5. Updated `/api/v1/auth/me` Endpoint
In S02, the authenticated `/me` endpoint is updated to return platform role context:

- **Method**: `GET`
- **Path**: `/api/v1/auth/me`
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
      "phone": "+91 9967xxxx01",
      "status": "ACTIVE",
      "platform_roles": ["SUPER_ADMIN"]
    },
    "meta": null
  }
  ```

---

## Related Documentation
- [[../architecture/06-platform-administration|Platform Administration Architecture]]
- [[../security/02-platform-administration-security|Platform Administration Security]]
- [[../database/02-platform-administration-schema|Platform Administration Schema]]
