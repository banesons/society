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

## Related Documents
- [[09-gate-foundation-and-staffing-api]]
- [[../architecture/14-gate-visitor-operations]]
- [[../decisions/023-visitor-lifecycle-and-gate-operations]]
- [[../security/09-gate-visitor-operations-security]]
