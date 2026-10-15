# 14 — Documents and Records API Contracts

## 1. Overview
Sprint S13 endpoints are mounted under `/api/v1/societies/{id}/documents`. All operations require valid JWT authentication, society tenant resolution, and RBAC authorization.

---

## 2. Categories Endpoints

### 2.1 List Document Categories
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents/categories`
- **Access**: `SOCIETY_RESIDENTS`, `COMMITTEE`, `SOC_ADMIN`, `SUPER_ADMIN`
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "categories": [
      {
        "id": "uuid",
        "name": "Registration & Legal",
        "description": "Statutory registrations and legal deeds",
        "is_system": true,
        "is_active": true,
        "display_order": 10
      }
    ]
  }
}
```

### 2.2 Create Document Category
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/documents/categories`
- **Access**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "name": "Vendor Contracts",
  "description": "Agreements with service contractors",
  "display_order": 50
}
```
- **Response** (`201 Created`)

---

## 3. Documents & Versions Endpoints

### 3.1 List Documents
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents`
- **Access**: `SOCIETY_RESIDENTS` (filters to resident-visible), `COMMITTEE` (filters to committee+resident), `SOC_ADMIN`, `SUPER_ADMIN` (can view all)
- **Query Params**:
  - `category_id`: UUID (optional)
  - `classification`: string (optional)
  - `status`: string (default `ACTIVE`)
  - `search`: string (title / description query)
  - `include_archived`: boolean (admin only)
  - `limit`: int (default 20, max 100)
  - `offset`: int (default 0)
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "documents": [
      {
        "id": "uuid",
        "document_number": "DOC-PUN-202610-10001",
        "title": "Society Bylaws (Registered 2026)",
        "description": "Official registered bylaws of the society",
        "category_id": "uuid",
        "category_name": "Bylaws & Rules",
        "classification": "SOCIETY_RESIDENTS",
        "status": "ACTIVE",
        "current_version_number": 1,
        "file_name": "bylaws-2026.pdf",
        "file_size": 245102,
        "mime_type": "application/pdf",
        "uploaded_by_name": "Vijay Godbole",
        "created_at": "2026-10-13T12:00:00Z",
        "updated_at": "2026-10-13T12:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### 3.2 Upload New Document (Version 1)
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/documents`
- **Access**: `SOC_ADMIN`, `SUPER_ADMIN`, or authorized `COMMITTEE` (only for `COMMITTEE` / `SOCIETY_RESIDENTS` classification)
- **Content-Type**: `multipart/form-data`
- **Form Fields**:
  - `title`: string (required)
  - `description`: string (optional)
  - `category_id`: UUID (required)
  - `classification`: `SOCIETY_RESIDENTS` | `COMMITTEE` | `SOC_ADMIN_ONLY` (required)
  - `change_summary`: string (optional)
  - `file`: binary file (required, max original 10MB, max accepted stored artifact 5MB)
- **Validation & Size Limits**:
  - Original file size must not exceed 10 MB (returns `FILE_TOO_LARGE` on violation).
  - Safe, type-aware optimization is applied (lossless PNG re-encoding with `png.BestCompression`).
  - Stored artifact must not exceed 5 MB (returns `FILE_EXCEEDS_ACCEPTED_LIMIT` on violation).
- **Response** (`201 Created`): Returns document record and version 1 details.

### 3.3 Get Document Details
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}`
- **Access**: Evaluated based on caller's role against document's `classification`.
- **Response** (`200 OK`)

### 3.4 Upload New Document Version
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}/versions`
- **Access**: `SOC_ADMIN`, `SUPER_ADMIN`, or authorized `COMMITTEE`
- **Content-Type**: `multipart/form-data`
- **Form Fields**:
  - `change_summary`: string (optional)
  - `file`: binary file (required, max original 10MB, max accepted stored artifact 5MB)
- **Version Limits**:
  - Maximum ceiling of 20 versions per document. If document already has 20 versions, returns `400 Bad Request` with code `MAX_VERSIONS_REACHED`.
  - Concurrency is protected via PostgreSQL `FOR UPDATE` row lock and database trigger `trg_enforce_document_version_limit`.
- **Response** (`201 Created`): Creates incremented `version_number` and atomically updates `current_version_id`.

### 3.5 List Document Versions
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}/versions`
- **Access**: Evaluated based on caller's role against document's `classification`.
- **Response** (`200 OK`): Array of versions with version numbers, timestamps, hashes, uploaders.

### 3.6 Download or Preview Document Version
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}/versions/{versionId}/download`
- **Access**: Evaluated based on caller's role against document's `classification`.
- **Query Params**:
  - `inline`: boolean (default `false`; `true` serves with `Content-Disposition: inline` for browser viewing, `false` serves with `Content-Disposition: attachment`).
- **Headers**: Enforces `X-Content-Type-Options: nosniff`.
- **Response**: Binary stream with validated `Content-Type` header.

### 3.7 Archive Document
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}/archive`
- **Access**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Request Body**:
```json
{
  "reason": "Superseded by revised 2027 deed"
}
```
- **Response** (`200 OK`)

### 3.8 Restore Archived Document
- **Method**: `POST`
- **Path**: `/api/v1/societies/{id}/documents/{documentId}/restore`
- **Access**: `SOC_ADMIN`, `SUPER_ADMIN`
- **Response** (`200 OK`)

### 3.9 Document Storage Observability Metrics
- **Method**: `GET`
- **Path**: `/api/v1/societies/{id}/documents/metrics`
- **Access**: `SOCIETY_RESIDENTS`, `COMMITTEE`, `SOC_ADMIN`, `SUPER_ADMIN`
- **Response** (`200 OK`):
```json
{
  "success": true,
  "data": {
    "metrics": {
      "total_documents": 12,
      "active_documents": 11,
      "archived_documents": 1,
      "total_versions": 25,
      "total_stored_bytes": 1458920
    }
  }
}
```
- **Semantics**: `total_stored_bytes` represents **metadata-derived expected storage bytes** calculated from PostgreSQL `SUM(file_size)` of committed versions. It reflects the application-tracked size of active and archived documents rather than physical cloud bucket size.
- **Batch Policy**: Upload endpoints accept a single file per document or version creation. Any 5-file / 25 MiB batch upload policy is a future roadmap item; no batch endpoint is active in S13.

---

## 4. Standard Error Codes

| HTTP Status | Error Code | Description |
| :--- | :--- | :--- |
| `400` | `UNSUPPORTED_FILE_TYPE` | Format not permitted (only PDF, PNG, JPEG, WebP, DOCX, XLSX allowed) |
| `400` | `MIME_MISMATCH` | Magic byte signature does not match declared MIME |
| `400` | `FILE_TOO_LARGE` | Selected original file exceeds 10 MB limit |
| `400` | `FILE_EXCEEDS_ACCEPTED_LIMIT` | File exceeds 5 MB accepted limit after type-aware optimization |
| `400` | `MAX_VERSIONS_REACHED` | Document has reached the ceiling of 20 versions |
| `400` | `DOCUMENT_ARCHIVED` | Cannot upload version to an archived document |
| `403` | `DOCUMENT_ACCESS_DENIED` | Caller role insufficient for document classification |
| `404` | `DOCUMENT_NOT_FOUND` | Document does not exist or belongs to another society |

