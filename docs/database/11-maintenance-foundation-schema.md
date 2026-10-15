# 11 — Maintenance Foundation Database Schema

## Overview
Sprint S11 introduces five core tables to manage society maintenance charges, billing periods, issued bills, and immutable line items with multi-tenant composite foreign keys and concurrency constraints.

---

## 1. Schema Definitions

### 1.1 `maintenance_charge_categories` Table
Defines society-scoped charge classifications (e.g., General Maintenance, Sinking Fund, Parking, Special Repair).

```sql
CREATE TABLE IF NOT EXISTS maintenance_charge_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_maintenance_charge_categories_name UNIQUE (society_id, name),
    CONSTRAINT uq_maintenance_charge_categories_id_society UNIQUE (id, society_id)
);
```

### 1.2 `maintenance_charge_configs` Table
Defines active and historical calculation rules for specific categories.

```sql
CREATE TABLE IF NOT EXISTS maintenance_charge_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    category_id UUID NOT NULL,
    name VARCHAR(150) NOT NULL,
    charge_type VARCHAR(50) NOT NULL CHECK (charge_type IN ('FLAT_RECURRING', 'UNIT_TYPE_RECURRING', 'ONE_OFF')),
    applicable_unit_type VARCHAR(50) NULL CHECK (
        applicable_unit_type IS NULL OR 
        applicable_unit_type IN ('RESIDENTIAL', 'COMMERCIAL', 'PENTHOUSE', 'PARKING', 'OTHER')
    ),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    billing_frequency VARCHAR(30) NOT NULL DEFAULT 'MONTHLY' CHECK (
        billing_frequency IN ('MONTHLY', 'QUARTERLY', 'ANNUALLY', 'ONE_OFF')
    ),
    effective_from DATE NOT NULL,
    effective_until DATE NULL CHECK (effective_until IS NULL OR effective_until >= effective_from),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_charge_configs_category FOREIGN KEY (category_id, society_id)
        REFERENCES maintenance_charge_categories(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maintenance_charge_configs_id_society UNIQUE (id, society_id)
);
```

### 1.3 `maintenance_billing_periods` Table
Defines temporal billing cycles with payment deadlines.

```sql
CREATE TABLE IF NOT EXISTS maintenance_billing_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    period_name VARCHAR(100) NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL CHECK (period_end >= period_start),
    due_date DATE NOT NULL CHECK (due_date >= period_start),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CLOSED')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_billing_periods_dates UNIQUE (society_id, period_start, period_end),
    CONSTRAINT uq_billing_periods_id_society UNIQUE (id, society_id)
);
```

### 1.4 `maintenance_bills` Table
Persists issued bills for individual units within a billing period.

```sql
CREATE TABLE IF NOT EXISTS maintenance_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    billing_period_id UUID NOT NULL,
    bill_number VARCHAR(80) NOT NULL,
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    subtotal NUMERIC(12, 2) NOT NULL CHECK (subtotal >= 0),
    total_amount NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'ISSUED' CHECK (status IN ('DRAFT', 'ISSUED', 'VOID')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    voided_by UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    voided_at TIMESTAMPTZ NULL,
    void_reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_bills_unit FOREIGN KEY (unit_id, society_id)
        REFERENCES society_units(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT fk_bills_period FOREIGN KEY (billing_period_id, society_id)
        REFERENCES maintenance_billing_periods(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_maintenance_bills_unit_period UNIQUE (society_id, billing_period_id, unit_id),
    CONSTRAINT uq_maintenance_bills_number UNIQUE (society_id, bill_number),
    CONSTRAINT uq_maintenance_bills_id_society UNIQUE (id, society_id)
);
```

### 1.5 `maintenance_bill_items` Table
Stores immutable financial line items for an issued bill.

```sql
CREATE TABLE IF NOT EXISTS maintenance_bill_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id UUID NOT NULL,
    society_id UUID NOT NULL,
    charge_category_id UUID NULL,
    category_name VARCHAR(100) NOT NULL,
    charge_config_id UUID NULL,
    description VARCHAR(255) NOT NULL,
    rate NUMERIC(12, 2) NOT NULL CHECK (rate >= 0),
    quantity NUMERIC(10, 2) NOT NULL DEFAULT 1.00 CHECK (quantity > 0),
    amount NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_bill_items_bill FOREIGN KEY (bill_id, society_id)
        REFERENCES maintenance_bills(id, society_id) ON DELETE CASCADE,
    CONSTRAINT fk_bill_items_category FOREIGN KEY (charge_category_id, society_id)
        REFERENCES maintenance_charge_categories(id, society_id) ON DELETE SET NULL,
    CONSTRAINT fk_bill_items_config FOREIGN KEY (charge_config_id, society_id)
        REFERENCES maintenance_charge_configs(id, society_id) ON DELETE SET NULL
);
```

---

## 2. Indexes for Query Performance
```sql
CREATE INDEX IF NOT EXISTS idx_maintenance_bills_society_status 
    ON maintenance_bills (society_id, status);

CREATE INDEX IF NOT EXISTS idx_maintenance_bills_unit 
    ON maintenance_bills (society_id, unit_id, due_date DESC);

CREATE INDEX IF NOT EXISTS idx_maintenance_bill_items_bill 
    ON maintenance_bill_items (bill_id);

CREATE INDEX IF NOT EXISTS idx_charge_configs_society_status 
    ON maintenance_charge_configs (society_id, status);

CREATE INDEX IF NOT EXISTS idx_billing_periods_society 
    ON maintenance_billing_periods (society_id, period_start DESC);
```
