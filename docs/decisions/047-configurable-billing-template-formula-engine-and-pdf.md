# ADR 047: Configurable Maintenance Billing, Formula Engine, and PDF Invoice Generator

## Status
Accepted

## Context
Cooperative Housing Society (CHS) Neu previously operated on Sprint S11/S12 maintenance foundation, which supported fixed flat and unit-type recurring charges. Real-world housing and commercial premises societies require:
1. **Dynamic Bill Templates**: Configurable layouts, customizable charge heads, SAC codes, tax classifications, bank instructions, and watermarks.
2. **Deterministic Formula Engine**: Complex charge calculation without code deployment (e.g., `area * maintenance_rate`, `consumption * rate`, non-occupancy percentages, interest on arrears).
3. **Tax & GST Compliance**: Separation of taxable (CGST, SGST, IGST) vs. exempt charges (sinking fund, municipal tax), place of supply, and reverse charge.
4. **Flexible Billing Runs**: Generation across all units, selected units (checkboxes), particular unit, or unit categories (residential/commercial).
5. **Unit-Specific Overrides**: Unit-level adjustments, rate overrides, meter readings, and fixed allowances preserving template integrity.
6. **Print-Ready PDF with Watermark**: High-fidelity PDF invoices matching society invoice standards, formatted in Indian Rupee denomination with words, containing the official CHS logo watermark behind content.

---

## Decisions

### 1. Technology & Architecture Boundaries
- **Backend**: Go 1.24 Chi REST API engine with PostgreSQL connection pooler (`pgx/v5`).
- **Frontend**: SvelteKit 2 + TailwindCSS v4 + Svelte 5 Runes (adhering to Swiss editorial minimalist tokens: `#f3f3f1` canvas, `#121212` primary text, `#e2e2de` borders, Hedvig Letters Sans/Serif typography).
- **Formula Engine**: Secure, deterministic AST-based expression evaluator in Go. Zero usage of `eval`, external interpreters, or shell executions. Supported operators: `+`, `-`, `*`, `/`, `()`, ternary conditionals, named numeric variables, division-by-zero safeguards, and circular dependency checks.
- **PDF Engine**: Pure Go server-side PDF generator (`jung-kurt/gofpdf`) generating vector A4 documents with transparent CHS logo watermark layer, Indian currency formatting (`₹`), and immutable generation snapshots.

### 2. Domain & Schema Entities
The billing domain extends into:
1. `maintenance_bill_templates`: Master configuration defining layout, header/footer, frequency, bank details, and watermark specs.
2. `maintenance_template_fields`: Dynamic field visibility, ordering, labels, and custom fields.
3. `maintenance_charge_heads`: Configurable charge heads with SAC codes, tax classification (TAXABLE, EXEMPT, NIL_RATED), default calculation formula, and accounting group.
4. `maintenance_formula_definitions`: Named formula expressions and input variable dependencies.
5. `maintenance_unit_billing_overrides`: Auditable unit-level overrides for rates, fixed amounts, or formula variables.
6. `maintenance_billing_batches`: Execution batch metadata for All / Selected / Single unit runs.
7. `maintenance_bills` (enhanced): Extended with template version snapshot, batch reference, GST breakdown, arrears, interest, and advance deduction.
8. `maintenance_bill_items`: Itemized line items linked to SAC codes and tax categories.

### 3. Execution Phases
1. **Phase 1**: Documentation-first specification (ADR 047, Architecture 22, Schema 16, API 18).
2. **Phase 2**: PostgreSQL migrations (tables, constraints, immutability triggers).
3. **Phase 3**: Go AST Formula Parser & Evaluator (TDD).
4. **Phase 4**: Backend Services & Chi Handlers (Templates, Charge Heads, Overrides, Batch Run).
5. **Phase 5**: Go PDF Invoice Generator with CHS Watermark & ZIP export.
6. **Phase 6**: SvelteKit 2 Frontend Console (Template Designer, Formula Editor, Billing Runner).
7. **Phase 7**: Quality Gates, verification scripts, and audit validation.

---

## Consequences

### Positive
- Accommodates both residential societies and commercial premises complexes.
- Zero code deployment needed to change rates, formulas, or charge heads.
- Strict financial immutability: finalized bills cannot be mutated by later template changes.
- High-performance PDF generation directly in Go backend without headless browser overhead.

### Negative
- Requires maintaining database migrations and backward-compatibility with S11/S12 bills.
- Formula evaluator must reject untrusted inputs and guard against recursion/division by zero.

---

## Related Documentation
- [[../architecture/22-configurable-billing-and-pdf-engine|Architecture 22: Configurable Billing Engine]]
- [[../architecture/16-maintenance-foundation|Architecture 16: Maintenance Foundation]]
- [[../architecture/17-maintenance-operations-and-collections|Architecture 17: Maintenance Collections]]
- [[../database/11-maintenance-foundation-schema|Database Schema 11]]
- [[../database/12-maintenance-operations-and-collections-schema|Database Schema 12]]
