package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CanonicalUserProfile represents the universal, tenant-independent personal profile of an authenticated user.
type CanonicalUserProfile struct {
	UserID          uuid.UUID `json:"id"`
	Phone           string    `json:"phone"`
	PhoneRaw        string    `json:"phone_raw"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	DisplayName     string    `json:"display_name"`
	Email           *string   `json:"email"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateProfileParams specifies allowable modifications to a user's canonical profile.
type UpdateProfileParams struct {
	FirstName       string  `json:"first_name"`
	LastName        *string `json:"last_name"`
	Email           *string `json:"email"`
	ProfilePhotoURL *string `json:"profile_photo_url,omitempty"`
}

// ResolveDisplayName enforces a deterministic display-name hierarchy:
// 1. Canonical global profile first_name + last_name (if both non-empty)
// 2. Canonical global profile first_name (if last_name is empty)
// 3. Fallback derived from verified account identity (formatted phone: e.g. "+91 99670 89101")
// Never returns generic role placeholders like "Verified Resident" as personal identity.
func ResolveDisplayName(firstName, lastName, rawPhone string) string {
	first := strings.TrimSpace(firstName)
	last := strings.TrimSpace(lastName)
	if first != "" && last != "" {
		return first + " " + last
	}
	if first != "" {
		return first
	}
	if last != "" {
		return last
	}
	if rawPhone != "" {
		return FormatDisplayPhone(rawPhone)
	}
	return "Account"
}

// GetProfile retrieves the authenticated user's canonical profile.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*CanonicalUserProfile, error) {
	if s.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT u.id, u.phone, u.status, u.created_at,
		       COALESCE(up.first_name, '') as first_name,
		       COALESCE(up.last_name, '') as last_name,
		       up.email,
		       up.profile_photo_url,
		       COALESCE(up.updated_at, u.updated_at) as updated_at
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE u.id = $1
	`
	var p CanonicalUserProfile
	var rawPhone string
	err := s.pool.QueryRow(ctx, query, userID).Scan(
		&p.UserID,
		&rawPhone,
		&p.Status,
		&p.CreatedAt,
		&p.FirstName,
		&p.LastName,
		&p.Email,
		&p.ProfilePhotoURL,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query canonical user profile: %w", err)
	}

	p.PhoneRaw = rawPhone
	p.Phone = MaskPhone(rawPhone)
	p.DisplayName = ResolveDisplayName(p.FirstName, p.LastName, rawPhone)

	return &p, nil
}

// UpdateProfile validates and transactionally persists the authenticated user's canonical profile.
// Guaranteed not to modify verified phone, session limits, or tenant-scoped society_members records.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, params UpdateProfileParams) (*CanonicalUserProfile, error) {
	trimmedFirst := strings.TrimSpace(params.FirstName)
	if trimmedFirst == "" {
		return nil, errors.New("first name cannot be empty")
	}
	if len(trimmedFirst) > 100 {
		return nil, errors.New("first name exceeds maximum length of 100 characters")
	}

	var trimmedLast string
	if params.LastName != nil {
		trimmedLast = strings.TrimSpace(*params.LastName)
		if len(trimmedLast) > 100 {
			return nil, errors.New("last name exceeds maximum length of 100 characters")
		}
	}

	var trimmedEmail *string
	if params.Email != nil {
		e := strings.TrimSpace(strings.ToLower(*params.Email))
		if e != "" {
			if len(e) > 255 {
				return nil, errors.New("email exceeds maximum length of 255 characters")
			}
			if !strings.Contains(e, "@") || !strings.Contains(e, ".") {
				return nil, errors.New("invalid email address format")
			}
			trimmedEmail = &e
		}
	}

	if s.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Verify user exists and acquire lock
	var rawPhone, status string
	var createdAt time.Time
	userQuery := `SELECT phone, status, created_at FROM users WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRow(ctx, userQuery, userID).Scan(&rawPhone, &status, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to verify user: %w", err)
	}

	var updatedFirstName, updatedLastName string
	var updatedEmail, profilePhotoURL *string
	var updatedAt time.Time

	if params.ProfilePhotoURL != nil {
		var photoVal *string
		p := strings.TrimSpace(*params.ProfilePhotoURL)
		if p != "" {
			photoVal = &p
		}

		upsertQuery := `
			INSERT INTO user_profiles (user_id, first_name, last_name, email, profile_photo_url, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (user_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				email = EXCLUDED.email,
				profile_photo_url = EXCLUDED.profile_photo_url,
				updated_at = NOW()
			RETURNING first_name, last_name, email, profile_photo_url, updated_at
		`
		err = tx.QueryRow(ctx, upsertQuery, userID, trimmedFirst, trimmedLast, trimmedEmail, photoVal).Scan(
			&updatedFirstName,
			&updatedLastName,
			&updatedEmail,
			&profilePhotoURL,
			&updatedAt,
		)
	} else {
		upsertQuery := `
			INSERT INTO user_profiles (user_id, first_name, last_name, email, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (user_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				email = EXCLUDED.email,
				updated_at = NOW()
			RETURNING first_name, last_name, email, profile_photo_url, updated_at
		`
		err = tx.QueryRow(ctx, upsertQuery, userID, trimmedFirst, trimmedLast, trimmedEmail).Scan(
			&updatedFirstName,
			&updatedLastName,
			&updatedEmail,
			&profilePhotoURL,
			&updatedAt,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to persist user profile: %w", err)
	}

	// Update user timestamp
	_, err = tx.Exec(ctx, `UPDATE users SET updated_at = NOW() WHERE id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user timestamp: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit profile transaction: %w", err)
	}

	return &CanonicalUserProfile{
		UserID:          userID,
		Phone:           MaskPhone(rawPhone),
		PhoneRaw:        rawPhone,
		FirstName:       updatedFirstName,
		LastName:        updatedLastName,
		DisplayName:     ResolveDisplayName(updatedFirstName, updatedLastName, rawPhone),
		Email:           updatedEmail,
		ProfilePhotoURL: profilePhotoURL,
		Status:          status,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}
