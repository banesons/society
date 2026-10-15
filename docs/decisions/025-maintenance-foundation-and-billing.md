# ADR 025: Maintenance Foundation and Billing Architecture

## Context & Problem Statement
Cooperative Housing Societies require an official, auditable financial foundation for recurring and special society maintenance dues. Historically, societies struggle with informal spreadsheets, arbitrary fee changes that retroactively distort past accounts, duplicate billing during concurrent administrative actions, and lack of line-item transparency for residents. 

CHS requires an official S11 Maintenance Foundation that establishes:
1. Charge categories (e.g., Monthly Maintenance, Sinking Fund, Parking, Repair Fund).
2. Charge configurations (flat recurring, unit-type specific, and one-off project charges).
3. Explicit billing periods with validated start, end, and due dates.
4. Concurrency-safe, idempotent bill generation creating immutable financial snapshots.
5. Line-item level transparency for both society administrators and residents.
6. Tenant isolation and fail-closed transactional audit logging.

Explicit non-goals for S11: payment gateway integration, payment collection, UPI/cards, receipts, payment allocation, bank reconciliation, and late-fee interest automation (all deferred to S12 or later).

## Decision Drivers
- **Financial Immutability**: Once a bill is issued, editing past charge configurations or category names must never mutate previously issued bills or line items.
- **Exact Precision**: Floating-point arithmetic is forbidden. Monetary values must use `NUMERIC(12, 2)` in PostgreSQL and exact fractional/integer arithmetic in application logic.
- **Idempotency & Concurrency**: Multiple administrators generating bills simultaneously or repeating requests after network timeouts must never produce duplicate bills for the same unit and period.
- **Derived Lifecycle**: Avoid brittle background cron jobs to update bills to "OVERDUE". Overdue status must be cleanly derived from `CURRENT_DATE > due_date` when status is `ISSUED`.
- **Tenant Isolation & Security**: Platform ADMIN has no implicit society access. Residents can only view bills for units where they hold an active, verified ownership or occupancy relationship.

## Architecture & Domain Model

### 1. Entity Hierarchy
```text
Society
 ├── maintenance_charge_categories (Society-scoped charge heads)
 ├── maintenance_charge_configs    (Active & historical charging rules)
 ├── maintenance_billing_periods   (Temporal billing windows with due dates)
 └── maintenance_bills             (Issued unit bills)
      └── maintenance_bill_items   (Immutable line-item financial snapshots)
```

### 2. Charge Models (S11 Scope)
- `FLAT_RECURRING`: Standard recurring fee applied equally to all active units in the society.
- `UNIT_TYPE_RECURRING`: Specific fee applied only to units of a matching `unit_type` (e.g., `COMMERCIAL` surcharge, `PENTHOUSE` rate).
- `ONE_OFF`: One-time fee valid specifically for a targeted billing period or date window.

### 3. Financial Precision & Calculations
- All monetary amounts in the database are stored as `NUMERIC(12, 2)` (scale 2, precision 12), supporting values up to ₹9,999,999,999.99 with exact paise precision.
- In Go, amounts are parsed, validated (`amount > 0`), and calculated using exact decimal addition.
- `subtotal` = Sum of all bill item amounts.
- `total_amount` = `subtotal` (no tax or late fees in S11).
- The client NEVER computes or submits monetary totals; all calculations are server-authoritative.

### 4. Bill Number Generation Strategy
To avoid sequence bottlenecks and concurrent duplication, bill numbers follow a deterministic, unique format:
```text
BILL-{SOC_CODE_SHORT}-{YYYYMM}-{UNIT_IDENTIFIER}
```
Example: `BILL-GVR-202610-A-101`
Enforced by database unique constraint `UNIQUE (society_id, bill_number)`.

### 5. Idempotent Bill Generation Workflow
Bill generation executes within a single database transaction (`tx`):
1. Acquires a transaction-level lock or checks `maintenance_billing_periods` status (`status != 'CLOSED'`).
2. Fetches all active units in the society (`status = 'ACTIVE'`).
3. Fetches existing bills for the period to filter out already-billed units.
4. Resolves applicable active charge configurations where `effective_from <= period_end` and (`effective_until IS NULL` or `effective_until >= period_start`).
5. For each unbilled unit, evaluates matching charges, generates immutable `maintenance_bill_items`, and inserts the `maintenance_bills` record.
6. Writes a `BILLS_GENERATED` audit entry to `platform_audit_logs`.
7. Database constraint `UNIQUE (society_id, billing_period_id, unit_id)` provides a fail-safe against concurrent insert collisions.

### 6. Bill Lifecycle State Machine
- `DRAFT`: Bill drafted but not yet finalized (optional).
- `ISSUED`: Active, authoritative bill awaiting resident payment.
- `VOID`: Nullified by administrator with documented `void_reason` and `voided_by`.
- *Overdue State*: Derived dynamically when `status = 'ISSUED'` AND `CURRENT_DATE > due_date`.

## Consequences
- **Positive**: Complete auditability and financial snapshot immutability. Zero data corruption from retroactively changed configurations.
- **Positive**: Strict tenant isolation via composite foreign keys `(society_id, ...)` across all five maintenance tables.
- **Negative**: Historical charge changes require creating new configuration records with updated effective dates rather than mutating active records in place.
