# 08 — Society Designations and Appointments API Specification

## 1. Overview
All designations and committee appointment endpoints are scoped under `/api/v1/societies/{id}/*` and require authentication (`Bearer <JWT>`) and society administrator authorization (`SOC_ADMIN` role or platform `SUPER_ADMIN`).

All endpoints return standardized JSON envelopes:
```json
{
  "success": true,
  "data": { ... },
  "meta": null
}
```

---

## 2. Endpoints

### 2.1 List Designations
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/designations`
- **Query Parameters**:
  - `status`: `string` (`active`, `inactive`, `all`, default: `all`)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "designations": [
      {
        "id": "d001-uuid",
        "society_id": "soc-uuid",
        "code": "CHAIRMAN",
        "name": "Chairman",
        "description": "Presiding officer of the managing committee",
        "allows_multiple_members": false,
        "allows_multiple_positions_per_member": true,
        "is_active": true,
        "active_appointments_count": 1,
        "created_at": "2026-10-15T10:00:00Z",
        "updated_at": "2026-10-15T10:00:00Z"
      }
    ]
  },
  "meta": null
}
```

### 2.2 Create Custom Designation
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/designations`
- **Request Body**:
```json
{
  "code": "VICE_CHAIRMAN",
  "name": "Vice Chairman",
  "description": "Deputizes for the Chairman",
  "allows_multiple_members": false,
  "allows_multiple_positions_per_member": true
}
```
- **Validation**:
  - `code` required, must match `^[A-Z0-9_]{2,50}$`.
  - `name` required, 2–100 chars.
  - Unique within the society.
- **Response**: `201 Created`

### 2.3 Update Designation
- **Method**: `PATCH`
- **Path**: `/api/v1/societies/{id}/designations/{designationId}`
- **Request Body**:
```json
{
  "name": "Executive Chairman",
  "description": "Updated duties",
  "allows_multiple_members": false,
  "allows_multiple_positions_per_member": true
}
```
- **Response**: `200 OK`

### 2.4 Toggle Designation Active Status
- **Method**: `PATCH`
- **Path**: `/api/v1/societies/{id}/designations/{designationId}/status`
- **Request Body**:
```json
{
  "is_active": false
}
```
- **Response**: `200 OK`

### 2.5 List Appointments / Committee Roster
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/appointments`
- **Query Parameters**:
  - `status`: `string` (`ACTIVE`, `ENDED`, `REVOKED`, `ALL`, default: `ACTIVE`)
  - `member_id`: `UUID` (optional filter)
  - `designation_id`: `UUID` (optional filter)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "appointments": [
      {
        "id": "apt-001",
        "society_id": "soc-uuid",
        "designation_id": "d001-uuid",
        "designation_code": "CHAIRMAN",
        "designation_name": "Chairman",
        "member_id": "mem-001",
        "member_name": "Ramesh Sharma",
        "member_phone": "919820011111",
        "start_date": "2026-04-01",
        "expected_end_date": "2027-03-31",
        "end_date": null,
        "status": "ACTIVE",
        "notes": "Elected at AGM 2026",
        "created_at": "2026-04-01T12:00:00Z"
      }
    ]
  },
  "meta": null
}
```

### 2.6 Create Appointment
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/appointments`
- **Request Body**:
```json
{
  "designation_id": "d001-uuid",
  "member_id": "mem-001",
  "start_date": "2026-04-01",
  "expected_end_date": "2027-03-31",
  "notes": "Elected at AGM"
}
```
- **Validation & Business Rules**:
  - `designation_id` and `member_id` must belong to the society.
  - Designation must be `is_active = true`.
  - Member must be in `ACTIVE` status in `society_members`.
  - If `designation.allows_multiple_members = false`, no other `ACTIVE` appointment may exist for that designation (`409 DESIGNATION_ALREADY_APPOINTED`).
  - Member cannot actively hold the exact same designation twice (`409 MEMBER_ALREADY_APPOINTED`).
  - Candidate appointment dates must not overlap historical appointments for singleton designations or same member (`409 CONFLICTING_DATE_RANGE`).
  - If designation disallows multiple positions per member (`allows_multiple_positions_per_member = false`), the member cannot hold other active positions (`409 MULTIPLE_DESIGNATIONS_DISALLOWED`).
  - `start_date <= expected_end_date`.
- **Response**: `201 Created`

### 2.7 End or Revoke Appointment
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/appointments/{appointmentId}/end`
- **Request Body**:
```json
{
  "action": "ENDED",
  "end_date": "2026-10-15",
  "notes": "Resigned due to relocation"
}
```
- **Validation**:
  - `action`: `ENDED` or `REVOKED`.
  - `end_date`: required (`YYYY-MM-DD`), $\ge start\_date$.
- **Response**: `200 OK`

### 2.8 Get Committee Overview
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/committee/overview`
- **Description**: Returns a consolidated governance overview including executive office-bearers (`CHAIRMAN`, `SECRETARY`, `TREASURER`), key metrics, and current active appointments.
- **Authorization**: `Bearer <JWT>` with `SOC_ADMIN` role or platform `SUPER_ADMIN`.
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "total_designations": 4,
    "active_designations": 4,
    "active_appointments": 5,
    "key_positions_filled": {
      "CHAIRMAN": true,
      "SECRETARY": true,
      "TREASURER": true
    },
    "current_appointments": [
      {
        "id": "apt-001",
        "society_id": "soc-uuid",
        "designation_id": "d001-uuid",
        "designation_code": "CHAIRMAN",
        "designation_name": "Chairman",
        "member_id": "mem-001",
        "member_name": "Ramesh Sharma",
        "member_phone": "+91 98200*****",
        "start_date": "2026-04-01",
        "expected_end_date": "2027-03-31",
        "end_date": null,
        "status": "ACTIVE",
        "notes": "Elected at AGM",
        "created_at": "2026-04-01T12:00:00Z",
        "updated_at": "2026-04-01T12:00:00Z"
      }
    ],
    "office_bearers": [
      {
        "designation_id": "d001-uuid",
        "designation_code": "CHAIRMAN",
        "designation_name": "Chairman",
        "member_id": "mem-001",
        "member_name": "Ramesh Sharma",
        "member_phone_masked": "+91 98200*****",
        "start_date": "2026-04-01",
        "expected_end_date": "2027-03-31",
        "status": "ACTIVE"
      },
      {
        "designation_id": "d002-uuid",
        "designation_code": "SECRETARY",
        "designation_name": "Secretary",
        "member_id": "mem-002",
        "member_name": "Milind Date",
        "member_phone_masked": "+91 98220*****",
        "start_date": "2026-01-01",
        "expected_end_date": "2027-12-31",
        "status": "ACTIVE"
      },
      {
        "designation_id": "d003-uuid",
        "designation_code": "TREASURER",
        "designation_name": "Treasurer",
        "member_id": null,
        "member_name": null,
        "member_phone_masked": null,
        "start_date": null,
        "expected_end_date": null,
        "status": null
      }
    ]
  },
  "meta": null
}
```
- **Behavior Notes**:
  - Valid, authorized societies with zero appointments return `200 OK` with `active_appointments: 0`, `current_appointments: []`, and `office_bearers` listing Chairman, Secretary, and Treasurer with vacant (`member_id: null`) entries.
  - Returns `401` for unauthenticated requests, `403` for unauthorized cross-society access or unassigned platform `ADMIN`.
  - Returns `404` only when the society tenant is genuinely not found.

---

## 3. Error Codes

| HTTP Status | Error Code | Description |
| :--- | :--- | :--- |
| `400` | `INVALID_PAYLOAD` | Malformed JSON or invalid parameter format |
| `400` | `INVALID_DATES` | End date is before start date |
| `400` | `DESIGNATION_INACTIVE` | Cannot appoint to an inactive/archived designation |
| `400` | `MEMBER_NOT_ACTIVE` | Member is not in ACTIVE status |
| `403` | `SOCIETY_ACCESS_DENIED` | Caller lacks `SOC_ADMIN` or `SUPER_ADMIN` privileges |
| `403` | `SOCIETY_SUSPENDED` | Society is in suspended state |
| `404` | `DESIGNATION_NOT_FOUND` | Designation does not exist in target society |
| `404` | `MEMBER_NOT_FOUND` | Member does not exist in target society |
| `404` | `APPOINTMENT_NOT_FOUND` | Appointment does not exist in target society |
| `409` | `DESIGNATION_ALREADY_EXISTS` | Designation code already exists in this society |
| `409` | `DESIGNATION_ALREADY_APPOINTED` | Singleton designation already has an active appointment |
| `409` | `MEMBER_ALREADY_APPOINTED` | Member already actively holds this designation |
| `409` | `CONFLICTING_DATE_RANGE` | Term dates conflict with historical appointment |
| `409` | `MULTIPLE_DESIGNATIONS_DISALLOWED` | Member or designation disallows multiple active positions |

---

## 4. Related Documentation
- [[../decisions/019-society-designations-and-appointments|ADR 019: Society Designations, Committee Appointments, and Role Segregation]]
- [[../architecture/11-society-designations-and-appointments|11 — Society Designations and Appointments Architecture]]
- [[../database/07-society-designations-and-appointments-schema|07 — Society Designations and Appointments Database Schema]]
- [[../security/07-society-designations-and-appointments-security|07 — Society Designations and Appointments Security Specification]]
