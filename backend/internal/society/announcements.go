package society

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Announcement represents a society communication notice.
type Announcement struct {
	ID               uuid.UUID  `json:"id"`
	SocietyID        uuid.UUID  `json:"society_id"`
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	Category         string     `json:"category"`
	Priority         string     `json:"priority"`
	Status           string     `json:"status"`
	AudienceType     string     `json:"audience_type"`
	TargetBuildingID *uuid.UUID `json:"target_building_id,omitempty"`
	TargetWingID     *uuid.UUID `json:"target_wing_id,omitempty"`
	TargetUnitID     *uuid.UUID `json:"target_unit_id,omitempty"`
	TargetLabel      *string    `json:"target_label,omitempty"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	PublishedBy      *uuid.UUID `json:"published_by,omitempty"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	ArchivedAt       *time.Time `json:"archived_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Resident feed enrichment
	IsRead *bool      `json:"is_read,omitempty"`
	ReadAt *time.Time `json:"read_at,omitempty"`
}

// CreateAnnouncementParams holds input for creating a draft announcement.
type CreateAnnouncementParams struct {
	Title            string     `json:"title"`
	Body             string     `json:"body"`
	Category         string     `json:"category"`
	Priority         string     `json:"priority"`
	AudienceType     string     `json:"audience_type"`
	TargetBuildingID *uuid.UUID `json:"target_building_id,omitempty"`
	TargetWingID     *uuid.UUID `json:"target_wing_id,omitempty"`
	TargetUnitID     *uuid.UUID `json:"target_unit_id,omitempty"`
}

// UpdateAnnouncementParams holds fields for updating an announcement.
type UpdateAnnouncementParams struct {
	Title            *string    `json:"title,omitempty"`
	Body             *string    `json:"body,omitempty"`
	Category         *string    `json:"category,omitempty"`
	Priority         *string    `json:"priority,omitempty"`
	AudienceType     *string    `json:"audience_type,omitempty"`
	TargetBuildingID *uuid.UUID `json:"target_building_id,omitempty"`
	TargetWingID     *uuid.UUID `json:"target_wing_id,omitempty"`
	TargetUnitID     *uuid.UUID `json:"target_unit_id,omitempty"`
}

// ListAdminAnnouncementsFilter holds admin listing options.
type ListAdminAnnouncementsFilter struct {
	Status   string `json:"status,omitempty"`
	Category string `json:"category,omitempty"`
	Search   string `json:"search,omitempty"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// ListResidentFeedFilter holds resident feed query options.
type ListResidentFeedFilter struct {
	Category   string `json:"category,omitempty"`
	UnreadOnly bool   `json:"unread_only,omitempty"`
	Search     string `json:"search,omitempty"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// AnnouncementListResult wraps paginated results.
type AnnouncementListResult struct {
	Items  []Announcement `json:"items"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// validateCategory verifies allowed category values.
func validateCategory(cat string) bool {
	switch cat {
	case CategoryGeneral, CategoryMaintenance, CategorySecurity, CategoryEvent, CategoryEmergency, CategoryAdministrative:
		return true
	default:
		return false
	}
}

// validatePriority verifies allowed priority values.
func validatePriority(pri string) bool {
	switch pri {
	case PriorityNormal, PriorityUrgent, PriorityCritical:
		return true
	default:
		return false
	}
}

// validateAudienceAndTargets validates audience type and target references.
func (r *Repository) validateAudienceAndTargets(ctx context.Context, tx pgx.Tx, societyID uuid.UUID, audienceType string, buildingID, wingID, unitID *uuid.UUID) error {
	switch audienceType {
	case AudienceTypeSociety:
		if buildingID != nil || wingID != nil || unitID != nil {
			return ErrAnnouncementTargetMismatch
		}
		return nil

	case AudienceTypeBuilding:
		if buildingID == nil || wingID != nil || unitID != nil {
			return ErrAnnouncementTargetMismatch
		}
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM society_buildings WHERE id = $1 AND society_id = $2)", *buildingID, societyID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to verify target building: %w", err)
		}
		if !exists {
			return ErrAnnouncementCrossSocietyTarget
		}
		return nil

	case AudienceTypeWing:
		if wingID == nil || buildingID != nil || unitID != nil {
			return ErrAnnouncementTargetMismatch
		}
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM society_wings WHERE id = $1 AND society_id = $2)", *wingID, societyID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to verify target wing: %w", err)
		}
		if !exists {
			return ErrAnnouncementCrossSocietyTarget
		}
		return nil

	case AudienceTypeUnit:
		if unitID == nil || buildingID != nil || wingID != nil {
			return ErrAnnouncementTargetMismatch
		}
		var exists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM society_units WHERE id = $1 AND society_id = $2)", *unitID, societyID).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to verify target unit: %w", err)
		}
		if !exists {
			return ErrAnnouncementCrossSocietyTarget
		}
		return nil

	default:
		return ErrAnnouncementTargetMismatch
	}
}

// CreateAnnouncementDraft creates a new draft announcement with transactional audit logging.
func (r *Repository) CreateAnnouncementDraft(ctx context.Context, societyID uuid.UUID, params CreateAnnouncementParams, actorID uuid.UUID, clientIP, userAgent string) (*Announcement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	trimmedTitle := strings.TrimSpace(params.Title)
	if trimmedTitle == "" {
		return nil, ErrAnnouncementTitleRequired
	}
	if len(trimmedTitle) > 200 {
		return nil, errors.New("announcement title must not exceed 200 characters")
	}

	trimmedBody := strings.TrimSpace(params.Body)
	if trimmedBody == "" {
		return nil, ErrAnnouncementBodyRequired
	}

	category := strings.TrimSpace(params.Category)
	if category == "" {
		category = CategoryGeneral
	}
	if !validateCategory(category) {
		return nil, errors.New("invalid announcement category")
	}

	priority := strings.TrimSpace(params.Priority)
	if priority == "" {
		priority = PriorityNormal
	}
	if !validatePriority(priority) {
		return nil, errors.New("invalid announcement priority")
	}

	audienceType := strings.TrimSpace(params.AudienceType)
	if audienceType == "" {
		audienceType = AudienceTypeSociety
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := r.validateAudienceAndTargets(ctx, tx, societyID, audienceType, params.TargetBuildingID, params.TargetWingID, params.TargetUnitID); err != nil {
		return nil, err
	}

	var a Announcement
	query := `
		INSERT INTO society_announcements (
			society_id, title, body, category, priority, status, audience_type,
			target_building_id, target_wing_id, target_unit_id,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, 'DRAFT', $6,
			$7, $8, $9,
			$10, NOW(), NOW()
		)
		RETURNING id, society_id, title, body, category, priority, status, audience_type,
		          target_building_id, target_wing_id, target_unit_id,
		          created_by, published_by, published_at, archived_at, created_at, updated_at
	`
	err = tx.QueryRow(ctx, query,
		societyID, trimmedTitle, trimmedBody, category, priority, audienceType,
		params.TargetBuildingID, params.TargetWingID, params.TargetUnitID,
		actorID,
	).Scan(
		&a.ID, &a.SocietyID, &a.Title, &a.Body, &a.Category, &a.Priority, &a.Status, &a.AudienceType,
		&a.TargetBuildingID, &a.TargetWingID, &a.TargetUnitID,
		&a.CreatedBy, &a.PublishedBy, &a.PublishedAt, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert announcement draft: %w", err)
	}

	// Transactional audit log
	auditDetails := map[string]interface{}{
		"announcement_id": a.ID.String(),
		"society_id":      societyID.String(),
		"title":           a.Title,
		"category":        a.Category,
		"priority":        a.Priority,
		"audience_type":   a.AudienceType,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionAnnouncementCreated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.populateTargetLabel(ctx, &a)
	return &a, nil
}

// UpdateAnnouncement modifies an announcement with lifecycle checks and audit logging.
func (r *Repository) UpdateAnnouncement(ctx context.Context, societyID, announcementID uuid.UUID, params UpdateAnnouncementParams, actorID uuid.UUID, clientIP, userAgent string) (*Announcement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current Announcement
	queryExisting := `
		SELECT id, society_id, title, body, category, priority, status, audience_type,
		       target_building_id, target_wing_id, target_unit_id,
		       created_by, published_by, published_at, archived_at, created_at, updated_at
		FROM society_announcements
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, queryExisting, announcementID, societyID).Scan(
		&current.ID, &current.SocietyID, &current.Title, &current.Body, &current.Category, &current.Priority, &current.Status, &current.AudienceType,
		&current.TargetBuildingID, &current.TargetWingID, &current.TargetUnitID,
		&current.CreatedBy, &current.PublishedBy, &current.PublishedAt, &current.ArchivedAt, &current.CreatedAt, &current.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("failed to query announcement for update: %w", err)
	}

	if current.Status == AnnouncementStatusArchived {
		return nil, ErrAnnouncementInvalidLifecycle
	}

	newTitle := current.Title
	if params.Title != nil {
		t := strings.TrimSpace(*params.Title)
		if t == "" {
			return nil, ErrAnnouncementTitleRequired
		}
		if len(t) > 200 {
			return nil, errors.New("announcement title must not exceed 200 characters")
		}
		newTitle = t
	}

	newBody := current.Body
	if params.Body != nil {
		b := strings.TrimSpace(*params.Body)
		if b == "" {
			return nil, ErrAnnouncementBodyRequired
		}
		newBody = b
	}

	newCategory := current.Category
	if params.Category != nil {
		c := strings.TrimSpace(*params.Category)
		if !validateCategory(c) {
			return nil, errors.New("invalid announcement category")
		}
		newCategory = c
	}

	newPriority := current.Priority
	if params.Priority != nil {
		p := strings.TrimSpace(*params.Priority)
		if !validatePriority(p) {
			return nil, errors.New("invalid announcement priority")
		}
		newPriority = p
	}

	newAudienceType := current.AudienceType
	newBuildingID := current.TargetBuildingID
	newWingID := current.TargetWingID
	newUnitID := current.TargetUnitID

	// Published announcements cannot modify audience targeting (immutable once published)
	if current.Status == AnnouncementStatusPublished {
		if (params.AudienceType != nil && *params.AudienceType != current.AudienceType) ||
			params.TargetBuildingID != nil || params.TargetWingID != nil || params.TargetUnitID != nil {
			return nil, errors.New("audience targeting cannot be modified on a published announcement")
		}
	} else {
		// DRAFT announcements can modify targeting
		if params.AudienceType != nil {
			newAudienceType = strings.TrimSpace(*params.AudienceType)
		}
		if params.TargetBuildingID != nil {
			newBuildingID = params.TargetBuildingID
		}
		if params.TargetWingID != nil {
			newWingID = params.TargetWingID
		}
		if params.TargetUnitID != nil {
			newUnitID = params.TargetUnitID
		}

		if newAudienceType == AudienceTypeSociety {
			newBuildingID = nil
			newWingID = nil
			newUnitID = nil
		}

		if err := r.validateAudienceAndTargets(ctx, tx, societyID, newAudienceType, newBuildingID, newWingID, newUnitID); err != nil {
			return nil, err
		}
	}

	var updated Announcement
	updateQuery := `
		UPDATE society_announcements
		SET title = $1, body = $2, category = $3, priority = $4,
		    audience_type = $5, target_building_id = $6, target_wing_id = $7, target_unit_id = $8,
		    updated_at = NOW()
		WHERE id = $9 AND society_id = $10
		RETURNING id, society_id, title, body, category, priority, status, audience_type,
		          target_building_id, target_wing_id, target_unit_id,
		          created_by, published_by, published_at, archived_at, created_at, updated_at
	`
	err = tx.QueryRow(ctx, updateQuery,
		newTitle, newBody, newCategory, newPriority,
		newAudienceType, newBuildingID, newWingID, newUnitID,
		announcementID, societyID,
	).Scan(
		&updated.ID, &updated.SocietyID, &updated.Title, &updated.Body, &updated.Category, &updated.Priority, &updated.Status, &updated.AudienceType,
		&updated.TargetBuildingID, &updated.TargetWingID, &updated.TargetUnitID,
		&updated.CreatedBy, &updated.PublishedBy, &updated.PublishedAt, &updated.ArchivedAt, &updated.CreatedAt, &updated.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update announcement: %w", err)
	}

	// Transactional audit log
	auditDetails := map[string]interface{}{
		"announcement_id": updated.ID.String(),
		"society_id":      societyID.String(),
		"title":           updated.Title,
		"category":        updated.Category,
		"priority":        updated.Priority,
		"audience_type":   updated.AudienceType,
		"status":          updated.Status,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionAnnouncementUpdated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.populateTargetLabel(ctx, &updated)
	return &updated, nil
}

// PublishAnnouncement transitions an announcement from DRAFT to PUBLISHED with audit logging.
func (r *Repository) PublishAnnouncement(ctx context.Context, societyID, announcementID uuid.UUID, actorID uuid.UUID, clientIP, userAgent string) (*Announcement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current Announcement
	queryExisting := `
		SELECT id, society_id, status, title, category, priority, audience_type
		FROM society_announcements
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, queryExisting, announcementID, societyID).Scan(
		&current.ID, &current.SocietyID, &current.Status, &current.Title, &current.Category, &current.Priority, &current.AudienceType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("failed to query announcement for publishing: %w", err)
	}

	if current.Status != AnnouncementStatusDraft {
		return nil, ErrAnnouncementInvalidLifecycle
	}

	var published Announcement
	publishQuery := `
		UPDATE society_announcements
		SET status = 'PUBLISHED', published_by = $1, published_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND society_id = $3
		RETURNING id, society_id, title, body, category, priority, status, audience_type,
		          target_building_id, target_wing_id, target_unit_id,
		          created_by, published_by, published_at, archived_at, created_at, updated_at
	`
	err = tx.QueryRow(ctx, publishQuery, actorID, announcementID, societyID).Scan(
		&published.ID, &published.SocietyID, &published.Title, &published.Body, &published.Category, &published.Priority, &published.Status, &published.AudienceType,
		&published.TargetBuildingID, &published.TargetWingID, &published.TargetUnitID,
		&published.CreatedBy, &published.PublishedBy, &published.PublishedAt, &published.ArchivedAt, &published.CreatedAt, &published.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish announcement: %w", err)
	}

	// Transactional audit log
	auditDetails := map[string]interface{}{
		"announcement_id": published.ID.String(),
		"society_id":      societyID.String(),
		"title":           published.Title,
		"category":        published.Category,
		"priority":        published.Priority,
		"audience_type":   published.AudienceType,
		"published_at":    published.PublishedAt,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionAnnouncementPublished, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	// Enqueue notification outbox events for eligible announcement recipients
	recipRows, qErr := tx.Query(ctx, `
		SELECT DISTINCT sm.user_id
		FROM society_members sm
		WHERE sm.society_id = $1 AND sm.status = 'ACTIVE'
		  AND (
		      $2 = 'SOCIETY'
		      OR (
		          $2 = 'BUILDING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE ur.member_id = sm.id AND ur.status = 'ACTIVE' AND su.building_id = $3
		          )
		      )
		      OR (
		          $2 = 'WING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE ur.member_id = sm.id AND ur.status = 'ACTIVE' AND su.wing_id = $4
		          )
		      )
		      OR (
		          $2 = 'UNIT' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              WHERE ur.member_id = sm.id AND ur.status = 'ACTIVE' AND ur.unit_id = $5
		          )
		      )
		  )
	`, societyID, published.AudienceType, published.TargetBuildingID, published.TargetWingID, published.TargetUnitID)
	if qErr == nil {
		for recipRows.Next() {
			var recipUID uuid.UUID
			if err := recipRows.Scan(&recipUID); err == nil {
				outboxPayload := map[string]interface{}{
					"recipient_user_id": recipUID.String(),
					"society_id":        societyID.String(),
					"announcement_id":   published.ID.String(),
					"title":             published.Title,
					"priority":          published.Priority,
				}
				_ = r.EnqueueNotificationOutbox(ctx, tx, "ANNOUNCEMENT_PUBLISHED", fmt.Sprintf("ANN_PUB:%s:%s", published.ID.String(), recipUID.String()), outboxPayload)
			}
		}
		recipRows.Close()
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.populateTargetLabel(ctx, &published)
	return &published, nil
}

// ArchiveAnnouncement transitions an announcement from PUBLISHED to ARCHIVED with audit logging.
func (r *Repository) ArchiveAnnouncement(ctx context.Context, societyID, announcementID uuid.UUID, actorID uuid.UUID, clientIP, userAgent string) (*Announcement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current Announcement
	queryExisting := `
		SELECT id, society_id, status, title, category, priority, audience_type
		FROM society_announcements
		WHERE id = $1 AND society_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, queryExisting, announcementID, societyID).Scan(
		&current.ID, &current.SocietyID, &current.Status, &current.Title, &current.Category, &current.Priority, &current.AudienceType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("failed to query announcement for archiving: %w", err)
	}

	if current.Status != AnnouncementStatusPublished {
		return nil, ErrAnnouncementInvalidLifecycle
	}

	var archived Announcement
	archiveQuery := `
		UPDATE society_announcements
		SET status = 'ARCHIVED', archived_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND society_id = $2
		RETURNING id, society_id, title, body, category, priority, status, audience_type,
		          target_building_id, target_wing_id, target_unit_id,
		          created_by, published_by, published_at, archived_at, created_at, updated_at
	`
	err = tx.QueryRow(ctx, archiveQuery, announcementID, societyID).Scan(
		&archived.ID, &archived.SocietyID, &archived.Title, &archived.Body, &archived.Category, &archived.Priority, &archived.Status, &archived.AudienceType,
		&archived.TargetBuildingID, &archived.TargetWingID, &archived.TargetUnitID,
		&archived.CreatedBy, &archived.PublishedBy, &archived.PublishedAt, &archived.ArchivedAt, &archived.CreatedAt, &archived.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to archive announcement: %w", err)
	}

	// Transactional audit log
	auditDetails := map[string]interface{}{
		"announcement_id": archived.ID.String(),
		"society_id":      societyID.String(),
		"title":           archived.Title,
		"category":        archived.Category,
		"priority":        archived.Priority,
		"audience_type":   archived.AudienceType,
		"archived_at":     archived.ArchivedAt,
	}
	detailsJSON, err := json.Marshal(auditDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionAnnouncementArchived, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to record audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.populateTargetLabel(ctx, &archived)
	return &archived, nil
}

// GetAnnouncementByID fetches an announcement by ID, applying audience isolation for non-administrators.
func (r *Repository) GetAnnouncementByID(ctx context.Context, societyID, announcementID uuid.UUID, userID uuid.UUID, isPrivileged bool) (*Announcement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if isPrivileged {
		query := `
			SELECT a.id, a.society_id, a.title, a.body, a.category, a.priority, a.status, a.audience_type,
			       a.target_building_id, a.target_wing_id, a.target_unit_id,
			       a.created_by, a.published_by, a.published_at, a.archived_at, a.created_at, a.updated_at,
			       COALESCE(b.name, w.wing_code, u.unit_identifier) as target_label
			FROM society_announcements a
			LEFT JOIN society_buildings b ON a.target_building_id = b.id
			LEFT JOIN society_wings w ON a.target_wing_id = w.id
			LEFT JOIN society_units u ON a.target_unit_id = u.id
			WHERE a.id = $1 AND a.society_id = $2
		`
		var a Announcement
		err := r.pool.QueryRow(ctx, query, announcementID, societyID).Scan(
			&a.ID, &a.SocietyID, &a.Title, &a.Body, &a.Category, &a.Priority, &a.Status, &a.AudienceType,
			&a.TargetBuildingID, &a.TargetWingID, &a.TargetUnitID,
			&a.CreatedBy, &a.PublishedBy, &a.PublishedAt, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
			&a.TargetLabel,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrAnnouncementNotFound
			}
			return nil, fmt.Errorf("failed to get announcement: %w", err)
		}
		return &a, nil
	}

	// Resident query: must be PUBLISHED and match audience resolution rules
	query := `
		SELECT a.id, a.society_id, a.title, a.body, a.category, a.priority, a.status, a.audience_type,
		       a.target_building_id, a.target_wing_id, a.target_unit_id,
		       a.created_by, a.published_by, a.published_at, a.archived_at, a.created_at, a.updated_at,
		       COALESCE(b.name, w.wing_code, u.unit_identifier) as target_label,
		       (r.id IS NOT NULL) as is_read,
		       r.read_at
		FROM society_announcements a
		LEFT JOIN society_announcement_reads r ON r.announcement_id = a.id AND r.user_id = $3
		LEFT JOIN society_buildings b ON a.target_building_id = b.id
		LEFT JOIN society_wings w ON a.target_wing_id = w.id
		LEFT JOIN society_units u ON a.target_unit_id = u.id
		WHERE a.id = $1 
		  AND a.society_id = $2
		  AND a.status = 'PUBLISHED'
		  AND (
		      EXISTS (
		          SELECT 1 FROM society_user_roles sur
		          WHERE sur.society_id = a.society_id AND sur.user_id = $3 AND sur.status = 'ACTIVE' AND sur.role IN ('SOC_ADMIN', 'SUPER_ADMIN')
		      )
		      OR a.audience_type = 'SOCIETY'
		      OR (
		          a.audience_type = 'BUILDING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE sm.user_id = $3 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.building_id = a.target_building_id
		          )
		      )
		      OR (
		          a.audience_type = 'WING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE sm.user_id = $3 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.wing_id = a.target_wing_id
		          )
		      )
		      OR (
		          a.audience_type = 'UNIT' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              WHERE sm.user_id = $3 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND ur.unit_id = a.target_unit_id
		          )
		      )
		  )
	`
	var a Announcement
	var isRead bool
	var readAt *time.Time
	err := r.pool.QueryRow(ctx, query, announcementID, societyID, userID).Scan(
		&a.ID, &a.SocietyID, &a.Title, &a.Body, &a.Category, &a.Priority, &a.Status, &a.AudienceType,
		&a.TargetBuildingID, &a.TargetWingID, &a.TargetUnitID,
		&a.CreatedBy, &a.PublishedBy, &a.PublishedAt, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
		&a.TargetLabel,
		&isRead,
		&readAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnnouncementNotFound
		}
		return nil, fmt.Errorf("failed to get resident announcement: %w", err)
	}
	a.IsRead = &isRead
	a.ReadAt = readAt
	return &a, nil
}

// ListAdminAnnouncements lists announcements for society administrators with filtering and pagination.
func (r *Repository) ListAdminAnnouncements(ctx context.Context, societyID uuid.UUID, filter ListAdminAnnouncementsFilter) (*AnnouncementListResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	whereClauses := []string{"a.society_id = $1"}
	args := []interface{}{societyID}
	argIdx := 2

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.category = $%d", argIdx))
		args = append(args, filter.Category)
		argIdx++
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(a.title ILIKE $%d OR a.body ILIKE $%d)", argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM society_announcements a WHERE %s", whereSQL)
	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count admin announcements: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.society_id, a.title, a.body, a.category, a.priority, a.status, a.audience_type,
		       a.target_building_id, a.target_wing_id, a.target_unit_id,
		       a.created_by, a.published_by, a.published_at, a.archived_at, a.created_at, a.updated_at,
		       COALESCE(b.name, w.wing_code, u.unit_identifier) as target_label
		FROM society_announcements a
		LEFT JOIN society_buildings b ON a.target_building_id = b.id
		LEFT JOIN society_wings w ON a.target_wing_id = w.id
		LEFT JOIN society_units u ON a.target_unit_id = u.id
		WHERE %s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list admin announcements: %w", err)
	}
	defer rows.Close()

	items := make([]Announcement, 0)
	for rows.Next() {
		var a Announcement
		err := rows.Scan(
			&a.ID, &a.SocietyID, &a.Title, &a.Body, &a.Category, &a.Priority, &a.Status, &a.AudienceType,
			&a.TargetBuildingID, &a.TargetWingID, &a.TargetUnitID,
			&a.CreatedBy, &a.PublishedBy, &a.PublishedAt, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
			&a.TargetLabel,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin announcement: %w", err)
		}
		items = append(items, a)
	}

	return &AnnouncementListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ListResidentAnnouncementFeed returns published announcements matching the user's audience scope.
func (r *Repository) ListResidentAnnouncementFeed(ctx context.Context, societyID uuid.UUID, userID uuid.UUID, filter ListResidentFeedFilter) (*AnnouncementListResult, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	audienceClause := `
		(
		    EXISTS (
		        SELECT 1 FROM society_user_roles sur
		        WHERE sur.society_id = a.society_id AND sur.user_id = $2 AND sur.status = 'ACTIVE' AND sur.role IN ('SOC_ADMIN', 'SUPER_ADMIN')
		    )
		    OR a.audience_type = 'SOCIETY'
		    OR (
		        a.audience_type = 'BUILDING' AND EXISTS (
		            SELECT 1 FROM unit_relationships ur
		            JOIN society_members sm ON ur.member_id = sm.id
		            JOIN society_units su ON ur.unit_id = su.id
		            WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.building_id = a.target_building_id
		        )
		    )
		    OR (
		        a.audience_type = 'WING' AND EXISTS (
		            SELECT 1 FROM unit_relationships ur
		            JOIN society_members sm ON ur.member_id = sm.id
		            JOIN society_units su ON ur.unit_id = su.id
		            WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.wing_id = a.target_wing_id
		        )
		    )
		    OR (
		        a.audience_type = 'UNIT' AND EXISTS (
		            SELECT 1 FROM unit_relationships ur
		            JOIN society_members sm ON ur.member_id = sm.id
		            WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND ur.unit_id = a.target_unit_id
		        )
		    )
		)
	`

	whereClauses := []string{
		"a.society_id = $1",
		"a.status = 'PUBLISHED'",
		audienceClause,
	}
	args := []interface{}{societyID, userID}
	argIdx := 3

	if filter.Category != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.category = $%d", argIdx))
		args = append(args, filter.Category)
		argIdx++
	}

	if filter.UnreadOnly {
		whereClauses = append(whereClauses, "r.id IS NULL")
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.TrimSpace(filter.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(a.title ILIKE $%d OR a.body ILIKE $%d)", argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM society_announcements a
		LEFT JOIN society_announcement_reads r ON r.announcement_id = a.id AND r.user_id = $2
		WHERE %s
	`, whereSQL)

	var total int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count resident feed: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.society_id, a.title, a.body, a.category, a.priority, a.status, a.audience_type,
		       a.target_building_id, a.target_wing_id, a.target_unit_id,
		       a.created_by, a.published_by, a.published_at, a.archived_at, a.created_at, a.updated_at,
		       COALESCE(b.name, w.wing_code, u.unit_identifier) as target_label,
		       (r.id IS NOT NULL) as is_read,
		       r.read_at
		FROM society_announcements a
		LEFT JOIN society_announcement_reads r ON r.announcement_id = a.id AND r.user_id = $2
		LEFT JOIN society_buildings b ON a.target_building_id = b.id
		LEFT JOIN society_wings w ON a.target_wing_id = w.id
		LEFT JOIN society_units u ON a.target_unit_id = u.id
		WHERE %s
		ORDER BY a.published_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list resident feed: %w", err)
	}
	defer rows.Close()

	items := make([]Announcement, 0)
	for rows.Next() {
		var a Announcement
		var isRead bool
		var readAt *time.Time
		err := rows.Scan(
			&a.ID, &a.SocietyID, &a.Title, &a.Body, &a.Category, &a.Priority, &a.Status, &a.AudienceType,
			&a.TargetBuildingID, &a.TargetWingID, &a.TargetUnitID,
			&a.CreatedBy, &a.PublishedBy, &a.PublishedAt, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
			&a.TargetLabel,
			&isRead,
			&readAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resident announcement: %w", err)
		}
		a.IsRead = &isRead
		a.ReadAt = readAt
		items = append(items, a)
	}

	return &AnnouncementListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// MarkAnnouncementRead records an idempotent read receipt for the authenticated user.
func (r *Repository) MarkAnnouncementRead(ctx context.Context, societyID, announcementID uuid.UUID, userID uuid.UUID) error {
	if r.pool == nil {
		return errors.New("database pool unavailable")
	}

	// Verify announcement exists, is PUBLISHED, and is visible to this user
	_, err := r.GetAnnouncementByID(ctx, societyID, announcementID, userID, false)
	if err != nil {
		if errors.Is(err, ErrAnnouncementNotFound) {
			return ErrAnnouncementAccessDenied
		}
		return err
	}

	query := `
		INSERT INTO society_announcement_reads (society_id, announcement_id, user_id, read_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (announcement_id, user_id) DO NOTHING
	`
	_, err = r.pool.Exec(ctx, query, societyID, announcementID, userID)
	if err != nil {
		return fmt.Errorf("failed to record announcement read receipt: %w", err)
	}

	return nil
}

// GetUnreadAnnouncementCount returns the count of unread published announcements for a user.
func (r *Repository) GetUnreadAnnouncementCount(ctx context.Context, societyID uuid.UUID, userID uuid.UUID) (int, error) {
	if r.pool == nil {
		return 0, errors.New("database pool unavailable")
	}

	query := `
		SELECT COUNT(*)
		FROM society_announcements a
		LEFT JOIN society_announcement_reads r ON r.announcement_id = a.id AND r.user_id = $2
		WHERE a.society_id = $1
		  AND a.status = 'PUBLISHED'
		  AND r.id IS NULL
		  AND (
		      EXISTS (
		          SELECT 1 FROM society_user_roles sur
		          WHERE sur.society_id = a.society_id AND sur.user_id = $2 AND sur.status = 'ACTIVE' AND sur.role IN ('SOC_ADMIN', 'SUPER_ADMIN')
		      )
		      OR a.audience_type = 'SOCIETY'
		      OR (
		          a.audience_type = 'BUILDING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.building_id = a.target_building_id
		          )
		      )
		      OR (
		          a.audience_type = 'WING' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              JOIN society_units su ON ur.unit_id = su.id
		              WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND su.wing_id = a.target_wing_id
		          )
		      )
		      OR (
		          a.audience_type = 'UNIT' AND EXISTS (
		              SELECT 1 FROM unit_relationships ur
		              JOIN society_members sm ON ur.member_id = sm.id
		              WHERE sm.user_id = $2 AND sm.society_id = a.society_id AND sm.status = 'ACTIVE' AND ur.status = 'ACTIVE' AND ur.unit_id = a.target_unit_id
		          )
		      )
		  )
	`
	var count int
	err := r.pool.QueryRow(ctx, query, societyID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread announcements: %w", err)
	}

	return count, nil
}

// populateTargetLabel fills the display label for an announcement target.
func (r *Repository) populateTargetLabel(ctx context.Context, a *Announcement) {
	if a == nil || r.pool == nil {
		return
	}
	switch a.AudienceType {
	case AudienceTypeBuilding:
		if a.TargetBuildingID != nil {
			var name string
			if err := r.pool.QueryRow(ctx, "SELECT name FROM society_buildings WHERE id = $1", *a.TargetBuildingID).Scan(&name); err == nil {
				a.TargetLabel = &name
			}
		}
	case AudienceTypeWing:
		if a.TargetWingID != nil {
			var code string
			if err := r.pool.QueryRow(ctx, "SELECT wing_code FROM society_wings WHERE id = $1", *a.TargetWingID).Scan(&code); err == nil {
				a.TargetLabel = &code
			}
		}
	case AudienceTypeUnit:
		if a.TargetUnitID != nil {
			var ident string
			if err := r.pool.QueryRow(ctx, "SELECT unit_identifier FROM society_units WHERE id = $1", *a.TargetUnitID).Scan(&ident); err == nil {
				a.TargetLabel = &ident
			}
		}
	}
}
