# ADR 017: Society Structure, Formation Workflow, and Unit Generation Model

## Status
Accepted (Sprint S05 Architecture Phase)

## Context
Sprint S03 established the Housing Society tenant boundary, lifecycle states (`PENDING`, `ACTIVE`, `SUSPENDED`), and canonical society codes (`SOC-{CITY}-{SUFFIX}`). Sprint S04 established administrative role mappings (`SOC_ADMIN`) and society-scoped authorization.

Sprint S05 introduces the physical structure of a housing society:
1. Societies contain physical buildings, optional wings, floors, and units/flats.
2. Entering hundreds of apartments manually is tedious and error-prone; a deterministic unit auto-generation engine is required.
3. Societies require gate configuration (vehicular/pedestrian gates) and gatekeeper staffing requirements.
4. Staffing requirements must be strictly decoupled from registered staff identity: no dummy or placeholder gatekeeper accounts may be fabricated.
5. Society formation must offer a guided multi-step wizard, but individual elements (buildings, wings, units, gates) must remain modular and editable without forcing society re-creation.

## Decision

### 1. Physical Structural Hierarchy
We adopt a normalized 3-tier hierarchy:
```text
Society
  └── Building (society_buildings)
       └── Wing (society_wings, optional)
            └── Unit (society_units)
```
- **Optional Wings**: A building may have multiple wings (e.g., Wing A, Wing B) or zero wings (direct units attached to the building with `wing_id = NULL`).
- **Floor Representation**: Floors are modeled as integer attributes (`floor_number` on `society_units` and `total_floors` count on buildings/wings). We explicitly reject a dedicated `society_floors` table to avoid redundant join tables that only store an integer count.

### 2. Deterministic Unit Generation Engine
We implement deterministic backend generation logic:
- Input parameters:
  - `building_id`, optional `wing_id`
  - `starting_floor` (integer, e.g., 1 or 0 for Ground)
  - `total_floors` (integer, 1..100)
  - `units_per_floor` (integer, 1..50)
  - `starting_unit_number` (integer pattern, e.g., 101)
  - `prefix` (optional, e.g., "A", "Tower-1")
  - `floor_overrides` (optional key-value map for non-uniform wings)
- **Numbering Algorithm**:
  - For floor $F$:
    $\text{Unit Sequence } u \in [1, \text{units\_per\_floor}]$
    $\text{Unit Number} = (F \times 100) + u$ (for 2-digit sequences)
    $\text{Identifier} = \text{Prefix} + \text{Unit Number}$ (e.g., Floor 9, Unit 3 with prefix "A" $\to$ `A903`).
- **Transactional Bulk Creation**: Both `/preview` and `/generate` endpoints share the identical pure generation function. Creating units runs in a single database transaction with safety limit capped at 1,000 units per request.

### 3. Unique Identifiers Scope
Unit identifiers (`unit_identifier`) must be unique across the entire society: `UNIQUE(society_id, unit_identifier)`. If two wings have unit 101, their identifiers must include wing/building qualification (e.g., `A-101` and `B-101`).

### 4. Gate Configuration & Staffing Model
- **`society_gates`**: Stores physical gates with `name`, `gate_code` (unique per society, e.g., `GATE-01`), `gate_type` (`VEHICLE_AND_PEDESTRIAN`, `VEHICLE_ONLY`, `PEDESTRIAN_ONLY`, `SERVICE`, `EMERGENCY`), `status` (`ACTIVE`, `INACTIVE`), and `required_staff_per_shift`.
- **`society_staffing_requirements`**: Stores overall staffing policy (shifts per day, total required headcount, notes).
- **Zero Fabrication**: Required staffing specifies headcount targets. Registered staff count is calculated dynamically against registered gatekeeper records (deferred to S08). No dummy user accounts are created.

### 5. Authorization & Audit Trail
- Supported callers: Platform `SUPER_ADMIN` and assigned active `SOC_ADMIN` of an `ACTIVE` society.
- All structural mutations record transactional audit logs in `platform_audit_logs`:
  - `BUILDING_CREATED`, `BUILDING_UPDATED`
  - `WING_CREATED`, `WING_UPDATED`
  - `UNITS_GENERATED`, `UNIT_UPDATED`
  - `GATE_CREATED`, `GATE_UPDATED`
  - `STAFFING_REQUIREMENTS_UPDATED`
  - `SOCIETY_FORMATION_COMMITTED`

## Consequences
- Flexible multi-building and multi-wing layout.
- Immediate deterministic generation of large residential societies without manual data entry.
- Honest, uncompromised staffing requirements model.
- Referential integrity preserved with `ON DELETE RESTRICT` preventing orphan units.

## Related Documentation
- [[../architecture/09-society-structure|09 — Society Structure Architecture]]
- [[../database/05-society-structure-schema|05 — Society Structure Database Schema]]
- [[../api/06-society-structure-api|06 — Society Structure API Specification]]
- [[../security/05-society-structure-security|05 — Society Structure Security Specification]]
- [[../sprints/S05-society-structure|S05 — Society Structure]]
