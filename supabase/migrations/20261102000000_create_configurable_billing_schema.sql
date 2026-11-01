-- Migration: 20261102000000_create_configurable_billing_schema.sql
-- Description: Configurable Maintenance Billing, Formula Engine, Unit Overrides, Batches, and PDF Artifacts Schema

-- ==============================================================================
-- 1. Sequences for Batch and Numbering
-- ==============================================================================
CREATE SEQUENCE IF NOT EXISTS maintenance_batch_seq START WITH 1001;

-- ==============================================================================
-- 2. Formula Definitions & Immutable Versions
-- ==============================================================================
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

-- ==============================================================================
-- 3. Configurable Charge Heads with SAC Codes and Tax Classifications
-- ==============================================================================
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

-- ==============================================================================
-- 4. Bill Templates, Versioning & Configurable Fields
-- ==============================================================================
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

CREATE TABLE IF NOT EXISTS maintenance_bill_template_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    template_id UUID NOT NULL,
    version INT NOT NULL CHECK (version >= 1),
    definition_snapshot JSONB NOT NULL,
    published_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_template_version_template FOREIGN KEY (template_id, society_id)
        REFERENCES maintenance_bill_templates(id, society_id) ON DELETE CASCADE,
    CONSTRAINT uq_maint_template_versions_ver UNIQUE (template_id, version),
    CONSTRAINT uq_maint_template_versions_id_society UNIQUE (id, society_id)
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

-- ==============================================================================
-- 5. Unit-Specific Billing Overrides
-- ==============================================================================
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

-- ==============================================================================
-- 6. Billing Batches & Batch Units
-- ==============================================================================
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

-- ==============================================================================
-- 7. Enhanced Columns for Existing maintenance_bills & maintenance_bill_items
-- ==============================================================================
ALTER TABLE maintenance_bills
    ADD COLUMN IF NOT EXISTS bill_type VARCHAR(50) NOT NULL DEFAULT 'MAINTENANCE',
    ADD COLUMN IF NOT EXISTS template_id UUID NULL,
    ADD COLUMN IF NOT EXISTS template_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS batch_id UUID NULL,
    ADD COLUMN IF NOT EXISTS taxable_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS cgst_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS sgst_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS igst_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS total_tax_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS arrears_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS interest_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS advance_adjusted NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS pdf_artifact_path TEXT NULL,
    ADD COLUMN IF NOT EXISTS pdf_generated_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS notes TEXT NULL;

-- Safe foreign keys on maintenance_bills
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_bills_template'
    ) THEN
        ALTER TABLE maintenance_bills
            ADD CONSTRAINT fk_bills_template FOREIGN KEY (template_id, society_id)
            REFERENCES maintenance_bill_templates(id, society_id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_bills_batch'
    ) THEN
        ALTER TABLE maintenance_bills
            ADD CONSTRAINT fk_bills_batch FOREIGN KEY (batch_id, society_id)
            REFERENCES maintenance_billing_batches(id, society_id) ON DELETE SET NULL;
    END IF;
END $$;

-- Upgrade Unique Constraint on maintenance_bills:
-- 1. Drop old constraint if present
ALTER TABLE maintenance_bills DROP CONSTRAINT IF EXISTS uq_maintenance_bills_unit_period;

-- 2. Create partial index ensuring uniqueness of active/issued/draft bills per (society_id, billing_period_id, unit_id, bill_type)
DROP INDEX IF EXISTS uq_idx_maintenance_bills_active_unit_period_type;
CREATE UNIQUE INDEX uq_idx_maintenance_bills_active_unit_period_type
    ON maintenance_bills (society_id, billing_period_id, unit_id, bill_type)
    WHERE status != 'VOID';

-- Enhanced Columns for maintenance_bill_items
ALTER TABLE maintenance_bill_items
    ADD COLUMN IF NOT EXISTS charge_head_id UUID NULL,
    ADD COLUMN IF NOT EXISTS sac_code VARCHAR(20) NULL,
    ADD COLUMN IF NOT EXISTS tax_classification VARCHAR(30) NOT NULL DEFAULT 'EXEMPT' CHECK (
        tax_classification IN ('TAXABLE', 'EXEMPT', 'NIL_RATED', 'NON_GST')
    ),
    ADD COLUMN IF NOT EXISTS tax_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00 CHECK (tax_rate >= 0),
    ADD COLUMN IF NOT EXISTS tax_amount NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (tax_amount >= 0),
    ADD COLUMN IF NOT EXISTS formula_expression TEXT NULL,
    ADD COLUMN IF NOT EXISTS display_order INT NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_bill_items_charge_head'
    ) THEN
        ALTER TABLE maintenance_bill_items
            ADD CONSTRAINT fk_bill_items_charge_head FOREIGN KEY (charge_head_id, society_id)
            REFERENCES maintenance_charge_heads(id, society_id) ON DELETE SET NULL;
    END IF;
END $$;

-- ==============================================================================
-- 8. Batch Units Table (Per-Unit Calculation Snapshot)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS maintenance_batch_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    batch_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    bill_id UUID NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING' CHECK (
        status IN ('PENDING', 'CALCULATED', 'GENERATED', 'EXCLUDED', 'FAILED')
    ),
    calculation_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_batch_units_batch FOREIGN KEY (batch_id, society_id)
        REFERENCES maintenance_billing_batches(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_batch_units_unit FOREIGN KEY (unit_id, society_id)
        REFERENCES society_units(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT fk_batch_units_bill FOREIGN KEY (bill_id, society_id)
        REFERENCES maintenance_bills(id, society_id) ON DELETE SET NULL,
    CONSTRAINT uq_maint_batch_units_unit UNIQUE (batch_id, unit_id),
    CONSTRAINT uq_maint_batch_units_id_society UNIQUE (id, society_id)
);

-- ==============================================================================
-- 9. PDF Artifacts Table
-- ==============================================================================
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

-- ==============================================================================
-- 10. Update Immutability Triggers to Guard New Financial Snapshot Columns
-- ==============================================================================
CREATE OR REPLACE FUNCTION trg_enforce_bill_immutability()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status != 'DRAFT' THEN
        -- Prevent modification of financial snapshot and structural attributes
        IF NEW.subtotal != OLD.subtotal OR 
           NEW.total_amount != OLD.total_amount OR
           NEW.taxable_amount != OLD.taxable_amount OR
           NEW.cgst_amount != OLD.cgst_amount OR
           NEW.sgst_amount != OLD.sgst_amount OR
           NEW.igst_amount != OLD.igst_amount OR
           NEW.total_tax_amount != OLD.total_tax_amount OR
           NEW.arrears_amount != OLD.arrears_amount OR
           NEW.interest_amount != OLD.interest_amount OR
           NEW.advance_adjusted != OLD.advance_adjusted OR
           NEW.society_id != OLD.society_id OR 
           NEW.unit_id != OLD.unit_id OR
           NEW.billing_period_id != OLD.billing_period_id OR 
           NEW.bill_number != OLD.bill_number OR
           NEW.bill_type != OLD.bill_type OR
           NEW.template_version != OLD.template_version OR
           NEW.issue_date != OLD.issue_date OR 
           NEW.currency != OLD.currency THEN
            RAISE EXCEPTION 'financial and structural attributes of issued maintenance bills are immutable';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ==============================================================================
-- 11. Performance & Multi-Tenant Isolation Indexes
-- ==============================================================================
CREATE INDEX IF NOT EXISTS idx_maint_templates_soc_status 
    ON maintenance_bill_templates (society_id, status);

CREATE INDEX IF NOT EXISTS idx_maint_template_fields_soc_tpl 
    ON maintenance_template_fields (society_id, template_id, display_order);

CREATE INDEX IF NOT EXISTS idx_maint_charge_heads_soc_active 
    ON maintenance_charge_heads (society_id, is_active);

CREATE INDEX IF NOT EXISTS idx_maint_formulas_soc_status 
    ON maintenance_formula_definitions (society_id, status);

CREATE INDEX IF NOT EXISTS idx_maint_unit_overrides_soc_unit 
    ON maintenance_unit_billing_overrides (society_id, unit_id, status);

CREATE INDEX IF NOT EXISTS idx_maint_batches_soc_period 
    ON maintenance_billing_batches (society_id, billing_period_id, status);

CREATE INDEX IF NOT EXISTS idx_maint_batch_units_soc_batch 
    ON maintenance_batch_units (society_id, batch_id, status);

CREATE INDEX IF NOT EXISTS idx_maint_bills_soc_batch 
    ON maintenance_bills (society_id, batch_id);

CREATE INDEX IF NOT EXISTS idx_maint_bills_soc_type 
    ON maintenance_bills (society_id, bill_type);

CREATE INDEX IF NOT EXISTS idx_maint_pdf_artifacts_soc_bill 
    ON maintenance_bill_pdf_artifacts (society_id, bill_id);
