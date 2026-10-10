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
    "phone": "9967089101"
  }
  ```
- **Success Response (`200 OK`)**:
  ```json
  {
    "success": true,
    "data": {
      "resend_after_seconds": 60
    },
    "meta": null
  }
  ```
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
    "phone": "9967089101",
    "otp": "123456"
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
        "phone": "+91 9967xxxx01",
        "status": "ACTIVE"
      }
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `INVALID_PHONE`: Invalid phone number format.
  - `OTP_INVALID`: OTP does not match.
  - `OTP_EXPIRED`: OTP expired (validity is 5 minutes).
  - `OTP_ATTEMPTS_EXCEEDED`: Max 3 attempts reached; OTP invalidated.
  - `SESSION_LIMIT_REACHED`: User already has 2 active devices. Must log out from an existing device first.

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
      "status": "ACTIVE"
    },
    "meta": null
  }
  ```
- **Error Codes**:
  - `UNAUTHORIZED`: Missing or invalid Bearer token.
  - `TOKEN_EXPIRED`: Access token expired.

## Related Documentation
- [[../architecture/05-authentication|Authentication Architecture]]
- [[../security/01-authentication-security|Authentication Security]]
