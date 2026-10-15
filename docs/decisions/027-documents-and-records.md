# ADR 027: Documents and Records Architecture

## Status
ACCEPTED

## Date
2026-10-13

## Context
Cooperative Housing Societies manage legal, structural, financial, operational, and administrative records. S13 establishes a multi-tenant document repository within the CHS modular monolith.

Key requirements:
1. **Multi-Tenancy**: Strict society isolation. Documents belonging to one society must never be accessible to users in another society.
2. **Access Classifications**:
   - `SOCIETY_RESIDENTS`: Public within the society; viewable by all active residents and society admins.
   - `COMMITTEE`: Restricted to active managing committee members and society admins.
   - `SOC_ADMIN_ONLY`: Restricted to society administrators and platform Super Admins.
3. **Private File Storage**: File objects must reside in a private Supabase Storage bucket (`society-documents`). Direct unauthenticated public access is prohibited.
4. **Backend-Mediated Security**: The Go backend validates tenant boundaries, roles, file signatures, size limits, and checksums before persisting metadata or streaming/downloading files.
5. **Version History**: Documents maintain an append-only version history. When a new version is uploaded, previous versions are preserved with immutable cryptographic hashes (SHA-256).
6. **Controlled Archival**: Documents are deactivated via an `ARCHIVED` status with mandatory audit recording; records are never hard-deleted during standard operations.
7. **Audit Logging**: Significant mutations (creation, updates, new versions, archival, restoration, downloads) are recorded in the platform audit log.

## Decision

### 1. Storage Architecture & Backend-Mediated Transfer
- Storage engine: Supabase Storage private bucket `society-documents`.
- Bucket configuration: `public = false`, `file_size_limit = 5242880` (5 MB accepted artifact limit), explicit MIME allowlist.
- Storage key layout: `societies/{society_id}/documents/{document_id}/versions/{version_id}/{filename_slug}`.
- Transfer pattern:
  - **Uploads**: Client submits `multipart/form-data` to Go backend (`/api/v1/societies/{id}/documents` or `/versions`). Backend inspects magic bytes, validates MIME against allowlist, applies safe type-aware optimization, verifies post-optimization size <= 5 MB, calculates SHA-256 on stored bytes, writes to Supabase Storage via authenticated internal endpoint, and commits metadata transactionally.
  - **Downloads & Previews**: Client requests `/download` from Go backend. Backend verifies caller authorization and classification. If authorized, backend streams the object with `Content-Disposition: inline` (for preview) or `attachment; filename="..."` (for download). `X-Content-Type-Options: nosniff` is enforced. No long-lived or public storage URLs are exposed.

### 2. File Validation, Optimization, and Security Controls
- **MIME Allowlist**:
  - `application/pdf`
  - `image/jpeg`
  - `image/png`
  - `image/webp`
  - `application/vnd.openxmlformats-officedocument.wordprocessingml.document` (DOCX)
  - `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` (XLSX)
- **Signature (Magic Byte) Verification**: Sniffs initial bytes to verify magic headers (`%PDF-`, `\xFF\xD8\xFF`, `\x89PNG`, `RIFF...WEBP`, `PK\x03\x04`). Rejects mismatching extensions, scripts, executables, HTML, or SVG.
- **Two-Tier File Size Policy**:
  - User-selected original input size limit: **10 MB** (`MaxOriginalFileSize`).
  - Stored artifact upload limit: **5 MB** (`MaxAcceptedFileSize` / 5,242,880 bytes).
  - Files larger than 5 MB up to 10 MB are accepted only if safe optimization reduces their size to <= 5 MB; otherwise rejected with clear actionable error `FILE_EXCEEDS_ACCEPTED_LIMIT`.
- **Safe, Bounded Type-Aware Optimization**:
  - `image/png`: Lossless re-encoding using standard library `png.BestCompression` preserving alpha channels and pixel dimensions. Retains original bytes if optimization does not yield reduction.
  - `application/pdf`, `image/jpeg`, `image/webp`, DOCX, XLSX: Retain original bytes without blind recompression or quality loss.
  - SHA-256 hash is computed on the exact stored bytes after optimization.
- **Batch Policy**: Single-file per document/version is currently implemented. Any 5-file / 25 MiB batch policy represents a reserved architectural roadmap concept for future multi-file extensions; no batch upload endpoint is implemented or active in S13.
- **Filename Sanitization**: Strip path traversal characters, directory separators, and control characters. Storage keys use UUIDs; original filenames preserved strictly as sanitized metadata.

### 3. Classification and Authorization Model
- Classifications:
  - `SOC_ADMIN_ONLY`: Caller must have active `SOC_ADMIN` role in `society_user_roles` or be platform `SUPER_ADMIN`.
  - `COMMITTEE`: Caller must have active appointment in `society_committee_appointments` (within valid term dates), or hold `SOC_ADMIN` / `SUPER_ADMIN`.
  - `SOCIETY_RESIDENTS`: Caller must hold active `unit_relationships` association with an active unit in the society, or hold `COMMITTEE` / `SOC_ADMIN` / `SUPER_ADMIN`.
- Archival visibility: Archived documents are visible only to `SOC_ADMIN` and `SUPER_ADMIN` with explicit `include_archived=true` filter.

### 4. Versioning Model and Concurrency Safety
- Sequential version numbering (`1, 2, 3...`) scoped to `document_id`.
- **Version Limit**: 20 versions maximum per document (`MaxDocumentVersions = 20`).
- **Concurrency Safety**:
  - Application layer: Row-level lock `SELECT ... FROM society_documents WHERE id = $1 AND society_id = $2 FOR UPDATE` precedes version count check and next version allocation within transaction.
  - Database engine layer: Trigger `trg_enforce_document_version_limit` actively prevents any insert exceeding 20 versions.
- `current_version_id` on `society_documents` points to the latest active version.
- **Compensatory Cleanup**: Storage uploads that succeed but encounter subsequent database commit failures immediately execute compensatory object deletion `deleteFileFromStorage` to prevent orphaned storage objects.
- **Observability**: `GET /api/v1/societies/{id}/documents/metrics` exposes total documents, active/archived counts, total versions, and `total_stored_bytes`. The stored byte metric represents **metadata-derived expected storage bytes** (calculated from `SUM(v.file_size)` of committed versions in PostgreSQL), not independently polled physical bucket usage.
- **Historical Preservation**: Historical files uploaded prior to the 5 MB policy remain intact, non-rewritten, and fully downloadable.

## Consequences

### Positive
- Strict tenant boundary isolation; zero possibility of leaking documents across societies.
- Strong defense against malware, active content (SVG/HTML execution), and file confusion attacks.
- Complete audit trail of all document actions and version changes.
- Seamless compatibility with existing frontend Swiss-minimalist design system and Svelte 5 components.

### Negative / Trade-offs
- File uploads stream through Go backend rather than client direct-to-S3, placing network transfer on the backend server. Given the 10 MB limit and society operational volumes, this is well within acceptable capacity and provides absolute authorization guarantees.
