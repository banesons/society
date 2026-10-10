package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ErrInvalidPhone indicates the phone number could not be normalized to Indian E.164 format.
	ErrInvalidPhone = errors.New("invalid Indian mobile phone number")

	// Canonical Indian mobile regex: +91 followed by 10 digits starting with 6, 7, 8, or 9
	e164IndianRegex = regexp.MustCompile(`^\+91[6-9]\d{9}$`)
)

// NormalizePhone cleans, standardizes, and validates an Indian mobile number into canonical E.164 format (+91XXXXXXXXXX).
func NormalizePhone(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", ErrInvalidPhone
	}

	// Step 1: Strip whitespace, hyphens, parentheses, dots, and formatting characters
	var digitsOnly strings.Builder
	hasLeadingPlus := strings.HasPrefix(strings.TrimSpace(input), "+")

	for _, ch := range input {
		if unicode.IsDigit(ch) {
			digitsOnly.WriteRune(ch)
		}
	}

	raw := digitsOnly.String()
	if raw == "" {
		return "", ErrInvalidPhone
	}

	// Step 2: Handle prefixes
	var nationalNumber string

	if hasLeadingPlus || strings.HasPrefix(raw, "91") && len(raw) == 12 {
		if strings.HasPrefix(raw, "91") && len(raw) == 12 {
			nationalNumber = raw[2:]
		} else {
			nationalNumber = raw
		}
	} else if strings.HasPrefix(raw, "0") && len(raw) == 11 {
		// Strip leading 0 (e.g. 09967089101 -> 9967089101)
		nationalNumber = raw[1:]
	} else if len(raw) == 10 {
		nationalNumber = raw
	} else {
		return "", ErrInvalidPhone
	}

	// Must be exactly 10 digits starting with 6, 7, 8, or 9
	if len(nationalNumber) != 10 {
		return "", ErrInvalidPhone
	}

	canonical := "+91" + nationalNumber

	if !e164IndianRegex.MatchString(canonical) {
		return "", ErrInvalidPhone
	}

	return canonical, nil
}

// MaskPhone returns a PII-safe representation of the canonical phone number.
// Example: +919967089101 -> +91 9967xxxx01
func MaskPhone(phone string) string {
	if len(phone) < 13 || !strings.HasPrefix(phone, "+91") {
		// Fallback masking if non-standard
		if len(phone) >= 4 {
			return phone[:2] + "****" + phone[len(phone)-2:]
		}
		return "****"
	}

	// Canonical format: +91 (3 chars) + 10 digits
	// Example: +919967089101
	// Country: +91
	// Prefix: 9967 (first 4 digits of 10)
	// Mask: xxxx (middle 4 digits)
	// Suffix: 01 (last 2 digits)
	country := phone[:3]       // +91
	prefix := phone[3:7]       // 9967
	suffix := phone[len(phone)-2:] // 01

	return country + " " + prefix + "xxxx" + suffix
}
