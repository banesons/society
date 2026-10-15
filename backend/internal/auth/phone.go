package auth

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ErrInvalidPhone indicates the phone number could not be normalized to Indian numeric format.
	ErrInvalidPhone = errors.New("invalid Indian mobile phone number")

	// Canonical Indian mobile regex: 91 followed by 10 digits starting with 6, 7, 8, or 9 (pure digits, no +)
	numericIndianRegex = regexp.MustCompile(`^91[6-9]\d{9}$`)
)

// NormalizePhone cleans, standardizes, trims, and validates an Indian mobile number into pure numeric digits (91XXXXXXXXXX).
// Removes '+', spaces, dashes, parentheses, dots.
func NormalizePhone(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ErrInvalidPhone
	}

	// Step 1: Strip whitespace, hyphens, parentheses, dots, '+' and formatting characters
	var digitsOnly strings.Builder
	for _, ch := range trimmed {
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

	if strings.HasPrefix(raw, "91") && len(raw) == 12 {
		nationalNumber = raw[2:]
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

	canonical := "91" + nationalNumber

	if !numericIndianRegex.MatchString(canonical) {
		return "", ErrInvalidPhone
	}

	return canonical, nil
}

// MaskPhone returns the phone number (pure digits or formatted for visibility as requested by user).
func MaskPhone(phone string) string {
	digits := ""
	for _, ch := range phone {
		if unicode.IsDigit(ch) {
			digits += string(ch)
		}
	}
	if len(digits) == 12 && strings.HasPrefix(digits, "91") {
		return digits
	}
	if len(digits) == 10 {
		return "91" + digits
	}
	return phone
}

// FormatDisplayPhone formats a numeric phone (e.g. 919967089101) with visual + prefix: +91 9967089101.
func FormatDisplayPhone(phone string) string {
	digits := ""
	for _, ch := range phone {
		if unicode.IsDigit(ch) {
			digits += string(ch)
		}
	}
	if len(digits) == 12 && strings.HasPrefix(digits, "91") {
		return "+91 " + digits[2:]
	}
	return phone
}
