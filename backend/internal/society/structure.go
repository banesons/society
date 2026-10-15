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
	"github.com/jackc/pgx/v5/pgconn"
)

// Structure audit actions
const (
	ActionBuildingCreated    = "BUILDING_CREATED"
	ActionBuildingUpdated    = "BUILDING_UPDATED"
	ActionWingCreated        = "WING_CREATED"
	ActionWingUpdated        = "WING_UPDATED"
	ActionUnitsGenerated     = "UNITS_GENERATED"
	ActionUnitUpdated        = "UNIT_UPDATED"
	ActionGateCreated        = "GATE_CREATED"
	ActionGateUpdated        = "GATE_UPDATED"
	ActionStaffingUpdated    = "STAFFING_REQUIREMENTS_UPDATED"
	ActionFormationCommitted = "SOCIETY_FORMATION_COMMITTED"
)

// Unit and Gate types
const (
	UnitTypeResidential = "RESIDENTIAL"
	UnitTypeCommercial  = "COMMERCIAL"
	UnitTypePenthouse   = "PENTHOUSE"
	UnitTypeParking     = "PARKING"
	UnitTypeOther       = "OTHER"

	GateTypeVehicleAndPedestrian = "VEHICLE_AND_PEDESTRIAN"
	GateTypeVehicleOnly          = "VEHICLE_ONLY"
	GateTypePedestrianOnly       = "PEDESTRIAN_ONLY"
	GateTypeService              = "SERVICE"
	GateTypeEmergency            = "EMERGENCY"
)

// Structure domain errors
var (
	ErrBuildingNotFound       = errors.New("building not found")
	ErrBuildingCodeExists     = errors.New("building code already exists in this society")
	ErrBuildingNameExists     = errors.New("building name already exists in this society")
	ErrWingNotFound           = errors.New("wing not found")
	ErrWingCodeExists         = errors.New("wing code already exists in this building")
	ErrWingNameExists         = errors.New("wing name already exists in this building")
	ErrUnitIdentifierExists   = errors.New("unit identifier already exists in this society")
	ErrGateCodeExists         = errors.New("gate code already exists in this society")
	ErrGateNameExists         = errors.New("gate name already exists in this society")
	ErrGateNotFound           = errors.New("gate not found")
	ErrUnitNotFound           = errors.New("unit not found")
	ErrInvalidFloorCount      = errors.New("floor count must be between 1 and 100")
	ErrInvalidUnitCount       = errors.New("units per floor must be between 1 and 50")
	ErrBulkLimitExceeded      = errors.New("cannot generate more than 1000 units in a single request")
	ErrParentBuildingMismatch = errors.New("referenced wing does not belong to the specified building")
	ErrInvalidGateType        = errors.New("invalid gate type")
	ErrInvalidShiftsPerDay    = errors.New("shifts per day must be between 1 and 4")
	ErrInvalidStaffCount      = errors.New("staff count cannot be negative")
)

// Building represents a physical building/tower.
type Building struct {
	ID           uuid.UUID `json:"id"`
	SocietyID    uuid.UUID `json:"society_id"`
	Name         string    `json:"name"`
	BuildingCode string    `json:"building_code"`
	TotalFloors  int       `json:"total_floors"`
	Wings        []Wing    `json:"wings,omitempty"`
	UnitCount    int       `json:"unit_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Wing represents an optional subdivision under a building.
type Wing struct {
	ID          uuid.UUID `json:"id"`
	SocietyID   uuid.UUID `json:"society_id"`
	BuildingID  uuid.UUID `json:"building_id"`
	Name        string    `json:"name"`
	WingCode    string    `json:"wing_code"`
	TotalFloors int       `json:"total_floors"`
	UnitCount   int       `json:"unit_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Unit represents a flat, apartment, or commercial space.
type Unit struct {
	ID             uuid.UUID  `json:"id"`
	SocietyID      uuid.UUID  `json:"society_id"`
	BuildingID     uuid.UUID  `json:"building_id"`
	WingID         *uuid.UUID `json:"wing_id,omitempty"`
	FloorNumber    int        `json:"floor_number"`
	UnitNumber     string     `json:"unit_number"`
	UnitIdentifier string     `json:"unit_identifier"`
	UnitType       string     `json:"unit_type"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Gate represents an entry/exit gate.
type Gate struct {
	ID                    uuid.UUID `json:"id"`
	SocietyID             uuid.UUID `json:"society_id"`
	Name                  string    `json:"name"`
	GateCode              string    `json:"gate_code"`
	GateType              string    `json:"gate_type"`
	Status                string    `json:"status"`
	Directionality        string    `json:"directionality"`
	OperationalStatus     string    `json:"operational_status"`
	RequiredStaffPerShift int       `json:"required_staff_per_shift"`
	Description           *string   `json:"description,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// StaffingRequirement represents staffing policy.
type StaffingRequirement struct {
	ID                     uuid.UUID  `json:"id"`
	SocietyID              uuid.UUID  `json:"society_id"`
	ShiftsPerDay           int        `json:"shifts_per_day"`
	TotalRequiredHeadcount int        `json:"total_required_headcount"`
	Notes                  *string    `json:"notes,omitempty"`
	UpdatedBy              *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// FormationSummary holds formation totals.
type FormationSummary struct {
	TotalBuildings         int `json:"total_buildings"`
	TotalWings             int `json:"total_wings"`
	TotalFloors            int `json:"total_floors"`
	TotalUnits             int `json:"total_units"`
	TotalGates             int `json:"total_gates"`
	RequiredStaffPerShift  int `json:"required_staff_per_shift"`
	TotalRequiredHeadcount int `json:"total_required_headcount"`
	RegisteredGatekeepers  int `json:"registered_gatekeepers"`
	AssignedGatekeepers    int `json:"assigned_gatekeepers"`
}

// GenerateUnitsParams contains inputs for unit generation.
type GenerateUnitsParams struct {
	BuildingID         uuid.UUID   `json:"building_id"`
	WingID             *uuid.UUID  `json:"wing_id,omitempty"`
	StartingFloor      int         `json:"starting_floor"`
	TotalFloors        int         `json:"total_floors"`
	UnitsPerFloor      int         `json:"units_per_floor"`
	StartingUnitNumber int         `json:"starting_unit_number"`
	Prefix             string      `json:"prefix,omitempty"`
	FloorOverrides     map[int]int `json:"floor_overrides,omitempty"`
	UnitType           string      `json:"unit_type,omitempty"`
}

// GeneratedUnitPreview represents a preview of a unit record.
type GeneratedUnitPreview struct {
	FloorNumber    int    `json:"floor_number"`
	UnitNumber     string `json:"unit_number"`
	UnitIdentifier string `json:"unit_identifier"`
	UnitType       string `json:"unit_type"`
}

// UnitPreviewResult contains the deterministic preview.
type UnitPreviewResult struct {
	TotalUnits int                    `json:"total_units"`
	Units      []GeneratedUnitPreview `json:"units"`
}

// GenerateUnitIdentifiers pure deterministic algorithm.
func GenerateUnitIdentifiers(params GenerateUnitsParams) (*UnitPreviewResult, error) {
	if params.TotalFloors <= 0 || params.TotalFloors > 100 {
		return nil, ErrInvalidFloorCount
	}
	if params.UnitsPerFloor <= 0 || params.UnitsPerFloor > 50 {
		return nil, ErrInvalidUnitCount
	}

	unitType := params.UnitType
	if unitType == "" {
		unitType = UnitTypeResidential
	}

	startFloor := params.StartingFloor
	if startFloor == 0 && params.TotalFloors > 0 && params.StartingUnitNumber >= 100 {
		// If user specified starting floor 0 with starting unit 101, normalize to 1 unless explicit
	}

	prefix := strings.TrimSpace(params.Prefix)

	var previews []GeneratedUnitPreview
	seenIdentifiers := make(map[string]bool)

	for f := 0; f < params.TotalFloors; f++ {
		floorNum := startFloor + f
		unitsOnFloor := params.UnitsPerFloor
		if override, ok := params.FloorOverrides[floorNum]; ok {
			if override < 0 || override > 50 {
				return nil, ErrInvalidUnitCount
			}
			unitsOnFloor = override
		}

		for u := 1; u <= unitsOnFloor; u++ {
			unitNumStr := fmt.Sprintf("%d%02d", floorNum, u)
			identifier := unitNumStr
			if prefix != "" {
				identifier = prefix + unitNumStr
			}

			if seenIdentifiers[identifier] {
				return nil, fmt.Errorf("duplicate generated unit identifier: %s", identifier)
			}
			seenIdentifiers[identifier] = true

			previews = append(previews, GeneratedUnitPreview{
				FloorNumber:    floorNum,
				UnitNumber:     unitNumStr,
				UnitIdentifier: identifier,
				UnitType:       unitType,
			})
		}
	}

	if len(previews) > 1000 {
		return nil, ErrBulkLimitExceeded
	}

	return &UnitPreviewResult{
		TotalUnits: len(previews),
		Units:      previews,
	}, nil
}

// CreateBuildingParams input for building creation.
type CreateBuildingParams struct {
	Name         string `json:"name"`
	BuildingCode string `json:"building_code"`
	TotalFloors  int    `json:"total_floors"`
}

// CreateBuilding creates a new building within the society.
func (r *Repository) CreateBuilding(ctx context.Context, societyID uuid.UUID, params CreateBuildingParams, actorID uuid.UUID, clientIP, userAgent string) (*Building, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, ErrRequiredFieldMissing
	}
	code := strings.ToUpper(strings.TrimSpace(params.BuildingCode))
	if code == "" {
		return nil, ErrRequiredFieldMissing
	}
	floors := params.TotalFloors
	if floors < 0 || floors > 100 {
		return nil, ErrInvalidFloorCount
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO society_buildings (society_id, name, building_code, total_floors)
		VALUES ($1, $2, $3, $4)
		RETURNING id, society_id, name, building_code, total_floors, created_at, updated_at
	`
	b := &Building{}
	err = tx.QueryRow(ctx, query, societyID, name, code, floors).Scan(
		&b.ID, &b.SocietyID, &b.Name, &b.BuildingCode, &b.TotalFloors, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "code") {
				return nil, ErrBuildingCodeExists
			}
			return nil, ErrBuildingNameExists
		}
		return nil, fmt.Errorf("failed to insert building: %w", err)
	}

	// Transactional audit log
	details := map[string]interface{}{
		"building_id":   b.ID.String(),
		"society_id":    societyID.String(),
		"name":          b.Name,
		"building_code": b.BuildingCode,
		"total_floors":  b.TotalFloors,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionBuildingCreated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit building creation: %w", err)
	}

	return b, nil
}

// ListBuildings lists all buildings and nested wings for a society.
func (r *Repository) ListBuildings(ctx context.Context, societyID uuid.UUID) ([]Building, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT b.id, b.society_id, b.name, b.building_code, b.total_floors, b.created_at, b.updated_at,
		       COUNT(u.id) as unit_count
		FROM society_buildings b
		LEFT JOIN society_units u ON b.id = u.building_id
		WHERE b.society_id = $1
		GROUP BY b.id
		ORDER BY b.name ASC
	`
	rows, err := r.pool.Query(ctx, query, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query buildings: %w", err)
	}
	defer rows.Close()

	buildings := make([]Building, 0)
	for rows.Next() {
		var b Building
		err := rows.Scan(&b.ID, &b.SocietyID, &b.Name, &b.BuildingCode, &b.TotalFloors, &b.CreatedAt, &b.UpdatedAt, &b.UnitCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan building: %w", err)
		}
		buildings = append(buildings, b)
	}

	// Fetch wings for each building
	for i := range buildings {
		wingsQuery := `
			SELECT w.id, w.society_id, w.building_id, w.name, w.wing_code, w.total_floors, w.created_at, w.updated_at,
			       COUNT(u.id) as unit_count
			FROM society_wings w
			LEFT JOIN society_units u ON w.id = u.wing_id
			WHERE w.building_id = $1
			GROUP BY w.id
			ORDER BY w.name ASC
		`
		wRows, err := r.pool.Query(ctx, wingsQuery, buildings[i].ID)
		if err == nil {
			wings := make([]Wing, 0)
			for wRows.Next() {
				var w Wing
				if err := wRows.Scan(&w.ID, &w.SocietyID, &w.BuildingID, &w.Name, &w.WingCode, &w.TotalFloors, &w.CreatedAt, &w.UpdatedAt, &w.UnitCount); err == nil {
					wings = append(wings, w)
				}
			}
			wRows.Close()
			buildings[i].Wings = wings
		}
	}

	return buildings, nil
}

// CreateWingParams input for wing creation.
type CreateWingParams struct {
	Name        string `json:"name"`
	WingCode    string `json:"wing_code"`
	TotalFloors int    `json:"total_floors"`
}

// CreateWing creates a new wing attached to a specific building.
func (r *Repository) CreateWing(ctx context.Context, societyID, buildingID uuid.UUID, params CreateWingParams, actorID uuid.UUID, clientIP, userAgent string) (*Wing, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	// Validate building belongs to this society
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_buildings WHERE id = $1 AND society_id = $2)`, buildingID, societyID).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrBuildingNotFound
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, ErrRequiredFieldMissing
	}
	code := strings.ToUpper(strings.TrimSpace(params.WingCode))
	if code == "" {
		return nil, ErrRequiredFieldMissing
	}
	floors := params.TotalFloors
	if floors < 0 || floors > 100 {
		return nil, ErrInvalidFloorCount
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO society_wings (society_id, building_id, name, wing_code, total_floors)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, society_id, building_id, name, wing_code, total_floors, created_at, updated_at
	`
	w := &Wing{}
	err = tx.QueryRow(ctx, query, societyID, buildingID, name, code, floors).Scan(
		&w.ID, &w.SocietyID, &w.BuildingID, &w.Name, &w.WingCode, &w.TotalFloors, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "code") {
				return nil, ErrWingCodeExists
			}
			return nil, ErrWingNameExists
		}
		return nil, fmt.Errorf("failed to insert wing: %w", err)
	}

	// Transactional audit log
	details := map[string]interface{}{
		"wing_id":      w.ID.String(),
		"building_id":  buildingID.String(),
		"society_id":   societyID.String(),
		"name":         w.Name,
		"wing_code":    w.WingCode,
		"total_floors": w.TotalFloors,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionWingCreated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit wing creation: %w", err)
	}

	return w, nil
}

// BulkGenerateUnits generates and saves units transactionally.
func (r *Repository) BulkGenerateUnits(ctx context.Context, societyID uuid.UUID, params GenerateUnitsParams, actorID uuid.UUID, clientIP, userAgent string) (int, error) {
	if r.pool == nil {
		return 0, errors.New("database pool unavailable")
	}

	// Verify building belongs to society
	var buildingExists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_buildings WHERE id = $1 AND society_id = $2)`, params.BuildingID, societyID).Scan(&buildingExists)
	if err != nil || !buildingExists {
		return 0, ErrBuildingNotFound
	}

	// Verify wing if supplied
	if params.WingID != nil {
		var wingValid bool
		err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM society_wings WHERE id = $1 AND building_id = $2 AND society_id = $3)`, *params.WingID, params.BuildingID, societyID).Scan(&wingValid)
		if err != nil || !wingValid {
			return 0, ErrParentBuildingMismatch
		}
	}

	preview, err := GenerateUnitIdentifiers(params)
	if err != nil {
		return 0, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	insertQuery := `
		INSERT INTO society_units (society_id, building_id, wing_id, floor_number, unit_number, unit_identifier, unit_type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE')
	`

	for _, u := range preview.Units {
		_, err := tx.Exec(ctx, insertQuery, societyID, params.BuildingID, params.WingID, u.FloorNumber, u.UnitNumber, u.UnitIdentifier, u.UnitType)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return 0, ErrUnitIdentifierExists
			}
			return 0, fmt.Errorf("failed to insert unit %s: %w", u.UnitIdentifier, err)
		}
	}

	// Transactional audit log
	details := map[string]interface{}{
		"society_id":   societyID.String(),
		"building_id":  params.BuildingID.String(),
		"wing_id":      params.WingID,
		"units_count":  preview.TotalUnits,
		"prefix":       params.Prefix,
		"total_floors": params.TotalFloors,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionUnitsGenerated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return 0, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit units generation: %w", err)
	}

	return preview.TotalUnits, nil
}

// ListUnitsFilter filter for listing society units.
type ListUnitsFilter struct {
	SocietyID  uuid.UUID
	BuildingID *uuid.UUID
	WingID     *uuid.UUID
	Limit      int
	Offset     int
}

// ListUnits returns paginated units for a society.
func (r *Repository) ListUnits(ctx context.Context, filter ListUnitsFilter) ([]Unit, int, error) {
	if r.pool == nil {
		return nil, 0, errors.New("database pool unavailable")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(*)
		FROM society_units
		WHERE society_id = $1
		  AND ($2::uuid IS NULL OR building_id = $2)
		  AND ($3::uuid IS NULL OR wing_id = $3)
	`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, filter.SocietyID, filter.BuildingID, filter.WingID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count units: %w", err)
	}

	query := `
		SELECT id, society_id, building_id, wing_id, floor_number, unit_number, unit_identifier, unit_type, status, created_at, updated_at
		FROM society_units
		WHERE society_id = $1
		  AND ($2::uuid IS NULL OR building_id = $2)
		  AND ($3::uuid IS NULL OR wing_id = $3)
		ORDER BY floor_number ASC, unit_number ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.pool.Query(ctx, query, filter.SocietyID, filter.BuildingID, filter.WingID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query units: %w", err)
	}
	defer rows.Close()

	items := make([]Unit, 0)
	for rows.Next() {
		var u Unit
		err := rows.Scan(
			&u.ID, &u.SocietyID, &u.BuildingID, &u.WingID, &u.FloorNumber,
			&u.UnitNumber, &u.UnitIdentifier, &u.UnitType, &u.Status,
			&u.CreatedAt, &u.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan unit: %w", err)
		}
		items = append(items, u)
	}

	return items, total, nil
}

// CreateGateParams input for gate creation.
type CreateGateParams struct {
	Name                  string  `json:"name"`
	GateCode              string  `json:"gate_code"`
	GateType              string  `json:"gate_type"`
	Status                string  `json:"status"`
	RequiredStaffPerShift int     `json:"required_staff_per_shift"`
	Description           *string `json:"description,omitempty"`
}

// CreateGate registers a physical gate.
func (r *Repository) CreateGate(ctx context.Context, societyID uuid.UUID, params CreateGateParams, actorID uuid.UUID, clientIP, userAgent string) (*Gate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, ErrRequiredFieldMissing
	}
	code := strings.ToUpper(strings.TrimSpace(params.GateCode))
	if code == "" {
		return nil, ErrRequiredFieldMissing
	}
	gateType := params.GateType
	if gateType == "" {
		gateType = GateTypeVehicleAndPedestrian
	}
	status := params.Status
	if status == "" {
		status = "ACTIVE"
	}
	if params.RequiredStaffPerShift < 0 {
		return nil, ErrInvalidStaffCount
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO society_gates (society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description)
		VALUES ($1, $2, $3, $4, $5, 'BIDIRECTIONAL', 'ACTIVE', $6, $7)
		RETURNING id, society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description, created_at, updated_at
	`
	g := &Gate{}
	err = tx.QueryRow(ctx, query, societyID, name, code, gateType, status, params.RequiredStaffPerShift, params.Description).Scan(
		&g.ID, &g.SocietyID, &g.Name, &g.GateCode, &g.GateType, &g.Status, &g.Directionality, &g.OperationalStatus, &g.RequiredStaffPerShift, &g.Description, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "code") {
				return nil, ErrGateCodeExists
			}
			return nil, ErrGateNameExists
		}
		return nil, fmt.Errorf("failed to insert gate: %w", err)
	}

	details := map[string]interface{}{
		"gate_id":                  g.ID.String(),
		"society_id":               societyID.String(),
		"name":                     g.Name,
		"gate_code":                g.GateCode,
		"gate_type":                g.GateType,
		"directionality":           g.Directionality,
		"operational_status":       g.OperationalStatus,
		"required_staff_per_shift": g.RequiredStaffPerShift,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionGateCreated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit gate creation: %w", err)
	}

	return g, nil
}

// ListGates retrieves all gates for a society.
func (r *Repository) ListGates(ctx context.Context, societyID uuid.UUID) ([]Gate, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_id, name, gate_code, gate_type, status, directionality, operational_status, required_staff_per_shift, description, created_at, updated_at
		FROM society_gates
		WHERE society_id = $1
		ORDER BY gate_code ASC
	`
	rows, err := r.pool.Query(ctx, query, societyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gates: %w", err)
	}
	defer rows.Close()

	items := make([]Gate, 0)
	for rows.Next() {
		var g Gate
		err := rows.Scan(
			&g.ID, &g.SocietyID, &g.Name, &g.GateCode, &g.GateType, &g.Status,
			&g.Directionality, &g.OperationalStatus,
			&g.RequiredStaffPerShift, &g.Description, &g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan gate: %w", err)
		}
		items = append(items, g)
	}

	return items, nil
}

// UpdateStaffingParams input for staffing configuration.
type UpdateStaffingParams struct {
	ShiftsPerDay           int     `json:"shifts_per_day"`
	TotalRequiredHeadcount int     `json:"total_required_headcount"`
	Notes                  *string `json:"notes,omitempty"`
}

// SetStaffingRequirement configures the society's gatekeeper staffing requirements.
func (r *Repository) SetStaffingRequirement(ctx context.Context, societyID uuid.UUID, params UpdateStaffingParams, actorID uuid.UUID, clientIP, userAgent string) (*StaffingRequirement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	if params.ShiftsPerDay < 1 || params.ShiftsPerDay > 4 {
		return nil, ErrInvalidShiftsPerDay
	}
	if params.TotalRequiredHeadcount < 0 {
		return nil, ErrInvalidStaffCount
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO society_staffing_requirements (society_id, shifts_per_day, total_required_headcount, notes, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (society_id) DO UPDATE SET
			shifts_per_day = EXCLUDED.shifts_per_day,
			total_required_headcount = EXCLUDED.total_required_headcount,
			notes = EXCLUDED.notes,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
		RETURNING id, society_id, shifts_per_day, total_required_headcount, notes, updated_by, created_at, updated_at
	`
	sr := &StaffingRequirement{}
	err = tx.QueryRow(ctx, query, societyID, params.ShiftsPerDay, params.TotalRequiredHeadcount, params.Notes, actorID).Scan(
		&sr.ID, &sr.SocietyID, &sr.ShiftsPerDay, &sr.TotalRequiredHeadcount, &sr.Notes, &sr.UpdatedBy, &sr.CreatedAt, &sr.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save staffing requirements: %w", err)
	}

	details := map[string]interface{}{
		"society_id":               societyID.String(),
		"shifts_per_day":           sr.ShiftsPerDay,
		"total_required_headcount": sr.TotalRequiredHeadcount,
	}
	detailsJSON, _ := json.Marshal(details)
	auditQuery := `
		INSERT INTO platform_audit_logs (actor_id, action, target_user_id, details, client_ip, user_agent)
		VALUES ($1, $2, NULL, $3::jsonb, $4, $5)
	`
	_, err = tx.Exec(ctx, auditQuery, actorID, ActionStaffingUpdated, string(detailsJSON), clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit staffing requirements: %w", err)
	}

	return sr, nil
}

// GetStaffingRequirement retrieves the staffing policy for a society.
func (r *Repository) GetStaffingRequirement(ctx context.Context, societyID uuid.UUID) (*StaffingRequirement, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	query := `
		SELECT id, society_id, shifts_per_day, total_required_headcount, notes, updated_by, created_at, updated_at
		FROM society_staffing_requirements
		WHERE society_id = $1
	`
	sr := &StaffingRequirement{}
	err := r.pool.QueryRow(ctx, query, societyID).Scan(
		&sr.ID, &sr.SocietyID, &sr.ShiftsPerDay, &sr.TotalRequiredHeadcount, &sr.Notes, &sr.UpdatedBy, &sr.CreatedAt, &sr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default empty policy
			return &StaffingRequirement{
				SocietyID:              societyID,
				ShiftsPerDay:           3,
				TotalRequiredHeadcount: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to query staffing requirements: %w", err)
	}

	return sr, nil
}

// GetFormationSummary calculates formation statistics dynamically.
func (r *Repository) GetFormationSummary(ctx context.Context, societyID uuid.UUID) (*FormationSummary, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	summary := &FormationSummary{}

	// Buildings & floors
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(total_floors), 0) FROM society_buildings WHERE society_id = $1`, societyID).Scan(&summary.TotalBuildings, &summary.TotalFloors)

	// Wings
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM society_wings WHERE society_id = $1`, societyID).Scan(&summary.TotalWings)

	// Units
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM society_units WHERE society_id = $1`, societyID).Scan(&summary.TotalUnits)

	// Gates & staff per shift
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(required_staff_per_shift), 0) FROM society_gates WHERE society_id = $1`, societyID).Scan(&summary.TotalGates, &summary.RequiredStaffPerShift)

	// Staffing policy
	_ = r.pool.QueryRow(ctx, `SELECT COALESCE(total_required_headcount, 0) FROM society_staffing_requirements WHERE society_id = $1`, societyID).Scan(&summary.TotalRequiredHeadcount)

	// Registered & assigned gatekeepers (currently 0, deferred to S08)
	summary.RegisteredGatekeepers = 0
	summary.AssignedGatekeepers = 0

	return summary, nil
}
