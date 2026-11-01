package billingpdf

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// RenderPDF produces an immutable, print-ready A4 PDF binary for the given BillDocument.
// It returns the rendered PDF byte array and its SHA-256 checksum.
func RenderPDF(doc BillDocument) ([]byte, string, error) {
	if doc.AmountInWords == "" {
		doc.AmountInWords = NumberToWords(doc.TotalPayable)
	}

	// A4 Portrait, millimeters
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 15)

	// Register official CHS logo watermark
	if doc.Watermark.Enabled {
		if err := RegisterWatermarkAsset(pdf); err != nil {
			return nil, "", fmt.Errorf("register watermark: %w", err)
		}
	}

	// Register header function to stamp watermark on every page start
	pdf.SetHeaderFunc(func() {
		if doc.Watermark.Enabled {
			DrawWatermark(pdf, doc.Watermark)
		}
	})

	// Register footer function for page numbering
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 7)
		pdf.SetTextColor(120, 120, 120)
		pageStr := fmt.Sprintf("Page %d of {nb}", pdf.PageNo())
		pdf.CellFormat(190, 8, pageStr, "", 0, "C", false, 0, "")
	})

	pdf.AliasNbPages("{nb}")
	pdf.AddPage()

	// 1. Header Section: Society Details (Left) + Invoice Metadata (Right)
	renderHeader(pdf, doc)

	// 2. Recipient & Unit Details Block
	renderRecipientSection(pdf, doc)

	// 3. Charge Items Table
	renderLineItemsTable(pdf, doc)

	// 4. Financial Summary, Bank Details & Amount in Words
	renderFinancialSummary(pdf, doc)

	// 5. Terms, Signatory & Footer
	renderFooterSection(pdf, doc)

	// Output buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, "", fmt.Errorf("pdf generation output error: %w", err)
	}

	pdfBytes := buf.Bytes()
	hash := sha256.Sum256(pdfBytes)
	checksum := hex.EncodeToString(hash[:])

	return pdfBytes, checksum, nil
}

func isFieldVisible(doc BillDocument, fieldKey string, defaultVisible bool) bool {
	if doc.FieldOverrides == nil {
		return defaultVisible
	}
	if conf, ok := doc.FieldOverrides[fieldKey]; ok {
		return conf.IsVisible
	}
	return defaultVisible
}

func getFieldLabel(doc BillDocument, fieldKey string, defaultLabel string) string {
	if doc.FieldOverrides == nil {
		return defaultLabel
	}
	if conf, ok := doc.FieldOverrides[fieldKey]; ok && conf.DisplayLabel != "" {
		return conf.DisplayLabel
	}
	return defaultLabel
}

func renderHeader(pdf *fpdf.Fpdf, doc BillDocument) {
	startY := pdf.GetY()

	// Left Column: Society Branding & Address
	pdf.SetXY(10, startY)
	if isFieldVisible(doc, "society_name", true) {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(110, 6, doc.SocietyName, "", 1, "L", false, 0, "")
	}

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(70, 70, 70)
	if doc.RegistrationNumber != "" && isFieldVisible(doc, "society_reg_no", true) {
		regLabel := getFieldLabel(doc, "society_reg_no", "Reg. No")
		pdf.CellFormat(110, 4, fmt.Sprintf("%s: %s", regLabel, doc.RegistrationNumber), "", 1, "L", false, 0, "")
	}

	if isFieldVisible(doc, "society_address", true) {
		var addrParts []string
		if doc.AddressLine1 != "" {
			addrParts = append(addrParts, doc.AddressLine1)
		}
		if doc.Locality != "" {
			addrParts = append(addrParts, doc.Locality)
		}
		if doc.City != "" {
			addrParts = append(addrParts, doc.City)
		}
		if doc.Pincode != "" {
			addrParts = append(addrParts, doc.Pincode)
		}
		if len(addrParts) > 0 {
			pdf.MultiCell(110, 3.8, strings.Join(addrParts, ", "), "", "L", false)
		}
	}

	if doc.SocietyGSTIN != "" && isFieldVisible(doc, "society_gstin", true) {
		gstLabel := getFieldLabel(doc, "society_gstin", "GSTIN")
		pdf.CellFormat(110, 4, fmt.Sprintf("%s: %s", gstLabel, doc.SocietyGSTIN), "", 1, "L", false, 0, "")
	}

	// Right Column: Tax Invoice Badge & Metadata
	pdf.SetXY(120, startY)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(18, 18, 18)

	billTitle := doc.BillTitle
	if billTitle == "" {
		billTitle = "TAX INVOICE / MAINTENANCE BILL"
	}
	pdf.CellFormat(80, 5, strings.ToUpper(billTitle), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(50, 50, 50)
	if isFieldVisible(doc, "bill_number", true) {
		renderMetaRow(pdf, 120, getFieldLabel(doc, "bill_number", "Bill Number") + ":", doc.BillNumber)
	}
	if isFieldVisible(doc, "bill_date", true) {
		renderMetaRow(pdf, 120, getFieldLabel(doc, "bill_date", "Bill Date") + ":", doc.BillDate.Format("02-Jan-2006"))
	}
	if isFieldVisible(doc, "due_date", true) {
		renderMetaRow(pdf, 120, getFieldLabel(doc, "due_date", "Due Date") + ":", doc.DueDate.Format("02-Jan-2006"))
	}
	if doc.PeriodName != "" && isFieldVisible(doc, "billing_period", true) {
		renderMetaRow(pdf, 120, getFieldLabel(doc, "billing_period", "Billing Period") + ":", doc.PeriodName)
	}

	currY := pdf.GetY()
	if currY < startY+30 {
		currY = startY + 30
	}
	pdf.SetY(currY)

	// Divider line
	pdf.SetDrawColor(210, 210, 210)
	pdf.SetLineWidth(0.3)
	pdf.Line(10, currY+2, 200, currY+2)
	pdf.SetY(currY + 4)
}

func renderMetaRow(pdf *fpdf.Fpdf, x float64, label, value string) {
	pdf.SetX(x)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.CellFormat(35, 4.2, label, "", 0, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(45, 4.2, value, "", 1, "R", false, 0, "")
}

func renderRecipientSection(pdf *fpdf.Fpdf, doc BillDocument) {
	y := pdf.GetY()

	// Subtle box container
	pdf.SetFillColor(250, 250, 249)
	pdf.SetDrawColor(225, 225, 222)
	pdf.SetLineWidth(0.2)
	pdf.Rect(10, y, 190, 19, "FD")

	// Left: Billed To (Member Info)
	pdf.SetXY(13, y+2)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(90, 4, getFieldLabel(doc, "billed_to", "BILLED TO"), "", 1, "L", false, 0, "")

	if isFieldVisible(doc, "member_name", true) {
		pdf.SetX(13)
		pdf.SetFont("Helvetica", "B", 9.5)
		pdf.SetTextColor(20, 20, 20)
		memberName := doc.MemberName
		if memberName == "" {
			memberName = "Resident"
		}
		pdf.CellFormat(90, 4.5, memberName, "", 1, "L", false, 0, "")
	}

	if (doc.MemberPhone != "" || doc.MemberEmail != "") && isFieldVisible(doc, "member_contact", true) {
		contact := doc.MemberPhone
		if doc.MemberEmail != "" {
			if contact != "" {
				contact += " | " + doc.MemberEmail
			} else {
				contact = doc.MemberEmail
			}
		}
		pdf.SetX(13)
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(90, 3.8, contact, "", 1, "L", false, 0, "")
	}

	// Right: Unit Details
	pdf.SetXY(110, y+2)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(85, 4, getFieldLabel(doc, "unit_particulars", "UNIT PARTICULARS"), "", 1, "R", false, 0, "")

	if isFieldVisible(doc, "unit_number", true) {
		pdf.SetXY(110, y+6.5)
		pdf.SetFont("Helvetica", "B", 9.5)
		pdf.SetTextColor(20, 20, 20)
		unitDesc := fmt.Sprintf("Unit No: %s", doc.UnitNumber)
		if doc.BuildingName != "" {
			unitDesc += " (" + doc.BuildingName + ")"
		}
		pdf.CellFormat(85, 4.5, unitDesc, "", 1, "R", false, 0, "")
	}

	pdf.SetXY(110, y+11)
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(80, 80, 80)
	areaInfo := fmt.Sprintf("Type: %s", doc.UnitType)
	if doc.AreaSqFt > 0 {
		areaInfo += fmt.Sprintf(" | Area: %.0f sq.ft.", doc.AreaSqFt)
	}
	pdf.CellFormat(85, 4, areaInfo, "", 1, "R", false, 0, "")

	pdf.SetY(y + 22)
}

func renderLineItemsTable(pdf *fpdf.Fpdf, doc BillDocument) {
	// Column widths totaling exactly 190mm
	// # (10mm), Charge Description (72mm), SAC (18mm), Tax Class (22mm), Rate (20mm), GST (22mm), Amount (26mm)
	pdf.SetFillColor(243, 243, 241)
	pdf.SetDrawColor(215, 215, 210)
	pdf.SetLineWidth(0.2)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(30, 30, 30)

	pdf.CellFormat(10, 6.5, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(72, 6.5, "PARTICULARS / CHARGE HEAD", "1", 0, "L", true, 0, "")
	pdf.CellFormat(18, 6.5, "SAC", "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 6.5, "TAX CLASS", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6.5, "RATE", "1", 0, "R", true, 0, "")
	pdf.CellFormat(22, 6.5, "GST (INR)", "1", 0, "R", true, 0, "")
	pdf.CellFormat(26, 6.5, "AMOUNT (INR)", "1", 1, "R", true, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(40, 40, 40)

	for i, item := range doc.LineItems {
		slNo := fmt.Sprintf("%d", i+1)
		sac := item.SACCode
		if sac == "" {
			sac = "-"
		}

		rateStr := "-"
		if item.Rate > 0 {
			rateStr = FormatINR(item.Rate)
		}

		taxStr := "-"
		if item.TaxAmount > 0 {
			taxStr = FormatINR(item.TaxAmount)
		}

		amtStr := FormatINR(item.Amount)

		// Alternate row background subtly
		fill := (i % 2) == 1
		if fill {
			pdf.SetFillColor(252, 252, 251)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(10, 6, slNo, "LRB", 0, "C", fill, 0, "")
		pdf.CellFormat(72, 6, item.ChargeHeadName, "LRB", 0, "L", fill, 0, "")
		pdf.CellFormat(18, 6, sac, "LRB", 0, "C", fill, 0, "")
		pdf.CellFormat(22, 6, item.TaxClassification, "LRB", 0, "C", fill, 0, "")
		pdf.CellFormat(20, 6, rateStr, "LRB", 0, "R", fill, 0, "")
		pdf.CellFormat(22, 6, taxStr, "LRB", 0, "R", fill, 0, "")
		pdf.CellFormat(26, 6, amtStr, "LRB", 1, "R", fill, 0, "")
	}

	pdf.Ln(2)
}

func renderFinancialSummary(pdf *fpdf.Fpdf, doc BillDocument) {
	startY := pdf.GetY()

	// Left Column: Amount in Words & Bank Details (Width: 105mm)
	pdf.SetXY(10, startY)
	if isFieldVisible(doc, "amount_in_words", true) {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(80, 80, 80)
		pdf.CellFormat(105, 4, getFieldLabel(doc, "amount_in_words", "AMOUNT IN WORDS"), "", 1, "L", false, 0, "")

		inWords := doc.AmountInWords
		if inWords == "" {
			inWords = NumberToWords(doc.TotalPayable)
		}

		pdf.SetFont("Helvetica", "BI", 8.5)
		pdf.SetTextColor(20, 20, 20)
		pdf.MultiCell(105, 4, inWords, "", "L", false)
		pdf.Ln(3)
	}

	// Bank Details Card
	if len(doc.BankDetails) > 0 && isFieldVisible(doc, "bank_details", true) {
		bankY := pdf.GetY()
		pdf.SetFillColor(250, 250, 249)
		pdf.SetDrawColor(230, 230, 226)
		pdf.Rect(10, bankY, 105, 26, "FD")

		pdf.SetXY(12, bankY+2)
		pdf.SetFont("Helvetica", "B", 7.5)
		pdf.SetTextColor(90, 90, 90)
		pdf.CellFormat(101, 3.5, getFieldLabel(doc, "bank_details", "BANK & PAYMENT DETAILS"), "", 1, "L", false, 0, "")

		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(40, 40, 40)
		if bankName, ok := doc.BankDetails["bank_name"].(string); ok && bankName != "" {
			pdf.SetX(12)
			pdf.CellFormat(101, 3.5, fmt.Sprintf("Bank: %s", bankName), "", 1, "L", false, 0, "")
		}
		if accNo, ok := doc.BankDetails["account_number"].(string); ok && accNo != "" {
			pdf.SetX(12)
			pdf.CellFormat(101, 3.5, fmt.Sprintf("A/C No: %s", accNo), "", 1, "L", false, 0, "")
		}
		if ifsc, ok := doc.BankDetails["ifsc_code"].(string); ok && ifsc != "" {
			pdf.SetX(12)
			pdf.CellFormat(101, 3.5, fmt.Sprintf("IFSC Code: %s", ifsc), "", 1, "L", false, 0, "")
		}
		if branch, ok := doc.BankDetails["branch"].(string); ok && branch != "" {
			pdf.SetX(12)
			pdf.CellFormat(101, 3.5, fmt.Sprintf("Branch: %s", branch), "", 1, "L", false, 0, "")
		}
	}

	// Right Column: Summary Breakdown Table (Width: 80mm, from x=120)
	pdf.SetXY(120, startY)
	renderSummaryLine(pdf, 120, getFieldLabel(doc, "subtotal", "Charges Subtotal:"), FormatINR(doc.ChargeSubtotal), false)

	if doc.TotalTaxAmount > 0 {
		if doc.CGSTAmount > 0 && isFieldVisible(doc, "cgst", true) {
			renderSummaryLine(pdf, 120, getFieldLabel(doc, "cgst", "CGST (Central Tax):"), FormatINR(doc.CGSTAmount), false)
		}
		if doc.SGSTAmount > 0 && isFieldVisible(doc, "sgst", true) {
			renderSummaryLine(pdf, 120, getFieldLabel(doc, "sgst", "SGST (State Tax):"), FormatINR(doc.SGSTAmount), false)
		}
		if doc.IGSTAmount > 0 && isFieldVisible(doc, "igst", true) {
			renderSummaryLine(pdf, 120, getFieldLabel(doc, "igst", "IGST (Integrated Tax):"), FormatINR(doc.IGSTAmount), false)
		}
	}

	if doc.ArrearsAmount > 0 && isFieldVisible(doc, "arrears", true) {
		renderSummaryLine(pdf, 120, getFieldLabel(doc, "arrears", "Previous Arrears:"), FormatINR(doc.ArrearsAmount), false)
	}

	if doc.InterestAmount > 0 && isFieldVisible(doc, "interest", true) {
		renderSummaryLine(pdf, 120, getFieldLabel(doc, "interest", "Interest on Overdue:"), FormatINR(doc.InterestAmount), false)
	}

	if doc.AdvanceDeduction > 0 && isFieldVisible(doc, "advance_adjusted", true) {
		renderSummaryLine(pdf, 120, getFieldLabel(doc, "advance_adjusted", "Less: Advance Adjusted:"), "-"+FormatINR(doc.AdvanceDeduction), false)
	}

	if doc.RoundingAdjustment != 0 {
		renderSummaryLine(pdf, 120, "Rounding Adjustment:", FormatINR(doc.RoundingAdjustment), false)
	}

	// Final Payable Double Line
	pdf.Ln(1)
	pdf.SetX(120)
	pdf.SetFillColor(243, 243, 241)
	pdf.SetDrawColor(20, 20, 20)
	pdf.SetLineWidth(0.4)

	pdf.SetFont("Helvetica", "B", 9.5)
	pdf.SetTextColor(10, 10, 10)
	totalLabel := getFieldLabel(doc, "total_payable", "TOTAL PAYABLE (INR)") + ":"
	pdf.CellFormat(45, 7, totalLabel, "TB", 0, "R", true, 0, "")
	pdf.CellFormat(35, 7, FormatINR(doc.TotalPayable), "TB", 1, "R", true, 0, "")
}

func renderSummaryLine(pdf *fpdf.Fpdf, x float64, label, value string, isBold bool) {
	pdf.SetX(x)
	fontStyle := ""
	if isBold {
		fontStyle = "B"
	}
	pdf.SetFont("Helvetica", fontStyle, 8)
	pdf.SetTextColor(60, 60, 60)
	pdf.CellFormat(45, 4.5, label, "", 0, "R", false, 0, "")

	pdf.SetTextColor(20, 20, 20)
	pdf.CellFormat(35, 4.5, value, "", 1, "R", false, 0, "")
}

func renderFooterSection(pdf *fpdf.Fpdf, doc BillDocument) {
	pdf.SetY(-40)

	// Left: Terms & Instructions
	if isFieldVisible(doc, "footer_notes", true) {
		pdf.SetXY(10, -38)
		pdf.SetFont("Helvetica", "B", 7)
		pdf.SetTextColor(90, 90, 90)
		pdf.CellFormat(110, 3.5, getFieldLabel(doc, "footer_notes", "TERMS & CONDITIONS"), "", 1, "L", false, 0, "")

		terms := doc.FooterText
		if terms == "" {
			terms = "1. Cheques subject to realization.\n2. Please mention Unit No on reverse of cheque.\n3. Late payment surcharge applicable as per society bye-laws."
		}
		pdf.SetFont("Helvetica", "", 6.5)
		pdf.SetTextColor(80, 80, 80)
		pdf.MultiCell(110, 3, terms, "", "L", false)
	}

	// Right: Authorized Signatory
	if isFieldVisible(doc, "authorized_signatory", true) {
		pdf.SetXY(140, -32)
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(60, 60, 60)
		pdf.CellFormat(60, 4, fmt.Sprintf("For %s", doc.SocietyName), "", 1, "C", false, 0, "")

		pdf.SetXY(140, -18)
		pdf.SetFont("Helvetica", "B", 7.5)
		pdf.SetTextColor(30, 30, 30)
		sigLabel := getFieldLabel(doc, "authorized_signatory", "Authorized Signatory")
		pdf.CellFormat(60, 4, sigLabel, "T", 1, "C", false, 0, "")
	}
}
