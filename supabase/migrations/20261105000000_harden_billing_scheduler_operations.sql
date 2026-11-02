-- Migration: 20261105000000_harden_billing_scheduler_operations.sql
-- Description: Phase 7.3 - Billing Automation Scheduler Operations, Observability & Recovery Telemetry

-- 1. Extend maintenance_billing_schedules with failure tracking
ALTER TABLE maintenance_billing_schedules
    ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error_code VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS last_error_message TEXT NULL;

-- 2. Extend maintenance_billing_schedule_executions with recovery and duration telemetry
ALTER TABLE maintenance_billing_schedule_executions
    ADD COLUMN IF NOT EXISTS duration_ms INT NULL,
    ADD COLUMN IF NOT EXISTS recovery_status VARCHAR(30) NOT NULL DEFAULT 'NONE',
    ADD COLUMN IF NOT EXISTS recovered_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS recovered_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS recovery_notes TEXT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_sched_exec_recovery_status'
    ) THEN
        ALTER TABLE maintenance_billing_schedule_executions
            ADD CONSTRAINT chk_sched_exec_recovery_status
            CHECK (recovery_status IN ('NONE', 'RECOVERED', 'REQUIRES_ATTENTION'));
    END IF;
END $$;

-- 3. Operational query performance indexes
CREATE INDEX IF NOT EXISTS idx_sched_exec_status 
    ON maintenance_billing_schedule_executions (society_id, status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_maint_schedules_overdue 
    ON maintenance_billing_schedules (society_id, status, next_run_at) 
    WHERE status = 'ACTIVE';
