package billingpdf

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNumberToWords(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{0.00, "Rupees Zero Only"},
		{500.00, "Rupees Five Hundred Only"},
		{7369.80, "Rupees Seven Thousand Three Hundred Sixty-Nine and Eighty Paise Only"},
		{100000.00, "Rupees One Lakh Only"},
		{123456.78, "Rupees One Lakh Twenty-Three Thousand Four Hundred Fifty-Six and Seventy-Eight Paise Only"},
		{10000000.50, "Rupees One Crore and Fifty Paise Only"},
		{-250.25, "Minus Rupees Two Hundred Fifty and Twenty-Five Paise Only"},
	}

	for _, tt := range tests {
		got := NumberToWords(tt.amount)
		if got != tt.want {
			t.Errorf("NumberToWords(%.2f) = %q; want %q", tt.amount, got, tt.want)
		}
	}
}

func TestFormatINR(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{0.0, "0.00"},
		{500.0, "500.00"},
		{7369.8, "7,369.80"},
		{123456.78, "1,23,456.78"},
		{1000000.00, "10,00,000.00"},
		{-5420.50, "-5,420.50"},
	}

	for _, tt := range tests {
		got := FormatINR(tt.amount)
		if got != tt.want {
			t.Errorf("FormatINR(%.2f) = %q; want %q", tt.amount, got, tt.want)
		}
	}
}

func sampleBillDocument() BillDocument {
	return BillDocument{
		SocietyID:          uuid.New(),
		SocietyName:        "Silver Palm Cooperative Housing Society Ltd",
		RegistrationNumber: "BOM/HSG/TC/12345/2012",
		AddressLine1:       "Plot 42, Sector 17, Palm Beach Road",
		Locality:           "Vashi",
		City:               "Navi Mumbai",
		State:              "Maharashtra",
		Pincode:            "400703",
		SocietyGSTIN:       "27AAAAA0000A1Z5",
		UnitID:             uuid.New(),
		UnitNumber:         "A-402",
		BuildingName:       "Tower Alpha",
		WingName:           "Wing A",
		FloorNumber:        4,
		UnitType:           "RESIDENTIAL",
		AreaSqFt:           1050,
		MemberName:         "Rajesh Sharma",
		MemberPhone:        "+91 98200 12345",
		MemberEmail:        "rajesh.sharma@example.com",
		BillID:             uuid.New(),
		BillNumber:         "BILL-202611-0042",
		BillType:           "REGULAR",
		BillTitle:          "Tax Invoice / Maintenance Bill",
		BillDate:           time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		DueDate:            time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		BillingPeriodID:    uuid.New(),
		PeriodName:         "November 2026",
		PeriodStart:        time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:          time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC),
		Currency:           "INR",
		TemplateVersion:    1,
		LineItems: []DocumentLineItem{
			{
				Index:             1,
				ChargeHeadName:    "Service & Maintenance Charges",
				Description:       "General society upkeep at Rs 3.50/sqft",
				SACCode:           "999511",
				TaxClassification: "TAXABLE",
				TaxRate:           18.0,
				Rate:              3.50,
				Quantity:          1050,
				Amount:            3675.00,
				TaxAmount:         661.50,
				DisplayOrder:      1,
			},
			{
				Index:             2,
				ChargeHeadName:    "Sinking Fund Contribution",
				Description:       "Statutory long-term reserve",
				SACCode:           "999512",
				TaxClassification: "EXEMPT",
				TaxRate:           0.0,
				Rate:              500.00,
				Quantity:          1,
				Amount:            500.00,
				TaxAmount:         0.00,
				DisplayOrder:      2,
			},
			{
				Index:             3,
				ChargeHeadName:    "Parking Maintenance (Stilt 1)",
				Description:       "Designated stilt parking bay upkeep",
				SACCode:           "999511",
				TaxClassification: "TAXABLE",
				TaxRate:           18.0,
				Rate:              400.00,
				Quantity:          1,
				Amount:            400.00,
				TaxAmount:         72.00,
				DisplayOrder:      3,
			},
		},
		ChargeSubtotal:     4575.00,
		TaxableAmount:      4075.00,
		CGSTAmount:         366.75,
		SGSTAmount:         366.75,
		IGSTAmount:         0.00,
		TotalTaxAmount:     733.50,
		ArrearsAmount:      1200.00,
		InterestAmount:     54.00,
		AdvanceDeduction:   0.00,
		RoundingAdjustment: -0.25,
		TotalPayable:       6562.00,
		AmountInWords:      "Rupees Six Thousand Five Hundred Sixty-Two Only",
		BankDetails: map[string]interface{}{
			"bank_name":      "State Bank of India",
			"account_number": "30491827461",
			"ifsc_code":      "SBIN0001234",
			"branch":         "Vashi Sector 17",
		},
		PaymentInstructions: "Please draw cheques in favor of Silver Palm CHS Ltd.",
		Watermark: WatermarkConfig{
			Enabled: true,
			Opacity: 0.08,
			Scale:   0.40,
		},
	}
}

func TestRenderPDF_SinglePage(t *testing.T) {
	doc := sampleBillDocument()

	pdfBytes, checksum, err := RenderPDF(doc)
	if err != nil {
		t.Fatalf("RenderPDF failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatalf("RenderPDF returned empty bytes")
	}

	// Verify PDF magic header "%PDF-"
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Errorf("expected PDF header %%PDF-, got %q", pdfBytes[:min(8, len(pdfBytes))])
	}

	// Verify SHA-256 checksum format (64 hex characters)
	if len(checksum) != 64 {
		t.Errorf("expected 64-char sha256 checksum, got %q (len %d)", checksum, len(checksum))
	}
}

func TestRenderPDF_WatermarkDisabled(t *testing.T) {
	doc := sampleBillDocument()
	doc.Watermark.Enabled = false

	pdfBytes, checksum, err := RenderPDF(doc)
	if err != nil {
		t.Fatalf("RenderPDF with watermark disabled failed: %v", err)
	}

	if len(pdfBytes) == 0 || len(checksum) != 64 {
		t.Fatalf("invalid rendered PDF output")
	}
}

func TestRenderPDF_MultiPageFlow(t *testing.T) {
	doc := sampleBillDocument()

	// Add 30 charge items to force multi-page rendering
	for i := 4; i <= 35; i++ {
		doc.LineItems = append(doc.LineItems, DocumentLineItem{
			Index:             i,
			ChargeHeadName:    "Additional Facility Maintenance Line",
			SACCode:           "999511",
			TaxClassification: "TAXABLE",
			TaxRate:           18.0,
			Rate:              100.00,
			Quantity:          1,
			Amount:            100.00,
			TaxAmount:         18.00,
			DisplayOrder:      i,
		})
	}
	doc.ChargeSubtotal += 3200.00
	doc.TotalPayable += 3200.00 * 1.18

	pdfBytes, _, err := RenderPDF(doc)
	if err != nil {
		t.Fatalf("RenderPDF multi-page failed: %v", err)
	}

	// Multi-page PDF must be larger than single-page
	if len(pdfBytes) < 5000 {
		t.Errorf("expected multi-page PDF to be >5KB, got %d bytes", len(pdfBytes))
	}
}

func TestRenderPDF_FieldOverrides(t *testing.T) {
	doc := sampleBillDocument()
	doc.FieldOverrides = map[string]TemplateFieldConfig{
		"society_gstin": {
			IsVisible: false,
		},
		"bill_number": {
			IsVisible:    true,
			DisplayLabel: "INVOICE REFERENCE",
		},
		"amount_in_words": {
			IsVisible: false,
		},
		"bank_details": {
			IsVisible: false,
		},
	}

	pdfBytes, checksum, err := RenderPDF(doc)
	if err != nil {
		t.Fatalf("RenderPDF with field overrides failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatal("expected non-empty pdfBytes")
	}
	if len(checksum) != 64 {
		t.Fatalf("expected 64-char sha256 checksum, got %d", len(checksum))
	}
}

func TestRenderPDF_ChecksumVerification(t *testing.T) {
	doc := sampleBillDocument()
	pdfBytes, checksum, err := RenderPDF(doc)
	if err != nil {
		t.Fatalf("RenderPDF failed: %v", err)
	}

	importHash := bytes.Equal([]byte{}, []byte{})
	_ = importHash

	// Compute independent sha256
	h := sha256Sum(pdfBytes)
	if h != checksum {
		t.Errorf("checksum mismatch: got %s, calculated %s", checksum, h)
	}
}

func sha256Sum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
