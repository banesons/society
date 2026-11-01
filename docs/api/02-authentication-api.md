# 02 — Authentication API Specification

## Overview

All authentication endpoints are versioned under `/api/v1/auth`.

---

### 1. Send OTP
Generates and dispatches a 6-digit OTP to the specified phone number.

- **Method**: `POST`
- **Path**: `/api/v1/auth/otp/send`
- **Authentication**: None (Public)
- **Request Body**:
  ```json
  {
    "phone": "919967089101"
  }
  ```
  *Note: Phone numbers are trimmed and stripped of formatting/`+`/spaces on both client and server, stored and communicated strictly as pure numeric integers (`91[6-9]\\d{9}`).*
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "resend_after_seconds": 60,
      "otp": "193298"
    },
    "meta": null
  }
  ```
  *Note: In non-production environments (`ENV!=production`), `data.otp` is returned directly in the response payload to allow instant verification and frontend automated testing without inspecting logs.*
- **Error Codes**:
  - `INVALID_PHONE`: Malformed or non-Indian phone number.
  - `OTP_COOLDOWN`: Resend attempted before 60-second cooldown expires.
  - `OTP_RATE_LIMITED`: Phone exceeded 5 requests/hour or IP exceeded 10 requests/hour.

---

### 2. Verify OTP
Verifies the OTP, provisions the user if new (JIT), checks active device limits, and establishes a new session.

- **Method**: `POST`
- **Path**: `/api/v1/auth/otp/verify`
- **Authentication**: None (Public)
- **Request Body**:
  ```json
  {
    "phone": "919967089101",
    "otp": "193298"
  }
  ```
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJFZERTQ...",
      "refresh_token": "9a8b7c6d...",
      "expires_in": 900,
      "user": {
        "id": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
        "phone": "919967089101",
        "status": "ACTIVE"
      },
      "session_replaced": true
    },
    "meta": null
  }
  ```
  *Note: `session_replaced` is `true` if an existing oldest session was automatically revoked via FIFO replacement to maintain the 2-device limit.*
- **Error Codes**:
  - `INVALID_PHONE`: Invalid phone number format.
  - `OTP_INVALID`: OTP does not match.
  - `OTP_EXPIRED`: OTP expired (validity is 5 minutes).
  - `OTP_ATTEMPTS_EXCEEDED`: Max 3 attempts reached; OTP invalidated.

---

### 3. Refresh Access Token
Rotates the refresh token and issues a new access token.

- **Method**: `POST`
- **Path**: `/api/v1/auth/refresh`
- **Authentication**: None (Requires valid refresh token)
- **Request Body**:
  ```json
  {
    "refresh_token": "9a8b7c6d..."
  }
  ```
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJFZERTQ...",
      "refresh_token": "3f4e5d6c...",
      "expires_in": 900
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `INVALID_TOKEN`: Malformed or missing refresh token.
  - `TOKEN_EXPIRED`: Refresh token expired (30 days).
  - `SESSION_REVOKED`: Session was previously logged out.
  - `REFRESH_TOKEN_REUSED`: Suspected token theft; session terminated.

---

### 4. Logout (Current Device)
Revokes the current active session.

- **Method**: `POST`
- **Path**: `/api/v1/auth/logout`
- **Authentication**: `Bearer <access_token>`
- **Request Body**: Empty
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "message": "Logged out successfully"
    },
    "meta": null
  }
  ```

---

### 5. Logout All Devices
Revokes all active sessions for the authenticated user.

- **Method**: `POST`
- **Path**: `/api/v1/auth/logout-all`
- **Authentication**: `Bearer <access_token>`
- **Request Body**: Empty
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "message": "All sessions revoked successfully"
    },
    "meta": null
  }
  ```

---

### 6. Get Current User (`/me`)
Retrieves the identity and canonical profile details of the authenticated user.

- **Method**: `GET`
- **Path**: `/api/v1/auth/me`
- **Authentication**: `Bearer <access_token>`
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "c7a8b4e2-6d1a-4c28-98e3-05f7c32a84b1",
      "phone": "+91 9967xxxx01",
      "first_name": "Shuubham",
      "last_name": "Bane",
      "display_name": "Shuubham Bane",
      "email": "shubham@banesons.com",
      "profile_photo_url": null,
      "status": "ACTIVE",
      "platform_roles": ["SUPER_ADMIN"],
      "society_assignments": [
        {
          "society_id": "0a2c153c-32ca-458e-a090-81a4687b7a76",
          "society_code": "SOC-PUN-GVR202",
          "society_name": "Greenview Residency",
          "role": "SOC_ADMIN",
          "status": "ACTIVE",
          "city": "Pune",
          "state": "Maharashtra"
        }
      ]
    },
    "meta": null
  }
  ```

---

### 7. Get Personal Profile
Retrieves the authenticated user's canonical personal profile and identity details.

- **Method**: `GET`
- **Path**: `/api/v1/auth/profile`
- **Authentication**: `Bearer <access_token>`
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "537082d1-e9ee-4f00-82d8-d050c7a6b432",
      "phone": "+91 9967xxxx01",
      "phone_raw": "919967089101",
      "first_name": "Shuubham",
      "last_name": "Bane",
      "display_name": "Shuubham Bane",
      "email": "shubham@banesons.com",
      "profile_photo_url": null,
      "status": "ACTIVE",
      "platform_roles": ["SUPER_ADMIN"],
      "society_assignments": []
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.
  - `PROFILE_NOT_FOUND`: User account record not found.

---

### 8. Update Personal Profile
Transactionally updates the authenticated user's canonical profile (`first_name`, `last_name`, `email`, `profile_photo_url`). Guaranteed not to modify verified phone number, platform roles, session limits, or tenant-scoped society member records.

- **Method**: `PATCH`
- **Path**: `/api/v1/auth/profile`
- **Authentication**: `Bearer <access_token>`
- **Request Body**:
  ```json
  {
    "first_name": "Shuubham",
    "last_name": "Bane",
    "email": "shubham@banesons.com",
    "profile_photo_url": "https://..."
  }
  ```
- **Validation Rules**:
  - `first_name`: Required, non-empty after trimming, maximum 100 characters.
  - `last_name`: Optional, maximum 100 characters.
  - `email`: Optional, normalized to lowercase, validated email format, maximum 255 characters.
  - `profile_photo_url`: Optional, validated image URL or data URI within system limits (max 5MB). Set to null or empty string to remove portrait.
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "id": "537082d1-e9ee-4f00-82d8-d050c7a6b432",
      "phone": "+91 9967xxxx01",
      "phone_raw": "919967089101",
      "first_name": "Shuubham",
      "last_name": "Bane",
      "display_name": "Shuubham Bane",
      "email": "shubham@banesons.com",
      "profile_photo_url": "https://...",
      "status": "ACTIVE",
      "platform_roles": ["SUPER_ADMIN"],
      "society_assignments": []
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `VALIDATION_FAILED`: Empty first name or bad payload.
  - `UPDATE_FAILED`: Validation error (e.g., invalid email format, excessive length).
---

### 9. User Activity Timeline
Retrieves a paginated, category-filtered chronological activity feed associated with the authenticated user's identity and phone number (across accounts, visitor movements, society changes, and security events). Routine device logins/logouts are excluded from the personal timeline (persisted internally in `platform_audit_logs`) while security-critical events like `SESSION_ROTATED`, `USER_LOGOUT_ALL`, and `SESSION_REVOKED` are displayed.

- **Method**: `GET`
- **Path**: `/api/v1/auth/profile/activity?category=ALL&limit=20&offset=0`
- **Authentication**: Bearer Token (`middleware.Authenticate`)
- **Query Parameters**:
  - `category`: Filter category. Supported values: `ALL` (default), `ACCOUNT`, `VISITOR`, `SOCIETY`, `SECURITY`.
  - `limit`: Number of items to return (1 to 50, default 20).
  - `offset`: Pagination offset (default 0).
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "events": [
        {
          "id": "e4b2d100-0000-0000-0000-000000000001",
          "action": "SESSION_ROTATED",
          "category": "SECURITY",
          "timestamp": "2026-10-18T10:45:00Z",
          "title": "Session Rotated",
          "summary": "Oldest active device retired under 2-device limit.",
          "display_title": "Session Rotated",
          "display_summary": "Oldest active device retired under 2-device limit.",
          "society_name": null,
          "metadata": {
            "reason": "FIFO_SESSION_LIMIT"
          }
        }
      ],
      "total": 1,
      "limit": 20,
      "offset": 0,
      "has_more": false
    },
    "meta": null
  }
  ```

---

### 10. List Active Sessions
Retrieves the authenticated user's currently active sessions (strictly `revoked_at IS NULL AND expires_at > NOW()`, subject to the invariant maximum of 2 active devices). Excludes raw IP addresses to protect user privacy.

- **Method**: `GET`
- **Path**: `/api/v1/auth/sessions`
- **Authentication**: `Bearer <access_token>`
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "sessions": [
        {
          "id": "84b7260f-1f8d-4f16-950c-e2f9d375a001",
          "device_label": "Chrome on macOS",
          "browser": "Chrome",
          "os": "macOS",
          "is_current": true,
          "created_at": "2026-10-18T20:30:00Z",
          "last_active_at": "2026-10-18T22:15:00Z"
        },
        {
          "id": "93c8371a-2a9e-5a27-061d-f3a0e486b002",
          "device_label": "Safari on iOS",
          "browser": "Safari",
          "os": "iOS",
          "is_current": false,
          "created_at": "2026-10-18T18:00:00Z",
          "last_active_at": "2026-10-18T19:30:00Z"
        }
      ],
      "max_active_sessions": 2
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.

---

### 11. Revoke Active Session
Allows the authenticated user to revoke an active session belonging to them (e.g. terminating access on another recognized device). Strictly validates session ownership (`user_id = auth.user_id`) to prevent cross-user session tampering.

- **Method**: `DELETE`
- **Path**: `/api/v1/auth/sessions/{id}`
- **Authentication**: `Bearer <access_token>`
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "message": "Session revoked successfully"
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.
  - `INVALID_SESSION_ID`: Invalid UUID format.
  - `SESSION_NOT_FOUND`: Active session does not exist, belongs to another user, or was already revoked.

---

## Device and Location Metadata Policy
1. **User Agent Processing**: Device labels, browsers, and operating systems are derived deterministically from the HTTP `User-Agent` header. When unparseable or absent, clean fallbacks (`Unknown Device`, `Unknown`) are returned without fabricating values.
2. **IP Minimization**: Raw client IP addresses are collected and stored in stateful `user_sessions` and internal `platform_audit_logs` for forensic and fraud-detection purposes only. They are NEVER exposed in user-facing session management or profile APIs.
3. **Geolocation Status**: Geolocation requires an external GeoIP database/provider that is not currently configured in the environment. In compliance with data integrity policies, location metadata is cleanly omitted / marked "Unknown" rather than fabricated or inferred from stock imagery.

## Common Authentication Errors on Protected Routes

Every protected route enforced by `middleware.Authenticate` performs cryptographic validation and server-side session checks. The standard error envelopes returned are:

| Code | HTTP Status | Meaning |
| :--- | :--- | :--- |
| `UNAUTHORIZED` | 401 | Missing or malformed `Authorization: Bearer <token>` header |
| `INVALID_TOKEN` | 401 | Invalid Ed25519 signature or corrupt JWT claims |
| `TOKEN_EXPIRED` | 401 | Access JWT 15-minute validity window expired |
| `SESSION_REVOKED` | 401 | Session was revoked via FIFO eviction, logout, or deactivation |
| `SESSION_EXPIRED` | 401 | Stateful session exceeded 30-day lifespan |
| `SESSION_INVALID` | 401 | Session ID does not exist in `user_sessions` or user mismatch |
| `INTERNAL_ERROR` | 500 | Database connection failure during session validation (fail-closed) |

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
- [[../decisions/041-gatekeeper-experience-visitor-identity-and-activity-timeline|ADR 041]]
- [[../decisions/042-active-session-visibility-and-profile-refinement|ADR 042]]
