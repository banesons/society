# 12. Maintenance Operations & Collections Security

## 1. Authentication & Tenancy Enforcement

All endpoints enforce society boundary through `RequireSocietyAccess` for administrative actions and `RequireSocietyMemberOrAdminAccess` for resident actions.

| Action | Allowed Roles | Tenancy Constraint |
| :--- | :--- | :--- |
| Record Payment | `SOC_ADMIN`, `SUPER_ADMIN` | Actor must hold active assignment in target society |
| Reverse Payment | `SOC_ADMIN`, `SUPER_ADMIN` | Actor must hold active assignment in target society |
| View Collections Register | `SOC_ADMIN`, `SUPER_ADMIN` | Scoped strictly to target `society_id` |
| View Receipt (Admin) | `SOC_ADMIN`, `SUPER_ADMIN` | Must belong to target `society_id` |
| View Resident Payments | `RESIDENT` | Scoped to active unit relationships for `user_id` |
| View Resident Receipt | `RESIDENT` | Scoped to active unit relationships; cross-unit returns `403` |

Platform `ADMIN` without explicit society assignment receives `403 FORBIDDEN`.

## 2. IDOR Prevention
1. Cross-Society IDOR:
   Every SQL mutation and selection includes `WHERE society_id = $1`. Even with a valid UUID from another society, the record is unreachable.
2. Cross-Unit IDOR:
   Resident endpoints query `society_unit_relationships` to confirm that the requesting user has an active `OWNER` or `TENANT` relationship to the unit linked with the bill or payment. If not, the request fails closed with `403 FORBIDDEN`.

## 3. Financial Integrity & Immutability
1. Database triggers enforce that once issued, a bill's `subtotal`, `total_amount`, and line items cannot be changed.
2. Allocations can only be created via the transactional recording workflow, which verifies `amount <= outstanding_balance` under a row-level lock (`SELECT ... FOR UPDATE`).
3. Reversals never delete records; they update status flags and log complete audit trail entries.

## 4. Audit Logging
Every financial mutation is recorded in `platform_audit_logs`:
- `MAINTENANCE_PAYMENT_RECORDED`
- `MAINTENANCE_PAYMENT_REVERSED`
- `MAINTENANCE_RECEIPT_ISSUED`
- `MAINTENANCE_BILL_VOID_REJECTED` (if void attempted on paid bill)
Each audit entry captures actor ID, society ID, client IP, user agent, and non-sensitive metadata (amount, references, reason).
