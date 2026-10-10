-- Migration: 20261015000000_create_society_structure_schema.sql
-- Description: Create society structure, units, gates, and staffing requirements (S05)

-- 1. Buildings Table
CREATE TABLE IF NOT EXISTS society_buildings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    building_code TEXT NOT NULL,
    total_floors INT NOT NULL DEFAULT 1 CHECK (total_floors >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_buildings_society_name_uq UNIQUE (society_id, name),
    CONSTRAINT society_buildings_society_code_uq UNIQUE (society_id, building_code)
);

-- 2. Wings Table (Optional subdivisions under a building)
CREATE TABLE IF NOT EXISTS society_wings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    building_id UUID NOT NULL REFERENCES society_buildings(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    wing_code TEXT NOT NULL,
    total_floors INT NOT NULL DEFAULT 1 CHECK (total_floors >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_wings_building_name_uq UNIQUE (building_id, name),
    CONSTRAINT society_wings_building_code_uq UNIQUE (building_id, wing_code)
);

-- 3. Units / Flats Table
CREATE TABLE IF NOT EXISTS society_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    building_id UUID NOT NULL REFERENCES society_buildings(id) ON DELETE RESTRICT,
    wing_id UUID NULL REFERENCES society_wings(id) ON DELETE RESTRICT,
    floor_number INT NOT NULL,
    unit_number TEXT NOT NULL,
    unit_identifier TEXT NOT NULL,
    unit_type TEXT NOT NULL DEFAULT 'RESIDENTIAL',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_units_type_check CHECK (unit_type IN ('RESIDENTIAL', 'COMMERCIAL', 'PENTHOUSE', 'PARKING', 'OTHER')),
    CONSTRAINT society_units_status_check CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT society_units_society_identifier_uq UNIQUE (society_id, unit_identifier)
);

-- 4. Gates Table
CREATE TABLE IF NOT EXISTS society_gates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    gate_code TEXT NOT NULL,
    gate_type TEXT NOT NULL DEFAULT 'VEHICLE_AND_PEDESTRIAN',
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    required_staff_per_shift INT NOT NULL DEFAULT 1 CHECK (required_staff_per_shift >= 0),
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_gates_type_check CHECK (gate_type IN ('VEHICLE_AND_PEDESTRIAN', 'VEHICLE_ONLY', 'PEDESTRIAN_ONLY', 'SERVICE', 'EMERGENCY')),
    CONSTRAINT society_gates_status_check CHECK (status IN ('ACTIVE', 'INACTIVE')),
    CONSTRAINT society_gates_society_code_uq UNIQUE (society_id, gate_code),
    CONSTRAINT society_gates_society_name_uq UNIQUE (society_id, name)
);

-- 5. Gatekeeper Staffing Requirements Table
CREATE TABLE IF NOT EXISTS society_staffing_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    shifts_per_day INT NOT NULL DEFAULT 3 CHECK (shifts_per_day BETWEEN 1 AND 4),
    total_required_headcount INT NOT NULL DEFAULT 0 CHECK (total_required_headcount >= 0),
    notes TEXT NULL,
    updated_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT society_staffing_requirements_society_uq UNIQUE (society_id)
);

-- Lookup Indexes
CREATE INDEX IF NOT EXISTS society_buildings_society_idx ON society_buildings (society_id, name);
CREATE INDEX IF NOT EXISTS society_wings_building_idx ON society_wings (building_id, name);
CREATE INDEX IF NOT EXISTS society_units_lookup_idx ON society_units (society_id, building_id, floor_number);
CREATE INDEX IF NOT EXISTS society_gates_society_idx ON society_gates (society_id, gate_code);
