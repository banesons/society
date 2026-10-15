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
Retrieves the identity details of the authenticated user.

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
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.
  - `TOKEN_EXPIRED`: Access token expired.
  - `SESSION_REVOKED`: Session was revoked via logout, FIFO replacement, or administrative deactivation.
  - `SESSION_EXPIRED`: Underlying server-side session expired.
  - `SESSION_INVALID`: Session record not found or mismatched with user identity.
  - `INTERNAL_ERROR`: Session verification failure (fails closed).

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
