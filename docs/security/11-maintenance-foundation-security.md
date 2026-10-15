# 11 — Maintenance Foundation Security

## 1. Overview
The Maintenance Foundation handles financial records, billing periods, charge configurations, and resident obligations. As a multi-tenant platform, CHS enforces strict isolation, multi-level authorization, financial immutability, and transactional audit trails.

---

## 2. Authorization Matrix

| Endpoint / Operation | `SUPER_ADMIN` | `SOC_ADMIN` | Platform `ADMIN` | Resident (Own Unit) | Resident (Other Unit) |
| :--- | :---: | :---: | :---: | :---: | :---: |
| Category / Config CRUD | ✅ | ✅ | ❌ (403) | ❌ (403) | ❌ (403) |
| Period CRUD | ✅ | ✅ | ❌ (403) | ❌ (403) | ❌ (403) |
| Bill Generation / Void | ✅ | ✅ | ❌ (403) | ❌ (403) | ❌ (403) |
| Admin Bills Register | ✅ | ✅ | ❌ (403) | ❌ (403) | ❌ (403) |
| Resident Bills (`/my-bills`) | ✅ | ✅ | ❌ (403) | ✅ | ❌ (403/404) |

---

## 3. Threat Modeling & Mitigations

### 3.1 Cross-Tenant Financial Exposure
- **Threat**: An administrator in Society A accesses or generates bills for units in Society B by tampering with IDs in URLs or bodies.
- **Mitigation**: Every database query enforces `WHERE society_id = $1`. Database tables use composite foreign keys `(category_id, society_id)`, `(billing_period_id, society_id)`, `(unit_id, society_id)` to physically reject cross-society associations at the schema layer.

### 3.2 Resident Insecure Direct Object References (IDOR)
- **Threat**: Resident of Unit 101 attempts to view Unit 102's maintenance bill by querying `/my-bills/{billId}`.
- **Mitigation**: The backend independently validates that the bill's `unit_id` matches an active `unit_relationships` record for the authenticated user's `member_id`. If not matching, the request is rejected with `403 Forbidden` / `404 Not Found`.

### 3.3 Financial Tampering & Race Conditions
- **Threat**: Submitting double-click generation or concurrent API calls creates duplicate bills or incorrect totals.
- **Mitigation**: Database-enforced unique constraint `UNIQUE (society_id, billing_period_id, unit_id)`. The server calculates all totals from active charge rules; client-provided subtotals are completely ignored.

### 3.4 Retroactive History Distortion
- **Threat**: An administrator alters last year's maintenance rate and unintentionally recalculates past bills.
- **Mitigation**: `maintenance_bill_items` stores literal snapshot copies of category names, descriptions, rates, and amounts. Historical bills are strictly read-only and immutable.

---

## 4. Audit Trail
All mutations trigger transactional audit records in `platform_audit_logs`:
- `CHARGE_CATEGORY_CREATED`
- `CHARGE_CATEGORY_UPDATED`
- `CHARGE_CONFIG_CREATED`
- `CHARGE_CONFIG_DEACTIVATED`
- `BILLING_PERIOD_CREATED`
- `BILLS_GENERATED`
- `BILL_VOIDED`

Failure to record the audit log aborts the transaction, guaranteeing zero untracked financial operations.
