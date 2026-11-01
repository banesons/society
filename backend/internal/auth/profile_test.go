package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestResolveDisplayName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		phone     string
		expected  string
	}{
		{
			name:      "both first and last name present",
			firstName: "Shuubham",
			lastName:  "Bane",
			phone:     "919967089101",
			expected:  "Shuubham Bane",
		},
		{
			name:      "first name only",
			firstName: "Shuubham",
			lastName:  "",
			phone:     "919967089101",
			expected:  "Shuubham",
		},
		{
			name:      "last name only",
			firstName: "",
			lastName:  "Bane",
			phone:     "919967089101",
			expected:  "Bane",
		},
		{
			name:      "first and last name with excess whitespace",
			firstName: "  Shuubham  ",
			lastName:  "   Bane   ",
			phone:     "919967089101",
			expected:  "Shuubham Bane",
		},
		{
			name:      "empty names fall back to verified account phone",
			firstName: "",
			lastName:  "",
			phone:     "919967089101",
			expected:  "+91 9967089101",
		},
		{
			name:      "whitespace-only names fall back to verified account phone",
			firstName: "   ",
			lastName:  "   ",
			phone:     "919967089101",
			expected:  "+91 9967089101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDisplayName(tt.firstName, tt.lastName, tt.phone)
			if got != tt.expected {
				t.Errorf("ResolveDisplayName(%q, %q, %q) = %q; want %q", tt.firstName, tt.lastName, tt.phone, got, tt.expected)
			}
			if got == "Verified Resident" {
				t.Errorf("ResolveDisplayName must never return hardcoded Verified Resident")
			}
		})
	}
}

func TestUpdateProfileValidation(t *testing.T) {
	svc := &Service{} // pool is nil; validation occurs before DB queries

	ctx := context.Background()
	randomID := uuid.New()

	t.Run("rejects empty first name", func(t *testing.T) {
		_, err := svc.UpdateProfile(ctx, randomID, UpdateProfileParams{
			FirstName: "   ",
		})
		if err == nil || err.Error() != "first name cannot be empty" {
			t.Errorf("expected 'first name cannot be empty', got %v", err)
		}
	})

	t.Run("rejects first name exceeding 100 characters", func(t *testing.T) {
		longName := string(make([]byte, 101))
		_, err := svc.UpdateProfile(ctx, randomID, UpdateProfileParams{
			FirstName: longName,
		})
		if err == nil || err.Error() != "first name exceeds maximum length of 100 characters" {
			t.Errorf("expected length error, got %v", err)
		}
	})

	t.Run("rejects last name exceeding 100 characters", func(t *testing.T) {
		longName := string(make([]byte, 101))
		_, err := svc.UpdateProfile(ctx, randomID, UpdateProfileParams{
			FirstName: "Valid",
			LastName:  &longName,
		})
		if err == nil || err.Error() != "last name exceeds maximum length of 100 characters" {
			t.Errorf("expected last name length error, got %v", err)
		}
	})

	t.Run("rejects invalid email format", func(t *testing.T) {
		badEmail := "not-an-email"
		_, err := svc.UpdateProfile(ctx, randomID, UpdateProfileParams{
			FirstName: "Valid",
			Email:     &badEmail,
		})
		if err == nil || err.Error() != "invalid email address format" {
			t.Errorf("expected 'invalid email address format', got %v", err)
		}
	})
}
