# API 16: Notifications API Contracts

## 1. Overview
All notification endpoints require authentication via Bearer JWT (`Authorization: Bearer <token>`). The user's identity is strictly derived from the authenticated token context. Clients can never supply arbitrary `recipient_user_id` values.

Base path: `/api/v1/notifications`

---

## 2. Endpoints

### 2.1 List Notifications
Retrieve the current authenticated user's notifications.

- **Method**: `GET`
- **Path**: `/api/v1/notifications`
- **Query Parameters**:
  - `page` (optional int, default: 1)
  - `limit` (optional int, default: 20, max: 100)
  - `unread_only` (optional bool, default: false)
  - `society_id` (optional UUID to filter notifications relevant to a specific society)
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "7a356249-f0bd-460d-85fa-76baec4c51f0",
        "recipient_user_id": "f5127606-2591-456d-88b9-53e34e56fe86",
        "society_id": "893c528f-7ee3-463d-82d2-8d76a5ebca57",
        "type": "VISITOR_APPROVAL_REQUESTED",
        "title": "Visitor Waiting: Delivery Agent",
        "body": "Delivery agent is at North Gate requesting entry to Unit 101.",
        "action_url": "/society/893c528f-7ee3-463d-82d2-8d76a5ebca57#visitors",
        "metadata": {
          "visit_id": "3c847589-9134-45aa-8e25-15a092da0254"
        },
        "reference_type": "society_visits",
        "reference_id": "3c847589-9134-45aa-8e25-15a092da0254",
        "read_at": null,
        "created_at": "2026-10-14T10:00:00Z"
      }
    ],
    "total": 1,
    "limit": 20,
    "offset": 0,
    "pagination": {
      "page": 1,
      "limit": 20,
      "total_items": 1,
      "total_pages": 1
    }
  }
}
```

---

### 2.2 Get Unread Notification Count
Returns the count of unread notifications for badge display.

- **Method**: `GET`
- **Path**: `/api/v1/notifications/unread-count`
- **Query Parameters**:
  - `society_id` (optional UUID)
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

### 2.3 Mark Single Notification Read
Mark a specific notification as read.

- **Method**: `PATCH`
- **Path**: `/api/v1/notifications/{id}/read`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "id": "7a356249-f0bd-460d-85fa-76baec4c51f0",
    "read_at": "2026-10-14T10:05:00Z"
  }
}
```
- **Errors**:
  - `404 NOT_FOUND`: Notification does not exist or does not belong to caller.

---

### 2.4 Mark All Notifications Read
Bulk mark all unread notifications as read for the authenticated caller.

- **Method**: `POST`
- **Path**: `/api/v1/notifications/mark-all-read`
- **Request Body** (optional):
```json
{
  "society_id": "893c528f-7ee3-463d-82d2-8d76a5ebca57"
}
```
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "marked_count": 5
  }
}
```

---

### 2.5 Register / Refresh Device Token
Register or refresh an active FCM device token for web push or native clients.

- **Method**: `POST`
- **Path**: `/api/v1/notifications/devices`
- **Request Body**:
```json
{
  "device_id": "browser_inst_987654321",
  "fcm_token": "fcm_token_string_example...",
  "platform": "WEB"
}
```
- **Validation**:
  - `device_id`: required string (non-empty)
  - `fcm_token`: required string (non-empty)
  - `platform`: `WEB`, `ANDROID`, or `IOS`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "device_id": "browser_inst_987654321",
    "status": "ACTIVE",
    "last_seen_at": "2026-10-14T10:00:00Z"
  }
}
```

---

### 2.6 Deactivate Device Token
Deactivate push notifications for a device on logout or permission revoke.

- **Method**: `DELETE`
- **Path**: `/api/v1/notifications/devices/{device_id}`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "device_id": "browser_inst_987654321",
    "status": "INACTIVE"
  }
}
```

---

### 2.7 List User Devices
List registered devices for the current user.

- **Method**: `GET`
- **Path**: `/api/v1/notifications/devices`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "devices": [
      {
        "id": "e43b174e-5b12-4f35-9051-7c9892ba095e",
        "device_id": "browser_inst_987654321",
        "platform": "WEB",
        "status": "ACTIVE",
        "last_seen_at": "2026-10-14T10:00:00Z"
      }
    ]
  }
}
```

---

## 3. Related Documentation
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../architecture/20-notifications|Architecture 20: Notification System]]
- [[../database/15-notifications-schema|Database Schema 15: Notifications Schema]]
- [[../security/15-notifications-security|Security 15: Notifications Security]]
