# 13 — Documents and Records Database Schema

## 1. Overview
Sprint S13 establishes the database tables, sequences, foreign keys, and indexes required for multi-tenant document and records management.

All tables are defined in the public schema, and the private object bucket `society-documents` is established in the `storage` schema.

---

## 2. Sequences

### `document_number_seq`
Sequence generating sequential numeric identifiers for document references formatted as `DOC-SOC-YYYYMM-XXXXX`.
```sql
CREATE SEQUENCE IF NOT EXISTS document_number_seq START WITH 10001 INCREMENT BY 1;
```

---

## 3. Tables

### 3.1 `document_categories`
Catalog of document categories per society.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal unique identifier |
| `society_id` | UUID | NOT NULL, REFERENCES `societies(id)` | Tenant society reference |
| `name` | TEXT | NOT NULL | Category name |
| `description` | TEXT | NULL | Detailed description |
| `is_system` | BOOLEAN | NOT NULL DEFAULT false | Whether seeded as a system default |
| `is_active` | BOOLEAN | NOT NULL DEFAULT true | Operational state |
| `display_order` | INT | NOT NULL DEFAULT 0 | Sorting priority in UI |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Timestamp of creation |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Timestamp of last update |

**Constraints & Indexes**:
- `CONSTRAINT uq_document_categories_society_name UNIQUE (society_id, name)`
- `CREATE INDEX idx_document_categories_society ON document_categories(society_id, is_active)`

---

### 3.2 `society_documents`
The root metadata record for an official society document.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Document unique identifier |
| `society_id` | UUID | NOT NULL, REFERENCES `societies(id)` | Tenant society reference |
| `category_id` | UUID | NOT NULL, REFERENCES `document_categories(id)` | Category reference |
| `document_number` | TEXT | NOT NULL UNIQUE | Formatted identifier `DOC-SOC-YYYYMM-XXXXX` |
| `title` | TEXT | NOT NULL | Human-readable document title |
| `description` | TEXT | NULL | Summary or notes |
| `classification` | TEXT | NOT NULL DEFAULT 'SOCIETY_RESIDENTS' | Access policy |
| `status` | TEXT | NOT NULL DEFAULT 'ACTIVE' | Lifecycle state |
| `current_version_id` | UUID | NULL | Pointer to current version |
| `building_id` | UUID | NULL, REFERENCES `society_buildings(id)` | Optional building association |
| `wing_id` | UUID | NULL, REFERENCES `society_wings(id)` | Optional wing association |
| `unit_id` | UUID | NULL, REFERENCES `society_units(id)` | Optional unit association |
| `member_id` | UUID | NULL, REFERENCES `society_members(id)` | Optional member association |
| `created_by` | UUID | NOT NULL REFERENCES `users(id)` | User who created the document |
| `updated_by` | UUID | NOT NULL REFERENCES `users(id)` | User who last updated metadata |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Record creation timestamp |
| `updated_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Record update timestamp |
| `archived_at` | TIMESTAMPTZ | NULL | Timestamp of archival |
| `archived_by` | UUID | NULL REFERENCES `users(id)` | Administrator who archived |
| `archived_reason` | TEXT | NULL | Mandatory reason for archival |

**Constraints & Indexes**:
- `CONSTRAINT chk_document_classification CHECK (classification IN ('SOCIETY_RESIDENTS', 'COMMITTEE', 'SOC_ADMIN_ONLY'))`
- `CONSTRAINT chk_document_status CHECK (status IN ('ACTIVE', 'ARCHIVED'))`
- `CONSTRAINT uq_society_documents_id_society UNIQUE (id, society_id)`
- `CREATE INDEX idx_society_documents_lookup ON society_documents(society_id, status, classification)`
- `CREATE INDEX idx_society_documents_category ON society_documents(society_id, category_id)`

---

### 3.3 `document_versions`
Historical and active revisions for documents.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Version unique identifier |
| `society_id` | UUID | NOT NULL | Tenant society reference |
| `document_id` | UUID | NOT NULL | Document parent reference |
| `version_number` | INT | NOT NULL | Sequential version counter |
| `storage_path` | TEXT | NOT NULL | Object key in storage bucket |
| `file_name` | TEXT | NOT NULL | Sanitized original filename |
| `file_size` | BIGINT | NOT NULL | File size in bytes |
| `mime_type` | TEXT | NOT NULL | Validated MIME type |
| `file_hash` | TEXT | NOT NULL | Cryptographic SHA-256 hex hash |
| `change_summary` | TEXT | NULL | Change summary or notes |
| `uploaded_by` | UUID | NOT NULL REFERENCES `users(id)` | Uploader reference |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | Upload timestamp |

**Constraints & Indexes**:
- `CONSTRAINT fk_document_versions_document FOREIGN KEY (document_id, society_id) REFERENCES society_documents(id, society_id) ON DELETE RESTRICT`
- `CONSTRAINT uq_document_versions_num UNIQUE (document_id, version_number)`
- `CREATE INDEX idx_document_versions_doc ON document_versions(document_id, version_number DESC)`
- **Version Ceiling Trigger**: `trg_enforce_document_version_limit` on `document_versions` fires `BEFORE INSERT` executing `fn_enforce_document_version_limit()` to guarantee that no document exceeds 20 versions.

---

## 4. Supabase Storage Configuration

Private bucket setup in `storage.buckets` (Hardened to 5 MB per `20261026000000_harden_document_storage_limits.sql`):
```sql
UPDATE storage.buckets
SET file_size_limit = 5242880, -- 5 MB accepted limit
    public = false,
    allowed_mime_types = ARRAY[
        'application/pdf',
        'image/jpeg',
        'image/png',
        'image/webp',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
    ]
WHERE id = 'society-documents';
```
- Bucket status: `public = false` (anonymous direct retrieval blocked).
- Stored artifact size limit: `5242880` bytes (5 MB).

