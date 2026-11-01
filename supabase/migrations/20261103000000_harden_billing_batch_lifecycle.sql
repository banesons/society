-- ==============================================================================
-- Migration: 20261103000000_harden_billing_batch_lifecycle.sql
-- Description: Hardens billing batch execution lifecycle, concurrency locking,
--              progress tracking, unit-level failure recording, and retry semantics.
-- ==============================================================================

-- 1. Update maintenance_billing_batches status check constraint
ALTER TABLE maintenance_billing_batches
    DROP CONSTRAINT IF EXISTS maintenance_billing_batches_status_check;

ALTER TABLE maintenance_billing_batches
    ADD CONSTRAINT maintenance_billing_batches_status_check CHECK (
        status IN (
            'DRAFT_PREVIEW',
            'QUEUED',
            'PROCESSING',
            'GENERATED',
            'COMPLETED_WITH_ERRORS',
            'FINALIZED',
            'FAILED',
            'CANCELLED'
        )
    );

-- 2. Add lifecycle timestamps and operational fields to maintenance_billing_batches
ALTER TABLE maintenance_billing_batches
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS cancelled_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cancellation_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS processed_unit_count INT NOT NULL DEFAULT 0;

-- 3. Add retry and failure diagnostic columns to maintenance_batch_units
ALTER TABLE maintenance_batch_units
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_attempted_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(50) NULL;

-- 4. Index for fast querying of failed or pending units during retries
CREATE INDEX IF NOT EXISTS idx_maint_batch_units_batch_status
    ON maintenance_batch_units (batch_id, status);

-- 5. Comments for schema self-documentation
COMMENT ON COLUMN maintenance_billing_batches.status IS 'Batch execution lifecycle status: DRAFT_PREVIEW, QUEUED, PROCESSING, GENERATED, COMPLETED_WITH_ERRORS, FINALIZED, FAILED, CANCELLED';
COMMENT ON COLUMN maintenance_billing_batches.started_at IS 'Timestamp when batch execution/processing began';
COMMENT ON COLUMN maintenance_billing_batches.completed_at IS 'Timestamp when batch calculation or bill generation completed';
COMMENT ON COLUMN maintenance_batch_units.retry_count IS 'Number of retry attempts made for this unit in the batch';
COMMENT ON COLUMN maintenance_batch_units.error_code IS 'Machine-readable error categorization: VALIDATION_ERROR, FORMULA_ERROR, DB_ERROR, TRANSIENT_ERROR';
