# S12 — Maintenance Operations & Collections

## Objective
Establish a reliable maintenance collection lifecycle: recording offline payments, allocating funds to bills, full and partial payment handling, immutable receipt generation, outstanding balance derivation, resident payment history & receipts, and payment reversal with financial integrity.

## Acceptance Criteria Matrix

| ID | Criterion | Status | Evidence |
| :--- | :--- | :--- | :--- |
| AC-S12-01 | The S11 maintenance console remains functional | PASSED | `cmd/s12_acceptance`: HTTP 200 on categories/periods/bills |
| AC-S12-02 | An authorized SOC_ADMIN can access the collection register | PASSED | `cmd/s12_acceptance`: HTTP 200, 96 bill records loaded |
| AC-S12-03 | An authorized SOC_ADMIN can record a valid offline payment | PASSED | `cmd/s12_acceptance`: HTTP 201, `PAY-PUN-202610-10004` & `REC-PUN-202610-10004` created |
| AC-S12-04 | A successful payment creates the required receipt exactly once | PASSED | `cmd/s12_acceptance`: Verified `COUNT(*) = 1` in `maintenance_receipts` |
| AC-S12-05 | A partial payment produces the correct outstanding balance | PASSED | `cmd/s12_acceptance`: ₹3200 bill - ₹500 paid = ₹2700 remaining balance |
| AC-S12-06 | A subsequent payment can settle the remaining balance | PASSED | `cmd/s12_acceptance`: ₹2700 settlement payment leaves ₹0.00 balance |
| AC-S12-07 | The collection register displays correct billed, paid, and outstanding amounts | PASSED | `cmd/s12_acceptance`: Register verifies billed ₹3200, paid ₹3200, out ₹0.00 |
| AC-S12-08 | A resident can view payment history for an authorized unit | PASSED | `cmd/s12_acceptance`: HTTP 200 on `/my-payments`, 2 payment records returned |
| AC-S12-09 | A resident can open an authorized receipt | PASSED | `cmd/s12_acceptance`: HTTP 200 on `/my-receipts/{id}`, receipt matches ID |
| AC-S12-10 | A resident cannot view another unit's payment history or receipt | PASSED | `cmd/s12_acceptance`: HTTP 403 on cross-society/unauthorized receipt query (IDOR defense) |
| AC-S12-11 | An unauthorized administrator cannot record or reverse payments | PASSED | `cmd/s12_acceptance`: HTTP 403 when unassigned admin attempts payment recording |
| AC-S12-12 | A payment cannot be allocated beyond the outstanding balance | PASSED | `cmd/s12_acceptance`: HTTP 400 `EXCEEDS_BALANCE` returned on overpayment attempt |
| AC-S12-13 | A payment reversal preserves the original record and receipt | PASSED | `cmd/s12_acceptance`: HTTP 200, both payment & receipt preserved with status `REVERSED` |
| AC-S12-14 | A reversal restores the correct outstanding balance | PASSED | `cmd/s12_acceptance`: Balance dynamically restored to ₹500.00 after reversing ₹500 allocation |
| AC-S12-15 | Duplicate submissions do not create duplicate payments or receipts | PASSED | `cmd/s12_acceptance`: Idempotency key reuse returns original record with 0 duplicate rows |
| AC-S12-16 | Concurrent allocation cannot overpay a bill | PASSED | `cmd/s12_acceptance`: 5 concurrent workers: exactly 1 succeeded, 4 rejected; total allocated ≤ bill amount |
| AC-S12-17 | Reporting excludes reversed payments from effective collection totals | PASSED | `cmd/s12_acceptance`: Reversals tracked in `reversed_amount`/`reversed_count`, excluded from effective collection |
| AC-S12-18 | The S11 overdue policy and issued-bill immutability remain correct | PASSED | `cmd/s12_acceptance`: `trg_bills_immutable` prevented direct UPDATE; overdue calculation intact |
| AC-S12-19 | No real payment gateway integration has been introduced | PASSED | Repository audit: 0 checkout SDKs (Razorpay/Stripe); offline methods only |
| AC-S12-20 | Existing society navigation and responsive layouts remain intact | PASSED | Frontend gates: `npm run check` (0 errors), `npm run lint` (0 errors), `npm test` (141/141), `npm run build` (0 errors). Browser UI verified: drawer, header, tabs intact |

## Browser Acceptance Verification

### Initial Automation Error & Diagnosis
- **Failure**: `failed to create browser context: failed to connect to browser via CDP even though the CDP port is responsive: http://127.0.0.1:9222: playwright: Protocol error (Browser.setDownloadBehavior): Browser context management is not supported.`
- **Root Cause**: Stale Playwright/CDP daemon session held a non-isolated target without multi-context capability.
- **Remediation**: IDE server environment restarted cleanly, releasing dead CDP listeners; backend API (`cmd/api` on `:8080`) and Vite dev server (`:5173`) restarted as clean background daemons.

### Browser Test Execution & Evidence
1. **Admin Collections Console**:
   - Authenticated as Super Admin (`+919967089101`).
   - Navigated to Greenview Residency Maintenance Console (`/society/0a2c153c-32ca-458e-a090-81a4687b7a76`).
   - Verified KPI Summary Cards: Total Collected, Total Outstanding, Collection Rate, Active Accounts.
   - Verified Collections Register Table: Bill numbers, unit identifiers, billed amounts, paid amounts, balance calculations, status badges.
   - Verified "Record Offline Payment" modal: Input validation, methods list (`CASH`, `CHEQUE`, `BANK_TRANSFER`, `UPI_RECORDED_OFFLINE`, `OTHER`), zero gateway checkouts.
   - Verified Printable Receipt modal: Official header, receipt reference, bill allocation breakdowns, print and close controls.
   - Recording artifact: `s12_admin_collections_1791873761312.webp`
   - Screenshot artifact: `society_console_1791873806681.png`
2. **Resident Bills & Receipts View**:
   - Authenticated as Resident Vijay Godbole (`+919822000101`).
   - Navigated to Maintenance module.
   - Verified "Assessments & Bills" tab and "Payment Receipts" tab.
   - Verified unit-scoped payment receipt view and printable receipt modal.
   - Recording artifact: `s12_resident_view_1791873929048.webp`
   - Screenshot artifact: `maintenance_console_1791874054642.png`

## Related Documentation
- [[../decisions/026-maintenance-operations-and-collections|ADR 026: Maintenance Operations and Collections]]
- [[../architecture/17-maintenance-operations-and-collections|Architecture 17: Maintenance Operations and Collections Architecture]]
- [[../database/12-maintenance-operations-and-collections-schema|Database Schema 12: Maintenance Operations and Collections Schema]]
- [[../api/13-maintenance-operations-and-collections-api|API 13: Maintenance Operations and Collections API]]
- [[../security/12-maintenance-operations-and-collections-security|Security 12: Maintenance Operations and Collections Security]]

## Status
COMPLETED


