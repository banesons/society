-- Migration: 20261025000000_create_documents_and_records_schema.sql
-- Description: Sprint S13 - Documents & Records Schema (Categories, Documents, Versions, Immutability & Seed Categories)

-- 1. Sequence for Concurrency-Safe Document Numbering
CREATE SEQUENCE IF NOT EXISTS document_number_seq START WITH 10001 INCREMENT BY 1;

-- 2. Document Categories Table
CREATE TABLE IF NOT EXISTS document_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT NULL,
    is_system BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    display_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_document_categories_society_name UNIQUE (society_id, name)
);

CREATE INDEX IF NOT EXISTS idx_document_categories_society 
    ON document_categories(society_id, is_active, display_order);

-- 3. Society Documents Root Metadata Table
CREATE TABLE IF NOT EXISTS society_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    category_id UUID NOT NULL REFERENCES document_categories(id) ON DELETE RESTRICT,
    document_number TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    description TEXT NULL,
    classification TEXT NOT NULL DEFAULT 'SOCIETY_RESIDENTS' 
        CHECK (classification IN ('SOCIETY_RESIDENTS', 'COMMITTEE', 'SOC_ADMIN_ONLY')),
    status TEXT NOT NULL DEFAULT 'ACTIVE' 
        CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    current_version_id UUID NULL,
    building_id UUID NULL REFERENCES society_buildings(id) ON DELETE RESTRICT,
    wing_id UUID NULL REFERENCES society_wings(id) ON DELETE RESTRICT,
    unit_id UUID NULL REFERENCES society_units(id) ON DELETE RESTRICT,
    member_id UUID NULL REFERENCES society_members(id) ON DELETE RESTRICT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    updated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL,
    archived_by UUID NULL REFERENCES users(id) ON DELETE RESTRICT,
    archived_reason TEXT NULL,
    CONSTRAINT uq_society_documents_id_society UNIQUE (id, society_id)
);

CREATE INDEX IF NOT EXISTS idx_society_documents_lookup 
    ON society_documents(society_id, status, classification, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_society_documents_category 
    ON society_documents(society_id, category_id);

-- 4. Document Versions Table (Append-Only)
CREATE TABLE IF NOT EXISTS document_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    society_id UUID NOT NULL REFERENCES societies(id) ON DELETE RESTRICT,
    document_id UUID NOT NULL,
    version_number INT NOT NULL,
    storage_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    file_hash TEXT NOT NULL,
    change_summary TEXT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_document_versions_document 
        FOREIGN KEY (document_id, society_id) 
        REFERENCES society_documents(id, society_id) ON DELETE RESTRICT,
    CONSTRAINT uq_document_versions_num UNIQUE (document_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_document_versions_doc 
    ON document_versions(document_id, version_number DESC);

-- 5. Foreign Key for Current Version Pointer
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_society_documents_current_version'
    ) THEN
        ALTER TABLE society_documents 
        ADD CONSTRAINT fk_society_documents_current_version 
        FOREIGN KEY (current_version_id) 
        REFERENCES document_versions(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- 6. Trigger to Enforce Document Version Immutability
CREATE OR REPLACE FUNCTION trg_enforce_document_version_immutability()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.storage_path != OLD.storage_path OR
           NEW.file_name != OLD.file_name OR
           NEW.file_size != OLD.file_size OR
           NEW.mime_type != OLD.mime_type OR
           NEW.file_hash != OLD.file_hash OR
           NEW.version_number != OLD.version_number OR
           NEW.document_id != OLD.document_id OR
           NEW.society_id != OLD.society_id THEN
            RAISE EXCEPTION 'document version binary attributes and metadata are immutable';
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'document version records cannot be deleted';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_document_versions_immutable ON document_versions;
CREATE TRIGGER trg_document_versions_immutable
BEFORE UPDATE OR DELETE ON document_versions
FOR EACH ROW EXECUTE FUNCTION trg_enforce_document_version_immutability();

-- 7. Ensure Storage Bucket Exists & Is Configured (Private)
INSERT INTO storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
VALUES (
    'society-documents',
    'society-documents',
    false,
    10485760,
    ARRAY[
        'application/pdf',
        'image/jpeg',
        'image/png',
        'image/webp',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ]
)
ON CONFLICT (id) DO UPDATE SET
    public = false,
    file_size_limit = 10485760,
    allowed_mime_types = EXCLUDED.allowed_mime_types;

-- 8. Seed Default System Categories for All Existing Societies
INSERT INTO document_categories (society_id, name, description, is_system, display_order)
SELECT s.id, c.name, c.description, true, c.ord
FROM societies s
CROSS JOIN (VALUES
    ('Registration and Legal', 'Society registration certificates, legal deeds, and property documents', 1),
    ('Bylaws and Rules', 'Society bylaws, house rules, and community guidelines', 2),
    ('Meetings and Minutes', 'Minutes of Annual General Meetings, SGMs, and Managing Committee meetings', 3),
    ('Financial Records', 'Audited balance sheets, financial statements, and tax filings', 4),
    ('Safety and Compliance', 'Fire-safety compliance certificates, structural audit reports, and NOCs', 5),
    ('Maintenance and Repairs', 'Major repair contracts, lift maintenance logs, and AMC records', 6),
    ('Insurance', 'Building and public liability insurance policies', 7),
    ('General Records', 'General society circulars, correspondence, and notices', 8)
) AS c(name, description, ord)
ON CONFLICT (society_id, name) DO NOTHING;
