package society

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFileContent(t *testing.T) {
	// 1. Valid PDF
	pdfBytes := []byte("%PDF-1.4 sample pdf content")
	mime, stored, hash, err := ValidateFileContent(pdfBytes, "application/pdf", "sample.pdf")
	if err != nil {
		t.Fatalf("expected valid PDF, got: %v", err)
	}
	if mime != "application/pdf" {
		t.Errorf("expected application/pdf, got: %s", mime)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char sha256 hex string, got: %s", hash)
	}
	if len(stored) != len(pdfBytes) {
		t.Errorf("expected stored length %d, got %d", len(pdfBytes), len(stored))
	}

	// 2. Valid PNG
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D}
	mime, _, _, err = ValidateFileContent(pngBytes, "image/png", "photo.png")
	if err != nil {
		t.Fatalf("expected valid PNG, got: %v", err)
	}
	if mime != "image/png" {
		t.Errorf("expected image/png, got: %s", mime)
	}

	// 3. Valid JPEG
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	mime, _, _, err = ValidateFileContent(jpegBytes, "image/jpeg", "photo.jpg")
	if err != nil {
		t.Fatalf("expected valid JPEG, got: %v", err)
	}
	if mime != "image/jpeg" {
		t.Errorf("expected image/jpeg, got: %s", mime)
	}

	// 4. Valid WebP
	webpBytes := append([]byte("RIFF1234WEBP"), []byte("VP8X...")...)
	mime, _, _, err = ValidateFileContent(webpBytes, "image/webp", "image.webp")
	if err != nil {
		t.Fatalf("expected valid WebP, got: %v", err)
	}
	if mime != "image/webp" {
		t.Errorf("expected image/webp, got: %s", mime)
	}

	// 5. Valid DOCX
	docxBytes := append([]byte{0x50, 0x4B, 0x03, 0x04}, []byte("dummy zip content")...)
	mime, _, _, err = ValidateFileContent(docxBytes, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "report.docx")
	if err != nil {
		t.Fatalf("expected valid DOCX, got: %v", err)
	}
	if mime != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("expected docx mime, got: %s", mime)
	}

	// 6. Valid XLSX
	xlsxBytes := append([]byte{0x50, 0x4B, 0x03, 0x04}, []byte("dummy zip content")...)
	mime, _, _, err = ValidateFileContent(xlsxBytes, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "sheet.xlsx")
	if err != nil {
		t.Fatalf("expected valid XLSX, got: %v", err)
	}
	if mime != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("expected xlsx mime, got: %s", mime)
	}

	// 7. Unsupported file format (e.g. text/plain, script, HTML)
	txtBytes := []byte("plain text content")
	_, _, _, err = ValidateFileContent(txtBytes, "text/plain", "doc.txt")
	if err != ErrUnsupportedFileType {
		t.Errorf("expected ErrUnsupportedFileType for txt, got: %v", err)
	}

	htmlBytes := []byte("<html><body>danger</body></html>")
	_, _, _, err = ValidateFileContent(htmlBytes, "text/html", "exploit.html")
	if err != ErrUnsupportedFileType {
		t.Errorf("expected ErrUnsupportedFileType for html, got: %v", err)
	}

	// 8. Empty file
	_, _, _, err = ValidateFileContent([]byte{}, "application/pdf", "empty.pdf")
	if err != ErrEmptyFile {
		t.Errorf("expected ErrEmptyFile, got: %v", err)
	}

	// 9. Original file exceeds 10MB
	hugeBytes := make([]byte, MaxOriginalFileSize+10)
	copy(hugeBytes, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(hugeBytes, "application/pdf", "huge.pdf")
	if err != ErrFileExceedsOriginalLimit {
		t.Errorf("expected ErrFileExceedsOriginalLimit, got: %v", err)
	}

	// 10. File between 5MB and 10MB without optimization exceeds accepted limit of 5MB
	sixMbPdf := make([]byte, 6*1024*1024)
	copy(sixMbPdf, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(sixMbPdf, "application/pdf", "sixmb.pdf")
	if err != ErrFileExceedsAcceptedLimit {
		t.Errorf("expected ErrFileExceedsAcceptedLimit for 6MB unoptimized PDF, got: %v", err)
	}

	// 11. MIME mismatch (PDF file bytes, but client claims image/png)
	_, _, _, err = ValidateFileContent(pdfBytes, "image/png", "sample.pdf")
	if err != ErrMimeMismatch {
		t.Errorf("expected ErrMimeMismatch, got: %v", err)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input       string
		expected    string
		expectError bool
	}{
		{"bylaws.pdf", "bylaws.pdf", false},
		{"../../etc/passwd", "passwd", false},
		{"folder\\subfolder\\report.docx", "report.docx", false},
		{"", "", true},
		{".", "", true},
		{"..", "", true},
		{"/root/bad\x00file.pdf", "", true},
		{strings.Repeat("a", 160) + ".pdf", strings.Repeat("a", 146) + ".pdf", false},
	}

	for _, tc := range tests {
		got, err := SanitizeFilename(tc.input)
		if tc.expectError && err == nil {
			t.Errorf("input %q: expected error, got nil (%s)", tc.input, got)
		} else if !tc.expectError && err != nil {
			t.Errorf("input %q: unexpected error: %v", tc.input, err)
		} else if !tc.expectError && got != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestOptimizeFileContent(t *testing.T) {
	// PDF should never be recompressed
	pdfOriginal := []byte("%PDF-1.4 sample content")
	optPdf, changed := OptimizeFileContent(pdfOriginal, "application/pdf")
	if changed {
		t.Error("expected changed == false for PDF")
	}
	if string(optPdf) != string(pdfOriginal) {
		t.Error("expected unchanged bytes for PDF")
	}

	// JPEG should never be recompressed
	jpegOriginal := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}
	optJpeg, changed := OptimizeFileContent(jpegOriginal, "image/jpeg")
	if changed {
		t.Error("expected changed == false for JPEG")
	}
	if len(optJpeg) != len(jpegOriginal) {
		t.Error("expected unchanged bytes for JPEG")
	}
}

func TestValidateFileContent_BoundaryLimits(t *testing.T) {
	// 1. Stored accepted limit boundary tests (5 MiB = 5,242,880 bytes)
	// 1a. Immediately below 5MB: 5,242,879 bytes
	below5MB := make([]byte, MaxAcceptedFileSize-1)
	copy(below5MB, []byte("%PDF-1.4"))
	mime, stored, hash, err := ValidateFileContent(below5MB, "application/pdf", "below5mb.pdf")
	if err != nil {
		t.Fatalf("expected 5MB-1 byte PDF to be accepted, got: %v", err)
	}
	if len(stored) != MaxAcceptedFileSize-1 {
		t.Errorf("expected stored length %d, got %d", MaxAcceptedFileSize-1, len(stored))
	}
	if mime != "application/pdf" || len(hash) != 64 {
		t.Errorf("invalid mime or hash: mime=%s, hash=%s", mime, hash)
	}

	// 1b. Exactly at 5MB: 5,242,880 bytes
	at5MB := make([]byte, MaxAcceptedFileSize)
	copy(at5MB, []byte("%PDF-1.4"))
	_, storedAt, _, err := ValidateFileContent(at5MB, "application/pdf", "at5mb.pdf")
	if err != nil {
		t.Fatalf("expected exactly 5MB PDF to be accepted, got: %v", err)
	}
	if len(storedAt) != MaxAcceptedFileSize {
		t.Errorf("expected stored length %d, got %d", MaxAcceptedFileSize, len(storedAt))
	}

	// 1c. Immediately above 5MB: 5,242,881 bytes
	above5MB := make([]byte, MaxAcceptedFileSize+1)
	copy(above5MB, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(above5MB, "application/pdf", "above5mb.pdf")
	if !errors.Is(err, ErrFileExceedsAcceptedLimit) {
		t.Fatalf("expected ErrFileExceedsAcceptedLimit for 5MB+1 byte, got: %v", err)
	}

	// 2. Original upload limit boundary tests (10 MiB = 10,485,760 bytes)
	// 2a. Immediately below 10MB: 10,485,759 bytes
	below10MB := make([]byte, MaxOriginalFileSize-1)
	copy(below10MB, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(below10MB, "application/pdf", "below10mb.pdf")
	// Since unoptimized PDF > 5MB, it should fail with ErrFileExceedsAcceptedLimit, NOT ErrFileExceedsOriginalLimit
	if !errors.Is(err, ErrFileExceedsAcceptedLimit) {
		t.Fatalf("expected ErrFileExceedsAcceptedLimit for 10MB-1 byte unoptimized PDF, got: %v", err)
	}

	// 2b. Exactly at 10MB: 10,485,760 bytes
	at10MB := make([]byte, MaxOriginalFileSize)
	copy(at10MB, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(at10MB, "application/pdf", "at10mb.pdf")
	if !errors.Is(err, ErrFileExceedsAcceptedLimit) {
		t.Fatalf("expected ErrFileExceedsAcceptedLimit for exactly 10MB unoptimized PDF, got: %v", err)
	}

	// 2c. Immediately above 10MB: 10,485,761 bytes
	above10MB := make([]byte, MaxOriginalFileSize+1)
	copy(above10MB, []byte("%PDF-1.4"))
	_, _, _, err = ValidateFileContent(above10MB, "application/pdf", "above10mb.pdf")
	if !errors.Is(err, ErrFileExceedsOriginalLimit) {
		t.Fatalf("expected ErrFileExceedsOriginalLimit for 10MB+1 byte, got: %v", err)
	}
}
