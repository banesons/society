# 11 — Society Communication API

## Overview
The Society Communication API provides administrative endpoints for notice composition, publishing, and archival, as well as resident-facing endpoints for feed consumption and read receipt tracking.

Base Path: `/api/v1/societies/{id}/announcements`

All endpoints require standard session authentication:
```http
Authorization: Bearer <JWT>
```

---

## 1. Administrative Endpoints (`SOC_ADMIN`)

### 1.1 Create Announcement (Draft)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/announcements`
- **Authorization**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
  ```json
  {
    "title": "Water Tank Cleaning Schedule",
    "body": "Overhead tanks in Tower A will be cleaned tomorrow between 10 AM and 2 PM.",
    "category": "MAINTENANCE",
    "priority": "URGENT",
    "audience_type": "BUILDING",
    "target_building_id": "9a009fb7-c8c3-42e7-91a3-ef07cb151da1"
  }
  ```
- **Response**: `201 Created`
  ```json
  {
    "success": true,
    "data": {
      "announcement": {
        "id": "e458e0a2-2b62-43bb-a309-8488e0c83e15",
        "society_id": "0a2c153c-32ca-458e-a090-81a4687b7a76",
        "title": "Water Tank Cleaning Schedule",
        "body": "...",
        "category": "MAINTENANCE",
        "priority": "URGENT",
        "status": "DRAFT",
        "audience_type": "BUILDING",
        "target_building_id": "9a009fb7-c8c3-42e7-91a3-ef07cb151da1",
        "created_by": "user-uuid",
        "created_at": "2026-10-12T10:00:00Z",
        "updated_at": "2026-10-12T10:00:00Z"
      }
    }
  }
  ```

### 1.2 List Administrative Announcements
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/announcements/admin`
- **Authorization**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Query Parameters**:
  - `status`: Optional filter (`DRAFT`, `PUBLISHED`, `ARCHIVED`, `ALL`). Default: `ALL`.
  - `category`: Optional filter (`GENERAL`, `MAINTENANCE`, `SECURITY`, etc.).
  - `limit`: Integer (default 20, max 100).
  - `offset`: Integer (default 0).
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "announcements": [...],
      "total": 12,
      "limit": 20,
      "offset": 0
    }
  }
  ```

### 1.3 Update Announcement
- **Method**: `PUT`
- **Path**: `/api/v1/societies/{id}/announcements/{announcementId}`
- **Authorization**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Rules**: Cannot modify `ARCHIVED` announcements. Updates to `PUBLISHED` announcements preserve `published_at`.
- **Response**: `200 OK`

### 1.4 Publish Announcement
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/announcements/{announcementId}/publish`
- **Authorization**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Rules**: Announcement must be in `DRAFT` status. Transitions status to `PUBLISHED`, assigns `published_at = NOW()` and `published_by = userID`.
- **Response**: `200 OK`

### 1.5 Archive Announcement
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/announcements/{announcementId}/archive`
- **Authorization**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Rules**: Transitions status from `PUBLISHED` to `ARCHIVED`, assigns `archived_at = NOW()`.
- **Response**: `200 OK`

---

## 2. Resident Feed Endpoints (`RESIDENT` / `SOC_ADMIN`)

### 2.1 Get Resident Feed
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/announcements/feed`
- **Authorization**: `RESIDENT`, `SOC_ADMIN`, `SUPER_ADMIN`, `GATEKEEPER`
- **Query Parameters**:
  - `category`: Optional filter.
  - `unread_only`: Boolean (`true` or `false`).
  - `limit`: Integer (default 20, max 50).
  - `offset`: Integer (default 0).
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "announcements": [
        {
          "id": "e458e0a2-2b62-43bb-a309-8488e0c83e15",
          "title": "Water Tank Cleaning Schedule",
          "body": "...",
          "category": "MAINTENANCE",
          "priority": "URGENT",
          "audience_type": "BUILDING",
          "target_name": "Tower A",
          "published_at": "2026-10-12T10:05:00Z",
          "is_read": false,
          "read_at": null
        }
      ],
      "total": 5,
      "unread_count": 2,
      "limit": 20,
      "offset": 0
    }
  }
  ```

### 2.2 Get Announcement Detail
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/announcements/{announcementId}`
- **Authorization**: Verified audience member or `SOC_ADMIN`. Residents cannot view `DRAFT` or `ARCHIVED` announcements through this route.
- **Response**: `200 OK`

### 2.3 Mark Announcement as Read
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/announcements/{announcementId}/read`
- **Authorization**: Accessible resident.
- **Idempotency**: Multiple calls return `200 OK` without duplicating records.
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "announcement_id": "e458e0a2-2b62-43bb-a309-8488e0c83e15",
      "read_at": "2026-10-12T10:30:00Z"
    }
  }
  ```

### 2.4 Get Unread Count
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/announcements/unread-count`
- **Authorization**: Verified resident/member.
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "unread_count": 3
    }
  }
  ```

---

## 3. Related Documentation
- [[../decisions/024-society-communication-and-announcements|ADR 024: Society Communication]]
- [[../architecture/15-society-communication-announcements|Architecture 15: Society Communication]]
- [[../database/10-society-communication-schema|Database Schema 10: Society Communication Schema]]
- [[../security/10-society-communication-security|Security 10: Society Communication Security]]
- [[../sprints/S10-society-communication|S10 Sprint Specification]]
