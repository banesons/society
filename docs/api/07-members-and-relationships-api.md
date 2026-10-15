# 07 — Members & Relationships API Specification

## 1. Overview

Sprint S06 introduces REST endpoints for managing society members and their relationships to physical units.
All endpoints require authentication and are guarded by the society access middleware:
`RequireSocietyAccess("SOC_ADMIN", ...)`.

---

## 2. Endpoints

### 2.1 List & Search Society Members
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/members`
- **Query Parameters**:
  - `limit`: `int` (default `20`, max `100`)
  - `offset`: `int` (default `0`)
  - `q`: `string` (optional name or canonical phone search)
  - `status`: `string` (`ACTIVE`, `INACTIVE`, `LEFT`, default `ACTIVE`)
- **Response**: `200 OK` (PII masked per project standards)
```json
{
  "members": [
    {
      "id": "e5b8d2a1-0000-0000-0000-000000000001",
      "society_id": "11111111-1111-1111-1111-111111111111",
      "user_id": "99999999-9999-9999-9999-999999999999",
      "first_name": "Ramesh",
      "last_name": "Sharma",
      "phone": "919820098201",
      "email": "ra****@example.com",
      "status": "ACTIVE",
      "is_registered_user": true,
      "active_units_count": 2,
      "created_at": "2026-10-15T10:00:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0
}
```

### 2.2 Register Society Member
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/members`
- **Request Body**:
```json
{
  "first_name": "Ramesh",
  "last_name": "Sharma",
  "phone": "919820098200",
  "email": "ramesh@example.com",
  "emergency_contact_name": "Sita Sharma",
  "emergency_contact_phone": "919820098201"
}
```
- **Validation**:
  - `phone` and `emergency_contact_phone` are validated and normalized to pure numeric 12-digit integers (`91[6-9]\d{9}`). Non-digit characters and spaces are stripped on both FE and BE. Duplicate active phone returns `409 MEMBER_ALREADY_EXISTS`.
  - If a verified user exists with that phone and is not yet linked to another member in this society, `user_id` is linked.
- **Response**: `201 Created`

### 2.3 Get Member Profile & Unit Associations
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/members/{memberId}`
- **Response**: `200 OK` (unmasked details for authorized admin)
```json
{
  "member": {
    "id": "e5b8d2a1-0000-0000-0000-000000000001",
    "society_id": "11111111-1111-1111-1111-111111111111",
    "user_id": "99999999-9999-9999-9999-999999999999",
    "first_name": "Ramesh",
    "last_name": "Sharma",
    "phone": "919820098200",
    "email": "ramesh@example.com",
    "status": "ACTIVE",
    "relationships": [
      {
        "id": "rel-001",
        "unit_id": "unit-101",
        "unit_identifier": "A101",
        "building_name": "Tower A",
        "relationship_type": "PRIMARY_OWNER",
        "is_occupant": false,
        "ownership_share": 50.0,
        "start_date": "2026-01-01",
        "end_date": null,
        "status": "ACTIVE"
      }
    ]
  }
}
```

### 2.4 Update Member Information & Administrative Status
- **Method**: `PATCH`
- **Path**: `/api/v1/societies/{id}/members/{memberId}`
- **Request Body**:
```json
{
  "first_name": "Ramesh",
  "last_name": "Sharma",
  "email": "ramesh.new@example.com",
  "status": "INACTIVE"
}
```
- **Rules**:
  - Updating `status` from `ACTIVE` $\to$ `INACTIVE` is a pure administrative hold and modifies **zero** unit relationships.
  - Updating `status` to `LEFT` directly via PATCH is rejected; the two-phase departure workflow must be used.
- **Response**: `200 OK`

### 2.5 Preview Member Departure (Phase 1)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/members/{memberId}/departure/preview`
- **Request Body**: Empty or `{}`
- **Response**: `200 OK`
```json
{
  "member_id": "e5b8d2a1-0000-0000-0000-000000000001",
  "current_status": "ACTIVE",
  "version_hash": "a1b2c3d4e5f6...7890",
  "preserved_ownership_relationships": [
    {
      "relationship_id": "rel-001",
      "unit_identifier": "A101",
      "relationship_type": "PRIMARY_OWNER",
      "ownership_share": 50.0,
      "action": "PRESERVED_AS_ABSENTEE_OWNER",
      "note": "Ownership title remains active; is_occupant will be set to false."
    }
  ],
  "targeted_terminations": [
    {
      "relationship_id": "rel-002",
      "unit_identifier": "A204",
      "relationship_type": "PRIMARY_TENANT",
      "action": "TERMINATED",
      "proposed_end_date": "2026-10-15"
    }
  ]
}
```

### 2.6 Confirm Member Departure (Phase 2)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/members/{memberId}/departure/confirm`
- **Request Body**:
```json
{
  "expected_version_hash": "a1b2c3d4e5f6...7890",
  "reason_code": "TENANT_MOVED_OUT",
  "reason_details": "Contract concluded; deposit returned.",
  "end_date": "2026-10-15"
}
```
- **Validation Rules**:
  - `reason_code` is required and must be one of:
    - `TENANT_MOVED_OUT` (Tenant Moved Out)
    - `LEASE_EXPIRED` (Lease Expired)
    - `OWNERSHIP_TRANSFERRED` (Ownership Transferred)
    - `MEMBER_RELOCATED` (Member Relocated)
    - `FAMILY_MEMBER_DEPARTED` (Family Member Departed)
    - `MEMBERSHIP_TERMINATED` (Membership Terminated)
    - `DECEASED` (Deceased)
    - `OTHER` (Other)
  - When `reason_code` is `OTHER`, `reason_details` is mandatory and must not be empty or whitespace-only.
  - `reason_details` must not exceed 500 characters.
  - Compares `expected_version_hash` against live state. If stale, returns `409 PREVIEW_STALE`.
  - Transitions confirmed non-ownership occupancies to `status = 'ENDED'`, `end_date = end_date`.
  - Preserves ownership records and sets `is_occupant = false` if resident.
  - Updates member status to `LEFT`, persisting `departure_reason_code`, `departure_reason_details`, and `departed_at`.
  - Runs in a single atomic transaction with audit logging.
- **Response**: `200 OK`
```json
{
  "message": "Member departed successfully; occupancies ended and ownerships preserved"
}
```

### 2.7 List Unit Relationships
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/units/{unitId}/relationships`
- **Query Parameters**:
  - `status`: `string` (`ACTIVE`, `ENDED`, default `ACTIVE`)
- **Response**: `200 OK`
```json
{
  "unit_id": "unit-101",
  "unit_identifier": "A101",
  "is_vacant": false,
  "total_ownership_share_recorded": 100.0,
  "relationships": [
    {
      "id": "rel-001",
      "member_id": "e5b8d2a1-0000-0000-0000-000000000001",
      "member_name": "Ramesh Sharma",
      "member_phone": "+91 98200xxxx1",
      "relationship_type": "PRIMARY_OWNER",
      "is_occupant": false,
      "ownership_share": 50.0,
      "start_date": "2026-01-01",
      "end_date": null,
      "status": "ACTIVE"
    },
    {
      "id": "rel-002",
      "member_id": "e5b8d2a1-0000-0000-0000-000000000002",
      "member_name": "Sita Sharma",
      "member_phone": "+91 98200xxxx2",
      "relationship_type": "PRIMARY_OWNER",
      "is_occupant": false,
      "ownership_share": 50.0,
      "start_date": "2026-01-01",
      "end_date": null,
      "status": "ACTIVE"
    },
    {
      "id": "rel-003",
      "member_id": "e5b8d2a1-0000-0000-0000-000000000003",
      "member_name": "Amit Verma",
      "member_phone": "+91 99300xxxx3",
      "relationship_type": "PRIMARY_TENANT",
      "is_occupant": true,
      "ownership_share": null,
      "start_date": "2026-02-01",
      "end_date": null,
      "status": "ACTIVE"
    }
  ]
}
```

### 2.8 Create Unit Relationship
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/units/{unitId}/relationships`
- **Request Body**:
```json
{
  "member_id": "e5b8d2a1-0000-0000-0000-000000000001",
  "relationship_type": "PRIMARY_OWNER",
  "is_occupant": false,
  "ownership_share": 50.0,
  "start_date": "2026-10-15",
  "notes": "Co-owner 50% share"
}
```
- **Concurrency & Validation**:
  - Acquires exclusive lock on parent unit row (`SELECT ... FOR UPDATE`).
  - `ownership_share` allowed only for `PRIMARY_OWNER` and `SECONDARY_OWNER`.
  - Duplicate active identical relationship returns `409 RELATIONSHIP_ALREADY_EXISTS`.
  - Rejects with `400 INVALID_OWNERSHIP_SHARE` if aggregate total exceeds 100.00%.
- **Response**: `201 Created`

### 2.9 End Unit Relationship
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/units/{unitId}/relationships/{relationshipId}/end`
- **Request Body**:
```json
{
  "end_date": "2026-10-15",
  "notes": "Tenancy contract concluded"
}
```
- **Response**: `200 OK`

---

## 3. Error Codes

| Status Code | Error Code | Description |
| :--- | :--- | :--- |
| `400` | `INVALID_PAYLOAD` | Malformed JSON or invalid parameter |
| `400` | `INVALID_PHONE` | Phone is not a valid E.164 number |
| `400` | `INVALID_DATES` | End date is before start date |
| `400` | `INVALID_OWNERSHIP_SHARE` | Share entered for non-owner or total exceeds 100.00% |
| `400` | `INVALID_DEPARTURE_REASON` | Missing or invalid departure reason code |
| `400` | `REASON_DETAILS_REQUIRED` | Reason details are required when reason code is OTHER |
| `400` | `REASON_DETAILS_TOO_LONG` | Reason details exceed maximum allowed length (500 chars) |
| `403` | `SOCIETY_ACCESS_DENIED` | Caller lacks `SOC_ADMIN` or `SUPER_ADMIN` authorization |
| `403` | `SOCIETY_SUSPENDED` | Target society is suspended |
| `404` | `MEMBER_NOT_FOUND` | Member ID does not exist in target society |
| `404` | `UNIT_NOT_FOUND` | Unit ID does not exist in target society |
| `409` | `MEMBER_ALREADY_EXISTS` | Active member with identical phone already exists |
| `409` | `RELATIONSHIP_ALREADY_EXISTS` | Identical active unit relationship already exists |
| `409` | `PREVIEW_STALE` | Relationships modified concurrently since departure preview |

---

## 4. References
- [[../decisions/018-member-and-relationship-model|ADR 018: Member and Relationship Model]]
- [[../architecture/10-members-and-relationships|10 — Members & Relationships Architecture]]
- [[../database/06-members-and-relationships-schema|06 — Members & Relationships Database Schema]]
