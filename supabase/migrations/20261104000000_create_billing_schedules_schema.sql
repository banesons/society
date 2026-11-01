-- Migration: 20261104000000_create_billing_schedules_schema.sql
-- Description: Phase 7.2 - Billing Automation Schedules and Execution History Schema

-- 1. Schedules Table
CREATE TABLE IF NOT EXISTS maintenance_billing_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name VARCHAR(150) NOT NULL,
    recurrence_type VARCHAR(20) NOT NULL CHECK (recurrence_type IN ('MONTHLY', 'QUARTERLY', 'ANNUAL')),
    template_id UUID NOT NULL,
    selection_mode VARCHAR(30) NOT NULL DEFAULT 'ALL_UNITS' CHECK (
        selection_mode IN ('ALL_UNITS', 'SELECTED_UNITS')
    ),
    filter_criteria JSONB NOT NULL DEFAULT '{}'::jsonb,
    day_of_month INT NOT NULL DEFAULT 1 CHECK (day_of_month BETWEEN 1 AND 31),
    due_days INT NOT NULL DEFAULT 15 CHECK (due_days BETWEEN 1 AND 90),
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Kolkata',
    auto_finalize BOOLEAN NOT NULL DEFAULT FALSE,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ NULL,
    last_batch_id UUID NULL REFERENCES maintenance_billing_batches(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAUSED', 'DISABLED')),
    locked_by TEXT NULL,
    locked_until TIMESTAMPTZ NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_schedules_template FOREIGN KEY (template_id, society_id)
        REFERENCES maintenance_bill_templates(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maint_schedules_name UNIQUE (society_id, name),
    CONSTRAINT uq_maint_schedules_id_society UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_maint_schedules_due 
    ON maintenance_billing_schedules (status, next_run_at) 
    WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_maint_schedules_society 
    ON maintenance_billing_schedules (society_id, status);

-- 2. Schedule Executions History Table
CREATE TABLE IF NOT EXISTS maintenance_billing_schedule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id UUID NOT NULL,
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE CASCADE,
    billing_period_id UUID NULL,
    batch_id UUID NULL,
    idempotency_key TEXT NOT NULL,
    trigger_source VARCHAR(20) NOT NULL CHECK (trigger_source IN ('SCHEDULED', 'MANUAL')),
    status VARCHAR(30) NOT NULL CHECK (status IN ('CLAIMED', 'PROCESSING', 'SUCCESS', 'FAILED', 'SKIPPED')),
    attempt_count INT NOT NULL DEFAULT 1,
    error_code VARCHAR(50) NULL,
    error_message TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_sched_exec_schedule FOREIGN KEY (schedule_id, society_id)
        REFERENCES maintenance_billing_schedules(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_sched_exec_period FOREIGN KEY (billing_period_id)
        REFERENCES maintenance_billing_periods(id) ON DELETE SET NULL,
    CONSTRAINT fk_sched_exec_batch FOREIGN KEY (batch_id)
        REFERENCES maintenance_billing_batches(id) ON DELETE SET NULL,
    CONSTRAINT uq_sched_exec_idempotency UNIQUE (society_id, idempotency_key),
    CONSTRAINT uq_sched_exec_id_society UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_sched_exec_schedule 
    ON maintenance_billing_schedule_executions (society_id, schedule_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sched_exec_batch 
    ON maintenance_billing_schedule_executions (batch_id) 
    WHERE batch_id IS NOT NULL;
