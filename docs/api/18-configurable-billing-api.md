# Configurable Billing Engine REST API

This document details the REST API endpoints for the Configurable Maintenance Billing Engine in CHS (S18).

All endpoints are mounted under `/api/v1/societies/{id}/billing` and are protected by role-based access control requiring society administration roles (`SOCIETY_ADMIN`, `CHAIRPERSON`, `SECRETARY`, `TREASURER`, or platform superadmin).

---

## 1. Template Management APIs

### 1.1 List Templates
- **Endpoint**: `GET /api/v1/societies/{id}/billing/templates`
- **Query Params**: `status` (optional: `DRAFT`, `ACTIVE`, `ARCHIVED`)
- **Response**: Array of `MaintenanceBillTemplate`

### 1.2 Create Template
- **Endpoint**: `POST /api/v1/societies/{id}/billing/templates`
- **Request Body**:
```json
{
  "name": "Standard Housing Maintenance Bill",
  "description": "Monthly residential billing layout",
  "billing_frequency": "MONTHLY",
  "currency": "INR",
  "bill_title": "Tax Invoice / Maintenance Bill",
  "payment_instructions": "Pay via NEFT/RTGS to Society Bank Account",
  "footer_text": "Cheques subject to realization. 2% interest on overdue charges.",
  "bank_details": {
    "bank_name": "HDFC Bank",
    "account_number": "50200012345678",
    "ifsc_code": "HDFC0000123",
    "branch": "Nariman Point"
  },
  "watermark_enabled": true,
  "watermark_opacity": 0.08,
  "fields": [
    {
      "field_key": "meter_reading",
      "field_label": "Water Meter Units",
      "field_category": "BILL_INFO",
      "is_visible": true,
      "is_required": false
    }
  ]
}
```
- **Response**: `201 Created` with created template object (status: `DRAFT`).

### 1.3 Get Template Details
- **Endpoint**: `GET /api/v1/societies/{id}/billing/templates/{templateId}`
- **Response**: Template object with associated template fields.

### 1.4 Update Draft Template
- **Endpoint**: `PATCH /api/v1/societies/{id}/billing/templates/{templateId}`
- **Constraint**: Only `DRAFT` templates can be updated directly. Attempting to update an `ACTIVE` (published) template returns `409 Conflict` (`TEMPLATE_IMMUTABLE`).

### 1.5 Duplicate Template
- **Endpoint**: `POST /api/v1/societies/{id}/billing/templates/{templateId}/duplicate`
- **Request Body**:
```json
{
  "new_name": "Standard Housing Maintenance Bill (Copy)"
}
```
- **Response**: `201 Created` with new cloned template in `DRAFT` status.

### 1.6 Publish Template
- **Endpoint**: `POST /api/v1/societies/{id}/billing/templates/{templateId}/publish`
- **Behavior**:
  - Increments template version counter (`version + 1`).
  - Serializes complete snapshot of template and fields into `maintenance_bill_template_versions`.
  - Sets template status to `ACTIVE`.
  - Published versions are completely immutable.

### 1.7 Archive Template
- **Endpoint**: `POST /api/v1/societies/{id}/billing/templates/{templateId}/archive`
- **Response**: `200 OK` (sets status to `ARCHIVED`).

### 1.8 List Template Versions
- **Endpoint**: `GET /api/v1/societies/{id}/billing/templates/{templateId}/versions`
- **Response**: Array of published immutable versions with snapshots.

---

## 2. Charge Head Management APIs

### 2.1 List Charge Heads
- **Endpoint**: `GET /api/v1/societies/{id}/billing/charge-heads`
- **Query Params**: `active_only` (boolean, default: false)
- **Response**: Array of `MaintenanceChargeHead`.

### 2.2 Create Charge Head
- **Endpoint**: `POST /api/v1/societies/{id}/billing/charge-heads`
- **Request Body**:
```json
{
  "code": "MAINT_AREA",
  "name": "Square Foot Maintenance Fee",
  "description": "Standard monthly maintenance",
  "sac_code": "999511",
  "tax_classification": "TAXABLE",
  "default_tax_rate": 18.0,
  "calculation_method": "AREA_MULTIPLIER",
  "display_order": 1,
  "is_active": true
}
```
- **Supported Calculation Methods**:
  - `FIXED`: Fixed standard amount.
  - `AREA_MULTIPLIER` (or `AREA`): Area-based rate per sq ft.
  - `METERED`: Consumption / meter based.
  - `PERCENTAGE`: Percentage based on specified base charge.
  - `FORMULA`: Dynamic deterministic formula evaluation.

### 2.3 Update Charge Head
- **Endpoint**: `PATCH /api/v1/societies/{id}/billing/charge-heads/{headId}`

### 2.4 Archive Charge Head
- **Endpoint**: `DELETE /api/v1/societies/{id}/billing/charge-heads/{headId}`
- **Behavior**: Soft-deactivates the charge head (`is_active = false`).

---

## 3. Unit Billing Overrides APIs

### 3.1 List Unit Overrides
- **Endpoint**: `GET /api/v1/societies/{id}/billing/units/{unitId}/overrides`
- **Response**: Array of active overrides for the unit.

### 3.2 Create Unit Override
- **Endpoint**: `POST /api/v1/societies/{id}/billing/units/{unitId}/overrides`
- **Request Body**:
```json
{
  "charge_head_id": "a1b2c3d4-...",
  "override_type": "RATE_OVERRIDE",
  "override_value": 2.80,
  "reason": "Discounted maintenance rate for ground floor unit"
}
```
- **Supported Override Types**:
  - `FIXED_AMOUNT`: Overrides line item to fixed amount.
  - `RATE_OVERRIDE`: Overrides rate (e.g. rate per sq ft).
  - `FORMULA_OVERRIDE`: Overrides formula expression.
  - `EXEMPTION`: Explicitly exempts the unit from this charge head (zero amount, exempt tax).

### 3.3 Update Unit Override
- **Endpoint**: `PATCH /api/v1/societies/{id}/billing/units/{unitId}/overrides/{overrideId}`

### 3.4 Delete Unit Override
- **Endpoint**: `DELETE /api/v1/societies/{id}/billing/units/{unitId}/overrides/{overrideId}`

---

## 4. Billing Batch & Calculation Pipeline APIs

### 4.1 List Batches
- **Endpoint**: `GET /api/v1/societies/{id}/billing/batches`
- **Response**: Array of `MaintenanceBillingBatch`.

### 4.2 Create Batch
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches`
- **Request Body**:
```json
{
  "billing_period_id": "c1d2e3f4-...",
  "template_id": "d1e2f3a4-...",
  "batch_name": "November 2026 Maintenance Billing Run",
  "selection_mode": "ALL_UNITS",
  "idempotency_key": "soc-9b1d-202611-batch",
  "notes": "Automated monthly batch"
}
```
- **Idempotency**: `idempotency_key` is unique per society (`uq_maint_batches_idempotency`). Re-submitting an identical key returns the existing batch without re-creating.

### 4.3 Get Batch Details
- **Endpoint**: `GET /api/v1/societies/{id}/billing/batches/{batchId}`
- **Response**: `MaintenanceBillingBatch` including lifecycle state, target unit counts, timestamps, and error summary.

### 4.4 Preview Calculation (Dry Run)
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/preview`
- **Behavior**:
  - Loads eligible units for the society based on `selection_mode` and `filter_criteria`.
  - Resolves unit overrides, charge heads, and tax configurations.
  - Deterministically evaluates charges, GST (intra-state CGST + SGST or inter-state IGST), arrears, interest, and advances.
  - Returns batch calculation summary with individual unit breakdown without modifying persistent bills.

### 4.5 Generate Draft Bills (Execute Batch)
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/generate`
- **Behavior**:
  - Acquires row-level pessimistic lock (`SELECT ... FOR UPDATE`) on the batch to prevent concurrent execution.
  - Enforces lifecycle state validation: batch must be in `DRAFT_PREVIEW`, `QUEUED`, or `FAILED`. If already `PROCESSING`, returns `409 Conflict` (`BATCH_ALREADY_PROCESSING`).
  - Transitions status to `PROCESSING` and records `started_at = NOW()`.
  - Runs deterministic calculation pipeline and creates persistent draft bills in `maintenance_bills` (status: `DRAFT`).
  - Creates immutable line items in `maintenance_bill_items`.
  - Enforces duplicate billing prevention via partial unique index `uq_idx_maintenance_bills_active_unit_period_type`.
  - Persists per-unit snapshots in `maintenance_batch_units`. If an individual unit fails (validation error, missing data), the failure is isolated to that unit (`status = 'FAILED'`, `error_message`, `error_code`) and does not abort the remaining units.
  - Records `completed_at = NOW()`.
  - Batch status transitions to `GENERATED` (if 0 failed), `COMPLETED_WITH_ERRORS` (if partial failures), or `FAILED` (if all failed).

### 4.6 Finalize Batch
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/finalize`
- **Behavior**:
  - Acquires row-level pessimistic lock (`FOR UPDATE`).
  - Validates that batch is in `GENERATED` or `COMPLETED_WITH_ERRORS` status.
  - Transitions bills from `DRAFT` to `ISSUED`.
  - Batch transitions to `FINALIZED` and records `finalized_at = NOW()` and `finalized_by`.
  - PostgreSQL trigger `trg_bills_immutable` enforces permanent immutability of issued bill amounts.

### 4.7 Cancel Batch
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/cancel`
- **Behavior**:
  - Validates that batch is NOT in `FINALIZED` state (returns `409 Conflict`).
  - Deletes all unissued `DRAFT` bills generated by this batch.
  - Batch transitions to `CANCELLED` and records `cancelled_at = NOW()`, `cancelled_by`, and `cancellation_reason`.

### 4.8 Get Batch Progress
- **Endpoint**: `GET /api/v1/societies/{id}/billing/batches/{batchId}/progress`
- **Response**: `200 OK`
```json
{
  "batch_id": "84310d54-1506-4fc6-bc2f-376c66657c91",
  "status": "PROCESSING",
  "target_unit_count": 100,
  "processed_unit_count": 80,
  "successful_unit_count": 78,
  "failed_unit_count": 2,
  "pending_unit_count": 20,
  "started_at": "2026-11-01T20:00:00Z",
  "completed_at": null,
  "finalized_at": null
}
```

### 4.9 List Batch Units
- **Endpoint**: `GET /api/v1/societies/{id}/billing/batches/{batchId}/units`
- **Query Params**: `status` (optional: `PENDING`, `CALCULATED`, `GENERATED`, `EXCLUDED`, `FAILED`), `limit`, `offset`
- **Response**: Array of `BatchUnit` records with diagnostic errors and snapshots.

### 4.10 Retry Failed Units
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/retry`
- **Behavior**:
  - Acquires row-level lock (`FOR UPDATE`).
  - Validates batch is in `COMPLETED_WITH_ERRORS` or `FAILED` status.
  - Transitions batch to `PROCESSING`.
  - Resolves only units in `FAILED` status (`status = 'FAILED'`).
  - Existing successful draft or finalized bills are strictly preserved and never recalculated or overwritten.
  - Re-evaluates formulas and creates draft bills for recovered units.
  - Increments `retry_count` and updates `last_attempted_at`.
  - Updates batch counters and transitions to `GENERATED` (if all failed units succeeded) or remains `COMPLETED_WITH_ERRORS`.

---

## 5. PDF Generation & Artifact Management APIs

### 5.1 Generate Bill PDF
- **Endpoint**: `POST /api/v1/societies/{id}/billing/bills/{billId}/pdf`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Behavior**:
  - Builds `BillDocument` from immutable bill snapshot, published template version, society branding, and resident details.
  - Renders print-ready A4 PDF with official CHS logo watermark.
  - Computes SHA-256 checksum and uploads to storage provider.
  - Records artifact in `maintenance_bill_pdf_artifacts` and updates `maintenance_bills.pdf_artifact_path`.
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "id": "c13ea4b8-f86a-4951-b847-f47ffea9c6a1",
    "society_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "bill_id": "f29db3f5-7d8b-4a5c-897b-89ff8a385f02",
    "artifact_type": "SINGLE_BILL_PDF",
    "file_name": "BILL-202611-0001.pdf",
    "storage_path": "societies/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d/billing/bills/f29db3f5-7d8b-4a5c-897b-89ff8a385f02/BILL-202611-0001.pdf",
    "file_size_bytes": 18450,
    "sha256_checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "mime_type": "application/pdf",
    "has_watermark": true,
    "created_at": "2026-11-01T17:30:00Z"
  }
}
```

### 5.2 Download Bill PDF
- **Endpoint**: `GET /api/v1/societies/{id}/billing/bills/{billId}/pdf` (also available via `GET /api/v1/societies/{id}/maintenance/my-bills/{billId}/pdf` for residents)
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`, or resident owning/occupying the billed unit.
- **Headers Returned**:
  - `Content-Type: application/pdf`
  - `Content-Disposition: inline; filename="BILL-202611-0001.pdf"`
  - `ETag: "{sha256_checksum}"`
  - `X-Content-Type-Options: nosniff`
- **Response**: Binary stream of the PDF file.

### 5.3 Generate Batch PDF Archive
- **Endpoint**: `POST /api/v1/societies/{id}/billing/batches/{batchId}/pdf`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Behavior**:
  - Generates individual PDFs for all finalized/issued bills in the batch.
  - Packages PDFs into an in-memory ZIP archive with deterministic naming.
  - Persists `BATCH_ZIP_ARCHIVE` artifact with SHA-256 checksum.
- **Response**: `200 OK` with artifact metadata.

### 5.4 Download Batch ZIP Archive
- **Endpoint**: `GET /api/v1/societies/{id}/billing/batches/{batchId}/pdf`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Headers Returned**:
  - `Content-Type: application/zip`
  - `Content-Disposition: attachment; filename="{batchNumber}.zip"`
  - `ETag: "{sha256_checksum}"`
- **Response**: Binary stream of the ZIP archive.

### 5.5 List Bill Artifacts
- **Endpoint**: `GET /api/v1/societies/{id}/billing/bills/{billId}/artifacts`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "artifacts": [ ... ]
  }
}
```

### 5.6 Get Artifact Details / Download by Artifact ID
- **Endpoint**: `GET /api/v1/societies/{id}/billing/artifacts/{artifactId}`
- **Query Params**: `download=true` (optional; if true, streams binary payload with appropriate Content-Type)
- **Response**: JSON metadata or binary stream.

---

## 6. Billing Automation Scheduler Endpoints

### 6.1 Create Billing Schedule
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Monthly Maintenance Automation",
  "recurrence_type": "MONTHLY",
  "template_id": "uuid",
  "selection_mode": "ALL_UNITS",
  "filter_criteria": {},
  "day_of_month": 1,
  "due_days": 15,
  "timezone": "Asia/Kolkata",
  "auto_finalize": false
}
```
- **Response**: `201 Created`

### 6.2 List Billing Schedules
- **Endpoint**: `GET /api/v1/societies/{id}/billing/schedules`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "schedules": [ ... ]
  }
}
```

### 6.3 Get Billing Schedule Details
- **Endpoint**: `GET /api/v1/societies/{id}/billing/schedules/{scheduleId}`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK`

### 6.4 Update Billing Schedule
- **Endpoint**: `PUT /api/v1/societies/{id}/billing/schedules/{scheduleId}`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Request Body**: Editable fields (`name`, `template_id`, `selection_mode`, `filter_criteria`, `day_of_month`, `due_days`, `timezone`, `auto_finalize`).
- **Response**: `200 OK`

### 6.5 Pause Billing Schedule
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/pause`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK` (sets status to `PAUSED`)

### 6.6 Resume Billing Schedule
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/resume`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK` (sets status to `ACTIVE`, recalculates `next_run_at` if past)

### 6.7 Disable Billing Schedule
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/disable`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK` (sets status to `DISABLED`)

### 6.8 Trigger Manual Schedule Run
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/run-now`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK` with execution details and generated batch ID.
- **Conflict (409)**: If schedule is already running or batch already active for intended period.

### 6.9 List Schedule Executions
- **Endpoint**: `GET /api/v1/societies/{id}/billing/schedules/{scheduleId}/executions`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "executions": [ ... ]
  }
}
```

### 6.10 Get Billing Automation Health & Diagnostics
- **Endpoint**: `GET /api/v1/societies/{id}/billing/schedules/health`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Response**: `200 OK`
```json
{
  "success": true,
  "data": {
    "worker_status": "RUNNING",
    "worker_id": "sched-worker-abc12345",
    "last_poll_at": "2026-11-02T10:00:00Z",
    "last_execution_at": "2026-11-02T09:00:00Z",
    "total_active_schedules": 5,
    "due_schedules": 0,
    "overdue_schedules": 0,
    "locked_schedules": 0,
    "expired_locks": 0,
    "stuck_executions": 0,
    "failed_executions_24h": 0,
    "recovery_required_count": 0
  }
}
```

### 6.11 Recover Expired Schedule Lock
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/recover-lock`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Description**: Safely clears dangling `locked_by` and `locked_until` timestamps after a process crash or hung execution.
- **Response**: `200 OK` with updated schedule details.

### 6.12 Retry Failed Schedule Execution
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/executions/{executionId}/retry`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Description**: Re-evaluates failed units or re-generates an incomplete batch draft without duplicating financial assessments or overwriting finalized bills.
- **Response**: `200 OK` with execution result and recovery status `RECOVERED`.

### 6.13 Re-run Missed / Overdue Schedule
- **Endpoint**: `POST /api/v1/societies/{id}/billing/schedules/{scheduleId}/re-run-missed`
- **Role**: `SOCIETY_ADMIN`, `SUPER_ADMIN`
- **Description**: Safely assesses and generates a batch for an overdue or unexecuted cycle, updating `next_run_at` to the proper current interval.
- **Response**: `200 OK` with generated execution record.


