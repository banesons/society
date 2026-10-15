-- Migration: 20261024000000_create_maintenance_operations_schema.sql
-- Description: Maintenance Operations & Collections (Payments, Allocations, Receipts, Sequences, Immutability Triggers)

-- 1. Sequences for Concurrency-Safe Numbering
CREATE SEQUENCE IF NOT EXISTS maintenance_bill_seq START WITH 10001;
CREATE SEQUENCE IF NOT EXISTS maintenance_payment_seq START WITH 10001;
CREATE SEQUENCE IF NOT EXISTS maintenance_receipt_seq START WITH 10001;

-- 2. Immutability Protection for Issued Maintenance Bills and Items (S11-GATE-02)
CREATE OR REPLACE FUNCTION trg_enforce_bill_immutability()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status != 'DRAFT' THEN
        -- Prevent modification of financial snapshot and structural associations
        IF NEW.subtotal != OLD.subtotal OR NEW.total_amount != OLD.total_amount OR
           NEW.society_id != OLD.society_id OR NEW.unit_id != OLD.unit_id OR
           NEW.billing_period_id != OLD.billing_period_id OR NEW.bill_number != OLD.bill_number OR
           NEW.issue_date != OLD.issue_date OR NEW.currency != OLD.currency THEN
            RAISE EXCEPTION 'financial and structural attributes of issued maintenance bills are immutable';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_bills_immutable ON maintenance_bills;
CREATE TRIGGER trg_bills_immutable
BEFORE UPDATE ON maintenance_bills
FOR EACH ROW EXECUTE FUNCTION trg_enforce_bill_immutability();

CREATE OR REPLACE FUNCTION trg_enforce_bill_item_immutability()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'maintenance_bill_items are immutable financial records';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_bill_items_immutable ON maintenance_bill_items;
CREATE TRIGGER trg_bill_items_immutable
BEFORE UPDATE OR DELETE ON maintenance_bill_items
FOR EACH ROW EXECUTE FUNCTION trg_enforce_bill_item_immutability();

-- 3. Maintenance Payments Table
CREATE TABLE IF NOT EXISTS maintenance_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    payment_reference VARCHAR(60) NOT NULL,
    payment_method VARCHAR(30) NOT NULL CHECK (payment_method IN ('CASH', 'CHEQUE', 'BANK_TRANSFER', 'UPI_RECORDED_OFFLINE', 'OTHER')),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    payment_date DATE NOT NULL,
    external_reference VARCHAR(100) NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'RECORDED' CHECK (status IN ('RECORDED', 'REVERSED')),
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    payer_name VARCHAR(150) NULL,
    notes TEXT NULL,
    reversed_at TIMESTAMPTZ NULL,
    reversed_by UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    reversal_reason TEXT NULL,
    idempotency_key VARCHAR(100) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_maintenance_payments_ref UNIQUE (society_id, payment_reference),
    CONSTRAINT uq_maintenance_payments_idem UNIQUE (society_id, idempotency_key),
    CONSTRAINT uq_maintenance_payments_id_society UNIQUE (id, society_id)
);

-- 4. Maintenance Payment Allocations Table
CREATE TABLE IF NOT EXISTS maintenance_payment_allocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    payment_id UUID NOT NULL,
    bill_id UUID NOT NULL,
    allocated_amount NUMERIC(12, 2) NOT NULL CHECK (allocated_amount > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'REVERSED')),
    allocated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reversed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_alloc_payment FOREIGN KEY (payment_id, society_id)
        REFERENCES maintenance_payments(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT fk_alloc_bill FOREIGN KEY (bill_id, society_id)
        REFERENCES maintenance_bills(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maintenance_alloc_id_society UNIQUE (id, society_id)
);

-- 5. Maintenance Receipts Table
CREATE TABLE IF NOT EXISTS maintenance_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    payment_id UUID NOT NULL,
    receipt_number VARCHAR(60) NOT NULL,
    receipt_date DATE NOT NULL,
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    issued_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'ISSUED' CHECK (status IN ('ISSUED', 'REVERSED')),
    reversed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_receipt_payment FOREIGN KEY (payment_id, society_id)
        REFERENCES maintenance_payments(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maintenance_receipts_number UNIQUE (society_id, receipt_number),
    CONSTRAINT uq_maintenance_receipts_payment UNIQUE (society_id, payment_id),
    CONSTRAINT uq_maintenance_receipts_id_society UNIQUE (id, society_id)
);

-- 6. Indexes for High-Traffic Queries
CREATE INDEX IF NOT EXISTS idx_maint_payments_soc_status ON maintenance_payments(society_id, status);
CREATE INDEX IF NOT EXISTS idx_maint_payments_soc_date ON maintenance_payments(society_id, payment_date);
CREATE INDEX IF NOT EXISTS idx_maint_alloc_soc_bill ON maintenance_payment_allocations(society_id, bill_id, status);
CREATE INDEX IF NOT EXISTS idx_maint_alloc_soc_payment ON maintenance_payment_allocations(society_id, payment_id);
CREATE INDEX IF NOT EXISTS idx_maint_receipts_soc_date ON maintenance_receipts(society_id, receipt_date);
