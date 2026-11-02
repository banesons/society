# 22 — Configurable Maintenance Billing & PDF Invoice Engine Architecture

## 1. System Purpose & Scope
This architecture defines the end-to-end subsystem for dynamic, configurable maintenance billing in CHS Neu. It decouples invoice structures from hardcoded business logic, allowing society administrators to author custom bill templates, define calculation formulas, configure SAC codes and GST classifications, apply unit-level overrides, execute selective or bulk billing batches, and generate print-ready PDFs with an official CHS watermark.

---

## 2. High-Level Architecture

```text
┌────────────────────────────────────────────────────────────────────────┐
│             SVELTEKIT 2 FRONTEND (Swiss Minimalist UI)                  │
│  - Bill Template Designer (Fields, Visibility, Layout, Watermark)      │
│  - Charge Heads & SAC Catalog Manager                                  │
│  - Interactive Formula Builder (AST Preview, Live Variable Validation) │
│  - Billing Operations (All Units, Selected Units Checkboxes, Single)   │
│  - Live Calculation Preview & PDF Viewer Modal                         │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ HTTPS REST (/api/v1/societies/{id}/billing/*)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                   GO 1.24 CHI BACKEND SERVICES                         │
│  ┌─────────────────────────────┐    ┌────────────────────────────────┐ │
│  │   Template & Config Engine  │    │   AST Formula Evaluator Engine │ │
│  │   - Field Order & Labels    │    │   - Lexer, Parser, AST Eval    │ │
│  │   - SAC / GST Classifications│   │   - Zero-Division & Cycle Proof│ │
│  └──────────────┬──────────────┘    └────────────────┬───────────────┘ │
│                 │                                    │                 │
│                 ▼                                    ▼                 │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                Billing Batch & Allocation Service                 │ │
│  │   - Unit Selector (All / Filtered / Single / Group)               │ │
│  │   - Unit Overrides Applier (Fixed / Area / Meter / Exemptions)    │ │
│  │   - Tax Splitter (CGST, SGST, IGST, Exempt Rules)                 │ │
│  │   - Arrears & Advance Adjustment Ledger                           │ │
│  └──────────────────────────────┬────────────────────────────────────┘ │
│                                 ▼                                      │
│  ┌───────────────────────────────────────────────────────────────────┐ │
│  │                  Pure Go PDF Invoice Generator                    │ │
│  │   - Vector A4 Layout Matching Indian Society Invoices             │ │
│  │   - Background Low-Opacity CHS Logo Watermark Layer               │ │
│  │   - Rupee Currency Number-to-Words Converter                      │ │
│  │   - Single PDF / Bulk Stream / ZIP Archive Exporter               │ │
│  └───────────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │ SQL (pgx/v5 Pooler)
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│                    SUPABASE POSTGRESQL CLUSTER                         │
│  - maintenance_bill_templates & fields                                 │
│  - maintenance_charge_heads (SAC, GST rules, accounting codes)         │
│  - maintenance_formula_definitions                                     │
│  - maintenance_unit_billing_overrides                                  │
│  - maintenance_billing_batches                                         │
│  - maintenance_bills & maintenance_bill_items (Immutable)              │
│  - platform_audit_logs (Transactional Audit Trail)                     │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Core Engine Specifications

### 3.1 Formula Engine (AST-Based)
Implemented in package `chs/backend/internal/formula`.
- **Pure Domain Engine**: Completely independent of HTTP routing, database layers, and external dependencies.
- **AST Architecture**:
  - `NumberNode`: Literal exact `Decimal` value.
  - `VariableNode`: Named identifier bound via `EvaluationContext`.
  - `UnaryNode`: Prefix operators (`+`, `-`).
  - `BinaryNode`: Arithmetic operators (`+`, `-`, `*`, `/`).
  - `ComparisonNode`: Relational expressions (`>`, `>=`, `<`, `<=`, `==`, `!=`). Returns `1` or `0`.
  - `ConditionalNode`: Ternary `if(cond, then, else)` with short-circuit evaluation.
  - `FunctionCallNode`: Whitelisted mathematical functions (`min`, `max`, `abs`, `round`, `floor`, `ceil`).
- **Numeric Semantics**:
  - Built on exact rational arithmetic (`math/big.Rat`). Zero IEEE-754 floating point drift during intermediate calculations.
  - Standard Half-Up rounding for monetary endpoints (`Round(places)`).
- **Security & Resource Guards**:
  - `MaxASTDepth`: Enforced at 30 levels to prevent stack exhaustion on deeply nested sub-expressions.
  - `MaxASTNodes`: Enforced at 200 nodes to reject excessively complex inputs.
  - Whitelist: Arbitrary function calls (`eval`, `exec`, `system`) are rejected with `UNKNOWN_FUNCTION`.
- **Dependency Resolution & Topological Ordering**:
  - `ResolveOrder(formulas)` constructs directed acyclic dependency graphs (DAG).
  - Detects cycles (e.g. `A -> B -> C -> A` or self-referential `A -> A`) and returns descriptive error `circular dependency detected: A -> B -> C -> A`.
  - Independent formula nodes are topologically sorted alphabetically for 100% reproducible execution order.
  - `EvaluateBatch(formulas, ctx)` sequentially calculates dependent formulas, feeding intermediate outputs to downstream consumers.
- **Benchmark Performance**:
  - Expression Parse & AST Build: **~1.65 µs/op** (Apple Silicon M2).
  - AST Evaluation: **~0.56 µs/op** (over 1.7 million evaluations/sec).

### 3.2 Tax & GST Model
- **Place of Supply**: Matches society state (default: intra-state CGST + SGST equal split).
- **Inter-State**: IGST applicability when member billing state differs from society state.
- **Exemptions**: Non-taxable heads (Sinking fund, municipal taxes, repair fund exempt under threshold) segregated from taxable heads.
- **Precision**: `NUMERIC(12, 2)` exact paise calculation with half-up rounding.

### 3.3 Watermark Specification
- **Asset**: Official CHS Logo (`backend/internal/billingpdf/assets/chs_logo.png` embedded from `frontend/static/icons/icon-512.png`).
- **Layer**: Drawn on background layer via `pdf.SetHeaderFunc()` before table grid and foreground content.
- **Opacity**: Configurable default 0.08 (8%) with alpha transparency (`pdf.SetAlpha(opacity, "Normal")`) to ensure crystal-clear text readability on screen and in print.
- **Coverage**: Applied uniformly to every page in single and multi-page invoices with 1:1 aspect ratio preservation.
- **Security**: Disabling watermark requires an explicit authorized template setting (`watermark_enabled: false`). Missing logo asset halts rendering safely.

### 3.4 PDF Invoice Generation Engine
Implemented in package `chs/backend/internal/billingpdf`.
- **Pure Go Rendering**: Zero CGo, zero headless browser/Chromium overhead. Built using `github.com/go-pdf/fpdf v0.9.0`. High throughput (<10ms per bill).
- **A4 Editorial Swiss Minimalist Layout**:
  - Off-white / white background with high contrast neutral typography.
  - Society header block (name, registration number, address, GSTIN).
  - Recipient block with member identity and unit details.
  - Tabular particulars: Sl No, Particulars/Charge Head, SAC, Tax Classification, Rate, GST, and Amount in INR.
  - Dual-column financial breakdown: Amount in words & society bank details on the left, line-by-line charges subtotal, CGST, SGST, IGST, arrears, interest, advance adjustments, and total payable on the right.
  - Terms & conditions and authorized signatory block.
- **Indian Financial Formatting**:
  - Decimal grouping matching Indian counting system (`FormatINR` produces `₹1,23,456.78` and `10,00,000.00`).
  - Indian English Number-to-Words (`NumberToWords` produces `Rupees Seven Thousand Three Hundred Sixty-Nine and Eighty Paise Only`).
  - Supports zero, negative adjustments, and amounts up to Crores.
- **Template Field Overrides**:
  - The renderer honors published template snapshots for field visibility and custom labels (`isFieldVisible`, `getFieldLabel`).
  - Preserves historical versioning: bills rendered years later use the published version snapshot active at their issuance.
- **Artifact Management & Packaging**:
  - Every generated PDF generates a 64-character SHA-256 checksum and metadata record in `maintenance_bill_pdf_artifacts`.
  - Batch generation bundles individual bills into a deterministic in-memory ZIP archive (`archive/zip`) naming files deterministically (e.g. `{billNumber}_{unitIdentifier}.pdf`).
  - Direct integration with Supabase Storage (`BucketSocietyDocuments`) with self-healing fallback to on-the-fly rendering when storage is offline.
- **Security & Multi-Tenancy**:
  - Strict society scoping on all artifact lookups.
  - RBAC: Residents can only access and download bills for units they actively own or occupy. Administrators can access all society bills.

### 3.5 Frontend Billing PDF & Artifact Integration (Phase 6.1)
The frontend connects the SvelteKit UI with Phase 5 PDF generation and binary delivery APIs:
- **Central API Client Integration (`src/lib/api/society.ts`)**:
  - `generateBillPdf(societyId, billId)`: Triggers PDF generation and returns artifact metadata.
  - `downloadBillPdf(societyId, billId)`: Streams binary PDF blob via `api.requestBlob()`.
  - `downloadResidentBillPdf(societyId, billId)`: Streams resident's own bill PDF blob from `/maintenance/my-bills/{id}/pdf`.
  - `generateBatchPdfZip(societyId, batchId)`: Triggers batch generation and ZIP artifact creation.
  - `downloadBatchZip(societyId, batchId)`: Streams batch ZIP binary archive.
  - `listBillArtifacts(societyId, billId)`: Retrieves immutable artifact audit records for a bill.
  - `getArtifactDetails(societyId, artifactId)`: Fetches artifact metadata.
  - `downloadArtifact(societyId, artifactId)`: Streams binary artifact by ID.
  - `listBillingBatches(societyId)` & `getBillingBatch(societyId, batchId)`: Batch tracking.
- **Client-Side Binary Delivery Utility (`src/lib/utils.ts`)**:
  - `triggerFileDownload(blob: Blob, filename: string)`: Safe, headless-compatible anchor dispatch and `URL.revokeObjectURL()` cleanup.
  - `formatBytes(bytes: number)`: Standardized file size display for artifacts.
- **Resident Bills Interface (`src/lib/components/resident/resident-bills-view.svelte`)**:
  - Direct PDF download buttons in both table row actions and the Bill Details modal.
  - Non-concurrent downloading indicator with disabled state prevention.
- **Admin Maintenance Console (`src/lib/components/admin/maintenance-console.svelte`)**:
  - Bills Register: One-click "PDF" download, "Generate" (or regenerate) PDF, and "Artifacts" modal launcher.
  - Bill Details Modal: Download, Regenerate, and Artifacts actions in modal footer.
  - Dedicated "Batches & ZIPs" Tab: Batch progress monitoring, target vs successful units, and "Download ZIP" / "Generate ZIP" actions.
- **Bill Artifacts Modal (`src/lib/components/admin/bill-artifacts-modal.svelte`)**:
  - Complete modal interface presenting artifact timeline, SHA-256 hash verification badge, MIME type, file size, watermark indicator, and direct download buttons.
- **Security & Multi-Tenant Boundary**:
  - Backend authorization remains strictly authoritative (handles 401/403/404).
  ### 3.6 PDF Preview & Billing Document UX (Phase 6.2)
Phase 6.2 delivers an in-browser PDF preview and refined document action UX across admin and resident surfaces:
- **PDF Viewing Engine (`src/lib/components/ui/pdf-preview-modal.svelte`)**:
  - Built with Mozilla's `pdfjs-dist` (v4) with web worker execution (`pdf.worker.min.mjs?url`) and fallback execution mode.
  - Multi-page document navigation with boundary-clamped pagination controls.
  - Granular zoom controls (0.4x to 3.0x) and automated Fit-to-Width / Fit-to-Page scaling.
  - HiDPI / Retina canvas scaling via `window.devicePixelRatio` for crystal-clear vector text.
  - Concurrency & memory management: active render cancellation (`activeRenderTask.cancel()`), document destruction (`pdfDoc.destroy()`), and cleanup on modal dismissal.
  - Keyboard accessibility: `Escape` dismisses viewer, `ArrowLeft` / `ArrowRight` / `PageUp` / `PageDown` flips pages, `+` / `-` adjusts zoom.
  - Mobile responsiveness: full-screen viewer with compact sticky control sub-bar ensuring zero horizontal overflow.
- **Admin Action Hierarchy (`src/lib/components/admin/maintenance-console.svelte`)**:
  - Bills table refactored to eliminate row clutter:
    - Primary Action: `Preview` (prominent button opening `PdfPreviewModal`).
    - Quick Action: `Download PDF` (icon button).
    - Compact Contextual Menu (`···`): Itemized Breakdown, Artifact History, Regenerate PDF, Void Bill.
  - Bill Details Modal: Prominent `Preview PDF` primary action in modal footer alongside download and regeneration.
- **Artifact History Inspection (`src/lib/components/admin/bill-artifacts-modal.svelte`)**:
  - Displays immutable generation history with explicit "Current Version" badge on the latest artifact.
  - Direct in-viewer `Preview` button for any historical or current PDF artifact without re-downloading.
  - Full audit metadata: generation timestamp, byte size, MIME type, SHA-256 checksum, watermark status.
- **Resident Experience (`src/lib/components/resident/resident-bills-view.svelte`)**:
  - Primary `View Bill` action opens the official PDF document directly in the in-browser viewer.
  - Quick `Download PDF` and `Breakdown` actions maintained with clean spacing.
  - Strict resident isolation: residents can only preview their own bills via `/maintenance/my-bills/{id}/pdf`.

### 3.7 Batch Execution Engine, Lifecycle & Idempotency (Phase 7.1)
Phase 7.1 introduces high-reliability batch execution, concurrency protection, idempotency enforcement, and partial failure isolation:

```text
                           ┌─────────────────┐
                           │  DRAFT_PREVIEW  │
                           └────────┬────────┘
                                    │ (enqueue / start)
                                    ▼
       ┌───────────────┐     ┌───────────────┐
       │    QUEUED     ├────►│  PROCESSING   │◄──────────────┐
       └───────┬───────┘     └───────┬───────┘               │
               │                     │                       │
               │ (cancel)            │ (eval & generate)     │ (retry failed)
               ▼                     ▼                       │
       ┌───────────────┐     ┌───────────────────────┐       │
       │   CANCELLED   │     │  COMPLETED_WITH_ERRORS├───────┘
       └───────────────┘     └───────┬───────────────┘
               ▲                     │
               │ (cancel)            │ (finalize)
               ├─────────────────────┼───────────────────────┐
               │                     ▼                       ▼
       ┌───────┴───────┐     ┌───────────────┐       ┌───────────────┐
       │    FAILED     │     │   GENERATED   ├──────►│   FINALIZED   │
       └───────▲───────┘     └───────────────┘       └───────────────┘
               │ (all failed)        │ (finalize)            (Immutable)
               └─────────────────────┘
```

- **Pessimistic Row-Level Concurrency Control**:
  - Execution (`/generate`, `/finalize`, `/retry`) wraps batch processing inside a PostgreSQL transaction that immediately issues `SELECT ... FROM maintenance_billing_batches WHERE id = $1 AND society_id = $2 FOR UPDATE`.
  - If another worker or process is currently running the batch (`status == 'PROCESSING'`), concurrent execution attempts are rejected with `409 Conflict` (`BATCH_ALREADY_PROCESSING`).
- **Idempotency Strategy**:
  - Deterministic batch key: `idempotency_key` enforced by unique constraint `uq_maint_batches_idempotency` per society.
  - Bill uniqueness boundary: PostgreSQL partial unique index `uq_idx_maintenance_bills_active_unit_period_type` on `(society_id, billing_period_id, unit_id, bill_type) WHERE status != 'VOID'` ensures that even in distributed execution, no unit receives duplicate bills for the same period.
  - Finalized immutability: PostgreSQL trigger `trg_enforce_bill_immutability` blocks any mutation to issued bills. Retries strictly preserve existing draft or issued bills and skip re-evaluating them.
- **Unit Failure Isolation & Bounded Retries**:
  - If calculation or member validation fails for an individual unit (e.g., malformed override, missing area variable), the engine catches the error, updates `maintenance_batch_units` (`status = 'FAILED'`, `error_message`, `error_code`, `retry_count`), and continues processing remaining units.
  - Batches with partial failures complete with `COMPLETED_WITH_ERRORS`.
  - The `/retry` endpoint queries solely `maintenance_batch_units WHERE batch_id = $1 AND status = 'FAILED'`, re-attempting only failed units without touching successful units.
- **Progress Tracking Metrics**:
  - The batch progress API exposes real-time counts: `target_unit_count`, `processed_unit_count`, `successful_unit_count`, `failed_unit_count`, `pending_unit_count`, alongside execution duration timestamps (`started_at`, `completed_at`, `finalized_at`).

### 3.8 Billing Automation Scheduler Architecture (Phase 7.2)

The Billing Automation Scheduler triggers automated recurring batch billing cycles per society without bypassing Phase 7.1's core batch generation engine.

```
┌────────────────────────────────┐
│   Scheduler Background Worker  │
│  (PostgreSQL Polling Ticker)   │
└───────────────┬────────────────┘
                │ 1. Claim due schedules (FOR UPDATE SKIP LOCKED)
                ▼
┌────────────────────────────────────────────────────────┐
│  maintenance_billing_schedules                         │
│  - next_run_at <= NOW()                                │
│  - status = 'ACTIVE'                                   │
│  - locked_by = worker_id, locked_until = NOW() + 5m   │
└───────────────┬────────────────────────────────────────┘
                │ 2. Derive intended period & idempotency key
                ▼
┌────────────────────────────────────────────────────────┐
│  Phase 7.1 Batch Execution Engine                      │
│  - Resolve/Create maintenance_billing_periods          │
│  - CreateBillingBatch(idempotencyKey)                  │
│  - GenerateBillingBatchDrafts()                        │
│  - FinalizeBillingBatch() (if auto_finalize = true)    │
└───────────────┬────────────────────────────────────────┘
                │ 3. Record outcome & advance schedule
                ▼
┌────────────────────────────────────────────────────────┐
│  maintenance_billing_schedule_executions (Audit log)   │
│  - status: SUCCESS / FAILED / SKIPPED                  │
│  - batch_id, error_code, started_at, completed_at      │
│  - advance next_run_at according to recurrence rules   │
└────────────────────────────────────────────────────────┘
```

#### Recurrence Semantics & Date Calculations
1. **Recurrence Intervals**:
   - `MONTHLY`: Generates period for current month `[YYYY-MM-01, YYYY-MM-LastDay]`.
   - `QUARTERLY`: Generates standard quarters `Q1: [Jan 1, Mar 31]`, `Q2: [Apr 1, Jun 30]`, `Q3: [Jul 1, Sep 30]`, `Q4: [Oct 1, Dec 31]`.
   - `ANNUAL`: Generates 1-year financial cycle based on society configuration.
2. **Date Safety**:
   - Timezone-aware date arithmetic using schedule's IANA timezone (default `Asia/Kolkata`).
   - Month-end dates safely handled via `time.Date(year, month+1, 0, ...)`, avoiding invalid date overflows on Feb 28/29 or 30-day months.
3. **Multi-Instance Coordination**:
   - Worker instances claim pending schedules via `WITH claimable AS (SELECT id ... FOR UPDATE SKIP LOCKED) UPDATE ... SET locked_by = $1, locked_until = NOW() + 5m`.
   - Prevents duplicate execution across clustered backend replicas.
   - Process crashes automatically self-heal when `locked_until` expires.
4. **Deterministic Idempotency**:
   - Unique execution key per period: `fmt.Sprintf("sched-%s-%s-%s", schedule.ID[:8], schedule.RecurrenceType, periodStart.Format("20060102"))`.
   - Prevents duplicate batch generation even if triggered simultaneously by timer and administrator manual run.

### 3.9 Billing Automation Operations & Reliability (Phase 7.3)

#### Operational Health Telemetry
The scheduler exposes aggregate and per-schedule diagnostics via `GET /billing/schedules/health`:
1. **Worker Liveness**: Real-time worker status (`RUNNING` / `STOPPED`), instance ID, and last poll timestamp.
2. **Threshold-Based Anomaly Detection**:
   - **Overdue Schedules**: Active schedules with `next_run_at < NOW() - INTERVAL '15 minutes'`.
   - **Stuck Executions**: Executions in `CLAIMED` or `PROCESSING` exceeding 15 minutes without completion.
   - **Expired Schedule Locks**: Dangling locks where `locked_until < NOW()` with non-null `locked_by`.
   - **Repeated Failures**: Consecutive failures recorded per schedule (`failure_count >= 3`).

#### Safe Administrator Recovery Workflows
```
┌─────────────────────────┐       ┌─────────────────────────┐
│   Expired Lock Detected │──────►│  POST /recover-lock     │──► Resets lock, leaves schedule intact
└─────────────────────────┘       └─────────────────────────┘

┌─────────────────────────┐       ┌─────────────────────────┐
│  Failed Unit Execution  │──────►│  POST /retry            │──► Reprocesses only failed units (Phase 7.1)
└─────────────────────────┘       └─────────────────────────┘

┌─────────────────────────┐       ┌─────────────────────────┐
│  Missed/Overdue Schedule│──────►│  POST /re-run-missed    │──► Assesses period idempotently, advances next run
└─────────────────────────┘       └─────────────────────────┘
```

1. **Lock Recovery**: Resets `locked_by` and `locked_until` after process crashes or worker restarts without mutating financial state.
2. **Execution Retry**: Safe retry reusing deterministic idempotency keys; prevents duplicate period assessments or overwriting finalized bills.
3. **Immutability Protection**: Retries against already `FINALIZED` batches are strictly rejected with `409 Conflict: BATCH_ALREADY_FINALIZED`.
4. **Alert Deduplication**: Alerts emit structured correlation IDs (`alert-billing-sched-{schedule_id}-{incident_type}-{hour}`) preventing notification storms during sustained operational degradation.

---

## 4. Related Documentation
- [[../decisions/047-configurable-billing-template-formula-engine-and-pdf|ADR 047]]
- [[../database/11-maintenance-foundation-schema|Database Schema 11]]
- [[../database/12-maintenance-operations-and-collections-schema|Database Schema 12]]
- [[../database/16-configurable-billing-schema|Database Schema 16 — Configurable Billing Schema]]
- [[../api/12-maintenance-foundation-api|API 12]]
- [[../api/18-configurable-billing-api|API 18 — Configurable Billing Engine API]]



