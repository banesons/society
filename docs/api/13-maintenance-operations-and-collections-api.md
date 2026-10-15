# 13. Maintenance Operations & Collections API

The Maintenance Operations & Collections API provides endpoints for recording offline payments, allocating funds, issuing receipts, processing reversals, querying collection registers and reports, and viewing resident payment history.

## 1. Administrative Endpoints (`SOC_ADMIN`, `SUPER_ADMIN`)

### 1.1 Record Payment
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/payments`
- **Request Body**:
```json
{
  "bill_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "amount": 2500.00,
  "payment_method": "BANK_TRANSFER",
  "payment_date": "2026-10-13",
  "external_reference": "NEFT12345678",
  "payer_name": "Vijay Godbole",
  "notes": "Bank payment verified in statement",
  "idempotency_key": "idem-uuid-1234"
}
```
- **Response** (`201 Created`):
```json
{
  "success": true,
  "data": {
    "payment": {
      "id": "pay-uuid",
      "payment_reference": "PAY-GVR-202610-10001",
      "amount": 2500.00,
      "status": "RECORDED",
      "payment_method": "BANK_TRANSFER",
      "payment_date": "2026-10-13"
    },
    "receipt": {
      "id": "rec-uuid",
      "receipt_number": "REC-GVR-202610-10001",
      "total_amount": 2500.00,
      "status": "ISSUED"
    },
    "allocation": {
      "id": "alloc-uuid",
      "bill_id": "bill-uuid",
      "allocated_amount": 2500.00,
      "remaining_balance": 0.00
    }
  }
}
```

### 1.2 List Collection Register
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/collections/register`
- **Query Params**:
  - `period_id`: Optional UUID
  - `status`: Optional (`UNPAID`, `PARTIALLY_PAID`, `PAID`)
  - `search`: Bill number or unit search
  - `limit`: Default 20, max 100
  - `offset`: Default 0
- **Response**: List of bills with `total_amount`, `total_paid`, `outstanding_balance`, `derived_status`, and `collection_status`.

### 1.3 Collection Summary & Reporting
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/collections/summary`
- **Query Params**: `period_id` (optional)
- **Response**:
```json
{
  "success": true,
  "data": {
    "summary": {
      "total_billed": 150000.00,
      "total_collected": 120000.00,
      "total_outstanding": 30000.00,
      "unpaid_bills_count": 5,
      "partially_paid_bills_count": 2,
      "paid_bills_count": 25,
      "reversed_payments_count": 1,
      "reversed_amount": 3000.00
    }
  }
}
```

### 1.4 Reverse Payment
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/maintenance/payments/{paymentId}/reverse`
- **Request Body**:
```json
{
  "reason": "Cheque bounced on 13 Oct 2026"
}
```
- **Response**: `200 OK` with updated payment, allocation, and restored bill balance.

### 1.5 Get Receipt Details
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/receipts/{receiptId}`
- **Response**: Complete receipt details, including society branding, payer name, payment instrument, bill numbers, and allocation breakdown.

---

## 2. Resident Endpoints (`RESIDENT`)

### 2.1 List Resident Payment History
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/my-payments`
- **Response**: Payments and receipts associated with units where the authenticated resident holds an active relationship.

### 2.2 Get Resident Receipt
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/maintenance/my-receipts/{receiptId}`
- **Auth**: Authorized occupant or owner of the associated unit. IDOR attempts return `403 Forbidden`.
