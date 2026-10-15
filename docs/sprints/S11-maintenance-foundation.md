# S11 — Maintenance Foundation

## Objective
Establish the financial structure for society maintenance: charge definitions, calculation rules, billing cycle schedules, unit-level fee configuration, immutable bill generation, and resident itemized visibility.

## Scope

### Backend
- Charge category management (e.g. Monthly Maintenance, Sinking Fund, Parking, Special Repair).
- Charge configuration engine (flat recurring, unit-type specific, one-off project charges).
- Billing periods with date validation and due dates.
- Concurrency-safe, idempotent bill generation engine.
- Financial snapshots and immutable bill line items (`maintenance_bill_items`).
- Derived bill lifecycles (`ISSUED`, `OVERDUE`, `VOID`).
- Maintenance bills register and resident-facing itemized visibility.
- Fail-closed transactional audit logging in `platform_audit_logs`.
- Multi-tenant isolation enforced via composite foreign keys and middleware.

### Frontend
- Society Admin Maintenance Console:
  - Charge Categories tab (CRUD & status).
  - Charge Configurations tab (CRUD & status).
  - Billing Periods tab (CRUD & status).
  - Bill Generation modal with billing scope and projected fee preview.
  - Bills Register with filters, pagination, bill detail modal, and void workflow.
- Resident Maintenance Bills View:
  - My Bills register scoped to authorized units.
  - Itemized line items breakdown modal.
  - Real-time overdue and due date status indicators.
- Navigation integration into society drawer and society shell.

## Explicit Exclusions (Deferred to S12 or Later)
- Payment collection, online checkout, payment gateways (UPI, Cards, NetBanking).
- Payment receipts, payment reconciliation, partial payment allocation.
- Late-fee interest calculations or payment reminders.
- Ledger accounting, vendor management, or Restro/POS billing.

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| AC-S11-01 | Authorized SOC_ADMIN can access the maintenance area | PASS | Verified in `cmd/s11_acceptance` and UI shell navigation guard. |
| AC-S11-02 | SOC_ADMIN can create a charge category | PASS | Verified via `POST /maintenance/categories` (`201 Created`, duplicate `409 Conflict`). |
| AC-S11-03 | SOC_ADMIN can create a valid charge configuration | PASS | Verified via `POST /maintenance/configs` with flat recurring rate. |
| AC-S11-04 | Invalid amounts and dates are rejected | PASS | Verified via rejection of non-positive amount (`400 INVALID_AMOUNT`) and inverted dates (`400 INVALID_DATES`). |
| AC-S11-05 | SOC_ADMIN can create a billing period | PASS | Verified via `POST /maintenance/periods` with cycle dates and due date. |
| AC-S11-06 | SOC_ADMIN can preview the billing scope before generation | PASS | Verified via `GET /maintenance/periods/{id}/preview` calculating eligible units and projected totals. |
| AC-S11-07 | SOC_ADMIN can generate bills for eligible units | PASS | Verified via `POST /maintenance/periods/{id}/generate` producing bills with sequential bill numbers. |
| AC-S11-08 | Generated bills contain correct line items and totals | PASS | Verified immutable snapshot in `maintenance_bill_items` matching configured charge amounts. |
| AC-S11-09 | Repeating bill generation does not create duplicate bills | PASS | Verified idempotent rerun: 0 bills created, 12 units skipped via `uq_maintenance_bills_unit_period`. |
| AC-S11-10 | The bills register displays generated bills correctly | PASS | Verified via `GET /maintenance/bills` filtering by period, status, unit search, and pagination. |
| AC-S11-11 | A resident can view a bill for an authorized unit | PASS | Verified via `GET /maintenance/my-bills` and `GET /maintenance/my-bills/{id}` for authenticated resident. |
| AC-S11-12 | A resident cannot access another unit's bill by changing URL, bill ID, or payload | PASS | Verified IDOR enforcement: cross-unit bill access returns `403 Forbidden`. |
| AC-S11-13 | A user without society authorization cannot manage that society's maintenance records | PASS | Verified authorization middleware: resident mutation attempts return `403 Forbidden`. |
| AC-S11-14 | Issued bills preserve their financial snapshot after a charge configuration changes | PASS | Verified: deactivating charge configuration left previously generated bills and line items intact. |
| AC-S11-15 | Voided or overdue states are displayed according to documented lifecycle rules | PASS | Verified: `POST /maintenance/bills/{id}/void` sets status to `VOID` with audit reason; query derives overdue status. |
| AC-S11-16 | The UI has useful loading, empty, success, and error states | PASS | Implemented in `maintenance-console.svelte` and `resident-bills-view.svelte` with loaders, empty states, and feedback banners. |
| AC-S11-17 | No payment-collection UI or payment mutation endpoints have been introduced by S11 | PASS | Verified: `/pay` endpoint returns `404 Not Found`; zero payment UI exists. |
| AC-S11-18 | Existing society navigation and responsive layouts remain intact | PASS | Verified: all 141 frontend tests pass, drawer integrates maintenance tab, `npm run build` succeeds. |

## Related Documentation
- [[../decisions/025-maintenance-foundation-and-billing|ADR 025: Maintenance Foundation and Billing]]
- [[../architecture/16-maintenance-foundation|Architecture 16: Maintenance Foundation Architecture]]
- [[../database/11-maintenance-foundation-schema|Database Schema 11: Maintenance Foundation Schema]]
- [[../api/12-maintenance-foundation-api|API 12: Maintenance Foundation API]]
- [[../security/11-maintenance-foundation-security|Security 11: Maintenance Foundation Security]]

## Status
COMPLETED
