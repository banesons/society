# ADR 026: Maintenance Operations and Collections

## Context
Sprint S11 established the maintenance billing foundation: charge heads, rates, billing cycles, idempotent bill generation, immutable line-item financial snapshots, and resident-facing itemized visibility.

However, societies require an operational collection workflow to record payments received, allocate funds against bills, handle partial payments, issue immutable receipts, support administrative reversals, and maintain precise outstanding balances.

## Decisions

### 1. Payment Recording & Offline Method Modeling
- CHS records payments received offline by the society administration.
- Supported methods:
  - `CASH`: Cash received at society office.
  - `CHEQUE`: Physical cheque handed over (recorded with cheque reference).
  - `BANK_TRANSFER`: NEFT/RTGS/IMPS verified in society bank account.
  - `UPI_RECORDED_OFFLINE`: UPI transfer to society QR/VPA verified in bank statements.
  - `OTHER`: Exceptional approved offline instruments.
- Strictly no payment gateway integrations (Razorpay, Stripe, UPI Intent) in S12.
- Payments must be recorded by an authorized `SOC_ADMIN` or `SUPER_ADMIN`. Resident-entered transaction references are not accepted as proof of payment.

### 2. Allocation Architecture
- Payments are recorded and allocated against maintenance bills within the same tenant society.
- A payment is allocated to a specific non-voided bill.
- Multiple partial payments can be allocated to a single bill until the outstanding balance reaches zero.
- Over-allocation is strictly prohibited at both repository and database levels:
  $$\sum \text{allocated\_amount} \le \text{payment.amount}$$
  $$\sum \text{allocated\_amount} \le \text{bill.outstanding\_amount}$$
- Canonical bill balance:
  $$\text{Outstanding Balance} = \text{bill.total\_amount} - \sum_{\text{status} = \text{'ACTIVE'}} \text{allocation.amount}$$
- Collection status is derived:
  - `UNPAID`: $\text{Paid} = 0$
  - `PARTIALLY_PAID`: $0 < \text{Paid} < \text{Total}$
  - `PAID`: $\text{Paid} \ge \text{Total}$

### 3. Concurrency-Safe Receipt & Number Allocation Strategy
- Bill numbers, payment references, and receipt numbers must never rely on unsafe `COUNT` queries or uncoordinated timestamps.
- Enforced via PostgreSQL sequences:
  - `maintenance_bill_seq` for bill numbering: `BILL-{SOC_SHORT}-{YYYYMM}-{UNIT}-{SEQ}`
  - `maintenance_payment_seq` for payment references: `PAY-{SOC_SHORT}-{YYYYMM}-{SEQ}`
  - `maintenance_receipt_seq` for receipt numbering: `REC-{SOC_SHORT}-{YYYYMM}-{SEQ}`
- Concurrent allocation against a bill uses row-level locking (`SELECT ... FOR UPDATE` on `maintenance_bills`) ensuring that concurrent payment allocation attempts cannot overpay a bill.

### 4. Idempotency
- Payment creation endpoints accept an `idempotency_key` (UUID or client token).
- Enforced by unique constraint `UNIQUE (society_id, idempotency_key)` on `maintenance_payments`.
- Repeated requests with the identical key and payload return the previously generated payment and receipt; conflicting payloads return `409 CONFLICT`.

### 5. Reversal vs. Destructive Edits
- Payments and receipts are immutable and never deleted.
- Administrators can reverse an erroneous payment by providing a mandatory non-empty reason.
- Reversal atomically updates:
  1. `maintenance_payments.status = 'REVERSED'` (with `reversed_by`, `reversed_at`, `reversal_reason`).
  2. `maintenance_payment_allocations.status = 'REVERSED'` (with `reversed_at`).
  3. `maintenance_receipts.status = 'REVERSED'` (with `reversed_at`).
  4. Platform audit trail transactionally logged.
- Original receipt records remain inspectable for audit integrity, flagged as `REVERSED`.
- Restores the bill's outstanding balance immediately.

### 6. Bill Void Guard
- S11 bill voiding is enhanced: a bill with active (non-reversed) payment allocations CANNOT be voided (`400 CANNOT_VOID_PAID_BILL`).
- Administrators must first reverse all associated payments before voiding an issued bill.

### 7. Overdue Boundary Rule
- A bill becomes overdue strictly starting on the calendar day *after* its due date:
  $$\text{CURRENT\_DATE} > \text{due\_date} \implies \text{OVERDUE}$$
- On the due date itself ($\text{CURRENT\_DATE} = \text{due\_date}$), the bill remains `ISSUED`.

## Consequences
- Financial integrity guaranteed by ACID transaction boundaries, exact NUMERIC(12, 2) arithmetic, and row locks.
- Complete separation of billed amount vs. collected amount vs. outstanding balance.
- Bulletproof multi-tenant isolation and IDOR protection across payments and receipts.
