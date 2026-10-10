-- Migration: 20261012000000_create_society_suspension_reasons.sql
-- Description: Controlled society suspension reason catalogue and initial seed data

-- 1. Create society_suspension_reasons Table
CREATE TABLE IF NOT EXISTS society_suspension_reasons (
    code TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL,
    allows_manual BOOLEAN NOT NULL DEFAULT TRUE,
    allows_automated BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Indexes
CREATE INDEX IF NOT EXISTS society_suspension_reasons_active_idx 
    ON society_suspension_reasons(is_active, allows_manual);

-- 3. Seed Canonical Suspension Reasons (Idempotent)
INSERT INTO society_suspension_reasons (code, label, description, category, allows_manual, allows_automated, is_active)
VALUES
    (
        'PAYMENT_OVERDUE',
        'Payment overdue',
        'Society payment obligations are overdue.',
        'PAYMENT',
        TRUE,
        TRUE,
        TRUE
    ),
    (
        'NON_PAYMENT_HOLD',
        'Non-payment hold',
        'Administrative hold placed due to outstanding dues.',
        'PAYMENT',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'LEGAL_DISPUTE',
        'Legal dispute',
        'Suspension related to an ongoing legal dispute or court order.',
        'LEGAL',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'COMPLIANCE_VIOLATION',
        'Compliance violation',
        'Failure to adhere to mandatory statutory or society bye-law compliance.',
        'COMPLIANCE',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'FRAUD_OR_MISUSE',
        'Suspected fraud or misuse',
        'Suspension triggered by suspected fraudulent activity or platform misuse.',
        'SECURITY',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'MANAGEMENT_REQUEST',
        'Management-requested suspension',
        'Suspension formally requested by society governing body or committee.',
        'ADMINISTRATIVE',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'DOCUMENTATION_ISSUE',
        'Documentation issue',
        'Critical documentation or registration paperwork is incomplete or invalid.',
        'COMPLIANCE',
        TRUE,
        FALSE,
        TRUE
    ),
    (
        'OTHER_REVIEW_REQUIRED',
        'Other — requires review',
        'Miscellaneous suspension reason requiring administrative review and mandatory details.',
        'ADMINISTRATIVE',
        TRUE,
        FALSE,
        TRUE
    )
ON CONFLICT (code) DO UPDATE
SET label = EXCLUDED.label,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    allows_manual = EXCLUDED.allows_manual,
    allows_automated = EXCLUDED.allows_automated,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
