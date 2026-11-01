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
- **Asset**: Official CHS Logo (`frontend/static/icons/icon-512.png` / vector geometry).
- **Layer**: Drawn on background layer before table grid and text.
- **Opacity**: Configurable default 0.08 (8%) to prevent obscuring numerical amounts.
- **Coverage**: Applied uniformly to every page in multi-page invoices.

---

## 4. Related Documentation
- [[../decisions/047-configurable-billing-template-formula-engine-and-pdf|ADR 047]]
- [[../database/11-maintenance-foundation-schema|Database Schema 11]]
- [[../database/12-maintenance-operations-and-collections-schema|Database Schema 12]]
- [[../api/12-maintenance-foundation-api|API 12]]
