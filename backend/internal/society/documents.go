package society

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// S13 Domain Errors
var (
	ErrDocumentNotFound         = errors.New("document not found in this society")
	ErrDocumentVersionNotFound  = errors.New("document version not found")
	ErrCategoryNotFound         = errors.New("document category not found in this society")
	ErrCategoryInactive         = errors.New("document category is inactive")
	ErrInvalidClassification    = errors.New("invalid document classification")
	ErrDocumentArchived         = errors.New("document is archived")
	ErrDocumentNotArchived      = errors.New("document is not archived")
	ErrArchivalReasonRequired   = errors.New("reason is required when archiving a document")
	ErrInvalidDocumentTitle     = errors.New("document title must be between 3 and 200 characters")
	ErrUnsupportedFileType      = errors.New("unsupported file format: only PDF, JPEG, PNG, WebP, DOCX, and XLSX are permitted")
	ErrMimeMismatch             = errors.New("file content signature does not match declared MIME type")
	ErrFileTooLarge             = errors.New("file size exceeds maximum allowed limit")
	ErrFileExceedsOriginalLimit = errors.New("original file size exceeds maximum limit of 10MB")
	ErrFileExceedsAcceptedLimit = errors.New("file size exceeds maximum accepted limit of 5MB after optimization")
	ErrMaxVersionsReached       = errors.New("maximum version limit of 20 reached for this document")
	ErrEmptyFile                = errors.New("uploaded file is empty")
	ErrInvalidFilename          = errors.New("invalid or unsafe filename")
	ErrDocumentAccessDenied     = errors.New("you do not have permission to access this document")
	ErrStorageOperationFailed   = errors.New("storage provider operation failed")
)

const (
	MaxOriginalFileSize    = 10 * 1024 * 1024    // 10MB user-selected limit
	MaxAcceptedFileSize    = 5 * 1024 * 1024     // 5MB stored artifact limit
	MaxDocumentFileSize    = MaxAcceptedFileSize // Backwards-compatible alias
	MaxDocumentVersions    = 20
	BucketSocietyDocuments = "society-documents"
)

var (
	safeFilenameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-\. ]+$`)
)

// Allowed MIME types map to expected signatures
type mimeValidator struct {
	allowedMime string
	matchFunc   func([]byte, string) bool
}

var allowedMimeValidators = []mimeValidator{
	{
		allowedMime: "application/pdf",
		matchFunc: func(data []byte, _ string) bool {
			return bytes.HasPrefix(data, []byte("%PDF-"))
		},
	},
	{
		allowedMime: "image/jpeg",
		matchFunc: func(data []byte, _ string) bool {
			return bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF})
		},
	},
	{
		allowedMime: "image/png",
		matchFunc: func(data []byte, _ string) bool {
			return bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
		},
	},
	{
		allowedMime: "image/webp",
		matchFunc: func(data []byte, _ string) bool {
			return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
		},
	},
	{
		allowedMime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		matchFunc: func(data []byte, filename string) bool {
			isZip := bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04})
			return isZip && strings.HasSuffix(strings.ToLower(filename), ".docx")
		},
	},
	{
		allowedMime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		matchFunc: func(data []byte, filename string) bool {
			isZip := bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04})
			return isZip && strings.HasSuffix(strings.ToLower(filename), ".xlsx")
		},
	},
}

// StorageConfig stores credentials and endpoint for Supabase Storage.
type StorageConfig struct {
	BaseURL string
	AnonKey string
}

// SetStorageConfig sets Supabase storage configuration on the repository.
func (r *Repository) SetStorageConfig(baseURL, anonKey string) {
	r.storageBaseURL = strings.TrimRight(baseURL, "/")
	r.storageAnonKey = anonKey
}

// SanitizeFilename cleans and validates original filename.
func SanitizeFilename(raw string) (string, error) {
	// Normalize Windows-style separators first
	normalized := strings.ReplaceAll(raw, "\\", "/")
	base := filepath.Base(normalized)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == ".." {
		return "", ErrInvalidFilename
	}
	if len(base) > 150 {
		ext := filepath.Ext(base)
		name := base[:150-len(ext)]
		base = name + ext
	}
	// Avoid any path traversal or control characters
	if strings.Contains(base, "/") || strings.Contains(base, "\\") || strings.Contains(base, "\x00") {
		return "", ErrInvalidFilename
	}
	return base, nil
}

// OptimizeFileContent applies safe, bounded, type-aware optimization.
// For PNG: performs lossless re-encoding using png.BestCompression. Keeps original if optimization is not smaller.
// For PDF, JPEG, WebP, DOCX, XLSX: retains original bytes without blind recompression or quality loss.
func OptimizeFileContent(content []byte, mimeType string) ([]byte, bool) {
	if mimeType == "image/png" && len(content) > 0 {
		img, err := png.Decode(bytes.NewReader(content))
		if err == nil {
			var buf bytes.Buffer
			enc := png.Encoder{CompressionLevel: png.BestCompression}
			if err := enc.Encode(&buf, img); err == nil {
				optBytes := buf.Bytes()
				if len(optBytes) < len(content) {
					return optBytes, true
				}
			}
		}
	}
	return content, false
}

// ValidateFileContent verifies file size, magic bytes, optimizes where supported, and computes SHA-256 hash.
// Returns validatedMime, storedBytes, fileHash, and error.
func ValidateFileContent(content []byte, declaredMime, filename string) (string, []byte, string, error) {
	if len(content) == 0 {
		return "", nil, "", ErrEmptyFile
	}
	if len(content) > MaxOriginalFileSize {
		return "", nil, "", ErrFileExceedsOriginalLimit
	}

	sanitizedName, err := SanitizeFilename(filename)
	if err != nil {
		return "", nil, "", err
	}

	// Detect matching allowed validator
	var matchedMime string
	for _, val := range allowedMimeValidators {
		if val.matchFunc(content, sanitizedName) {
			matchedMime = val.allowedMime
			break
		}
	}

	if matchedMime == "" {
		return "", nil, "", ErrUnsupportedFileType
	}

	// If client provided a declared MIME, verify it doesn't conflict
	if declaredMime != "" && declaredMime != "application/octet-stream" {
		if !strings.EqualFold(declaredMime, matchedMime) {
			// For images, if client says image/jpg but validator returns image/jpeg, allow
			if !(declaredMime == "image/jpg" && matchedMime == "image/jpeg") {
				return "", nil, "", ErrMimeMismatch
			}
		}
	}

	// Apply safe, type-aware optimization
	storedBytes, _ := OptimizeFileContent(content, matchedMime)
	if len(storedBytes) > MaxAcceptedFileSize {
		return "", nil, "", ErrFileExceedsAcceptedLimit
	}

	hash := sha256.Sum256(storedBytes)
	fileHash := hex.EncodeToString(hash[:])

	return matchedMime, storedBytes, fileHash, nil
}

// uploadFileToStorage sends the binary to private Supabase Storage bucket.
func (r *Repository) uploadFileToStorage(ctx context.Context, storagePath, mimeType string, content []byte) error {
	if r.storageBaseURL == "" || r.storageAnonKey == "" {
		return errors.New("storage credentials not configured")
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", r.storageBaseURL, BucketSocietyDocuments, storagePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to create storage request: %w", err)
	}

	req.Header.Set("apikey", r.storageAnonKey)
	req.Header.Set("Authorization", "Bearer "+r.storageAnonKey)
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "false")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("storage request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: storage upload returned status %d: %s", ErrStorageOperationFailed, resp.StatusCode, string(body))
	}

	return nil
}

// DownloadFileFromStorage streams the file binary from private Supabase Storage.
func (r *Repository) DownloadFileFromStorage(ctx context.Context, storagePath string) (io.ReadCloser, string, int64, error) {
	if r.storageBaseURL == "" || r.storageAnonKey == "" {
		return nil, "", 0, errors.New("storage credentials not configured")
	}

	url := fmt.Sprintf("%s/storage/v1/object/authenticated/%s/%s", r.storageBaseURL, BucketSocietyDocuments, storagePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create storage request: %w", err)
	}

	req.Header.Set("apikey", r.storageAnonKey)
	req.Header.Set("Authorization", "Bearer "+r.storageAnonKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("storage download failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", 0, fmt.Errorf("%w: download status %d", ErrStorageOperationFailed, resp.StatusCode)
	}

	return resp.Body, resp.Header.Get("Content-Type"), resp.ContentLength, nil
}

// deleteFileFromStorage deletes an object from private Supabase Storage (used for rollback).
func (r *Repository) deleteFileFromStorage(ctx context.Context, storagePath string) error {
	if r.storageBaseURL == "" || r.storageAnonKey == "" {
		return nil
	}

	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", r.storageBaseURL, BucketSocietyDocuments, storagePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", r.storageAnonKey)
	req.Header.Set("Authorization", "Bearer "+r.storageAnonKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// CreateDocument uploads the initial file to storage and persists document + version 1.
func (r *Repository) CreateDocument(ctx context.Context, societyID uuid.UUID, params CreateDocumentParams) (*SocietyDocument, error) {
	title := strings.TrimSpace(params.Title)
	if len(title) < 3 || len(title) > 200 {
		return nil, ErrInvalidDocumentTitle
	}

	switch params.Classification {
	case ClassificationSocietyResidents, ClassificationCommittee, ClassificationSocAdminOnly:
	default:
		return nil, ErrInvalidClassification
	}

	// 1. Verify Category exists in society and is active
	var categoryName string
	var categoryActive bool
	err := r.pool.QueryRow(ctx, `
		SELECT name, is_active FROM document_categories
		WHERE id = $1 AND society_id = $2
	`, params.CategoryID, societyID).Scan(&categoryName, &categoryActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to verify category: %w", err)
	}
	if !categoryActive {
		return nil, ErrCategoryInactive
	}

	// 2. Validate associations belong to the same society
	if params.BuildingID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_buildings WHERE id = $1 AND society_id = $2)`, params.BuildingID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("building not found in this society")
		}
	}
	if params.WingID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_wings WHERE id = $1 AND society_id = $2)`, params.WingID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("wing not found in this society")
		}
	}
	if params.UnitID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_units WHERE id = $1 AND society_id = $2)`, params.UnitID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("unit not found in this society")
		}
	}
	if params.MemberID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_members WHERE id = $1 AND society_id = $2)`, params.MemberID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("member not found in this society")
		}
	}

	// 3. Validate file content and signature
	validatedMime, storedBytes, fileHash, err := ValidateFileContent(params.FileContent, params.MimeType, params.FileName)
	if err != nil {
		return nil, err
	}
	sanitizedFilename, _ := SanitizeFilename(params.FileName)

	// 4. Generate IDs and document number
	documentID := uuid.New()
	versionID := uuid.New()

	// Get society code for reference
	var socCode string
	err = r.pool.QueryRow(ctx, `SELECT code FROM societies WHERE id = $1`, societyID).Scan(&socCode)
	if err != nil {
		socCode = "SOC"
	}

	var seqVal int64
	err = r.pool.QueryRow(ctx, `SELECT nextval('document_number_seq')`).Scan(&seqVal)
	if err != nil {
		return nil, fmt.Errorf("failed to generate document number: %w", err)
	}
	docNumber := fmt.Sprintf("DOC-%s-%s-%05d", socCode, time.Now().Format("200601"), seqVal)

	// 5. Storage path: societies/{society_id}/documents/{document_id}/versions/{version_id}/{clean_name}
	storagePath := fmt.Sprintf("societies/%s/documents/%s/versions/%s/%s", societyID, documentID, versionID, sanitizedFilename)

	// 6. Upload to private Supabase Storage
	if err := r.uploadFileToStorage(ctx, storagePath, validatedMime, storedBytes); err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	// 7. Insert metadata in PostgreSQL transaction with rollback cleanup
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert society_documents with current_version_id NULL initially
	insertDocQuery := `
		INSERT INTO society_documents (
			id, society_id, category_id, document_number, title, description,
			classification, status, current_version_id, building_id, wing_id,
			unit_id, member_id, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE', NULL, $8, $9, $10, $11, $12, $12)
	`
	_, err = tx.Exec(ctx, insertDocQuery,
		documentID, societyID, params.CategoryID, docNumber, title, params.Description,
		params.Classification, params.BuildingID, params.WingID, params.UnitID, params.MemberID, params.ActorID,
	)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Insert version 1
	insertVersionQuery := `
		INSERT INTO document_versions (
			id, society_id, document_id, version_number, storage_path, file_name,
			file_size, mime_type, file_hash, change_summary, uploaded_by
		) VALUES ($1, $2, $3, 1, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(ctx, insertVersionQuery,
		versionID, societyID, documentID, storagePath, sanitizedFilename,
		int64(len(storedBytes)), validatedMime, fileHash, params.ChangeSummary, params.ActorID,
	)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to insert document version: %w", err)
	}

	// Point current_version_id to version 1
	_, err = tx.Exec(ctx, `UPDATE society_documents SET current_version_id = $1 WHERE id = $2`, versionID, documentID)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to update current version reference: %w", err)
	}

	// Insert audit log
	auditPayload, _ := json.Marshal(map[string]any{
		"document_id":     documentID,
		"document_number": docNumber,
		"title":           title,
		"category_id":     params.CategoryID,
		"classification":  params.Classification,
		"version_number":  1,
		"file_name":       sanitizedFilename,
		"file_size":       len(storedBytes),
		"file_hash":       fileHash,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, params.ActorID, ActionDocumentCreated, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to commit document creation: %w", err)
	}

	return r.GetDocumentByID(ctx, societyID, documentID)
}

// UploadVersion adds a new version to an existing document.
func (r *Repository) UploadVersion(ctx context.Context, societyID uuid.UUID, params UploadVersionParams) (*DocumentVersion, error) {
	// 1. Validate file content and signature first
	validatedMime, storedBytes, fileHash, err := ValidateFileContent(params.FileContent, params.MimeType, params.FileName)
	if err != nil {
		return nil, err
	}
	sanitizedFilename, _ := SanitizeFilename(params.FileName)

	versionID := uuid.New()
	storagePath := fmt.Sprintf("societies/%s/documents/%s/versions/%s/%s", societyID, params.DocumentID, versionID, sanitizedFilename)

	// 2. Start database transaction with FOR UPDATE row lock to ensure concurrency safety
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	var docTitle string
	var docNumber string
	err = tx.QueryRow(ctx, `
		SELECT status, title, document_number FROM society_documents
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`, params.DocumentID, societyID).Scan(&status, &docTitle, &docNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to query document: %w", err)
	}
	if status == DocumentStatusArchived {
		return nil, ErrDocumentArchived
	}

	// 3. Check version count and allocate next sequential version number
	var currentCount int
	var maxVer int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(MAX(version_number), 0)
		FROM document_versions
		WHERE document_id = $1
	`, params.DocumentID).Scan(&currentCount, &maxVer)
	if err != nil {
		return nil, fmt.Errorf("failed to check document version count: %w", err)
	}

	if currentCount >= MaxDocumentVersions {
		return nil, ErrMaxVersionsReached
	}

	nextVer := maxVer + 1

	// 4. Upload file to Supabase Storage
	if err := r.uploadFileToStorage(ctx, storagePath, validatedMime, storedBytes); err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	// 5. Insert version in database
	insertVersionQuery := `
		INSERT INTO document_versions (
			id, society_id, document_id, version_number, storage_path, file_name,
			file_size, mime_type, file_hash, change_summary, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = tx.Exec(ctx, insertVersionQuery,
		versionID, societyID, params.DocumentID, nextVer, storagePath, sanitizedFilename,
		int64(len(storedBytes)), validatedMime, fileHash, params.ChangeSummary, params.ActorID,
	)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		if strings.Contains(err.Error(), "maximum version limit of 20 reached") {
			return nil, ErrMaxVersionsReached
		}
		return nil, fmt.Errorf("failed to insert new version: %w", err)
	}

	// Atomically switch current_version_id and update updated_by/updated_at
	_, err = tx.Exec(ctx, `
		UPDATE society_documents
		SET current_version_id = $1, updated_by = $2, updated_at = NOW()
		WHERE id = $3 AND society_id = $4
	`, versionID, params.ActorID, params.DocumentID, societyID)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to update current version reference: %w", err)
	}

	// Audit log
	auditPayload, _ := json.Marshal(map[string]any{
		"document_id":     params.DocumentID,
		"document_number": docNumber,
		"title":           docTitle,
		"version_id":      versionID,
		"version_number":  nextVer,
		"file_name":       sanitizedFilename,
		"file_size":       len(storedBytes),
		"file_hash":       fileHash,
		"change_summary":  params.ChangeSummary,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, params.ActorID, ActionDocumentVersionUploaded, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		_ = r.deleteFileFromStorage(ctx, storagePath)
		return nil, fmt.Errorf("failed to commit version upload: %w", err)
	}

	return r.GetDocumentVersionByID(ctx, societyID, params.DocumentID, versionID)
}

// GetDocumentByID retrieves a single society document with its current version and associations.
func (r *Repository) GetDocumentByID(ctx context.Context, societyID, documentID uuid.UUID) (*SocietyDocument, error) {
	query := `
		SELECT 
			d.id, d.society_id, d.category_id, c.name AS category_name,
			d.document_number, d.title, d.description, d.classification, d.status,
			d.current_version_id, d.building_id, b.name AS building_name,
			d.wing_id, w.name AS wing_name, d.unit_id, u.unit_number,
			d.member_id, TRIM(CONCAT(m.first_name, ' ', m.last_name)) AS member_name,
			d.created_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = d.created_by AND sm.society_id = d.society_id LIMIT 1), cu.phone, 'User') AS created_by_name, d.updated_by,
			d.created_at, d.updated_at, d.archived_at, d.archived_by, d.archived_reason,
			(SELECT COUNT(*) FROM document_versions dv WHERE dv.document_id = d.id) AS version_count,
			v.id, v.version_number, v.storage_path, v.file_name, v.file_size, v.mime_type, v.file_hash,
			v.change_summary, v.uploaded_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = v.uploaded_by AND sm.society_id = v.society_id LIMIT 1), vu.phone, 'User') AS uploaded_by_name, v.created_at
		FROM society_documents d
		JOIN document_categories c ON c.id = d.category_id
		LEFT JOIN society_buildings b ON b.id = d.building_id
		LEFT JOIN society_wings w ON w.id = d.wing_id
		LEFT JOIN society_units u ON u.id = d.unit_id
		LEFT JOIN society_members m ON m.id = d.member_id
		LEFT JOIN users cu ON cu.id = d.created_by
		LEFT JOIN document_versions v ON v.id = d.current_version_id
		LEFT JOIN users vu ON vu.id = v.uploaded_by
		WHERE d.id = $1 AND d.society_id = $2
	`

	var doc SocietyDocument
	var curVer DocumentVersion
	var curVerID *uuid.UUID
	var vID *uuid.UUID
	var vNum *int
	var vPath, vName, vMime, vHash *string
	var vSize *int64
	var vChange *string
	var vUploadedBy *uuid.UUID
	var vUploadedByName *string
	var vCreatedAt *time.Time

	err := r.pool.QueryRow(ctx, query, documentID, societyID).Scan(
		&doc.ID, &doc.SocietyID, &doc.CategoryID, &doc.CategoryName,
		&doc.DocumentNumber, &doc.Title, &doc.Description, &doc.Classification, &doc.Status,
		&curVerID, &doc.BuildingID, &doc.BuildingName,
		&doc.WingID, &doc.WingName, &doc.UnitID, &doc.UnitNumber,
		&doc.MemberID, &doc.MemberName,
		&doc.CreatedBy, &doc.CreatedByName, &doc.UpdatedBy,
		&doc.CreatedAt, &doc.UpdatedAt, &doc.ArchivedAt, &doc.ArchivedBy, &doc.ArchivedReason,
		&doc.VersionCount,
		&vID, &vNum, &vPath, &vName, &vSize, &vMime, &vHash,
		&vChange, &vUploadedBy, &vUploadedByName, &vCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.CurrentVersionID = curVerID
	if vID != nil {
		curVer = DocumentVersion{
			ID:             *vID,
			SocietyID:      doc.SocietyID,
			DocumentID:     doc.ID,
			VersionNumber:  *vNum,
			StoragePath:    *vPath,
			FileName:       *vName,
			FileSize:       *vSize,
			MimeType:       *vMime,
			FileHash:       *vHash,
			ChangeSummary:  vChange,
			UploadedBy:     *vUploadedBy,
			UploadedByName: *vUploadedByName,
			CreatedAt:      *vCreatedAt,
		}
		doc.CurrentVersion = &curVer
	}

	return &doc, nil
}

// GetDocumentVersionByID retrieves metadata of a specific version.
func (r *Repository) GetDocumentVersionByID(ctx context.Context, societyID, documentID, versionID uuid.UUID) (*DocumentVersion, error) {
	query := `
		SELECT 
			v.id, v.society_id, v.document_id, v.version_number, v.storage_path,
			v.file_name, v.file_size, v.mime_type, v.file_hash, v.change_summary,
			v.uploaded_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = v.uploaded_by AND sm.society_id = v.society_id LIMIT 1), u.phone, 'User') AS uploaded_by_name, v.created_at
		FROM document_versions v
		JOIN users u ON u.id = v.uploaded_by
		WHERE v.id = $1 AND v.document_id = $2 AND v.society_id = $3
	`

	var v DocumentVersion
	err := r.pool.QueryRow(ctx, query, versionID, documentID, societyID).Scan(
		&v.ID, &v.SocietyID, &v.DocumentID, &v.VersionNumber, &v.StoragePath,
		&v.FileName, &v.FileSize, &v.MimeType, &v.FileHash, &v.ChangeSummary,
		&v.UploadedBy, &v.UploadedByName, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentVersionNotFound
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	return &v, nil
}

// ListDocumentVersions retrieves all versions of a document.
func (r *Repository) ListDocumentVersions(ctx context.Context, societyID, documentID uuid.UUID) ([]DocumentVersion, error) {
	// First verify document belongs to society
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_documents WHERE id = $1 AND society_id = $2)`, documentID, societyID).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrDocumentNotFound
	}

	query := `
		SELECT 
			v.id, v.society_id, v.document_id, v.version_number, v.storage_path,
			v.file_name, v.file_size, v.mime_type, v.file_hash, v.change_summary,
			v.uploaded_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = v.uploaded_by AND sm.society_id = v.society_id LIMIT 1), u.phone, 'User') AS uploaded_by_name, v.created_at
		FROM document_versions v
		JOIN users u ON u.id = v.uploaded_by
		WHERE v.document_id = $1 AND v.society_id = $2
		ORDER BY v.version_number DESC
	`

	rows, err := r.pool.Query(ctx, query, documentID, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	versions := make([]DocumentVersion, 0)
	for rows.Next() {
		var v DocumentVersion
		if err := rows.Scan(
			&v.ID, &v.SocietyID, &v.DocumentID, &v.VersionNumber, &v.StoragePath,
			&v.FileName, &v.FileSize, &v.MimeType, &v.FileHash, &v.ChangeSummary,
			&v.UploadedBy, &v.UploadedByName, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan version row: %w", err)
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// ListDocuments returns filtered, paginated society documents.
func (r *Repository) ListDocuments(ctx context.Context, societyID uuid.UUID, filter ListDocumentsFilter, callerClassification string) (*ListDocumentsResult, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("d.society_id = $%d", argIdx))
	args = append(args, societyID)
	argIdx++

	// Filter based on caller classification visibility
	switch callerClassification {
	case ClassificationSocietyResidents:
		// Ordinary residents can ONLY see SOCIETY_RESIDENTS classification and ACTIVE documents
		conditions = append(conditions, fmt.Sprintf("d.classification = $%d", argIdx))
		args = append(args, ClassificationSocietyResidents)
		argIdx++

		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
		args = append(args, DocumentStatusActive)
		argIdx++

	case ClassificationCommittee:
		// Committee members can see SOCIETY_RESIDENTS and COMMITTEE
		conditions = append(conditions, fmt.Sprintf("d.classification IN ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, ClassificationSocietyResidents, ClassificationCommittee)
		argIdx += 2

		// Non-admins only see ACTIVE unless explicitly requested
		if filter.Status != "" {
			conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
			args = append(args, filter.Status)
			argIdx++
		} else {
			conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
			args = append(args, DocumentStatusActive)
			argIdx++
		}

	case ClassificationSocAdminOnly, "SUPER_ADMIN":
		// Admins can see all classifications
		if filter.Classification != "" {
			conditions = append(conditions, fmt.Sprintf("d.classification = $%d", argIdx))
			args = append(args, filter.Classification)
			argIdx++
		}
		if filter.Status != "" {
			conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
			args = append(args, filter.Status)
			argIdx++
		}
	default:
		// Default to resident restrictions
		conditions = append(conditions, fmt.Sprintf("d.classification = $%d", argIdx))
		args = append(args, ClassificationSocietyResidents)
		argIdx++
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", argIdx))
		args = append(args, DocumentStatusActive)
		argIdx++
	}

	if filter.CategoryID != nil {
		conditions = append(conditions, fmt.Sprintf("d.category_id = $%d", argIdx))
		args = append(args, *filter.CategoryID)
		argIdx++
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.TrimSpace(filter.Search) + "%"
		conditions = append(conditions, fmt.Sprintf("(d.title ILIKE $%d OR d.description ILIKE $%d OR d.document_number ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	if filter.BuildingID != nil {
		conditions = append(conditions, fmt.Sprintf("d.building_id = $%d", argIdx))
		args = append(args, *filter.BuildingID)
		argIdx++
	}
	if filter.WingID != nil {
		conditions = append(conditions, fmt.Sprintf("d.wing_id = $%d", argIdx))
		args = append(args, *filter.WingID)
		argIdx++
	}
	if filter.UnitID != nil {
		conditions = append(conditions, fmt.Sprintf("d.unit_id = $%d", argIdx))
		args = append(args, *filter.UnitID)
		argIdx++
	}
	if filter.MemberID != nil {
		conditions = append(conditions, fmt.Sprintf("d.member_id = $%d", argIdx))
		args = append(args, *filter.MemberID)
		argIdx++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM society_documents d %s`, whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			d.id, d.society_id, d.category_id, c.name AS category_name,
			d.document_number, d.title, d.description, d.classification, d.status,
			d.current_version_id, d.building_id, b.name AS building_name,
			d.wing_id, w.name AS wing_name, d.unit_id, u.unit_number,
			d.member_id, TRIM(CONCAT(m.first_name, ' ', m.last_name)) AS member_name,
			d.created_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = d.created_by AND sm.society_id = d.society_id LIMIT 1), cu.phone, 'User') AS created_by_name, d.updated_by,
			d.created_at, d.updated_at, d.archived_at, d.archived_by, d.archived_reason,
			(SELECT COUNT(*) FROM document_versions dv WHERE dv.document_id = d.id) AS version_count,
			v.id, v.version_number, v.storage_path, v.file_name, v.file_size, v.mime_type, v.file_hash,
			v.change_summary, v.uploaded_by, COALESCE((SELECT TRIM(CONCAT(sm.first_name, ' ', sm.last_name)) FROM society_members sm WHERE sm.user_id = v.uploaded_by AND sm.society_id = v.society_id LIMIT 1), vu.phone, 'User') AS uploaded_by_name, v.created_at
		FROM society_documents d
		JOIN document_categories c ON c.id = d.category_id
		LEFT JOIN society_buildings b ON b.id = d.building_id
		LEFT JOIN society_wings w ON w.id = d.wing_id
		LEFT JOIN society_units u ON u.id = d.unit_id
		LEFT JOIN society_members m ON m.id = d.member_id
		LEFT JOIN users cu ON cu.id = d.created_by
		LEFT JOIN document_versions v ON v.id = d.current_version_id
		LEFT JOIN users vu ON vu.id = v.uploaded_by
		%s
		ORDER BY d.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	items := make([]SocietyDocument, 0)
	for rows.Next() {
		var doc SocietyDocument
		var curVerID *uuid.UUID
		var vID *uuid.UUID
		var vNum *int
		var vPath, vName, vMime, vHash *string
		var vSize *int64
		var vChange *string
		var vUploadedBy *uuid.UUID
		var vUploadedByName *string
		var vCreatedAt *time.Time

		err := rows.Scan(
			&doc.ID, &doc.SocietyID, &doc.CategoryID, &doc.CategoryName,
			&doc.DocumentNumber, &doc.Title, &doc.Description, &doc.Classification, &doc.Status,
			&curVerID, &doc.BuildingID, &doc.BuildingName,
			&doc.WingID, &doc.WingName, &doc.UnitID, &doc.UnitNumber,
			&doc.MemberID, &doc.MemberName,
			&doc.CreatedBy, &doc.CreatedByName, &doc.UpdatedBy,
			&doc.CreatedAt, &doc.UpdatedAt, &doc.ArchivedAt, &doc.ArchivedBy, &doc.ArchivedReason,
			&doc.VersionCount,
			&vID, &vNum, &vPath, &vName, &vSize, &vMime, &vHash,
			&vChange, &vUploadedBy, &vUploadedByName, &vCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		doc.CurrentVersionID = curVerID
		if vID != nil {
			doc.CurrentVersion = &DocumentVersion{
				ID:             *vID,
				SocietyID:      doc.SocietyID,
				DocumentID:     doc.ID,
				VersionNumber:  *vNum,
				StoragePath:    *vPath,
				FileName:       *vName,
				FileSize:       *vSize,
				MimeType:       *vMime,
				FileHash:       *vHash,
				ChangeSummary:  vChange,
				UploadedBy:     *vUploadedBy,
				UploadedByName: *vUploadedByName,
				CreatedAt:      *vCreatedAt,
			}
		}

		items = append(items, doc)
	}

	return &ListDocumentsResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// UpdateDocument updates mutable metadata on a document.
func (r *Repository) UpdateDocument(ctx context.Context, societyID uuid.UUID, params UpdateDocumentParams) (*SocietyDocument, error) {
	title := strings.TrimSpace(params.Title)
	if len(title) < 3 || len(title) > 200 {
		return nil, ErrInvalidDocumentTitle
	}

	switch params.Classification {
	case ClassificationSocietyResidents, ClassificationCommittee, ClassificationSocAdminOnly:
	default:
		return nil, ErrInvalidClassification
	}

	// Verify category
	var catActive bool
	err := r.pool.QueryRow(ctx, `SELECT is_active FROM document_categories WHERE id = $1 AND society_id = $2`, params.CategoryID, societyID).Scan(&catActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, err
	}
	if !catActive {
		return nil, ErrCategoryInactive
	}

	// Verify associations
	if params.BuildingID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_buildings WHERE id = $1 AND society_id = $2)`, params.BuildingID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("building not found in this society")
		}
	}
	if params.WingID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_wings WHERE id = $1 AND society_id = $2)`, params.WingID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("wing not found in this society")
		}
	}
	if params.UnitID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_units WHERE id = $1 AND society_id = $2)`, params.UnitID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("unit not found in this society")
		}
	}
	if params.MemberID != nil {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_members WHERE id = $1 AND society_id = $2)`, params.MemberID, societyID).Scan(&exists); err != nil || !exists {
			return nil, fmt.Errorf("member not found in this society")
		}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	updateQuery := `
		UPDATE society_documents
		SET title = $1, description = $2, category_id = $3, classification = $4,
		    building_id = $5, wing_id = $6, unit_id = $7, member_id = $8,
		    updated_by = $9, updated_at = NOW()
		WHERE id = $10 AND society_id = $11
	`
	tag, err := tx.Exec(ctx, updateQuery,
		title, params.Description, params.CategoryID, params.Classification,
		params.BuildingID, params.WingID, params.UnitID, params.MemberID,
		params.ActorID, params.DocumentID, societyID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrDocumentNotFound
	}

	auditPayload, _ := json.Marshal(map[string]any{
		"document_id":    params.DocumentID,
		"title":          title,
		"category_id":    params.CategoryID,
		"classification": params.Classification,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, params.ActorID, ActionDocumentUpdated, string(auditPayload), params.ClientIP, params.UserAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetDocumentByID(ctx, societyID, params.DocumentID)
}

// ArchiveDocument marks a document as ARCHIVED with a required reason.
func (r *Repository) ArchiveDocument(ctx context.Context, societyID, documentID uuid.UUID, reason string, actorID uuid.UUID, clientIP, userAgent string) (*SocietyDocument, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrArchivalReasonRequired
	}

	var status string
	var docTitle string
	var docNumber string
	err := r.pool.QueryRow(ctx, `SELECT status, title, document_number FROM society_documents WHERE id = $1 AND society_id = $2`, documentID, societyID).Scan(&status, &docTitle, &docNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	if status == DocumentStatusArchived {
		return nil, errors.New("document is already archived")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE society_documents
		SET status = 'ARCHIVED', archived_at = NOW(), archived_by = $1, archived_reason = $2, updated_by = $1, updated_at = NOW()
		WHERE id = $3 AND society_id = $4
	`, actorID, reason, documentID, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to archive document: %w", err)
	}

	auditPayload, _ := json.Marshal(map[string]any{
		"document_id":     documentID,
		"document_number": docNumber,
		"title":           docTitle,
		"reason":          reason,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionDocumentArchived, string(auditPayload), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetDocumentByID(ctx, societyID, documentID)
}

// RestoreDocument brings an archived document back to ACTIVE status.
func (r *Repository) RestoreDocument(ctx context.Context, societyID, documentID uuid.UUID, actorID uuid.UUID, clientIP, userAgent string) (*SocietyDocument, error) {
	var status string
	var docTitle string
	var docNumber string
	err := r.pool.QueryRow(ctx, `SELECT status, title, document_number FROM society_documents WHERE id = $1 AND society_id = $2`, documentID, societyID).Scan(&status, &docTitle, &docNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDocumentNotFound
		}
		return nil, err
	}
	if status != DocumentStatusArchived {
		return nil, ErrDocumentNotArchived
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE society_documents
		SET status = 'ACTIVE', archived_at = NULL, archived_by = NULL, archived_reason = NULL, updated_by = $1, updated_at = NOW()
		WHERE id = $2 AND society_id = $3
	`, actorID, documentID, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to restore document: %w", err)
	}

	auditPayload, _ := json.Marshal(map[string]any{
		"document_id":     documentID,
		"document_number": docNumber,
		"title":           docTitle,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionDocumentRestored, string(auditPayload), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetDocumentByID(ctx, societyID, documentID)
}

// RecordDocumentDownloadAudit logs access to a document version.
func (r *Repository) RecordDocumentDownloadAudit(ctx context.Context, societyID, documentID, versionID uuid.UUID, actorID uuid.UUID, clientIP, userAgent string) error {
	auditPayload, _ := json.Marshal(map[string]any{
		"society_id":  societyID,
		"document_id": documentID,
		"version_id":  versionID,
	})
	_, err := r.pool.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionDocumentDownloaded, string(auditPayload), clientIP, userAgent)
	return err
}

// ListCategories returns document categories for the given society.
func (r *Repository) ListCategories(ctx context.Context, societyID uuid.UUID, includeInactive bool) ([]DocumentCategory, error) {
	query := `
		SELECT 
			c.id, c.society_id, c.name, c.description, c.is_system, c.is_active, c.display_order,
			c.created_at, c.updated_at,
			(SELECT COUNT(*) FROM society_documents d WHERE d.category_id = c.id AND d.status = 'ACTIVE') AS document_count
		FROM document_categories c
		WHERE c.society_id = $1
	`
	if !includeInactive {
		query += " AND c.is_active = true"
	}
	query += " ORDER BY c.display_order ASC, c.name ASC"

	rows, err := r.pool.Query(ctx, query, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	categories := make([]DocumentCategory, 0)
	for rows.Next() {
		var c DocumentCategory
		if err := rows.Scan(
			&c.ID, &c.SocietyID, &c.Name, &c.Description, &c.IsSystem, &c.IsActive,
			&c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt, &c.DocumentCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// CreateCategory adds a custom category to a society.
func (r *Repository) CreateCategory(ctx context.Context, societyID uuid.UUID, name string, description *string, displayOrder int, actorID uuid.UUID, clientIP, userAgent string) (*DocumentCategory, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return nil, errors.New("category name must be between 2 and 100 characters")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var cat DocumentCategory
	cat.ID = uuid.New()
	cat.SocietyID = societyID
	cat.Name = name
	cat.Description = description
	cat.IsSystem = false
	cat.IsActive = true
	cat.DisplayOrder = displayOrder

	insertQuery := `
		INSERT INTO document_categories (id, society_id, name, description, is_system, is_active, display_order)
		VALUES ($1, $2, $3, $4, false, true, $5)
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, insertQuery, cat.ID, societyID, name, description, displayOrder).Scan(&cat.CreatedAt, &cat.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	auditPayload, _ := json.Marshal(map[string]any{
		"category_id":   cat.ID,
		"category_name": cat.Name,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`, actorID, ActionDocumentCategoryCreated, string(auditPayload), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &cat, nil
}

// IsUserInCommittee checks if a user holds an active committee appointment in the society.
func (r *Repository) IsUserInCommittee(ctx context.Context, societyID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM society_committee_appointments a
			JOIN society_members m ON a.member_id = m.id
			WHERE a.society_id = $1 
			  AND m.user_id = $2 
			  AND a.status = 'ACTIVE'
			  AND m.status = 'ACTIVE'
			  AND a.start_date <= CURRENT_DATE
			  AND (a.expected_end_date IS NULL OR a.expected_end_date >= CURRENT_DATE)
		)
	`
	var inCommittee bool
	err := r.pool.QueryRow(ctx, query, societyID, userID).Scan(&inCommittee)
	return inCommittee, err
}

// ResolveUserClassificationScope resolves the caller's highest classification scope:
// "SUPER_ADMIN", "SOC_ADMIN_ONLY", "COMMITTEE", or "SOCIETY_RESIDENTS".
func (r *Repository) ResolveUserClassificationScope(ctx context.Context, societyID, userID uuid.UUID, roleHint string) (string, error) {
	if roleHint == "SUPER_ADMIN" {
		return "SUPER_ADMIN", nil
	}

	// Check SOC_ADMIN
	var isSocAdmin bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM society_user_roles
			WHERE society_id = $1 AND user_id = $2 AND role = 'SOC_ADMIN' AND status = 'ACTIVE'
		)
	`, societyID, userID).Scan(&isSocAdmin)
	if err == nil && isSocAdmin {
		return ClassificationSocAdminOnly, nil
	}

	// Check Committee
	inCommittee, err := r.IsUserInCommittee(ctx, societyID, userID)
	if err == nil && inCommittee {
		return ClassificationCommittee, nil
	}

	// Otherwise resident
	return ClassificationSocietyResidents, nil
}

// DocumentStorageMetrics holds aggregated storage observability data for a society.
type DocumentStorageMetrics struct {
	TotalDocuments    int64 `json:"total_documents"`
	ActiveDocuments   int64 `json:"active_documents"`
	ArchivedDocuments int64 `json:"archived_documents"`
	TotalVersions     int64 `json:"total_versions"`
	TotalStoredBytes  int64 `json:"total_stored_bytes"`
}

// GetDocumentStorageMetrics retrieves document storage observability metrics for a society.
func (r *Repository) GetDocumentStorageMetrics(ctx context.Context, societyID uuid.UUID) (*DocumentStorageMetrics, error) {
	query := `
		SELECT
			COUNT(DISTINCT d.id) AS total_documents,
			COUNT(DISTINCT CASE WHEN d.status = 'ACTIVE' THEN d.id END) AS active_documents,
			COUNT(DISTINCT CASE WHEN d.status = 'ARCHIVED' THEN d.id END) AS archived_documents,
			COUNT(v.id) AS total_versions,
			COALESCE(SUM(v.file_size), 0) AS total_stored_bytes
		FROM society_documents d
		LEFT JOIN document_versions v ON v.document_id = d.id AND v.society_id = d.society_id
		WHERE d.society_id = $1
	`
	var m DocumentStorageMetrics
	err := r.pool.QueryRow(ctx, query, societyID).Scan(
		&m.TotalDocuments,
		&m.ActiveDocuments,
		&m.ArchivedDocuments,
		&m.TotalVersions,
		&m.TotalStoredBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query document storage metrics: %w", err)
	}
	return &m, nil
}
