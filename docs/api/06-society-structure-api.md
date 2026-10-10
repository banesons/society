# 06 — Society Structure & Formation API Specification

## Overview

This document specifies the endpoints introduced in Sprint S05 for physical structure, unit generation, gates, and society formation.

All endpoints adhere to the unified envelope [[../decisions/002-api-contract|ADR 002]] and require standard Bearer JWT authentication.

---

## 1. Buildings & Wings

### 1.1 List Buildings
```http
GET /api/v1/societies/{society_id}/buildings
```
**Authorization**: `SUPER_ADMIN` or active assigned `SOC_ADMIN`.  
**Response (`200 OK`)**:
```json
{
  "success": true,
  "data": [
    {
      "id": "b1b1b1b1-1111-1111-1111-111111111111",
      "society_id": "7b63fcf0-3889-4bc2-84cf-d5e6834ff7f1",
      "name": "Building A",
      "building_code": "A",
      "total_floors": 9,
      "wings": [
        {
          "id": "w1w1w1w1-1111-1111-1111-111111111111",
          "building_id": "b1b1b1b1-1111-1111-1111-111111111111",
          "name": "Wing A",
          "wing_code": "A",
          "total_floors": 9
        }
      ],
      "unit_count": 27
    }
  ],
  "meta": null
}
```

### 1.2 Create Building
```http
POST /api/v1/societies/{society_id}/buildings
Content-Type: application/json

{
  "name": "Building A",
  "building_code": "A",
  "total_floors": 9
}
```
**Response (`201 Created`)**

### 1.3 Create Wing under Building
```http
POST /api/v1/societies/{society_id}/buildings/{building_id}/wings
Content-Type: application/json

{
  "name": "Wing A",
  "wing_code": "A",
  "total_floors": 9
}
```
**Response (`201 Created`)**

---

## 2. Unit Generation & Inventory

### 2.1 Preview Unit Generation
Calculates unit records deterministically without persisting to the database.

```http
POST /api/v1/societies/{society_id}/units/preview
Content-Type: application/json

{
  "building_id": "b1b1b1b1-1111-1111-1111-111111111111",
  "wing_id": "w1w1w1w1-1111-1111-1111-111111111111",
  "starting_floor": 1,
  "total_floors": 9,
  "units_per_floor": 3,
  "starting_unit_number": 101,
  "prefix": "A",
  "floor_overrides": {}
}
```
**Response (`200 OK`)**:
```json
{
  "success": true,
  "data": {
    "total_units": 27,
    "units": [
      { "floor_number": 1, "unit_number": "101", "unit_identifier": "A101" },
      { "floor_number": 1, "unit_number": "102", "unit_identifier": "A102" },
      { "floor_number": 1, "unit_number": "103", "unit_identifier": "A103" },
      { "floor_number": 2, "unit_number": "201", "unit_identifier": "A201" },
      { "floor_number": 9, "unit_number": "903", "unit_identifier": "A903" }
    ]
  },
  "meta": null
}
```

### 2.2 Bulk Generate Units
Atomically commits generated units in a single transaction.

```http
POST /api/v1/societies/{society_id}/units/generate
Content-Type: application/json

{
  "building_id": "b1b1b1b1-1111-1111-1111-111111111111",
  "wing_id": "w1w1w1w1-1111-1111-1111-111111111111",
  "starting_floor": 1,
  "total_floors": 9,
  "units_per_floor": 3,
  "starting_unit_number": 101,
  "prefix": "A",
  "floor_overrides": {}
}
```
**Response (`201 Created`)**

### 2.3 List Units
```http
GET /api/v1/societies/{society_id}/units?building_id=...&wing_id=...&limit=50&offset=0
```
**Response (`200 OK`)**

---

## 3. Gates & Staffing Policy

### 3.1 List Gates
```http
GET /api/v1/societies/{society_id}/gates
```

### 3.2 Create Gate
```http
POST /api/v1/societies/{society_id}/gates
Content-Type: application/json

{
  "name": "Main Gate",
  "gate_code": "GATE-01",
  "gate_type": "VEHICLE_AND_PEDESTRIAN",
  "status": "ACTIVE",
  "required_staff_per_shift": 2,
  "description": "Primary society entrance on North Road"
}
```
**Response (`201 Created`)**

### 3.3 Get / Update Staffing Policy
```http
GET /api/v1/societies/{society_id}/staffing
PUT /api/v1/societies/{society_id}/staffing
Content-Type: application/json

{
  "shifts_per_day": 3,
  "total_required_headcount": 10,
  "notes": "2 guards at Main Gate, 1 at Service Gate, 1 roaming per shift"
}
```

---

## 4. Formation Summary & Commit

### 4.1 Formation Summary
```http
GET /api/v1/societies/{society_id}/formation-summary
```
**Response (`200 OK`)**:
```json
{
  "success": true,
  "data": {
    "total_buildings": 3,
    "total_wings": 7,
    "total_floors": 45,
    "total_units": 315,
    "total_gates": 4,
    "required_staff_per_shift": 6,
    "total_required_headcount": 10,
    "registered_gatekeepers": 0,
    "assigned_gatekeepers": 0
  },
  "meta": null
}
```

---

## 5. Related Documentation
- [[../decisions/017-society-structure-and-formation|ADR 017: Society Structure, Formation Workflow, and Unit Generation Model]]
- [[../architecture/09-society-structure|09 — Society Structure Architecture]]
- [[../database/05-society-structure-schema|05 — Society Structure Database Schema]]
- [[../security/05-society-structure-security|05 — Society Structure Security Specification]]
- [[../sprints/S05-society-structure|S05 — Society Structure]]
