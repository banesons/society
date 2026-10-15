# S16 — Dashboard & Reporting Specification

## 1. Executive Summary & Objective
S16 provides actionable operational, financial, and compliance intelligence across CHS without compromising tenant isolation, data minimization, or transaction performance. It introduces role-tailored dashboard views for:
1. **Platform Administrators (`SUPER_ADMIN`, `ADMIN`)**: Platform-wide health, society growth, registration pipelines, system throughput, and audit volume.
2. **Society Administrators (`SOC_ADMIN`)**: Society occupancy, resident counts, pending approvals, overdue collections, active visitors, and announcement reach.
3. **Committee Officers (Secretary, Treasurer, Chairman)**: Dedicated financial recovery scorecards, ledger summaries, and security shift coverage.
4. **Gatekeepers (`GATEKEEPER`)**: Immediate shift metrics (active visitors on premises, today's check-ins, pending resident approvals).

---

## 2. Metric Catalog & Calculation Rules

### 2.1 Platform Dashboard Metrics (Platform-Wide)
| Metric Name | Calculation Rule / SQL Formula | Source Tables & Columns | Schema Support & Gaps |
| :--- | :--- | :--- | :--- |
| **Total Societies** | `COUNT(*)` grouped by `status` (`ACTIVE`, `PENDING`, `SUSPENDED`) | `societies.status` | **Supported**: Indexed by `status`. |
| **New Societies (30d)** | `COUNT(*) WHERE created_at >= NOW() - INTERVAL '30 days'` | `societies.created_at` | **Supported**. |
| **Total Registered Units** | `COUNT(*)` across all societies | `society_units.id` | **Supported**. |
| **Global Member Accounts** | `COUNT(DISTINCT user_id) WHERE status = 'ACTIVE' AND user_id IS NOT NULL` | `society_members.user_id`, `status` | **Supported**. |
| **Platform Financial Velocity (30d)** | `ROUND(COALESCE(SUM(amount), 0) * 100)::BIGINT WHERE status != 'REVERSED' AND payment_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date - INTERVAL '30 days'` | `maintenance_payments.amount`, `status`, `payment_date` | **Supported**: Stored as `NUMERIC(12,2)`, returned as integer paise. Excludes reversed payments. |
| **Platform Audit Events (24h)** | `COUNT(*) WHERE created_at >= NOW() - INTERVAL '24 hours'` | `platform_audit_logs.created_at` | **Supported**: Covered by `idx_platform_audit_logs_action_created`. |

### 2.2 Society Operational & Occupancy Metrics (Per Society)
| Metric Name | Calculation Rule / SQL Formula | Source Tables & Columns | Schema Support & Gaps |
| :--- | :--- | :--- | :--- |
| **Total Units** | `COUNT(*)` where `society_id = $soc_id AND status = 'ACTIVE'` | `society_units.society_id`, `status` | **Supported**. |
| **Occupancy Rate (%)** | `(COUNT(DISTINCT ur.unit_id) * 100.0) / NULLIF(COUNT(DISTINCT u.id), 0)` where `ur.status = 'ACTIVE' AND ur.is_occupant = true` | `society_units`, `unit_relationships` | **Supported**: Occupancy is derived from active `unit_relationships.is_occupant = true` (not `society_units.status`). |
| **Total Active Members** | `COUNT(*) WHERE society_id = $soc_id AND status = 'ACTIVE'` | `society_members.status` | **Supported**. |
| **Owner vs Tenant Units** | `COUNT(DISTINCT unit_id)` grouped by relationship prefix (`PRIMARY_OWNER`/`SECONDARY_OWNER`/`OWNER_FAMILY` vs `PRIMARY_TENANT`/`SECONDARY_TENANT`/`TENANT_FAMILY`/`AUTHORIZED_OCCUPANT`) | `unit_relationships.relationship_type`, `status = 'ACTIVE'`, `is_occupant = true` | **Supported**: Uses `unit_relationships` (not `society_unit_relationships`). |
| **Active Announcements** | `COUNT(*) WHERE society_id = $soc_id AND status = 'PUBLISHED'` | `society_announcements.status` | **Supported**: `expires_at` does not exist; publication status determines activity. |

### 2.3 Gate Operations & Footfall Metrics (Per Society)
| Metric Name | Calculation Rule / SQL Formula | Source Tables & Columns | Schema Support & Gaps |
| :--- | :--- | :--- | :--- |
| **Visitors Currently On-Premises** | `COUNT(*) WHERE society_id = $soc_id AND status = 'CHECKED_IN'` | `society_visits.status` | **Supported**: Status `CHECKED_IN` indicates entry without exit. |
| **Today's Total Entries** | `COUNT(*) WHERE society_id = $soc_id AND check_in_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')` | `society_visits.check_in_at` | **Supported**: Column is `check_in_at` (not `checked_in_at`). |
| **Today's Total Exits** | `COUNT(*) WHERE society_id = $soc_id AND check_out_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')` | `society_visits.check_out_at` | **Supported**: Column is `check_out_at` (not `checked_out_at`). |
| **Pending Resident Approvals** | `COUNT(*) WHERE society_id = $soc_id AND status = 'PENDING_APPROVAL'` | `society_visits.status` | **Supported**. |
| **Active Guards On Duty** | `COUNT(DISTINCT user_id) WHERE society_id = $soc_id AND status = 'ACTIVE' AND ended_at IS NULL` | `society_gate_sessions.user_id`, `status`, `ended_at` | **Supported**: Table is `society_gate_sessions` (not `society_gate_staffing`). |

### 2.4 Maintenance & Collections Financial Metrics (Per Society)
| Metric Name | Calculation Rule / SQL Formula | Source Tables & Columns | Schema Support & Gaps |
| :--- | :--- | :--- | :--- |
| **Current Period Billed (Paise)** | `ROUND(COALESCE(SUM(total_amount), 0) * 100)::BIGINT WHERE society_id = $soc_id AND billing_period_id = $curr_period_id AND status = 'ISSUED'` | `maintenance_bills.total_amount`, `status`, `billing_period_id` | **Supported**: Status is `'ISSUED'` (not `'PAID'` or `'PARTIALLY_PAID'`). Amount stored as NUMERIC(12,2). |
| **Current Period Collected (Paise)** | `ROUND(COALESCE(SUM(a.allocated_amount), 0) * 100)::BIGINT FROM maintenance_payment_allocations a JOIN maintenance_bills b ON a.bill_id = b.id WHERE b.society_id = $soc_id AND b.billing_period_id = $curr_period_id AND a.status = 'ACTIVE'` | `maintenance_payment_allocations.allocated_amount`, `status`, `maintenance_bills.billing_period_id` | **Supported**: Stored as `NUMERIC(12,2)` on allocations. Reversals excluded via `a.status = 'ACTIVE'`. |
| **Collection Efficiency (%)** | `(Collected Paise * 100.0) / NULLIF(Billed Paise, 0)` | Computed from bills & allocations | **Supported**. |
| **Total Overdue Arrears (Paise)** | `ROUND(COALESCE(SUM(GREATEST(0.00, b.total_amount - COALESCE(alloc.paid, 0.00))), 0) * 100)::BIGINT WHERE b.society_id = $soc_id AND b.status = 'ISSUED' AND b.due_date < (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date` | `maintenance_bills.due_date`, `status`, `total_amount`, joined with active allocations sum | **Supported**: Calculates remaining dues by subtracting active allocations from issued bills. |
| **Payments Collected (Month-to-Date)**| `ROUND(COALESCE(SUM(amount), 0) * 100)::BIGINT WHERE society_id = $soc_id AND status = 'RECORDED' AND payment_date >= DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date` | `maintenance_payments.amount`, `status`, `payment_date` | **Supported**: Status `'RECORDED'` excludes `'REVERSED'`. |

---

## 3. Date, Timezone, and Monetary Conventions
1. **Timezone Anchor**: Standard timezone for all cooperative housing societies in CHS is Indian Standard Time (`Asia/Kolkata`, UTC+05:30).
   - SQL queries grouping by day/month must truncate using: `DATE_TRUNC('day', timestamp AT TIME ZONE 'Asia/Kolkata')`.
2. **Monetary Storage & Transport**:
   - Stored and queried strictly in **paise** (integers). Never floating-point rupees in SQL.
   - API JSON returns:
     ```json
     { "amount_paise": 250000, "formatted_inr": "₹2,500.00" }
     ```
3. **Date Range Filters**:
   - `today`: `[start_of_day_ist, end_of_day_ist]`
   - `7d`: `NOW() - INTERVAL '7 days'`
   - `30d`: `NOW() - INTERVAL '30 days'`
   - `mtd`: Month-to-date starting 1st of current month in IST.
   - `ytd`: Year-to-date starting April 1st (Indian financial year) or Jan 1st.

---

## 4. Role-Based Access Control (RBAC) Matrix

| Endpoint / View | `SUPER_ADMIN` | `ADMIN` (Platform) | `SOC_ADMIN` | Secretary / Treasurer | `GATEKEEPER` | `RESIDENT` |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **Platform Overview Dashboard** | **ALLOW** | **ALLOW** | DENY | DENY | DENY | DENY |
| **Society Overview KPI Hub** | **ALLOW** | **ALLOW** (Scoped) | **ALLOW** | **ALLOW** | DENY | DENY |
| **Financial & Dues Scorecard** | **ALLOW** | **ALLOW** (Scoped) | **ALLOW** | **ALLOW** | DENY | DENY |
| **Gate Operations Dashboard** | **ALLOW** | **ALLOW** (Scoped) | **ALLOW** | **ALLOW** | **ALLOW** | DENY |
| **Audit Activity Ledger Summary** | **ALLOW** | **ALLOW** (Scoped) | **ALLOW** | DENY | DENY | DENY |
| **Unit Personal Dues (Resident)**| DENY | DENY | DENY | DENY | DENY | **ALLOW** (Own Unit) |

---

## 5. Tenant Isolation & Security Architecture
1. **Explicit Society ID Binding**: All society metrics queries strictly bind `WHERE society_id = $1`.
2. **Role Verification Middleware**: `RequireSocietyRole(db, "SOC_ADMIN")` or designations middleware prevents cross-society data leakage.
3. **Data Minimization in Summary APIs**:
   - Dashboards return aggregated counts, rates, and sums.
   - PII (names, phone numbers, flat owner identities) is never loaded or exposed in aggregate metrics endpoints.
4. **Anti-Tampering & Read-Only Execution**: Dashboard reporting queries are strictly read-only (`SELECT`) without row mutation.

---

## 6. Query and Index Performance Analysis
- **Existing Indexes Utilized**:
  - `idx_society_units_society` on `society_units(society_id)`
  - `idx_society_members_society` on `society_members(society_id, status)`
  - `idx_society_visits_society` on `society_visits(society_id, status, checked_in_at)`
  - `uq_maintenance_bills_id_society` on `maintenance_bills(society_id)`
  - `idx_maintenance_payments_society` on `maintenance_payments(society_id, payment_date)`
  - `idx_platform_audit_logs_society_created` on `platform_audit_logs(society_id, created_at DESC)`
- **Optimization Strategy**:
  - All single-society dashboard queries run in sub-50ms using indexed scans.
  - Queries avoid full table scans by bracketing time filters (`created_at >= NOW() - INTERVAL '30 days'`).
  - No new schema migration is required. Existing database structures accurately support all defined KPIs.

---

## 7. Proposed API Contracts

### 7.1 Platform Dashboard
`GET /api/v1/platform/dashboard/summary`
```json
{
  "success": true,
  "data": {
    "societies": { "total": 12, "active": 10, "pending": 1, "suspended": 1 },
    "members": { "total_active_accounts": 1420 },
    "collections_30d_paise": 48500000,
    "audit_events_24h": 348
  }
}
```

### 7.2 Society Dashboard Overview
`GET /api/v1/societies/{id}/dashboard/summary`
```json
{
  "success": true,
  "data": {
    "occupancy": {
      "total_units": 120,
      "occupied_units": 108,
      "vacant_units": 12,
      "occupancy_rate_percent": 90.0
    },
    "gate": {
      "visitors_on_premises": 4,
      "today_entries": 32,
      "pending_approvals": 1
    },
    "finance": {
      "current_period_name": "October 2026",
      "billed_paise": 60000000,
      "collected_paise": 51000000,
      "collection_efficiency_percent": 85.0,
      "overdue_arrears_paise": 9000000
    },
    "communication": {
      "active_announcements_count": 2
    }
  }
}
```

### 7.3 Gate Shift Operations Summary
`GET /api/v1/societies/{id}/gate/dashboard`
```json
{
  "success": true,
  "data": {
    "active_visitors_count": 4,
    "today_checkins_count": 32,
    "today_checkouts_count": 28,
    "pending_approvals_count": 1,
    "active_guards_count": 2
  }
}
```

---

## 8. Frontend Component & Visual Route Plan

Following the established **Swiss Editorial Typography & Architectural Hierarchy**:
- Clean stark monochromatic grays (`neutral-950`, `neutral-500`, `#f3f3f1`).
- High-contrast numerical scorecards with tabular numerals (`font-mono tracking-tight`).
- No neon or distracting chart gradients; minimal clean bar sparklines and progress rings.

### Routes
1. `/admin/dashboard`: Platform overview mounted in Platform Admin console.
2. `/society/[id]/dashboard`: Society operations & financial overview.
3. `/society/[id]/gate/console`: Embedded gate operator real-time dashboard widget.

### Components
- `kpi-card.svelte`: Compact metric scorecard with title, primary value, trend badge, and subtitle.
- `collection-gauge.svelte`: Clean progress bar displaying billing vs recovery percentage.
- `visitor-footfall-widget.svelte`: Real-time entry counter with live badge.
- `occupancy-distribution.svelte`: Unit breakdown by owner-occupied, tenant-occupied, and vacant.

---

## 9. Empty States, Error Handling, and Loading States
- **Loading**: Skeleton shimmer matching exact card proportions (`animate-pulse bg-neutral-200/60`).
- **Empty States**:
  - Newly onboarded society with 0 units: *"Society structure pending configuration — add units in structure settings."*
  - Zero billing periods: *"No billing periods generated — generate bills in maintenance foundation."*
  - Zero visitors today: *"No gate visitors recorded for today."*
- **Error States**: Inline error card with retry trigger (`Try again`) without unmounting shell layout.

---

## 10. Automated Tests & Acceptance Criteria Matrix

| ID | Criterion | Verification Target | Status | Verified Evidence |
| :--- | :--- | :--- | :---: | :--- |
| **AC-S16-01** | Platform metrics endpoint | `GET /api/v1/platform/dashboard/summary` restricted to `SUPER_ADMIN`/`ADMIN`. | **PASS** | `TestDashboardReportingSuite/Platform_Dashboard_-_SuperAdmin_Success` (200 OK); Non-admin rejected with `403 FORBIDDEN`; Anonymous rejected with `401 UNAUTHORIZED`. Live verified in `cmd/s16_acceptance`. |
| **AC-S16-02** | Society dashboard summary endpoint | `GET /api/v1/societies/{id}/dashboard/summary` scoped by `society_id`. | **PASS** | `TestDashboardReportingSuite/Society_Dashboard_-_Occupancy_Calculations` (200 OK); empty state verified; live verified in `cmd/s16_acceptance` against active society ID. |
| **AC-S16-03** | Gate shift metrics endpoint | `GET /api/v1/societies/{id}/gate/dashboard` with active visitor counts. | **PASS** | `TestDashboardReportingSuite/Gate_Dashboard_-_Gatekeeper_and_SocAdmin_Access` (200 OK for both roles); live verified in `cmd/s16_acceptance`. |
| **AC-S16-04** | Monetary precision & formatting | All financial sums return exact integer paise with correct INR formatting. | **PASS** | `TestFormatINR` verified across 8 scales; `TestFinancialCorrectnessAndModuleComparison` verified Billed (300,000 paise = ₹3,000.00), Collected (150,000 paise = ₹1,500.00), Overdue (150,000 paise = ₹1,500.00). |
| **AC-S16-05** | Timezone IST alignment | Daily and monthly aggregations bracket at midnight Asia/Kolkata. | **PASS** | `TestTimezoneBoundaries` verified gate entry midnight IST cutoff (+1s included, -1s excluded), overdue due_date boundary (today not overdue, yesterday overdue), and MTD month boundary (1st of month included, previous month excluded). |
| **AC-S16-06** | Tenant isolation enforcement | Calling society dashboard with cross-society token yields `403 FORBIDDEN`. | **PASS** | `TestCrossSocietyAuthorization` verified multi-tenant matrix: assigned SOC_ADMIN A accesses Soc A (200 OK), denied Soc B (`403 FORBIDDEN`); unassigned GATEKEEPER A denied Soc B (`403 FORBIDDEN`); GATEKEEPER denied Soc Dashboard (`403 FORBIDDEN`); missing token yields `401 UNAUTHORIZED`. |
| **AC-S16-07** | Read-only non-mutating execution | Running dashboard queries mutates 0 rows in PostgreSQL. | **PASS** | `TestDashboardReportingSuite/Non-Mutating_Execution` verified audit log count unchanged across consecutive dashboard queries. |
| **AC-S16-08** | Frontend society dashboard view | Responsive layout with KPI cards, collection efficiency, and footfall counters. | **PASS** | Verified via browser subagent with visual evidence: `admin_platform_dashboard_loaded_1791982768855.png` and `society_dashboard_authenticated_1791982700263.png`. |
| **AC-S16-09** | Quality gates passing | `gofmt`, `go vet`, `go test -count=1 ./...`, `npm run check`, `npm run lint`, `npm test`, `npm run build`. | **PASS** | All Go tests clean (`ok chs/backend/internal/server 2.987s`), `gofmt -s -w .` and `go vet ./...` clean; `npm test` 153/153 passed; `npm run check` 0 errors; `npm run lint` 0 errors; `npm run build` production bundle succeeded in 5.49s. |
| **AC-S16-10** | Zero schema migrations | Completed without schema mutation or data truncation. | **PASS** | Existing schema verified 100% sufficient; zero migrations added; full audit ledger of 5,701 events strictly preserved. |

---

## 11. Related Documentation
- [[../decisions/022-society-application-shell-and-navigation|ADR 022: Society Shell Navigation]]
- [[../decisions/025-maintenance-foundation-and-billing|ADR 025: Maintenance Foundation]]
- [[../architecture/03-platform-administration|Architecture 03: Platform Administration]]
- [[../architecture/14-gate-visitor-operations|Architecture 14: Gate Visitor Operations]]
- [[../architecture/17-maintenance-operations-and-collections|Architecture 17: Maintenance Collections]]
- [[../sprints/S15-notifications|Sprint S15: Notifications]]
- [[../api/17-dashboard-reporting-api|API 17: Dashboard & Reporting API Contracts]]

---

## 12. Status
COMPLETE (Implementation, automated test suite, live API acceptance, cross-tenant isolation, financial module parity, and browser visual verification verified).

### Verification Evidence
1. **Backend Tests**: `go test -count=1 ./...` passed across all packages (`society`, `server`, `auth`, `middleware`, `notification`, `platform`, `response`).
   - `TestCrossSocietyAuthorization`: Verified assigned SOC_ADMIN accesses own society (200 OK) but denied other societies (403 Forbidden); GATEKEEPER accesses permitted gate metrics (200 OK) but denied other societies (403 Forbidden) and society dashboard (403 Forbidden); SUPER_ADMIN maintains governance access; missing auth yields 401 Unauthorized.
   - `TestFinancialCorrectnessAndModuleComparison`: Proved VOID bills excluded, REVERSED allocations excluded, partial allocations correctly tracked, paise/INR alignment exact, and 100% mathematical parity with `societyRepo.GetCollectionSummary`.
   - `TestTimezoneBoundaries`: Proved IST midnight bracketing for gate entries, due date boundaries for overdue arrears, and month boundaries for MTD collections.
2. **Frontend Quality Gates**:
   - `npm run check`: 0 errors.
   - `npm run lint`: Prettier and ESLint clean with 0 errors.
   - `npm test`: 153 tests passed across 11 test files (including Vitest dashboard client tests).
   - `npm run build`: Production bundle built cleanly in 5.49s.
3. **Live Acceptance**: `cmd/s16_acceptance/main.go` verified live endpoints against Supabase development instance:
   - AC-S16-01: Platform dashboard summary (`/api/v1/platform/dashboard/summary`) with Super Admin auth.
   - AC-S16-02: Society dashboard summary (`/api/v1/societies/{id}/dashboard/summary`) with tenant-isolated data.
   - AC-S16-03: Gate dashboard summary (`/api/v1/societies/{id}/gate/dashboard`) with active shift metrics.
   - AC-S16-06: 401/403 authorization and tenant isolation checks.
4. **Visual & Browser Verification**:
   - Platform Dashboard verified at `http://localhost:5173/admin` (`admin_platform_dashboard_loaded_1791982768855.png`).
   - Society Dashboard verified at `http://localhost:5173/society/f5eded8a-5eca-4e3e-9673-cbc092ab06a3#reports` (`society_dashboard_authenticated_1791982700263.png`).
   - Browser interaction session recorded (`s16_dashboard_acceptance_1791982552588.webp`, `s16_audit_browser_verification_1791987643268.webp`).
5. **Database Preservation & Integrity**:
   - Confirmed shared database target across acceptance and verification harnesses (`DATABASE_URL` in `backend/.env`).
   - Verified active dataset intact: 576 societies (522 active, 54 pending), 733 registered units, 128 active member accounts, 5,701 platform audit logs.
   - 0 schema migrations added, 0 records deleted or truncated.
