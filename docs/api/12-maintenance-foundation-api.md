# 12 — Maintenance Foundation API

## Overview
The Maintenance Foundation API provides administrative endpoints for managing charge categories, configurations, billing cycles, bill generation, and the bills register, as well as resident-facing endpoints for viewing unit maintenance bills and line items.

Base Path: `/api/v1/societies/{id}`

---

## 1. Charge Categories Endpoints (Admin)

### 1.1 List Charge Categories
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/categories`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Query Params**:
  - `status`: Optional filter (`ACTIVE`, `INACTIVE`)
- **Response**:
```json
{
  "success": true,
  "data": {
    "categories": [
      {
        "id": "c1a2b3c4-...",
        "society_id": "0a2c153c-...",
        "name": "General Maintenance",
        "description": "Monthly recurring upkeep fee",
        "status": "ACTIVE",
        "created_at": "2026-10-12T10:00:00Z"
      }
    ]
  }
}
```

### 1.2 Create Charge Category
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/categories`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Sinking Fund",
  "description": "Long-term structural reserve fund"
}
```
- **Response**: `201 Created` with created category object.

### 1.3 Update Charge Category
- **Method**: `PUT`
- **Path**: `/api/v1/societies/{id}/maintenance/categories/{categoryId}`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Sinking Reserve Fund",
  "description": "Updated description",
  "status": "ACTIVE"
}
```

---

## 2. Charge Configurations Endpoints (Admin)

### 2.1 List Charge Configurations
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/configs`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Response**: List of configurations with joined category name.

### 2.2 Create Charge Configuration
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/configs`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "category_id": "c1a2b3c4-...",
  "name": "Flat Residential Maintenance",
  "charge_type": "FLAT_RECURRING",
  "applicable_unit_type": "RESIDENTIAL",
  "amount": 2500.00,
  "billing_frequency": "MONTHLY",
  "effective_from": "2026-10-01",
  "effective_until": null
}
```

### 2.3 Deactivate Charge Configuration
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/configs/{configId}/deactivate`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`

---

## 3. Billing Periods Endpoints (Admin)

### 3.1 List Billing Periods
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/periods`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`

### 3.2 Create Billing Period
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/periods`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "period_name": "October 2026",
  "period_start": "2026-10-01",
  "period_end": "2026-10-31",
  "due_date": "2026-11-10"
}
```

---

## 4. Bill Generation & Preview Endpoints (Admin)

### 4.1 Preview Bill Generation
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/periods/{periodId}/preview`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Response**:
```json
{
  "success": true,
  "data": {
    "period_id": "p1-...",
    "period_name": "October 2026",
    "total_units": 40,
    "already_billed_units": 0,
    "eligible_units": 40,
    "projected_total_amount": 100000.00,
    "applicable_configs": [
      {
        "name": "Flat Residential Maintenance",
        "amount": 2500.00,
        "charge_type": "FLAT_RECURRING"
      }
    ]
  }
}
```

### 4.2 Generate Bills
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/periods/{periodId}/generate`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**: Optional `{ "note": "Monthly billing run" }`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "period_id": "p1-...",
    "bills_created": 40,
    "units_skipped_existing": 0,
    "total_billed_amount": 100000.00
  }
}
```

---

## 5. Bills Register Endpoints (Admin)

### 5.1 List All Bills (Admin Register)
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/bills`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Query Params**:
  - `period_id`: Optional UUID
  - `unit_id`: Optional UUID
  - `status`: Optional (`ISSUED`, `VOID`, `OVERDUE`)
  - `search`: Unit identifier search
  - `limit`: Default 20, max 100
  - `offset`: Default 0

### 5.2 Get Bill Details with Line Items
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/bills/{billId}`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN` (or authorized resident)
- **Response**:
```json
{
  "success": true,
  "data": {
    "bill": {
      "id": "b1-...",
      "bill_number": "BILL-GVR-202610-A-101",
      "unit_identifier": "A-101",
      "period_name": "October 2026",
      "issue_date": "2026-10-12",
      "due_date": "2026-11-10",
      "currency": "INR",
      "subtotal": 2500.00,
      "total_amount": 2500.00,
      "status": "ISSUED",
      "derived_status": "ISSUED",
      "line_items": [
        {
          "id": "li-1",
          "category_name": "General Maintenance",
          "description": "Flat Residential Maintenance",
          "rate": 2500.00,
          "quantity": 1.00,
          "amount": 2500.00
        }
      ]
    }
  }
}
```

### 5.3 Void Bill
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/bills/{billId}/void`
- **Auth**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "reason": "Incorrect calculation, will regenerate"
}
```

---

## 6. Resident Bills Endpoints

### 6.1 List Resident Bills
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/my-bills`
- **Auth**: Resident occupant or owner in society
- **Response**: Returns only bills for units where the authenticated resident holds an active authorized relationship.

### 6.2 Get Resident Bill Details
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/my-bills/{billId}`
- **Auth**: Resident occupant or owner of the specific unit
