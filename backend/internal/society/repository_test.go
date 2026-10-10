package society

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"chs/backend/internal/config"
	"chs/backend/internal/database"
)

func setupTestDB(t *testing.T) *Repository {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping database integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	return NewRepository(db.Pool())
}

func getOrCreateTestSuperAdmin(t *testing.T, repo *Repository) uuid.UUID {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userID uuid.UUID
	query := `SELECT id FROM users LIMIT 1`
	err := repo.pool.QueryRow(ctx, query).Scan(&userID)
	if err != nil {
		insertQuery := `INSERT INTO users (phone, status) VALUES ('+919967089101', 'ACTIVE') RETURNING id`
		err = repo.pool.QueryRow(ctx, insertQuery).Scan(&userID)
		if err != nil {
			t.Fatalf("failed to get/create test super admin: %v", err)
		}
	}
	return userID
}

func TestSocietyLifecycleAndImmutability(t *testing.T) {
	repo := setupTestDB(t)
	actorID := getOrCreateTestSuperAdmin(t, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. Create Society
	regNum := "MUM/REG/12345"
	email := "contact@gokuldham.org"
	phone := "+919820123456"
	addr2 := "Near Film City"

	soc, err := repo.CreateSociety(ctx, CreateSocietyParams{
		Name:               "Gokuldham CHS",
		RegistrationNumber: &regNum,
		AddressLine1:       "Plot 10, Goregaon East",
		AddressLine2:       &addr2,
		Locality:           "Film City Road",
		City:               "Mumbai",
		State:              "Maharashtra",
		Pincode:            "400063",
		ContactEmail:       &email,
		ContactPhone:       &phone,
	}, actorID, "127.0.0.1", "TestRunner")

	if err != nil {
		t.Fatalf("CreateSociety failed: %v", err)
	}

	if soc.Status != StatusPending {
		t.Errorf("expected initial status PENDING, got %s", soc.Status)
	}
	if !strings.HasPrefix(soc.SocietyCode, "SOC-MUM-") {
		t.Errorf("expected SOC-MUM- prefix in code %s", soc.SocietyCode)
	}

	initialCode := soc.SocietyCode

	// 2. Immutability Trigger Test: direct UPDATE to society_code must fail
	_, err = repo.pool.Exec(ctx, `UPDATE societies SET society_code = 'SOC-MUM-999999' WHERE id = $1`, soc.ID)
	if err == nil {
		t.Fatalf("expected database immutability trigger to reject society_code update, but it succeeded")
	}
	if !strings.Contains(err.Error(), "immutable") {
		t.Errorf("expected trigger error message mentioning 'immutable', got %v", err)
	}

	// 3. Invalid Transition: PENDING -> SUSPENDED must fail
	legalCode := "LEGAL_DISPUTE"
	_, err = repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{ReasonCode: &legalCode}, actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrInvalidStatusTransition.Error()) {
		t.Errorf("expected ErrInvalidStatusTransition for PENDING -> SUSPENDED, got %v", err)
	}

	// 4. Valid Transition: PENDING -> ACTIVE (Activate)
	activated, err := repo.ActivateSociety(ctx, soc.ID, "Onboarding approved", actorID, "127.0.0.1", "TestRunner")
	if err != nil {
		t.Fatalf("ActivateSociety failed: %v", err)
	}
	if activated.Status != StatusActive {
		t.Errorf("expected status ACTIVE, got %s", activated.Status)
	}

	// 5. Invalid Transition: ACTIVE -> ACTIVE must return ErrAlreadyInStatus
	_, err = repo.ActivateSociety(ctx, soc.ID, "Redundant", actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrAlreadyInStatus.Error()) {
		t.Errorf("expected ErrAlreadyInStatus, got %v", err)
	}

	// 6. Suspension without reason code/ID must fail
	_, err = repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{}, actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrSuspensionReasonRequired.Error()) {
		t.Errorf("expected ErrSuspensionReasonRequired for empty reason, got %v", err)
	}

	// 7. Unknown reason code must fail
	badCode := "INVALID_CODE"
	_, err = repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{ReasonCode: &badCode}, actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrInvalidSuspensionReason.Error()) {
		t.Errorf("expected ErrInvalidSuspensionReason for unknown code, got %v", err)
	}

	// 8. Unknown reason ID must fail
	badID := int16(9999)
	_, err = repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{ReasonID: &badID}, actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrInvalidSuspensionReason.Error()) {
		t.Errorf("expected ErrInvalidSuspensionReason for unknown ID, got %v", err)
	}

	// 9. OTHER_REVIEW_REQUIRED without details must fail
	otherCode := "OTHER_REVIEW_REQUIRED"
	_, err = repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{ReasonCode: &otherCode}, actorID, "127.0.0.1", "TestRunner")
	if err == nil || !strings.Contains(err.Error(), ErrSuspensionDetailsRequired.Error()) {
		t.Errorf("expected ErrSuspensionDetailsRequired, got %v", err)
	}

	// 10. Valid Transition: ACTIVE -> SUSPENDED with mandatory reason by numeric ReasonID
	reasonID1 := int16(1) // PAYMENT_OVERDUE
	suspended, err := repo.SuspendSociety(ctx, soc.ID, SuspendSocietyParams{ReasonID: &reasonID1}, actorID, "127.0.0.1", "TestRunner")
	if err != nil {
		t.Fatalf("SuspendSociety failed: %v", err)
	}
	if suspended.Status != StatusSuspended {
		t.Errorf("expected status SUSPENDED, got %s", suspended.Status)
	}

	// 8. Valid Transition: SUSPENDED -> ACTIVE (Reactivate)
	reactivated, err := repo.ReactivateSociety(ctx, soc.ID, "Audit cleared", actorID, "127.0.0.1", "TestRunner")
	if err != nil {
		t.Fatalf("ReactivateSociety failed: %v", err)
	}
	if reactivated.Status != StatusActive {
		t.Errorf("expected status ACTIVE, got %s", reactivated.Status)
	}

	// 9. Profile update must preserve society code even if city changes
	newName := "Gokuldham CHS Ltd"
	newCity := "Pune" // changing city must NOT change society code
	updated, err := repo.UpdateSocietyProfile(ctx, soc.ID, UpdateSocietyParams{
		Name:         newName,
		AddressLine1: soc.AddressLine1,
		Locality:     soc.Locality,
		City:         newCity,
		State:        soc.State,
		Pincode:      soc.Pincode,
	}, actorID, "127.0.0.1", "TestRunner")
	if err != nil {
		t.Fatalf("UpdateSocietyProfile failed: %v", err)
	}
	if updated.SocietyCode != initialCode {
		t.Errorf("expected society code to remain %s, got %s", initialCode, updated.SocietyCode)
	}
	if updated.City != newCity {
		t.Errorf("expected city to update to %s, got %s", newCity, updated.City)
	}

	// 10. List & Pagination check
	items, total, err := repo.ListSocieties(ctx, ListSocietiesFilter{
		Search: "Gokuldham",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListSocieties failed: %v", err)
	}
	if total == 0 || len(items) == 0 {
		t.Errorf("expected to find at least 1 society matching 'Gokuldham', got %d", total)
	}
}
