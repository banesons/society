-- Migration: 20261026000000_harden_document_storage_limits.sql
-- Description: Enforce 5MB storage bucket limit and max 20 versions per document trigger

-- 1. Update storage bucket file size limit to 5MB (5242880 bytes)
UPDATE storage.buckets
SET file_size_limit = 5242880
WHERE id = 'society-documents';

-- 2. Database trigger enforcing max 20 versions per document
CREATE OR REPLACE FUNCTION trg_check_document_version_limit()
RETURNS TRIGGER AS $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM document_versions
    WHERE document_id = NEW.document_id;

    IF v_count >= 20 THEN
        RAISE EXCEPTION 'maximum version limit of 20 reached for this document'
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_enforce_document_version_limit ON document_versions;
CREATE TRIGGER trg_enforce_document_version_limit
    BEFORE INSERT ON document_versions
    FOR EACH ROW
    EXECUTE FUNCTION trg_check_document_version_limit();
