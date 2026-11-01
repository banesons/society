# 16 — Configurable Maintenance Billing & PDF Engine Database Schema

## 1. Overview
Sprint S19 / Configurable Billing Engine introduces 9 new tables, enhances 2 existing tables (`maintenance_bills` and `maintenance_bill_items`), establishes concurrency sequences, and applies immutability triggers to support dynamic bill templates, AST formula evaluation, SAC codes, GST calculations, unit-specific overrides, and PDF artifact tracking.

---

## 2. Entity Specifications

### 2.1 `maintenance_formula_definitions` & `maintenance_formula_versions`
Stores formula expressions (e.g. `area * maintenance_rate`) and their published immutable revisions.

```sql
CREATE TABLE IF NOT EXISTS maintenance_formula_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    code VARCHAR(60) NOT NULL,
    name VARCHAR(120) NOT NULL,
    description TEXT NULL,
    expression TEXT NOT NULL,
    required_variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED')),
    current_version INT NOT NULL DEFAULT 1 CHECK (current_version >= 1),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_maint_formulas_code UNIQUE (society_id, code),
    CONSTRAINT uq_maint_formulas_id_society UNIQUE (id, society_id)
);

CREATE TABLE IF NOT EXISTS maintenance_formula_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    formula_id UUID NOT NULL,
    version INT NOT NULL CHECK (version >= 1),
    expression TEXT NOT NULL,
    required_variables JSONB NOT NULL DEFAULT '[]'::jsonb,
    published_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_formula_version_formula FOREIGN KEY (formula_id, society_id)
        REFERENCES maintenance_formula_definitions(id, society_id) ON DELETE CASCADE,
    CONSTRAINT uq_maint_formula_versions_ver UNIQUE (formula_id, version),
    CONSTRAINT uq_maint_formula_versions_id_society UNIQUE (id, society_id)
);
```

### 2.2 `maintenance_charge_heads`
Dynamic charge heads with SAC code, tax classification, and calculation method.

```sql
CREATE TABLE IF NOT EXISTS maintenance_charge_heads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    category_id UUID NULL,
    code VARCHAR(60) NOT NULL,
    name VARCHAR(120) NOT NULL,
    description TEXT NULL,
    sac_code VARCHAR(20) NULL,
    tax_classification VARCHAR(30) NOT NULL DEFAULT 'EXEMPT' CHECK (
        tax_classification IN ('TAXABLE', 'EXEMPT', 'NIL_RATED', 'NON_GST')
    ),
    default_tax_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00 CHECK (default_tax_rate >= 0),
    calculation_method VARCHAR(40) NOT NULL DEFAULT 'FIXED' CHECK (
        calculation_method IN ('FIXED', 'AREA_MULTIPLIER', 'METERED', 'PERCENTAGE', 'FORMULA')
    ),
    default_formula_id UUID NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    display_order INT NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_charge_head_category FOREIGN KEY (category_id, society_id)
        REFERENCES maintenance_charge_categories(id, society_id) ON DELETE SET NULL,
    CONSTRAINT fk_charge_head_formula FOREIGN KEY (default_formula_id, society_id)
        REFERENCES maintenance_formula_definitions(id, society_id) ON DELETE SET NULL,
    CONSTRAINT uq_maint_charge_heads_code UNIQUE (society_id, code),
    CONSTRAINT uq_maint_charge_heads_id_society UNIQUE (id, society_id)
);
```

### 2.3 `maintenance_bill_templates` & `maintenance_template_fields`
Configurable invoice layout, watermark, frequencies, bank details, and field order/visibility.

```sql
CREATE TABLE IF NOT EXISTS maintenance_bill_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name VARCHAR(120) NOT NULL,
    description TEXT NULL,
    bill_type VARCHAR(50) NOT NULL DEFAULT 'MAINTENANCE',
    billing_frequency VARCHAR(30) NOT NULL DEFAULT 'MONTHLY' CHECK (
        billing_frequency IN ('MONTHLY', 'QUARTERLY', 'HALF_YEARLY', 'ANNUALLY', 'ONE_OFF')
    ),
    applicable_unit_types JSONB NOT NULL DEFAULT '["RESIDENTIAL", "COMMERCIAL", "PENTHOUSE", "PARKING", "OTHER"]'::jsonb,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    bill_title VARCHAR(150) NOT NULL DEFAULT 'Tax Invoice / Maintenance Bill',
    header_notes TEXT NULL,
    footer_text TEXT NULL,
    payment_instructions TEXT NULL,
    bank_details JSONB NOT NULL DEFAULT '{}'::jsonb,
    watermark_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    watermark_opacity NUMERIC(3, 2) NOT NULL DEFAULT 0.08 CHECK (watermark_opacity >= 0.00 AND watermark_opacity <= 1.00),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED')),
    current_version INT NOT NULL DEFAULT 1 CHECK (current_version >= 1),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_maint_templates_name UNIQUE (society_id, name),
    CONSTRAINT uq_maint_templates_id_society UNIQUE (id, society_id)
);

CREATE TABLE IF NOT EXISTS maintenance_template_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    template_id UUID NOT NULL,
    field_key VARCHAR(80) NOT NULL,
    field_label VARCHAR(120) NOT NULL,
    field_category VARCHAR(40) NOT NULL DEFAULT 'BILL_INFO' CHECK (
        field_category IN ('SOCIETY_INFO', 'BILL_INFO', 'MEMBER_INFO', 'FINANCIAL', 'CUSTOM')
    ),
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INT NOT NULL DEFAULT 0,
    default_value TEXT NULL,
    formatting JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_template_fields_template FOREIGN KEY (template_id, society_id)
        REFERENCES maintenance_bill_templates(id, society_id) ON DELETE CASCADE,
    CONSTRAINT uq_maint_template_fields_key UNIQUE (template_id, field_key),
    CONSTRAINT uq_maint_template_fields_id_society UNIQUE (id, society_id)
);
```

### 2.4 `maintenance_unit_billing_overrides`
Auditable unit-level overrides for rates, fixed amounts, or formula variables.

```sql
CREATE TABLE IF NOT EXISTS maintenance_unit_billing_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    charge_head_id UUID NOT NULL,
    override_type VARCHAR(40) NOT NULL CHECK (
        override_type IN ('FIXED_AMOUNT', 'RATE_OVERRIDE', 'FORMULA_OVERRIDE', 'EXEMPTION', 'PERCENTAGE_DISCOUNT')
    ),
    override_value NUMERIC(12, 2) NULL,
    formula_override_expression TEXT NULL,
    effective_from DATE NOT NULL,
    effective_until DATE NULL CHECK (effective_until IS NULL OR effective_until >= effective_from),
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_unit_overrides_unit FOREIGN KEY (unit_id, society_id)
        REFERENCES society_units(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT fk_unit_overrides_charge_head FOREIGN KEY (charge_head_id, society_id)
        REFERENCES maintenance_charge_heads(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maint_unit_overrides_id_society UNIQUE (id, society_id)
);
```

### 2.5 `maintenance_billing_batches` & `maintenance_batch_units`
Batch generation execution tracking and unit-level calculation snapshots.

```sql
CREATE TABLE IF NOT EXISTS maintenance_billing_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    billing_period_id UUID NOT NULL,
    template_id UUID NOT NULL,
    batch_number VARCHAR(60) NOT NULL,
    selection_mode VARCHAR(30) NOT NULL DEFAULT 'ALL_UNITS' CHECK (
        selection_mode IN ('ALL_UNITS', 'SELECTED_UNITS', 'SINGLE_UNIT', 'UNIT_GROUP')
    ),
    filter_criteria JSONB NOT NULL DEFAULT '{}'::jsonb,
    target_unit_count INT NOT NULL DEFAULT 0,
    successful_unit_count INT NOT NULL DEFAULT 0,
    failed_unit_count INT NOT NULL DEFAULT 0,
    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT_PREVIEW' CHECK (
        status IN ('DRAFT_PREVIEW', 'PROCESSING', 'GENERATED', 'FINALIZED', 'FAILED')
    ),
    idempotency_key VARCHAR(100) NULL,
    error_summary JSONB NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    finalized_by UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    finalized_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_batches_period FOREIGN KEY (billing_period_id, society_id)
        REFERENCES maintenance_billing_periods(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT fk_batches_template FOREIGN KEY (template_id, society_id)
        REFERENCES maintenance_bill_templates(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maint_batches_number UNIQUE (society_id, batch_number),
    CONSTRAINT uq_maint_batches_idempotency UNIQUE (society_id, idempotency_key),
    CONSTRAINT uq_maint_batches_id_society UNIQUE (id, society_id)
);
```

### 2.6 `maintenance_bill_pdf_artifacts`
Tracks vector A4 invoices, hash sums, and watermark metadata.

```sql
CREATE TABLE IF NOT EXISTS maintenance_bill_pdf_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    bill_id UUID NULL,
    batch_id UUID NULL,
    artifact_type VARCHAR(40) NOT NULL CHECK (
        artifact_type IN ('SINGLE_BILL_PDF', 'BATCH_COMBINED_PDF', 'BATCH_ZIP_ARCHIVE')
    ),
    file_name VARCHAR(255) NOT NULL,
    storage_path TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL CHECK (file_size_bytes >= 0),
    sha256_checksum VARCHAR(64) NOT NULL,
    mime_type VARCHAR(50) NOT NULL DEFAULT 'application/pdf',
    has_watermark BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_pdf_artifacts_bill FOREIGN KEY (bill_id, society_id)
        REFERENCES maintenance_bills(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_pdf_artifacts_batch FOREIGN KEY (batch_id, society_id)
        REFERENCES maintenance_billing_batches(id, society_id) ON DELETE CASCADE,
    CONSTRAINT uq_maint_pdf_artifacts_id_society UNIQUE (id, society_id)
);
```

---

## 3. Critical Uniqueness & Immutability Rules
1. **No Duplicate Finalized Bills**:
   Enforced via partial unique index:
   ```sql
   CREATE UNIQUE INDEX uq_idx_maintenance_bills_active_unit_period_type
       ON maintenance_bills (society_id, billing_period_id, unit_id, bill_type)
       WHERE status != 'VOID';
   ```
   Prevents duplicate active/draft bills for the same unit, period, and bill type, while allowing reissuing voided bills.
2. **Immutable Snapshot Protection**:
   `trg_enforce_bill_immutability()` prevents modifying `subtotal`, `total_amount`, `taxable_amount`, `cgst_amount`, `sgst_amount`, `igst_amount`, `total_tax_amount`, `arrears_amount`, `interest_amount`, `advance_adjusted`, `bill_type`, and `template_version` once a bill is no longer `DRAFT`.
3. **Immutable Line Items**:
   `trg_enforce_bill_item_immutability()` blocks all updates or deletions on `maintenance_bill_items`.

---

## 4. Related Documentation
- [[../architecture/22-configurable-billing-and-pdf-engine|Architecture 22: Configurable Billing Engine]]
- [[../decisions/047-configurable-billing-template-formula-engine-and-pdf|ADR 047]]
- [[11-maintenance-foundation-schema|Database Schema 11]]
- [[12-maintenance-operations-and-collections-schema|Database Schema 12]]
