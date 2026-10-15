# 13 — Documents and Records Security

## 1. Threat Model & Security Perimeter

Sprint S13 handles sensitive administrative, financial, and legal records. The security model addresses the following core risks:

1. **Horizontal Privilege Escalation (Cross-Society IDOR)**:
   A malicious user in Society A attempts to access or mutate documents belonging to Society B.
2. **Vertical Privilege Escalation**:
   An ordinary resident attempts to view or upload `COMMITTEE` or `SOC_ADMIN_ONLY` documents.
3. **Malicious File Ingestion**:
   Attackers attempt to upload executable payloads (PHP, JS, shell scripts), polyglot binaries, HTML/SVG active scripts, or oversized memory exhaustion vectors.
4. **Direct Storage Path Guessing**:
   Attackers attempt to bypass application authorization by directly requesting storage objects from Supabase Storage.
5. **Historical Tampering / Overwrite**:
   Attempting to overwrite previous document revisions to conceal alterations or audit discrepancies.

---

## 2. Defensive Controls & Mitigation Strategy

### 2.1 Multi-Tenant Isolation
- Every database query scopes `WHERE society_id = $1`.
- Foreign key constraints include composite society references (`FOREIGN KEY (document_id, society_id) REFERENCES society_documents(id, society_id)`).
- Document and version endpoints resolve caller society context first via `RequireSocietyAccess` or `RequireResidentAccess`.

### 2.2 Classification Access Matrix
| Caller Role | `SOCIETY_RESIDENTS` | `COMMITTEE` | `SOC_ADMIN_ONLY` | Archived Documents |
| :--- | :--- | :--- | :--- | :--- |
| **Resident** (active unit rel) | **Read Only** | **Deny (403)** | **Deny (403)** | **Deny (404/403)** |
| **Committee Member** (active appt) | **Read + Upload** | **Read + Upload** | **Deny (403)** | **Deny (404/403)** |
| **Society Admin** (`SOC_ADMIN`) | **Full Access** | **Full Access** | **Full Access** | **Read + Restore** |
| **Super Admin** (`SUPER_ADMIN`) | **Full Access** | **Full Access** | **Full Access** | **Read + Restore** |
| **Other Society Resident/Admin** | **Deny (403/404)** | **Deny (403/404)** | **Deny (403/404)** | **Deny (403/404)** |

### 2.3 Storage Hardening & Private Bucket Configuration
- The `society-documents` bucket is strictly private (`public = false`) with `file_size_limit = 5242880` (5 MB).
- Direct public URL access (`/object/public/...`) returns `404 NoSuchBucket` or `400 InvalidRequest`.
- No service-role keys or storage credentials are leaked to frontend code.
- All downloads stream through the Go backend, which inspects authorization on every request.
- Downloads set `X-Content-Type-Options: nosniff` and quote sanitized filenames in `Content-Disposition`.
- Storage administrative guarantees: Acknowledged that platform infrastructure DB/Storage administrators retain low-level infrastructure access, but platform application roles and tenants have zero privilege escalation vectors.

### 2.4 File Signature, Optimization & Ingestion Validation
- **Two-Tier Size Limit**:
  - Original selection limit: 10 MB (10,485,760 bytes) enforced via `io.LimitReader`.
  - Stored artifact limit: 5 MB (5,242,880 bytes) enforced post-optimization.
- **Magic Bytes Verification**:
  - `application/pdf`: starts with `%PDF-`
  - `image/jpeg`: starts with `\xFF\xD8\xFF`
  - `image/png`: starts with `\x89PNG\r\n\x1a\n`
  - `image/webp`: starts with `RIFF` and contains `WEBP`
  - DOCX / XLSX: starts with `PK\x03\x04`
- **Safe Type-Aware Optimization Security**:
  - Optimization is performed in-memory using Go's standard library `image/png` without invoking external shell tools, cgo, or sending data to third parties.
  - Bounded in memory and CPU execution time.
  - Retains original bytes for formats where optimization risks document integrity (PDF, Office, WebP, JPEG).
- **Active Content Prohibition**: SVG, HTML, XML, executable binaries, macros, and script files are rejected unconditionally.
- **Filename Sanitization**: Path separators (`/`, `\`, `..`), control characters, and null bytes are stripped via `SanitizeFilename`.

### 2.5 Immutability, Concurrency Safety & Audit Integrity
- Revisions are strictly append-only.
- **20-Version Ceiling**: Documents cannot exceed 20 revisions (`MaxDocumentVersions = 20`), enforced both via row-level locking `SELECT ... FOR UPDATE` in Go and via the database trigger `trg_enforce_document_version_limit`.
- `UNIQUE (document_id, version_number)` prevents duplicate version numbers under concurrency.
- Cryptographic SHA-256 digests are calculated and recorded on the exact stored bytes.
- **Compensatory Cleanup**: In the event of a database transaction failure following a successful storage upload, `deleteFileFromStorage` is immediately executed to prevent orphaned objects.
- Deletions are forbidden by default; documents use the `ARCHIVED` lifecycle state with a mandatory reason.
- All mutations emit an immutable entry to `platform_audit_logs`.

