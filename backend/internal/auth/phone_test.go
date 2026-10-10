package auth

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{
			name:      "10 digits standard",
			input:     "9967089101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "leading zero 11 digits",
			input:     "09967089101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "already E.164 with plus",
			input:     "+919967089101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "hyphens and spaces",
			input:     "+91-9967-089-101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "parentheses formatting",
			input:     "+91 (9967) 089101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "91 prefix without plus 12 digits",
			input:     "919967089101",
			expected:  "+919967089101",
			expectErr: false,
		},
		{
			name:      "second test number +918422862001",
			input:     "8422862001",
			expected:  "+918422862001",
			expectErr: false,
		},
		{
			name:      "invalid: starts with 5 (invalid Indian telecom range)",
			input:     "5555555555",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "invalid: too short (9 digits)",
			input:     "996708910",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "invalid: too long (13 digits)",
			input:     "9967089101111",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "invalid: empty string",
			input:     "",
			expected:  "",
			expectErr: true,
		},
		{
			name:      "invalid: non-numeric garbage",
			input:     "abcdefghij",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := NormalizePhone(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil (normalized: %s)", tt.input, normalized)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if normalized != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, normalized)
				}
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "+919967089101",
			expected: "+91 9967xxxx01",
		},
		{
			input:    "+918422862001",
			expected: "+91 8422xxxx01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			masked := MaskPhone(tt.input)
			if masked != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, masked)
			}
		})
	}
}
