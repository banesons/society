# 10 — Gate Visitor Operations API

## Overview
This document specifies the RESTful API endpoints for visitor registration, approvals, pass verification, check-in, check-out, and operational dashboard analytics under `/api/v1/societies/{id}/...`.

All endpoints require standard CHS authentication headers:
```http
Authorization: Bearer <JWT>
X-Session-ID: <session-id>
```

Responses adhere strictly to the unified envelope contract (`ADR 002`):
```json
{
  "success": true,
  "data": { ... },
  "request_id": "<uuid>"
}
```

---

## Endpoints Summary

| Method | Path | Allowed Roles | Description |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/societies/{id}/units/my` | Unit Resident | List units authorized for authenticated resident |
| `PATCH` | `/api/v1/societies/{id}/units/{unitId}/visitor-policy` | Unit Resident | Update unit's visitor approval preference |
| `GET` | `/api/v1/societies/{id}/visitors/pending` | Unit Resident, `SOC_ADMIN` | List pending approval visits (scoped to resident units) |
| `POST` | `/api/v1/societies/{id}/visitors/expected` | Unit Resident, `SOC_ADMIN` | Register an expected visitor |
| `POST` | `/api/v1/societies/{id}/visitors/walk-in` | `GATEKEEPER`, `SOC_ADMIN` | Register a walk-in visitor at gate |
| `GET` | `/api/v1/societies/{id}/visitors` | `GATEKEEPER`, `SOC_ADMIN` | List and search visits with pagination & filters |
| `GET` | `/api/v1/societies/{id}/visitors/dashboard` | `GATEKEEPER`, `SOC_ADMIN` | Operational counts (expected, awaiting approval, inside) |
| `GET` | `/api/v1/societies/{id}/visitors/{visitId}` | Unit Resident, `GATEKEEPER`, `SOC_ADMIN` | Get visit details, passes, and events |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/cancel` | Visit Creator, Resident, `SOC_ADMIN` | Cancel an unentered visit |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/approve` | Unit Resident (Target Unit), `SOC_ADMIN` | Approve a pending visit |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/reject` | Unit Resident (Target Unit), `SOC_ADMIN` | Reject a pending visit |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/passes` | Unit Resident, `SOC_ADMIN` | Issue a visitor pass token |
| `POST` | `/api/v1/societies/{id}/visitors/passes/verify` | `GATEKEEPER`, `SOC_ADMIN` | Verify pass by code or bearer token |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/check-in` | Stationed `GATEKEEPER`, `SOC_ADMIN` | Record entry at specific gate |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/check-out` | Stationed `GATEKEEPER`, `SOC_ADMIN` | Record departure at specific gate |
| `GET` | `/public/visit-qr/{token}` | Public / Anonymous | Resolve public entrance registration context |
| `POST` | `/public/visit-qr/send-otp` | Public / Anonymous | Send OTP to visitor mobile number |
| `POST` | `/public/visit-qr/verify-otp` | Public / Anonymous | Verify OTP & receive stateless verification token |
| `POST` | `/public/visit-qr/register` | Public / Anonymous | Submit self-registration and issue Visitor Pass |
| `GET` | `/public/visit-qr/pass/{passReference}` | Public / Anonymous | Track public visitor pass status live |
| `GET` | `/api/v1/societies/{id}/visitors/pending-qr` | `GATEKEEPER`, `SOC_ADMIN` | List pending QR self-registrations (masked phones) |
| `POST` | `/api/v1/societies/{id}/visitors/{visitId}/admit-qr` | Stationed `GATEKEEPER`, `SOC_ADMIN` | Explicit Allow/Reject admission at gate |
| `POST` | `/api/v1/societies/{id}/gates/{gateId}/qr` | `SOC_ADMIN`, `SUPER_ADMIN` | Generate entrance QR for physical gate |
| `GET` | `/api/v1/societies/{id}/visitor-qrs` | `SOC_ADMIN`, `SUPER_ADMIN` | List entrance QR configurations |
| `POST` | `/api/v1/societies/{id}/visitor-qrs/{qrId}/rotate` | `SOC_ADMIN`, `SUPER_ADMIN` | Rotate QR token (invalidates old printed QR) |
| `DELETE` | `/api/v1/societies/{id}/visitor-qrs/{qrId}` | `SOC_ADMIN`, `SUPER_ADMIN` | Revoke entrance QR code |
| `GET` | `/api/v1/visitors/my-history` | Authenticated User | Cross-society visitor pass and visit history |

---

## Detailed Endpoint Contracts

### 1. Register Expected Visitor
`POST /api/v1/societies/{id}/visitors/expected`

**Request Body**:
```json
{
  "unit_id": "038bf8c8-b219-4aa8-9f3b-fa2a549a1f11",
  "visitor_name": "Rajesh Sharma",
  "visitor_phone": "+919820011223",
  "purpose": "GUEST",
  "vehicle_number": "MH02CB1234",
  "expected_arrival": "2026-10-12T14:00:00Z",
  "valid_until": "2026-10-12T22:00:00Z"
}
```

**Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "visit": {
      "id": "e6f47701-4475-4702-8a90-3b60098f41e0",
      "society_id": "0a2c153c-32ca-458e-a090-81a4687b7a76",
      "unit_id": "038bf8c8-b219-4aa8-9f3b-fa2a549a1f11",
      "visitor_name": "Rajesh Sharma",
      "visitor_phone_masked": "+919820xxxx23",
      "purpose": "GUEST",
      "visit_type": "EXPECTED",
      "status": "APPROVED",
      "expected_arrival": "2026-10-12T14:00:00Z",
      "created_at": "2026-10-12T00:15:00Z"
    },
    "pass": {
      "id": "d0e2101b-c12e-4b2a-b684-29d911e3892f",
      "pass_code": "GP-4912",
      "bearer_token": "chs_pass_8a01bf2849e...",
      "valid_from": "2026-10-12T14:00:00Z",
      "valid_until": "2026-10-12T22:00:00Z"
    }
  },
  "request_id": "req-1"
}
```

---

### 2. Register Walk-In Visitor
`POST /api/v1/societies/{id}/visitors/walk-in`

**Request Body**:
```json
{
  "unit_id": "038bf8c8-b219-4aa8-9f3b-fa2a549a1f11",
  "visitor_name": "Amit Courier",
  "visitor_phone": "+919822334455",
  "purpose": "DELIVERY",
  "vehicle_number": "MH12AB9876",
  "requires_approval": true
}
```

**Response (201 Created)**:
```json
{
  "success": true,
  "data": {
    "visit": {
      "id": "11ab3c29-37f2-4ad5-93df-40cb1f419812",
      "society_id": "0a2c153c-32ca-458e-a090-81a4687b7a76",
      "unit_id": "038bf8c8-b219-4aa8-9f3b-fa2a549a1f11",
      "visitor_name": "Amit Courier",
      "visitor_phone_masked": "+919822xxxx55",
      "purpose": "DELIVERY",
      "visit_type": "WALK_IN",
      "status": "PENDING_APPROVAL",
      "created_at": "2026-10-12T00:16:00Z"
    }
  },
  "request_id": "req-2"
}
```

---

### 3. Approve Visit
`POST /api/v1/societies/{id}/visitors/{visitId}/approve`

**Request Body**:
```json
{
  "notes": "Approved for delivery drop-off"
}
```

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "visit": {
      "id": "11ab3c29-37f2-4ad5-93df-40cb1f419812",
      "status": "APPROVED",
      "updated_at": "2026-10-12T00:17:00Z"
    }
  },
  "request_id": "req-3"
}
```

---

### 4. Check-In Visitor
`POST /api/v1/societies/{id}/visitors/{visitId}/check-in`

**Request Body**:
```json
{
  "gate_id": "f8a08d24-3868-45e0-82a9-7c15e8557e4b",
  "vehicle_number": "MH12AB9876",
  "notes": "Delivering package to Flat 101"
}
```

**Validation & Error Mapping**:
- 400 `INVALID_GATE`: Target gate does not exist or does not belong to society.
- 400 `GATE_INACTIVE`: Gate is `INACTIVE` or `MAINTENANCE`.
- 400 `GATE_DIRECTION_MISMATCH`: Gate is `EGRESS_ONLY`.
- 403 `DUTY_SESSION_REQUIRED`: Operating gatekeeper does not hold an active duty session at target gate.
- 409 `VISIT_NOT_APPROVED`: Visit is in `PENDING_APPROVAL`, `REJECTED`, `CANCELLED`, or `EXPIRED`.
- 409 `VISIT_ALREADY_CHECKED_IN`: Visit has already been checked in.

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "visit": {
      "id": "11ab3c29-37f2-4ad5-93df-40cb1f419812",
      "status": "CHECKED_IN",
      "check_in_at": "2026-10-12T00:18:00Z",
      "entry_gate_id": "f8a08d24-3868-45e0-82a9-7c15e8557e4b"
    }
  },
  "request_id": "req-4"
}
```

---

### 5. Check-Out Visitor
`POST /api/v1/societies/{id}/visitors/{visitId}/check-out`

**Request Body**:
```json
{
  "gate_id": "f8a08d24-3868-45e0-82a9-7c15e8557e4b",
  "notes": "Departed on scooter"
}
```

**Validation & Error Mapping**:
- 400 `GATE_DIRECTION_MISMATCH`: Gate is `INGRESS_ONLY`.
- 409 `VISIT_NOT_CHECKED_IN`: Visit is not currently inside (`CHECKED_IN`).
- 409 `VISIT_ALREADY_CHECKED_OUT`: Visit was already checked out.

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "visit": {
      "id": "11ab3c29-37f2-4ad5-93df-40cb1f419812",
      "status": "CHECKED_OUT",
      "check_out_at": "2026-10-12T00:25:00Z",
      "exit_gate_id": "f8a08d24-3868-45e0-82a9-7c15e8557e4b"
    }
  },
  "request_id": "req-5"
}
```

---

### 6. Operational Dashboard Metrics
`GET /api/v1/societies/{id}/visitors/dashboard`

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "expected_today": 8,
    "awaiting_approval": 2,
    "currently_inside": 3,
    "completed_today": 14
  },
  "request_id": "req-6"
}
```

---

### 7. Active Visits (Currently Inside)
`GET /api/v1/societies/{id}/visitors/active`

Retrieves all visits currently in `CHECKED_IN` status across the entire society, enabling fast egress lookup and cross-gate checkout.

**Allowed Roles**: `GATEKEEPER`, `SOC_ADMIN`

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "visits": [
      {
        "id": "11ab3c29-37f2-4ad5-93df-40cb1f419812",
        "visitor_name": "Karan Mehra",
        "visitor_phone": "+91 98*** **321",
        "visitor_phone_masked": "+91 98*** **321",
        "status": "CHECKED_IN",
        "purpose": "GUEST",
        "vehicle_number": "MH02CB1234",
        "unit_number": "A-402",
        "check_in_at": "2026-10-18T10:15:00Z",
        "entry_gate_name": "Main Gate 1"
      }
    ]
  },
  "request_id": "req-7"
}
```

---

### 8. Privacy-Preserving Visitor Identity Search
`GET /api/v1/societies/{id}/visitors/identities/search?phone=+919876543210`

Enables authorized gatekeepers to look up previous visitor identities by phone number. For `GATEKEEPER` callers, the backend enforces server-side masking (`visitor_phone_masked`) and never reveals raw phone numbers or external society histories.

**Allowed Roles**: `GATEKEEPER`, `SOC_ADMIN`

**Response (200 OK)**:
```json
{
  "success": true,
  "data": {
    "identities": [
      {
        "id": "f83b2a11-0000-0000-0000-000000000001",
        "full_name": "Karan Mehra",
        "phone_number": "+91 98*** **210",
        "phone_masked": "+91 98*** **210",
        "user_id": null
      }
    ]
  },
  "request_id": "req-8"
}
```

---

### 9. Gatekeeper Privacy Boundary
When a `GATEKEEPER` interacts with any visitor endpoint (`GET /visitors`, `GET /visitors/active`, `GET /visitors/{id}`, `GET /visitors/identities/search`, `POST /visitors/walk-in`, `POST /visitors/{id}/check-in`, `POST /visitors/{id}/check-out`, `POST /visitors/passes/verify`), the Go backend automatically overwrites `visitor_phone` with `visitor_phone_masked` prior to JSON serialization. The raw phone number is never transmitted over HTTP to gatekeeper clients. Unrestricted user profiles, email addresses, and other societies' records are completely excluded.

---

### 10. Cross-Gate Entry and Exit
A visit checked in at Gate 1 (`INGRESS` or `BIDIRECTIONAL`) can be checked out at Gate 2 (`EGRESS` or `BIDIRECTIONAL`) within the same society. The check-out handler records the distinct `exit_gate_id` and the acting gatekeeper, preserving the entry gate ID untouched.

---

### 11. Public Entrance QR Resolution & Live Pass Tracking
Public endpoints allowing anonymous visitors to resolve physical entrance QR codes and track issued visitor passes without requiring permanent account credentials.

#### A. Resolve Entrance QR Context
`GET /public/visit-qr/{token}`

Returns society and gate details bound to the scanned physical QR token:
```json
{
  "success": true,
  "data": {
    "society_id": "038bf8c8-b219-4aa8-9f3b-fa2a549a1f11",
    "society_name": "Palm Grove Residency",
    "society_code": "PALM01",
    "gate_id": "8f121d51-8728-4ef9-bb6d-9610fdf4aa83",
    "gate_name": "Main Entrance Gate",
    "token": "vqr_a98f12..."
  }
}
```

#### B. Track Public Visitor Pass
`GET /public/visit-qr/pass/{passReference}`

Allows the visitor to monitor live admission status (`REGISTERED_PENDING_GATE_VERIFICATION`, `CHECKED_IN`, `REJECTED`, `CHECKED_OUT`).
- **Security & Entropy**: Uses a 128-bit cryptographically secure, unpredictable bearer reference (`VP-` followed by 32 hex characters, e.g. `VP-9B7C12E4A8F03D6175C2E9B410D8F3A6`) with $2^{128}$ combinations, rendering brute-force enumeration computationally impossible.
- **Rate Limiting**: Protected by client IP rate limiting (maximum 60 lookups/minute per IP). Violations return `429 Too Many Requests`.
- **Response Minimization**: Returns minimal public data. Phone numbers are strictly masked (`+91 98200 *****`), and raw internal host profile details or database identifiers are never exposed.
```json
{
  "success": true,
  "data": {
    "visit_id": "c39a...",
    "pass_reference": "VP-9B7C12E4A8F03D6175C2E9B410D8F3A6",
    "visitor_name": "Rajesh Sharma",
    "visitor_phone_masked": "+91 98200 *****",
    "society_name": "Palm Grove Residency",
    "gate_name": "Main Entrance Gate",
    "purpose": "GUEST",
    "status": "REGISTERED_PENDING_GATE_VERIFICATION",
    "registered_at": "2026-10-19T10:00:00Z"
  }
}
```

---

### 12. Public Visitor OTP Verification & Self-Registration

#### A. Send Visitor OTP
`POST /public/visit-qr/send-otp`

Dispatches a 6-digit verification code to the visitor's mobile number:
```json
{
  "phone": "9820011223",
  "qr_token": "vqr_a98f12..."
}
```

#### B. Verify Visitor OTP
`POST /public/visit-qr/verify-otp`

Verifies the mobile OTP and issues a cryptographically bound HMAC-SHA256 verification token valid for 15 minutes.
The token binds the normalized phone number, SHA-256 hash prefix of the `qr_token`, a 128-bit random nonce (`jti`), and expiration timestamp (`phone|qrHash|nonce|expiresAt|sig`).
```json
{
  "phone": "9820011223",
  "otp": "123456",
  "qr_token": "chs_qr_a98f12..."
}
```

#### C. Register Visitor by QR
`POST /public/visit-qr/register`

Creates or safely references the visitor identity, creates a `society_visits` record in state `REGISTERED_PENDING_GATE_VERIFICATION`, and issues a cryptographically unpredictable bearer reference (`VP-` followed by 32 hex chars with 128-bit cryptographic entropy).
- **Single-Use Enforcement**: Enforces atomic token consumption; replay attempts return `400 Bad Request`.
- **Strict Context Binding**: Token issued for one QR cannot be used at another QR or society.
- **Identity Isolation**: Unauthenticated public registrations NEVER automatically bind or merge visitor identities to platform user accounts.
- **Canonical Identity Preservation**: Deduplication preserves the original person's canonical identity name when phone numbers are shared across individuals.
- **Payload Validation**: Server enforces maximum photo payload length (2.5MB) and strict MIME type checking (`data:image/jpeg`, `data:image/png`, `data:image/webp`, or secure `https://`). No server-side HTTP fetch is performed (preventing SSRF).
```json
{
  "qr_token": "chs_qr_a98f12...",
  "full_name": "Rajesh Sharma",
  "phone": "9820011223",
  "verification_token": "<signed-hmac-token>",
  "purpose": "GUEST",
  "coming_from": "Andheri West",
  "visitor_photo_url": "data:image/jpeg;base64,..."
}
```

---

### 13. Gatekeeper QR Queue & Admission Decision

#### A. List Pending QR Registrations
`GET /api/v1/societies/{id}/visitors/pending-qr?limit=50&gate_id={gateId}`

Allowed Roles: `GATEKEEPER`, `SOC_ADMIN`, `SUPER_ADMIN`.
Enforces strict data minimization: full phone numbers and IP addresses are completely excluded.
```json
{
  "success": true,
  "data": {
    "pending_registrations": [
      {
        "id": "c39a...",
        "pass_reference": "VP-9B7C12",
        "visitor_name": "Rajesh Sharma",
        "visitor_phone_masked": "98200****3",
        "visitor_photo_url": "...",
        "purpose": "GUEST",
        "coming_from": "Andheri West",
        "status": "REGISTERED_PENDING_GATE_VERIFICATION",
        "created_at": "2026-10-19T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

#### B. Admit or Reject QR Visitor
`POST /api/v1/societies/{id}/visitors/{visitId}/admit-qr`

Performs an atomic state transition (`SELECT ... FOR UPDATE`):
- When `decision: "APPROVE"`: Status transitions to `CHECKED_IN`, actual check-in time recorded, gatekeeper identity logged.
- When `decision: "REJECT"`: Status transitions to `REJECTED`, reason captured, visit kept out of active counts.
- Concurrent conflicts automatically return `HTTP 409 Conflict`.
```json
{
  "decision": "APPROVE",
  "gate_id": "8f121d51-8728-4ef9-bb6d-9610fdf4aa83"
}
```

---

### 14. Society Admin QR Code Management & Token Rotation

#### A. Generate Gate QR
`POST /api/v1/societies/{id}/gates/{gateId}/qr`
Allowed Roles: `SOC_ADMIN`, `SUPER_ADMIN`.

#### B. Rotate QR Code
`POST /api/v1/societies/{id}/visitor-qrs/{qrId}/rotate`
Invalidates the old token and issues a new active token. Historical visits remain intact.

#### C. Revoke QR Code
`DELETE /api/v1/societies/{id}/visitor-qrs/{qrId}`
Deactivates the entrance QR immediately.

---

### 15. Authenticated Visitor History
`GET /api/v1/visitors/my-history?limit=20&offset=0`

Allows authenticated residents/visitors to inspect their own cross-society visitor passes and check-in history matching their verified phone number.

#### Identity Disambiguation & Privacy Guarantees
- **Verified Attribution**: Returns visits where ownership is established via:
  1. Explicit authenticated creation (`v.created_by = user_id`), OR
  2. Matching verified phone with matching canonical profile name (`user_profiles.first_name`) and zero conflicting visitor names on that phone number.
- **Shared Phone Protection**: Phone number equality alone is NEVER used as proof that historical visits belong to the caller.
- **Ambiguity Exclusion**: If multiple distinct visitor names have registered using the same phone number (e.g. family members or roommates sharing a mobile), or if the authenticated user has not configured their canonical profile name, unauthenticated historical visits are strictly excluded from the response.
- **Preservation & Non-Destructive Storage**: Excluded historical records remain intact and attributed in `society_visits`; visitor identities are not automatically merged.
- **Operational Limitation & Future Workflow**: For unauthenticated visits on shared phone numbers, users cannot view historical records until they complete an explicit, future manual identity-linking claim workflow.

---

## Related Documents
- [[09-gate-foundation-and-staffing-api]]
- [[../architecture/14-gate-visitor-operations]]
- [[../decisions/023-visitor-lifecycle-and-gate-operations]]
- [[../decisions/041-gatekeeper-experience-visitor-identity-and-activity-timeline]]
- [[../decisions/044-qr-based-visitor-pass-and-gatekeeper-admission]]
- [[../security/09-gate-visitor-operations-security]]

