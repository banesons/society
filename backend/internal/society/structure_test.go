package society

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGenerateUnitIdentifiers_ExampleA(t *testing.T) {
	// Example A from specifications:
	// Building: A, Wing: A, Number of floors: 9, Starting floor: 1, Units per floor: 3, Starting unit: 101, Prefix: A
	// Expected units: A101 through A903, total: 27 units
	params := GenerateUnitsParams{
		BuildingID:         uuid.New(),
		StartingFloor:      1,
		TotalFloors:        9,
		UnitsPerFloor:      3,
		StartingUnitNumber: 101,
		Prefix:             "A",
	}

	res, err := GenerateUnitIdentifiers(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalUnits != 27 {
		t.Fatalf("expected 27 units, got %d", res.TotalUnits)
	}

	expectedFirst := "A101"
	if res.Units[0].UnitIdentifier != expectedFirst {
		t.Errorf("expected first unit %s, got %s", expectedFirst, res.Units[0].UnitIdentifier)
	}

	expectedLast := "A903"
	if res.Units[26].UnitIdentifier != expectedLast {
		t.Errorf("expected last unit %s, got %s", expectedLast, res.Units[26].UnitIdentifier)
	}

	// Verify floor 1 units
	floor1Units := []string{"A101", "A102", "A103"}
	for i, expected := range floor1Units {
		if res.Units[i].UnitIdentifier != expected {
			t.Errorf("expected unit[%d] = %s, got %s", i, expected, res.Units[i].UnitIdentifier)
		}
		if res.Units[i].FloorNumber != 1 {
			t.Errorf("expected unit[%d] floor 1, got %d", i, res.Units[i].FloorNumber)
		}
	}

	// Verify floor 9 units
	floor9Units := []string{"A901", "A902", "A903"}
	for i, expected := range floor9Units {
		idx := 24 + i
		if res.Units[idx].UnitIdentifier != expected {
			t.Errorf("expected unit[%d] = %s, got %s", idx, expected, res.Units[idx].UnitIdentifier)
		}
		if res.Units[idx].FloorNumber != 9 {
			t.Errorf("expected unit[%d] floor 9, got %d", idx, res.Units[idx].FloorNumber)
		}
	}
}

func TestGenerateUnitIdentifiers_ExampleB(t *testing.T) {
	// Example B from specifications:
	// Building: B, Wing: B, 5 floors, 4 units per floor, Prefix: B
	// Expected: B101..B104 up to B501..B504 (20 units)
	params := GenerateUnitsParams{
		BuildingID:         uuid.New(),
		StartingFloor:      1,
		TotalFloors:        5,
		UnitsPerFloor:      4,
		StartingUnitNumber: 101,
		Prefix:             "B",
	}

	res, err := GenerateUnitIdentifiers(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalUnits != 20 {
		t.Fatalf("expected 20 units, got %d", res.TotalUnits)
	}

	if res.Units[0].UnitIdentifier != "B101" {
		t.Errorf("expected B101, got %s", res.Units[0].UnitIdentifier)
	}
	if res.Units[19].UnitIdentifier != "B504" {
		t.Errorf("expected B504, got %s", res.Units[19].UnitIdentifier)
	}
}

func TestGenerateUnitIdentifiers_ExampleC_Overrides(t *testing.T) {
	// Example C: Non-uniform wing
	// Building: C, Wing: C, 4 floors
	// Floor 1: 2 units
	// Floor 2: 3 units
	// Floor 3: 4 units
	// Floor 4: 2 units
	// Total: 11 units
	params := GenerateUnitsParams{
		BuildingID:         uuid.New(),
		StartingFloor:      1,
		TotalFloors:        4,
		UnitsPerFloor:      2, // baseline
		StartingUnitNumber: 101,
		Prefix:             "C",
		FloorOverrides: map[int]int{
			2: 3,
			3: 4,
		},
	}

	res, err := GenerateUnitIdentifiers(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalUnits != 11 {
		t.Fatalf("expected 11 units, got %d", res.TotalUnits)
	}

	// Verify counts per floor
	floorCounts := make(map[int]int)
	for _, u := range res.Units {
		floorCounts[u.FloorNumber]++
	}

	if floorCounts[1] != 2 {
		t.Errorf("floor 1 count = %d, want 2", floorCounts[1])
	}
	if floorCounts[2] != 3 {
		t.Errorf("floor 2 count = %d, want 3", floorCounts[2])
	}
	if floorCounts[3] != 4 {
		t.Errorf("floor 3 count = %d, want 4", floorCounts[3])
	}
	if floorCounts[4] != 2 {
		t.Errorf("floor 4 count = %d, want 2", floorCounts[4])
	}
}

func TestGenerateUnitIdentifiers_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		params  GenerateUnitsParams
		wantErr error
	}{
		{
			name: "Zero floors",
			params: GenerateUnitsParams{
				TotalFloors:   0,
				UnitsPerFloor: 4,
			},
			wantErr: ErrInvalidFloorCount,
		},
		{
			name: "Negative floors",
			params: GenerateUnitsParams{
				TotalFloors:   -5,
				UnitsPerFloor: 4,
			},
			wantErr: ErrInvalidFloorCount,
		},
		{
			name: "Over 100 floors",
			params: GenerateUnitsParams{
				TotalFloors:   101,
				UnitsPerFloor: 2,
			},
			wantErr: ErrInvalidFloorCount,
		},
		{
			name: "Zero units per floor",
			params: GenerateUnitsParams{
				TotalFloors:   5,
				UnitsPerFloor: 0,
			},
			wantErr: ErrInvalidUnitCount,
		},
		{
			name: "Over 50 units per floor",
			params: GenerateUnitsParams{
				TotalFloors:   5,
				UnitsPerFloor: 51,
			},
			wantErr: ErrInvalidUnitCount,
		},
		{
			name: "Bulk limit exceeded",
			params: GenerateUnitsParams{
				StartingFloor: 1,
				TotalFloors:   25,
				UnitsPerFloor: 45, // 25 * 45 = 1125 > 1000 limit
			},
			wantErr: ErrBulkLimitExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateUnitIdentifiers(tt.params)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGenerateUnitIdentifiers_NoPrefix(t *testing.T) {
	// Without prefix: "101", "102", "201", etc.
	params := GenerateUnitsParams{
		StartingFloor: 1,
		TotalFloors:   2,
		UnitsPerFloor: 2,
		Prefix:        "",
	}

	res, err := GenerateUnitIdentifiers(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalUnits != 4 {
		t.Fatalf("expected 4 units, got %d", res.TotalUnits)
	}

	expected := []string{"101", "102", "201", "202"}
	for i, exp := range expected {
		if res.Units[i].UnitIdentifier != exp {
			t.Errorf("unit[%d] = %s, want %s", i, res.Units[i].UnitIdentifier, exp)
		}
	}
}
