package society

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrInvalidCity = errors.New("city name cannot be converted to a valid city code")
)

// SuffixCharset defines unambiguous uppercase alphanumeric characters for code suffixes.
// Excludes confusing pairs where practical (e.g. 0/O, 1/I). Length = 32.
const SuffixCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

const SuffixLength = 6

var nonAlphanumericRegex = regexp.MustCompile(`[^A-Z0-9]`)

// CityCodeMap contains deterministic 3-letter codes for major Indian metropolitan areas.
var CityCodeMap = map[string]string{
	"MUMBAI":      "MUM",
	"PUNE":        "PUN",
	"BANGALORE":   "BLR",
	"BENGALURU":   "BLR",
	"DELHI":       "DEL",
	"NEW DELHI":   "DEL",
	"HYDERABAD":   "HYD",
	"CHENNAI":     "CHE",
	"KOLKATA":     "CCU",
	"AHMEDABAD":   "AMD",
	"JAIPUR":      "JAI",
	"LUCKNOW":     "LKO",
	"SURAT":       "SUR",
	"NAGPUR":      "NAG",
	"INDORE":      "IND",
	"THANE":       "THA",
	"NAVI MUMBAI": "NVI",
	"NOIDA":       "NOI",
	"GURUGRAM":    "GUR",
	"GURGAON":     "GUR",
	"CHANDIGARH":  "CHD",
	"BHOPAL":      "BHO",
	"PATNA":       "PAT",
	"VADODARA":    "BDQ",
	"GHAZIABAD":   "GZB",
	"LUDHIANA":    "LDH",
	"AGRA":        "AGR",
	"NASHIK":      "NSK",
	"FARIDABAD":   "FBD",
	"MEERUT":      "MRT",
	"RAJKOT":      "RAJ",
	"VARANASI":    "VNS",
	"SRINAGAR":    "SXR",
	"AURANGABAD":  "IXU",
	"DHANBAD":     "DHN",
	"AMRITSAR":    "ATQ",
	"RANCHI":      "IXR",
	"COIMBATORE":  "CJB",
	"JABALPUR":    "JLR",
	"GWALIOR":     "GWL",
	"VIJAYAWADA":  "VGA",
	"JODHPUR":     "JDH",
	"MADURAI":     "IXM",
	"RAIPUR":      "RPR",
	"KOTA":        "KOT",
	"GUWAHATI":    "GAU",
	"KOCHI":       "COK",
	"COCHIN":      "COK",
}

// DeriveCityCode converts a city string to a deterministic 3-character uppercase code.
func DeriveCityCode(city string) (string, error) {
	trimmed := strings.TrimSpace(city)
	if trimmed == "" {
		return "", ErrInvalidCity
	}

	normalized := strings.ToUpper(trimmed)
	if code, found := CityCodeMap[normalized]; found {
		return code, nil
	}

	// Fallback: strip non-ASCII / non-alphanumeric characters
	var sb strings.Builder
	for _, r := range normalized {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if unicode.IsLetter(r) {
			// Convert accented unicode letters if any
			sb.WriteRune(unicode.ToUpper(r))
		}
	}

	clean := sb.String()
	if len(clean) < 3 {
		return "", fmt.Errorf("%w: city '%s' has insufficient alphanumeric characters", ErrInvalidCity, city)
	}

	return clean[:3], nil
}

// GenerateRandomSuffix produces a cryptographically secure 6-character uppercase alphanumeric string.
func GenerateRandomSuffix() (string, error) {
	result := make([]byte, SuffixLength)
	charsetLen := big.NewInt(int64(len(SuffixCharset)))

	for i := 0; i < SuffixLength; i++ {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate secure random suffix: %w", err)
		}
		result[i] = SuffixCharset[n.Int64()]
	}

	return string(result), nil
}

// GenerateSocietyCode derives a full society code in the format SOC-{CITY_CODE}-{RANDOM_SUFFIX}.
func GenerateSocietyCode(city string) (string, error) {
	cityCode, err := DeriveCityCode(city)
	if err != nil {
		return "", err
	}

	suffix, err := GenerateRandomSuffix()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SOC-%s-%s", cityCode, suffix), nil
}
