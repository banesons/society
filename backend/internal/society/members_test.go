package society

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Standard Indian with plus", "+919820098200", "919820098200", false},
		{"Spaces in input", "+91 98200 98200", "919820098200", false},
		{"Dashes and parentheses", "+91-(98200)-98200", "919820098200", false},
		{"10 digits without prefix", "9820098200", "919820098200", false},
		{"12 digits with 91", "919820098200", "919820098200", false},
		{"Invalid characters", "+9198200abcde", "", true},
		{"Too short", "+123", "", true},
		{"Empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhone(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizePhone(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMaskPhoneAndEmail(t *testing.T) {
	phoneTests := []struct {
		input string
		want  string
	}{
		{"+919820098200", "919820098200"},
		{"919967089101", "919967089101"},
		{"9820098200", "919820098200"},
	}

	for _, tt := range phoneTests {
		got := MaskPhone(tt.input)
		if got != tt.want {
			t.Errorf("MaskPhone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}

	emailTests := []struct {
		input string
		want  string
	}{
		{"ramesh@example.com", "ra****@example.com"},
		{"ab@example.com", "ab****@example.com"},
		{"a@example.com", "a****@example.com"},
		{"", ""},
	}

	for _, tt := range emailTests {
		got := MaskEmail(tt.input)
		if got != tt.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestComputeRelationshipsVersionHash(t *testing.T) {
	relID1 := uuid.New()
	relID2 := uuid.New()
	now := time.Now().UTC()

	relsA := []UnitRelationship{
		{ID: relID1, RelationshipType: RelTypePrimaryOwner, Status: RelStatusActive, IsOccupant: true, UpdatedAt: now},
		{ID: relID2, RelationshipType: RelTypePrimaryTenant, Status: RelStatusActive, IsOccupant: true, UpdatedAt: now},
	}

	// Order swapped
	relsB := []UnitRelationship{
		{ID: relID2, RelationshipType: RelTypePrimaryTenant, Status: RelStatusActive, IsOccupant: true, UpdatedAt: now},
		{ID: relID1, RelationshipType: RelTypePrimaryOwner, Status: RelStatusActive, IsOccupant: true, UpdatedAt: now},
	}

	hashA := computeRelationshipsVersionHash(relsA)
	hashB := computeRelationshipsVersionHash(relsB)

	if hashA != hashB {
		t.Errorf("expected deterministic hash regardless of slice order, got %q vs %q", hashA, hashB)
	}

	// Modified relationship
	relsC := []UnitRelationship{
		{ID: relID1, RelationshipType: RelTypePrimaryOwner, Status: RelStatusActive, IsOccupant: false, UpdatedAt: now.Add(time.Second)},
		{ID: relID2, RelationshipType: RelTypePrimaryTenant, Status: RelStatusActive, IsOccupant: true, UpdatedAt: now},
	}
	hashC := computeRelationshipsVersionHash(relsC)

	if hashA == hashC {
		t.Errorf("expected different hash for modified relationship, got identical %q", hashA)
	}
}
