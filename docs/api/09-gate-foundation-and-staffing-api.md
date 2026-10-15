# 09 — Gate Foundation and Staffing API Specification

## 1. Overview
Sprint S08 introduces endpoints to configure gates operationally, manage the gatekeeper staff roster, define shift templates, and track duty sessions.

Base Path: `/api/v1/societies/{id}`

Authentication:
```http
Authorization: Bearer <access_token>
```

---

## 2. Endpoints

### 2.1 Update Gate Operational Configuration
- **Method**: `PATCH`
- **Path**: `/api/v1/societies/{id}/gates/{gateId}`
- **Authorization**: `SOC_ADMIN` or `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Main Entrance Gate",
  "directionality": "BIDIRECTIONAL",
  "operational_status": "ACTIVE",
  "required_staff_per_shift": 2,
  "description": "Primary 24/7 entry and exit point"
}
```
- **Validation**:
  - `directionality` in `['BIDIRECTIONAL', 'INGRESS_ONLY', 'EGRESS_ONLY']`.
  - `operational_status` in `['ACTIVE', 'INACTIVE', 'MAINTENANCE', 'EMERGENCY_OPEN']`.
  - `required_staff_per_shift` $\ge 0$.
- **Response**: `200 OK`

### 2.2 List Gatekeepers
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/gatekeepers`
- **Authorization**: `SOC_ADMIN` or `SUPER_ADMIN`
- **Query Parameters**:
  - `status`: `string` (`ACTIVE`, `REVOKED`, `ALL`, default: `ACTIVE`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "gatekeepers": [
      {
        "id": "assignment-uuid",
        "user_id": "user-uuid",
        "phone": "+919820099999",
        "role": "GATEKEEPER",
        "status": "ACTIVE",
        "created_at": "2026-10-18T10:00:00Z"
      }
    ]
  }
}
```

### 2.3 Assign Gatekeeper
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/gatekeepers`
- **Authorization**: `SOC_ADMIN` or `SUPER_ADMIN`
- **Request Body**:
```json
{
  "phone": "+919820099999"
}
```
- **Behavior**:
  - Finds or creates user by normalized phone.
  - Inserts record into `society_user_roles` with `role = 'GATEKEEPER'`, `status = 'ACTIVE'`.
  - Transactionally logs `GATEKEEPER_ASSIGNED` in `platform_audit_logs`.
- **Response**: `201 Created`

### 2.4 Revoke Gatekeeper
- **Method**: `DELETE`
- **Path**: `/api/v1/societies/{id}/gatekeepers/{assignmentId}`
- **Authorization**: `SOC_ADMIN` or `SUPER_ADMIN`
- **Behavior**:
  - Updates `society_user_roles.status = 'REVOKED'`.
  - Closes any active duty sessions for that guard (`status = 'ENDED'`).
  - Transactionally logs `GATEKEEPER_REVOKED` in `platform_audit_logs`.
- **Response**: `200 OK`

### 2.5 List & Manage Shifts
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/shifts`
- **Authorization**: `SOC_ADMIN`, `GATEKEEPER`, or `SUPER_ADMIN`
- **Response**: `200 OK`

- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/shifts`
- **Authorization**: `SOC_ADMIN` or `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Morning Shift",
  "start_time": "06:00:00",
  "end_time": "14:00:00"
}
```
- **Response**: `201 Created`

### 2.6 Active Duty Sessions Roster
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/duty-sessions`
- **Authorization**: `SOC_ADMIN`, `GATEKEEPER`, or `SUPER_ADMIN`
- **Query Parameters**:
  - `gate_id`: `UUID` (optional)
  - `status`: `ACTIVE`, `ENDED`, `ALL` (default: `ACTIVE`)
- **Response**: `200 OK`

### 2.7 Gatekeeper Check-In (Start Duty Session)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/gates/{gateId}/sessions/check-in`
- **Authorization**: `GATEKEEPER` (assigned to this society), `SOC_ADMIN`, or `SUPER_ADMIN`
- **Request Body**:
```json
{
  "shift_id": "shift-uuid"
}
```
- **Behavior**:
  - Verifies caller has active `GATEKEEPER` or admin role in society.
  - Verifies gate exists, belongs to society, and is `status = 'ACTIVE'`.
  - Checks if user is already checked into any gate in this society (`409 SESSION_ALREADY_ACTIVE`).
  - Creates record in `society_gate_sessions` with `status = 'ACTIVE'`, `started_at = NOW()`.
  - Writes `GATE_DUTY_STARTED` in `platform_audit_logs`.
- **Response**: `201 Created`

### 2.8 Gatekeeper Check-Out (End Duty Session)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/gates/{gateId}/sessions/check-out`
- **Authorization**: Stationed `GATEKEEPER`, `SOC_ADMIN`, or `SUPER_ADMIN`
- **Request Body**:
```json
{
  "handover_notes": "All perimeter gates locked; no incidents reported."
}
```
- **Behavior**:
  - Resolves caller's active session at this gate.
  - Updates `status = 'ENDED'`, `ended_at = NOW()`, `handover_notes`.
  - Writes `GATE_DUTY_ENDED` in `platform_audit_logs`.
- **Response**: `200 OK`

---

## 3. Error Codes

| HTTP Status | Error Code | Description |
| :--- | :--- | :--- |
| `400` | `INVALID_PAYLOAD` | Malformed JSON or invalid parameter format |
| `400` | `INVALID_PHONE` | Invalid E.164 phone number |
| `400` | `INVALID_TIMES` | End time must be valid format |
| `400` | `GATE_INACTIVE` | Gate is inactive or closed for operations |
| `403` | `SOCIETY_ACCESS_DENIED` | Caller lacks required role in society |
| `403` | `SOCIETY_SUSPENDED` | Target society is suspended |
| `404` | `GATE_NOT_FOUND` | Gate not found in this society |
| `404` | `GATEKEEPER_NOT_FOUND` | Gatekeeper assignment not found |
| `404` | `SESSION_NOT_FOUND` | No active duty session found to end |
| `409` | `GATEKEEPER_ALREADY_ASSIGNED` | User already holds an active GATEKEEPER assignment |
| `409` | `SESSION_ALREADY_ACTIVE` | Guard is already checked into an active duty session |
| `409` | `SHIFT_NAME_EXISTS` | Shift with this name already exists |

---

## 4. Related Documentation
- [[../decisions/020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation, Gatekeeper Roles, and Staffing Operations]]
- [[../architecture/12-gate-foundation-and-staffing|12 — Gate Foundation and Staffing Architecture]]
- [[../database/08-gate-foundation-and-staffing-schema|08 — Gate Foundation and Staffing Database Schema]]
- [[../security/08-gate-foundation-and-staffing-security|08 — Gate Foundation and Staffing Security Specification]]
- [[../sprints/S08-gate-foundation|S08 — Gate Foundation Sprint]]
