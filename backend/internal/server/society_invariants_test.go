package server

import (
	"context"
	"testing"
	"time"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
	"chs/backend/internal/testutil"
)

func TestCanonicalSocietiesInvariant(t *testing.T) {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping database invariant test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure cleanup runs first in case transient data exists
	_ = testutil.CleanExtraSocieties(ctx, db.Pool())

	var count int
	err = db.Pool().QueryRow(ctx, "SELECT COUNT(*) FROM societies").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query societies count: %v", err)
	}

	if count != 2 {
		t.Errorf("invariant violated: expected exactly 2 canonical societies in persistent database, got %d", count)
	}

	// Verify exact codes
	rows, err := db.Pool().Query(ctx, "SELECT society_code, name, status FROM societies ORDER BY society_code")
	if err != nil {
		t.Fatalf("failed to query societies: %v", err)
	}
	defer rows.Close()

	expectedCodes := map[string]string{
		testutil.CanonicalSociety1Code: "Shree Ram Cooperative Housing Society",
		testutil.CanonicalSociety2Code: "Greenview Residency",
	}

	for rows.Next() {
		var code, name, status string
		if err := rows.Scan(&code, &name, &status); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}

		expectedName, ok := expectedCodes[code]
		if !ok {
			t.Errorf("unexpected society code found in database: %s (%s)", code, name)
		} else if name != expectedName {
			t.Errorf("society %s: expected name %q, got %q", code, expectedName, name)
		}
		if status != "ACTIVE" {
			t.Errorf("society %s: expected status ACTIVE, got %s", code, status)
		}
	}
}
