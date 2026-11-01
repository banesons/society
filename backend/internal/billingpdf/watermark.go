package billingpdf

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	"github.com/go-pdf/fpdf"
)

//go:embed assets/chs_logo.png
var EmbeddedCHSLogo []byte

const (
	LogoImageIdentifier = "chs_official_logo_watermark"
)

// ErrLogoAssetMissing is returned if the official CHS logo asset is unavailable.
var ErrLogoAssetMissing = errors.New("official CHS logo asset is missing or empty")

// RegisterWatermarkAsset registers the embedded official CHS logo with the fpdf document.
func RegisterWatermarkAsset(pdf *fpdf.Fpdf) error {
	if len(EmbeddedCHSLogo) == 0 {
		return ErrLogoAssetMissing
	}

	opt := fpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   false,
	}

	pdf.RegisterImageOptionsReader(LogoImageIdentifier, opt, bytes.NewReader(EmbeddedCHSLogo))
	return nil
}

// DrawWatermark stamps the CHS logo watermark on the current page if enabled.
func DrawWatermark(pdf *fpdf.Fpdf, cfg WatermarkConfig) {
	if !cfg.Enabled || len(EmbeddedCHSLogo) == 0 {
		return
	}

	pageWidth, pageHeight := pdf.GetPageSize()

	// Default scale = 0.40 of page width (~84mm on A4)
	scale := cfg.Scale
	if scale <= 0 || scale > 1.0 {
		scale = 0.40
	}

	targetW := pageWidth * scale
	// Square 1:1 aspect ratio for official CHS icon
	targetH := targetW

	// Opacity default 0.08
	opacity := cfg.Opacity
	if opacity <= 0 {
		opacity = 0.08
	} else if opacity > 1.0 {
		opacity = 1.0
	}

	// Calculate center position
	posX := (pageWidth - targetW) / 2.0
	posY := (pageHeight - targetH) / 2.0

	// Set transparency
	pdf.SetAlpha(opacity, "Normal")

	// Render background watermark
	pdf.ImageOptions(
		LogoImageIdentifier,
		posX, posY,
		targetW, targetH,
		false,
		fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false},
		0,
		"",
	)

	// Restore full opacity for foreground content
	pdf.SetAlpha(1.0, "Normal")
}

// FormatWatermarkSummary returns a string description for diagnostics.
func FormatWatermarkSummary(cfg WatermarkConfig) string {
	if !cfg.Enabled {
		return "DISABLED"
	}
	return fmt.Sprintf("ENABLED(opacity=%.2f, scale=%.2f)", cfg.Opacity, cfg.Scale)
}
