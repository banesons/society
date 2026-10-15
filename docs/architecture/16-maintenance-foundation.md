# 16 — Maintenance Foundation Architecture

## 1. Overview
Sprint S11 introduces the financial maintenance backbone of the CHS platform. It provides society administrators with the tools to configure charge heads, establish billing cycles, and execute idempotent bill generation runs, while giving residents direct, line-item visibility into their maintenance obligations.

---

## 2. Core Subsystems

```text
┌────────────────────────────────────────────────────────────────────────┐
│                      S11 MAINTENANCE FOUNDATION                        │
├────────────────────────────────┬───────────────────────────────────────┤
│    Admin Configuration Engine  │       Bill Generation & Register      │
│  - Charge Categories           │  - Billing Periods (Dates & Due)      │
│  - Flat & Unit-Type Charges    │  - Idempotent Generation Engine       │
│  - Effective Dates & Status    │  - Immutable Snapshot Line Items      │
│                                │  - Void Workflow & Audit Trail        │
├────────────────────────────────┴───────────────────────────────────────┤
│                       Resident Access Pipeline                         │
│  - Active Unit Relationship Authorization                              │
│  - Chronological Bill History & Real-Time Overdue Badges               │
│  - Line-Item Itemized Breakdown Modal (Zero Payment UI in S11)         │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Financial Precision & Exact Arithmetic
- Currency: Indian Rupee (`INR`).
- Precision: 2 decimal places (paise precision).
- Storage: `NUMERIC(12, 2)` across `amount`, `rate`, `quantity`, `subtotal`, and `total_amount`.
- Rounding: Standard half-up rounding where divisions occur. All addition sums are exact.
- Client Trust: The backend strictly rejects client-supplied totals and computes all sums directly from matched charge line items.

---

## 4. Immutability & Financial Snapshots
A core tenet of CHS financial architecture is **Historical Immutability**:
- When an administrator modifies a charge category (e.g., changes "Monthly Maintenance" to "Society Maintenance") or updates an amount (e.g., changes ₹2,000 to ₹2,500), this edit only impacts future bill generation runs.
- Every `maintenance_bill_items` row persists:
  - `category_name`: Literal string snapshot at issuance time.
  - `description`: Literal charge description snapshot.
  - `rate` and `quantity`: Exact multiplier inputs.
  - `amount`: Exact line-item subtotal.
- Once a bill transitions to `ISSUED`, its line items cannot be updated or deleted. To correct errors, administrators must void the bill with an explicit reason and reissue.

---

## 5. Billing Period & Concurrency Management
1. **Period Definition**:
   - `period_start` <= `period_end`.
   - `due_date` >= `period_start`.
   - Unique per society: `(society_id, period_start, period_end)`.
2. **Concurrency Control**:
   - Bill generation uses transactional isolation with composite unique constraints on `(society_id, billing_period_id, unit_id)`.
   - Concurrent generation runs cannot insert duplicate bills for the same unit in the same period; one succeeds and the other skips or safely fails with an idempotency guarantee.

---

## 6. Derived Status & Lifecycles
Bills use a simple, robust lifecycle model:
- `DRAFT`: Bill drafted but not formally issued.
- `ISSUED`: Active official bill.
- `VOID`: Annulled bill with recorded `void_reason`, `voided_by`, and `voided_at`.

### Derived Display Status:
```text
if status == 'VOID' -> VOID
if status == 'ISSUED':
    if CURRENT_DATE > due_date -> OVERDUE
    else -> ISSUED
```
No brittle cron jobs are required to update rows every midnight.

---

## 7. Resident Authorization
Resident access to maintenance bills is governed by verified unit relationships (`unit_relationships`):
- A resident user can only view bills where `unit_relationships` has:
  - `status = 'ACTIVE'`
  - `sm.user_id = <authenticated_user_id>`
  - `ur.is_occupant = true` OR `ur.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER')`
- Cross-unit URL tampering (`/bills/{billId}`) returns 404/403.

---

## 8. Related Documentation
- [[../decisions/025-maintenance-foundation-and-billing|ADR 025: Maintenance Foundation and Billing]]
- [[../database/11-maintenance-foundation-schema|Database Schema 11: Maintenance Foundation Schema]]
- [[../api/12-maintenance-foundation-api|API 12: Maintenance Foundation API]]
- [[../security/11-maintenance-foundation-security|Security 11: Maintenance Foundation Security]]
- [[../sprints/S11-maintenance-foundation|Sprint S11 Tracker]]
