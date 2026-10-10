package society

import (
	"strings"
	"testing"
)

func TestDeriveCityCode(t *testing.T) {
	tests := []struct {
		city     string
		expected string
		wantErr  bool
	}{
		{"Mumbai", "MUM", false},
		{"MUMBAI", "MUM", false},
		{"  Pune  ", "PUN", false},
		{"pune", "PUN", false},
		{"Bangalore", "BLR", false},
		{"Bengaluru", "BLR", false},
		{"New Delhi", "DEL", false},
		{"Hyderabad", "HYD", false},
		{"Chennai", "CHE", false},
		{"Kolkata", "CCU", false},
		// Fallbacks
		{"Nagapattinam", "NAG", false}, // Fallback first 3 letters
		{"Goa", "GOA", false},          // Exactly 3 chars
		{"St. Louis", "STL", false},    // Strips spaces and dots
		{"", "", true},                 // Empty
		{"   ", "", true},              // Whitespace only
		{"AB", "", true},               // Too short (< 3 alphanumeric)
		{"!@#", "", true},              // No alphanumeric
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			got, err := DeriveCityCode(tt.city)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DeriveCityCode(%q) error = %v, wantErr %v", tt.city, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("DeriveCityCode(%q) = %q, want %q", tt.city, got, tt.expected)
			}
		})
	}
}

func TestGenerateSocietyCode(t *testing.T) {
	t.Run("Mumbai generates SOC-MUM-XXXXXX", func(t *testing.T) {
		code, err := GenerateSocietyCode("Mumbai")
		if err != nil {
			t.Fatalf("GenerateSocietyCode failed: %v", err)
		}
		if !strings.HasPrefix(code, "SOC-MUM-") {
			t.Errorf("expected SOC-MUM- prefix, got %s", code)
		}
		parts := strings.Split(code, "-")
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %v", parts)
		}
		if len(parts[2]) != 6 {
			t.Errorf("expected suffix length 6, got %d in %s", len(parts[2]), parts[2])
		}
	})

	t.Run("Pune generates SOC-PUN-XXXXXX", func(t *testing.T) {
		code, err := GenerateSocietyCode("Pune")
		if err != nil {
			t.Fatalf("GenerateSocietyCode failed: %v", err)
		}
		if !strings.HasPrefix(code, "SOC-PUN-") {
			t.Errorf("expected SOC-PUN- prefix, got %s", code)
		}
		parts := strings.Split(code, "-")
		if len(parts) != 3 || len(parts[2]) != 6 {
			t.Errorf("malformed code %s", code)
		}
	})

	t.Run("Suffix characters are within SuffixCharset", func(t *testing.T) {
		for i := 0; i < 50; i++ {
			suffix, err := GenerateRandomSuffix()
			if err != nil {
				t.Fatalf("GenerateRandomSuffix error: %v", err)
			}
			if len(suffix) != 6 {
				t.Fatalf("expected 6 chars, got %d", len(suffix))
			}
			for _, ch := range suffix {
				if !strings.ContainsRune(SuffixCharset, ch) {
					t.Fatalf("invalid char %c in suffix %s", ch, suffix)
				}
			}
		}
	})
}
