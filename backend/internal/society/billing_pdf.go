package society

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"chs/backend/internal/billingpdf"
)

var (
	ErrArtifactNotFound = errors.New("billing pdf artifact not found")
)

// MaintenanceBillPDFArtifact represents a stored PDF document or archive artifact.
type MaintenanceBillPDFArtifact struct {
	ID             uuid.UUID  `json:"id"`
	SocietyID      uuid.UUID  `json:"society_id"`
	BillID         *uuid.UUID `json:"bill_id,omitempty"`
	BatchID        *uuid.UUID `json:"batch_id,omitempty"`
	ArtifactType   string     `json:"artifact_type"` // SINGLE_BILL_PDF, BATCH_COMBINED_PDF, BATCH_ZIP_ARCHIVE
	FileName       string     `json:"file_name"`
	StoragePath    string     `json:"storage_path"`
	FileSizeBytes  int64      `json:"file_size_bytes"`
	SHA256Checksum string     `json:"sha256_checksum"`
	MimeType       string     `json:"mime_type"`
	HasWatermark   bool       `json:"has_watermark"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CanUserAccessBill checks whether an actor has permission to access the bill.
func (r *Repository) CanUserAccessBill(ctx context.Context, societyID, billID, userID uuid.UUID, role string) (bool, error) {
	if role == RoleSocietyAdmin || role == "SUPER_ADMIN" {
		return true, nil
	}
	var count int
	query := `
		SELECT COUNT(1)
		FROM maintenance_bills b
		JOIN unit_relationships ur ON b.unit_id = ur.unit_id AND b.society_id = ur.society_id
		JOIN society_members sm ON ur.member_id = sm.id AND ur.society_id = sm.society_id
		WHERE b.id = $1 AND b.society_id = $2 AND sm.user_id = $3
		  AND ur.status = 'ACTIVE' AND sm.status = 'ACTIVE'
	`
	err := r.pool.QueryRow(ctx, query, billID, societyID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check bill access: %w", err)
	}
	return count > 0, nil
}

// GenerateBillPDF generates, signs with SHA-256, stores, and records a PDF artifact for an immutable bill.
func (r *Repository) GenerateBillPDF(ctx context.Context, societyID, billID, actorID uuid.UUID) (*MaintenanceBillPDFArtifact, []byte, error) {
	doc, err := r.buildBillDocument(ctx, societyID, billID)
	if err != nil {
		return nil, nil, fmt.Errorf("build bill document model: %w", err)
	}

	pdfBytes, checksum, err := billingpdf.RenderPDF(*doc)
	if err != nil {
		return nil, nil, fmt.Errorf("render bill pdf: %w", err)
	}

	// Deterministic naming: {billNumber}.pdf sanitized
	cleanBillNum := strings.ReplaceAll(doc.BillNumber, "/", "-")
	cleanBillNum = strings.ReplaceAll(cleanBillNum, " ", "_")
	fileName := fmt.Sprintf("%s.pdf", cleanBillNum)
	storagePath := fmt.Sprintf("societies/%s/billing/bills/%s/%s", societyID, billID, fileName)

	// Attempt upload to Supabase storage if credentials configured
	if err := r.uploadFileToStorage(ctx, storagePath, "application/pdf", pdfBytes); err != nil {
		// Log or tolerate missing credentials in local offline environments
		_ = err
	}

	artifactID := uuid.New()
	var artifact MaintenanceBillPDFArtifact
	query := `
		INSERT INTO maintenance_bill_pdf_artifacts (
			id, society_id, bill_id, batch_id, artifact_type, file_name,
			storage_path, file_size_bytes, sha256_checksum, mime_type,
			has_watermark, created_at
		) VALUES (
			$1, $2, $3, NULL, 'SINGLE_BILL_PDF', $4,
			$5, $6, $7, 'application/pdf',
			$8, NOW()
		)
		RETURNING id, society_id, bill_id, batch_id, artifact_type, file_name,
		          storage_path, file_size_bytes, sha256_checksum, mime_type,
		          has_watermark, created_at
	`
	err = r.pool.QueryRow(ctx, query,
		artifactID, societyID, billID, fileName,
		storagePath, int64(len(pdfBytes)), checksum,
		doc.Watermark.Enabled,
	).Scan(
		&artifact.ID, &artifact.SocietyID, &artifact.BillID, &artifact.BatchID,
		&artifact.ArtifactType, &artifact.FileName, &artifact.StoragePath,
		&artifact.FileSizeBytes, &artifact.SHA256Checksum, &artifact.MimeType,
		&artifact.HasWatermark, &artifact.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert bill pdf artifact: %w", err)
	}

	// Update maintenance_bills record with path and timestamp
	_, _ = r.pool.Exec(ctx, `
		UPDATE maintenance_bills
		SET pdf_artifact_path = $1, pdf_generated_at = NOW()
		WHERE id = $2 AND society_id = $3
	`, storagePath, billID, societyID)

	return &artifact, pdfBytes, nil
}

// GetBillPDF retrieves the PDF binary and artifact metadata for a bill.
func (r *Repository) GetBillPDF(ctx context.Context, societyID, billID uuid.UUID) (*MaintenanceBillPDFArtifact, []byte, error) {
	var artifact MaintenanceBillPDFArtifact
	query := `
		SELECT id, society_id, bill_id, batch_id, artifact_type, file_name,
		       storage_path, file_size_bytes, sha256_checksum, mime_type,
		       has_watermark, created_at
		FROM maintenance_bill_pdf_artifacts
		WHERE society_id = $1 AND bill_id = $2 AND artifact_type = 'SINGLE_BILL_PDF'
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.pool.QueryRow(ctx, query, societyID, billID).Scan(
		&artifact.ID, &artifact.SocietyID, &artifact.BillID, &artifact.BatchID,
		&artifact.ArtifactType, &artifact.FileName, &artifact.StoragePath,
		&artifact.FileSizeBytes, &artifact.SHA256Checksum, &artifact.MimeType,
		&artifact.HasWatermark, &artifact.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Generate on-the-fly if no prior artifact exists
			return r.GenerateBillPDF(ctx, societyID, billID, uuid.Nil)
		}
		return nil, nil, fmt.Errorf("query bill pdf artifact: %w", err)
	}

	// Attempt download from storage
	rc, _, _, err := r.DownloadFileFromStorage(ctx, artifact.StoragePath)
	if err == nil && rc != nil {
		defer rc.Close()
		content, readErr := io.ReadAll(rc)
		if readErr == nil && len(content) > 0 {
			return &artifact, content, nil
		}
	}

	// Fallback: render from immutable snapshot if storage is offline
	doc, err := r.buildBillDocument(ctx, societyID, billID)
	if err != nil {
		return nil, nil, err
	}
	pdfBytes, _, err := billingpdf.RenderPDF(*doc)
	if err != nil {
		return nil, nil, err
	}

	return &artifact, pdfBytes, nil
}

// GenerateBatchPDFs generates PDFs for all issued bills in a batch and bundles them into a ZIP archive.
func (r *Repository) GenerateBatchPDFs(ctx context.Context, societyID, batchID, actorID uuid.UUID) (*MaintenanceBillPDFArtifact, []byte, error) {
	var batchNumber string
	err := r.pool.QueryRow(ctx, `
		SELECT batch_number
		FROM maintenance_billing_batches
		WHERE id = $1 AND society_id = $2
	`, batchID, societyID).Scan(&batchNumber)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrBillingBatchNotFound
		}
		return nil, nil, fmt.Errorf("query billing batch: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id
		FROM maintenance_bills
		WHERE batch_id = $1 AND society_id = $2
		ORDER BY bill_number ASC
	`, batchID, societyID)
	if err != nil {
		return nil, nil, fmt.Errorf("query batch bills: %w", err)
	}
	defer rows.Close()

	var billIDs []uuid.UUID
	for rows.Next() {
		var bID uuid.UUID
		if err := rows.Scan(&bID); err != nil {
			return nil, nil, err
		}
		billIDs = append(billIDs, bID)
	}
	if len(billIDs) == 0 {
		return nil, nil, errors.New("no bills found in this billing batch")
	}

	// Create ZIP buffer
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	for _, bID := range billIDs {
		art, pdfBytes, err := r.GenerateBillPDF(ctx, societyID, bID, actorID)
		if err != nil {
			return nil, nil, fmt.Errorf("generate bill pdf for bill %s: %w", bID, err)
		}

		fw, err := zipWriter.Create(art.FileName)
		if err != nil {
			return nil, nil, fmt.Errorf("create zip entry: %w", err)
		}
		if _, err := fw.Write(pdfBytes); err != nil {
			return nil, nil, fmt.Errorf("write zip entry: %w", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, nil, fmt.Errorf("close zip writer: %w", err)
	}

	zipBytes := zipBuf.Bytes()
	hash := sha256.Sum256(zipBytes)
	checksum := hex.EncodeToString(hash[:])

	cleanBatchNum := strings.ReplaceAll(batchNumber, "/", "-")
	cleanBatchNum = strings.ReplaceAll(cleanBatchNum, " ", "_")
	fileName := fmt.Sprintf("%s.zip", cleanBatchNum)
	storagePath := fmt.Sprintf("societies/%s/billing/batches/%s/%s", societyID, batchID, fileName)

	// Attempt upload
	_ = r.uploadFileToStorage(ctx, storagePath, "application/zip", zipBytes)

	artifactID := uuid.New()
	var artifact MaintenanceBillPDFArtifact
	query := `
		INSERT INTO maintenance_bill_pdf_artifacts (
			id, society_id, bill_id, batch_id, artifact_type, file_name,
			storage_path, file_size_bytes, sha256_checksum, mime_type,
			has_watermark, created_at
		) VALUES (
			$1, $2, NULL, $3, 'BATCH_ZIP_ARCHIVE', $4,
			$5, $6, $7, 'application/zip',
			TRUE, NOW()
		)
		RETURNING id, society_id, bill_id, batch_id, artifact_type, file_name,
		          storage_path, file_size_bytes, sha256_checksum, mime_type,
		          has_watermark, created_at
	`
	err = r.pool.QueryRow(ctx, query,
		artifactID, societyID, batchID, fileName,
		storagePath, int64(len(zipBytes)), checksum,
	).Scan(
		&artifact.ID, &artifact.SocietyID, &artifact.BillID, &artifact.BatchID,
		&artifact.ArtifactType, &artifact.FileName, &artifact.StoragePath,
		&artifact.FileSizeBytes, &artifact.SHA256Checksum, &artifact.MimeType,
		&artifact.HasWatermark, &artifact.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert batch zip artifact: %w", err)
	}

	return &artifact, zipBytes, nil
}

// GetBatchZipArtifact retrieves the ZIP archive artifact for a batch.
func (r *Repository) GetBatchZipArtifact(ctx context.Context, societyID, batchID uuid.UUID) (*MaintenanceBillPDFArtifact, []byte, error) {
	var artifact MaintenanceBillPDFArtifact
	query := `
		SELECT id, society_id, bill_id, batch_id, artifact_type, file_name,
		       storage_path, file_size_bytes, sha256_checksum, mime_type,
		       has_watermark, created_at
		FROM maintenance_bill_pdf_artifacts
		WHERE society_id = $1 AND batch_id = $2 AND artifact_type = 'BATCH_ZIP_ARCHIVE'
		ORDER BY created_at DESC
		LIMIT 1
	`
	err := r.pool.QueryRow(ctx, query, societyID, batchID).Scan(
		&artifact.ID, &artifact.SocietyID, &artifact.BillID, &artifact.BatchID,
		&artifact.ArtifactType, &artifact.FileName, &artifact.StoragePath,
		&artifact.FileSizeBytes, &artifact.SHA256Checksum, &artifact.MimeType,
		&artifact.HasWatermark, &artifact.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Generate on-the-fly
			return r.GenerateBatchPDFs(ctx, societyID, batchID, uuid.Nil)
		}
		return nil, nil, fmt.Errorf("query batch zip artifact: %w", err)
	}

	rc, _, _, err := r.DownloadFileFromStorage(ctx, artifact.StoragePath)
	if err == nil && rc != nil {
		defer rc.Close()
		content, readErr := io.ReadAll(rc)
		if readErr == nil && len(content) > 0 {
			return &artifact, content, nil
		}
	}

	// Fallback to re-generating on-the-fly if storage is unconfigured
	return r.GenerateBatchPDFs(ctx, societyID, batchID, uuid.Nil)
}

// ListBillPDFArtifacts returns all artifacts for a bill.
func (r *Repository) ListBillPDFArtifacts(ctx context.Context, societyID, billID uuid.UUID) ([]MaintenanceBillPDFArtifact, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, society_id, bill_id, batch_id, artifact_type, file_name,
		       storage_path, file_size_bytes, sha256_checksum, mime_type,
		       has_watermark, created_at
		FROM maintenance_bill_pdf_artifacts
		WHERE society_id = $1 AND bill_id = $2
		ORDER BY created_at DESC
	`, societyID, billID)
	if err != nil {
		return nil, fmt.Errorf("query bill artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []MaintenanceBillPDFArtifact
	for rows.Next() {
		var a MaintenanceBillPDFArtifact
		if err := rows.Scan(
			&a.ID, &a.SocietyID, &a.BillID, &a.BatchID,
			&a.ArtifactType, &a.FileName, &a.StoragePath,
			&a.FileSizeBytes, &a.SHA256Checksum, &a.MimeType,
			&a.HasWatermark, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}
	return artifacts, nil
}

// GetArtifactByID retrieves an artifact metadata and content by ID.
func (r *Repository) GetArtifactByID(ctx context.Context, societyID, artifactID uuid.UUID) (*MaintenanceBillPDFArtifact, []byte, error) {
	var a MaintenanceBillPDFArtifact
	err := r.pool.QueryRow(ctx, `
		SELECT id, society_id, bill_id, batch_id, artifact_type, file_name,
		       storage_path, file_size_bytes, sha256_checksum, mime_type,
		       has_watermark, created_at
		FROM maintenance_bill_pdf_artifacts
		WHERE id = $1 AND society_id = $2
	`, artifactID, societyID).Scan(
		&a.ID, &a.SocietyID, &a.BillID, &a.BatchID,
		&a.ArtifactType, &a.FileName, &a.StoragePath,
		&a.FileSizeBytes, &a.SHA256Checksum, &a.MimeType,
		&a.HasWatermark, &a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrArtifactNotFound
		}
		return nil, nil, fmt.Errorf("get artifact: %w", err)
	}

	rc, _, _, err := r.DownloadFileFromStorage(ctx, a.StoragePath)
	if err == nil && rc != nil {
		defer rc.Close()
		content, readErr := io.ReadAll(rc)
		if readErr == nil && len(content) > 0 {
			return &a, content, nil
		}
	}

	// If single bill, fallback to re-render
	if a.BillID != nil {
		doc, err := r.buildBillDocument(ctx, societyID, *a.BillID)
		if err == nil {
			pdfBytes, _, err := billingpdf.RenderPDF(*doc)
			if err == nil {
				return &a, pdfBytes, nil
			}
		}
	}

	return &a, nil, nil
}

// buildBillDocument constructs the comprehensive BillDocument from immutable database records.
func (r *Repository) buildBillDocument(ctx context.Context, societyID, billID uuid.UUID) (*billingpdf.BillDocument, error) {
	doc := &billingpdf.BillDocument{
		SocietyID: societyID,
		BillID:    billID,
		Currency:  "INR",
		Watermark: billingpdf.WatermarkConfig{
			Enabled:  true,
			Opacity:  0.08,
			Scale:    0.40,
			Position: "CENTER",
		},
	}

	var templateID *uuid.UUID
	var templateVersion int
	var notes string

	billQuery := `
		SELECT b.unit_id, b.billing_period_id, b.bill_number, b.bill_type,
		       b.template_id, b.template_version, b.issue_date, b.due_date, b.currency,
		       b.subtotal, b.taxable_amount, b.cgst_amount, b.sgst_amount, b.igst_amount,
		       b.total_tax_amount, b.arrears_amount, b.interest_amount, b.advance_adjusted,
		       b.total_amount, COALESCE(b.notes, ''),
		       s.name, COALESCE(s.registration_number, ''), s.address_line1,
		       COALESCE(s.locality, ''), s.city, s.state, s.pincode,
		       COALESCE(s.contact_email, ''), COALESCE(s.contact_phone, ''),
		       u.unit_identifier, COALESCE(bld.name, ''), COALESCE(u.floor_number, 0),
		       u.unit_type,
		       bp.period_name, bp.period_start, bp.period_end
		FROM maintenance_bills b
		JOIN societies s ON b.society_id = s.id
		JOIN society_units u ON b.unit_id = u.id AND b.society_id = u.society_id
		LEFT JOIN society_buildings bld ON u.building_id = bld.id AND u.society_id = bld.society_id
		JOIN maintenance_billing_periods bp ON b.billing_period_id = bp.id AND b.society_id = bp.society_id
		WHERE b.id = $1 AND b.society_id = $2
	`
	err := r.pool.QueryRow(ctx, billQuery, billID, societyID).Scan(
		&doc.UnitID, &doc.BillingPeriodID, &doc.BillNumber, &doc.BillType,
		&templateID, &templateVersion, &doc.BillDate, &doc.DueDate, &doc.Currency,
		&doc.ChargeSubtotal, &doc.TaxableAmount, &doc.CGSTAmount, &doc.SGSTAmount, &doc.IGSTAmount,
		&doc.TotalTaxAmount, &doc.ArrearsAmount, &doc.InterestAmount, &doc.AdvanceDeduction,
		&doc.TotalPayable, &notes,
		&doc.SocietyName, &doc.RegistrationNumber, &doc.AddressLine1,
		&doc.Locality, &doc.City, &doc.State, &doc.Pincode,
		&doc.SocietyEmail, &doc.SocietyPhone,
		&doc.UnitNumber, &doc.BuildingName, &doc.FloorNumber,
		&doc.UnitType,
		&doc.PeriodName, &doc.PeriodStart, &doc.PeriodEnd,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBillNotFound
		}
		return nil, fmt.Errorf("fetch bill details: %w", err)
	}

	doc.AreaSqFt = 1000.0

	doc.TemplateVersion = templateVersion
	doc.BillTitle = "TAX INVOICE / MAINTENANCE BILL"

	// Fetch primary owner/occupant member info
	memberQuery := `
		SELECT sm.first_name, COALESCE(sm.last_name, ''), sm.phone, COALESCE(sm.email, '')
		FROM unit_relationships ur
		JOIN society_members sm ON ur.member_id = sm.id AND sm.society_id = ur.society_id
		WHERE ur.unit_id = $1 AND ur.society_id = $2 AND ur.status = 'ACTIVE'
		ORDER BY CASE WHEN ur.relationship_type = 'PRIMARY_OWNER' THEN 1 ELSE 2 END, ur.created_at ASC
		LIMIT 1
	`
	var fName, lName, phone, email string
	if err := r.pool.QueryRow(ctx, memberQuery, doc.UnitID, societyID).Scan(&fName, &lName, &phone, &email); err == nil {
		doc.MemberName = strings.TrimSpace(fName + " " + lName)
		doc.MemberPhone = phone
		doc.MemberEmail = email
	}

	// Fetch line items
	itemsQuery := `
		SELECT item.description, item.rate, item.quantity, item.amount,
		       COALESCE(item.sac_code, ''), item.tax_classification, item.tax_rate, item.tax_amount,
		       COALESCE(item.formula_expression, ''), item.display_order,
		       COALESCE(ch.name, item.category_name, 'Maintenance Charge')
		FROM maintenance_bill_items item
		LEFT JOIN maintenance_charge_heads ch ON item.charge_head_id = ch.id AND ch.society_id = item.society_id
		WHERE item.bill_id = $1 AND item.society_id = $2
		ORDER BY item.display_order ASC, item.created_at ASC
	`
	itemRows, err := r.pool.Query(ctx, itemsQuery, billID, societyID)
	if err != nil {
		return nil, fmt.Errorf("fetch line items: %w", err)
	}
	defer itemRows.Close()

	var idx = 1
	for itemRows.Next() {
		var it billingpdf.DocumentLineItem
		it.Index = idx
		if err := itemRows.Scan(
			&it.Description, &it.Rate, &it.Quantity, &it.Amount,
			&it.SACCode, &it.TaxClassification, &it.TaxRate, &it.TaxAmount,
			&it.FormulaExpression, &it.DisplayOrder, &it.ChargeHeadName,
		); err != nil {
			return nil, fmt.Errorf("scan line item: %w", err)
		}
		doc.LineItems = append(doc.LineItems, it)
		idx++
	}

	// Resolve template snapshot
	if templateID != nil {
		var snapshotJSON []byte
		err := r.pool.QueryRow(ctx, `
			SELECT definition_snapshot
			FROM maintenance_bill_template_versions
			WHERE template_id = $1 AND version = $2 AND society_id = $3
		`, templateID, templateVersion, societyID).Scan(&snapshotJSON)
		if err == nil && len(snapshotJSON) > 0 {
			var snap map[string]interface{}
			if jsonErr := json.Unmarshal(snapshotJSON, &snap); jsonErr == nil {
				applyTemplateSnapshot(doc, snap)
			}
		} else {
			// Fallback: check current template if version record is missing
			var tpl MaintenanceBillTemplate
			if currentTpl, tplErr := r.GetBillTemplate(ctx, societyID, *templateID); tplErr == nil && currentTpl != nil {
				tpl = *currentTpl
				doc.BillTitle = tpl.BillTitle
				if tpl.FooterText != nil {
					doc.FooterText = *tpl.FooterText
				}
				if tpl.HeaderNotes != nil {
					doc.HeaderNotes = *tpl.HeaderNotes
				}
				if tpl.PaymentInstructions != nil {
					doc.PaymentInstructions = *tpl.PaymentInstructions
				}
				doc.BankDetails = tpl.BankDetails
				doc.Watermark.Enabled = tpl.WatermarkEnabled
				if tpl.WatermarkOpacity > 0 {
					doc.Watermark.Opacity = tpl.WatermarkOpacity
				}
				doc.FieldOverrides = make(map[string]billingpdf.TemplateFieldConfig)
				for _, f := range tpl.Fields {
					doc.FieldOverrides[f.FieldKey] = billingpdf.TemplateFieldConfig{
						IsVisible:    f.IsVisible,
						DisplayLabel: f.FieldLabel,
					}
				}
			}
		}
	}

	// Always compute AmountInWords deterministically
	doc.AmountInWords = billingpdf.NumberToWords(doc.TotalPayable)

	return doc, nil
}

func applyTemplateSnapshot(doc *billingpdf.BillDocument, snap map[string]interface{}) {
	if title, ok := snap["bill_title"].(string); ok && title != "" {
		doc.BillTitle = title
	}
	if notes, ok := snap["header_notes"].(string); ok {
		doc.HeaderNotes = notes
	}
	if footer, ok := snap["footer_text"].(string); ok {
		doc.FooterText = footer
	}
	if pay, ok := snap["payment_instructions"].(string); ok {
		doc.PaymentInstructions = pay
	}
	if bank, ok := snap["bank_details"].(map[string]interface{}); ok {
		doc.BankDetails = bank
	}
	if wm, ok := snap["watermark_enabled"].(bool); ok {
		doc.Watermark.Enabled = wm
	}
	if opacity, ok := snap["watermark_opacity"].(float64); ok && opacity > 0 {
		doc.Watermark.Opacity = opacity
	}
	if fieldsRaw, ok := snap["fields"].([]interface{}); ok {
		doc.FieldOverrides = make(map[string]billingpdf.TemplateFieldConfig)
		for _, fRaw := range fieldsRaw {
			if fMap, ok := fRaw.(map[string]interface{}); ok {
				k, _ := fMap["field_key"].(string)
				l, _ := fMap["field_label"].(string)
				vis, visOk := fMap["is_visible"].(bool)
				if !visOk {
					vis = true
				}
				if k != "" {
					doc.FieldOverrides[k] = billingpdf.TemplateFieldConfig{
						IsVisible:    vis,
						DisplayLabel: l,
					}
				}
			}
		}
	}
}
