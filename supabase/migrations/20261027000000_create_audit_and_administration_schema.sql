-- Migration: 20261027000000_create_audit_and_administration_schema.sql
-- Description: S14 Audit and Administration Schema Extension
-- Adds optional society_id foreign key column to platform_audit_logs and performance indexes for society-scoped audit queries

-- 1. Add nullable society_id column to existing platform_audit_logs
ALTER TABLE platform_audit_logs
ADD COLUMN IF NOT EXISTS society_id UUID REFERENCES societies(id) ON DELETE SET NULL;

-- 2. Index on first-class society_id and created_at
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_society_created
ON platform_audit_logs (society_id, created_at DESC);

-- 3. Functional index on historical details->>'society_id' for fast backward-compatible society queries
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_details_society_created
ON platform_audit_logs ((details->>'society_id'), created_at DESC)
WHERE details->>'society_id' IS NOT NULL;

-- 4. Index on action and created_at for action filtering
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_action_created
ON platform_audit_logs (action, created_at DESC);

-- 5. Index on actor_id and created_at for actor filtering
CREATE INDEX IF NOT EXISTS idx_platform_audit_logs_actor_created
ON platform_audit_logs (actor_id, created_at DESC);
