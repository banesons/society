# 09 — Society Structure Architecture

## Overview

Sprint S05 provides physical modeling of housing societies, allowing authorized administrators (`SUPER_ADMIN` and `SOC_ADMIN`) to define buildings, optional wings, unit generation patterns, gate configurations, and staffing requirements.

---

## 1. Domain Hierarchy

```text
Housing Society (societies)
 │
 ├── Buildings (society_buildings)
 │    └── Wings (society_wings, optional)
 │         └── Units (society_units)
 │
 ├── Gates (society_gates)
 │
 └── Staffing Policy (society_staffing_requirements)
```

### Hierarchy Rules
1. **Buildings**: Every unit must belong to a building.
2. **Wings**: Wings are optional children of a building. When a building has wings, units reference both `building_id` and `wing_id`. When a building has no wings, `wing_id` is `NULL`.
3. **Floors**: Floors are numeric values stored directly on unit records (`floor_number`).
4. **Units**: Leaf node representing a flat, apartment, penthouse, shop, or commercial office.
5. **Gates**: Physical entry/exit points independent of buildings.

---

## 2. Unit Auto-Generation Engine

Entering hundreds of flats manually is eliminated through a deterministic auto-generation algorithm.

### Algorithmic Flow
```text
Inputs:
(Prefix: "A", StartFloor: 1, Floors: 9, UnitsPerFloor: 3, StartUnit: 101)
      │
      ▼
Loop Floor F from StartFloor to (StartFloor + Floors - 1):
      │
      ├─ Check if floor F has override in FloorOverrides map
      │  If yes, unit_count = FloorOverrides[F]
      │  If no,  unit_count = UnitsPerFloor
      │
      └─ Loop Unit Index u from 1 to unit_count:
            UnitNumber = (F * 100) + u
            UnitIdentifier = Prefix + UnitNumber
            Emit Unit Record (floor=F, unit_number=UnitNumber, identifier=UnitIdentifier)
      │
      ▼
Return Array of Units (e.g., A101, A102, A103 ... A901, A902, A903 = 27 units)
```

---

## 3. Society Formation Workflow

The Society Formation experience guides administrators through 5 steps:
1. **Step 1 — Society**: Confirm existing society profile, canonical code, and status.
2. **Step 2 — Buildings & Wings**: Define buildings, optional wings, and floor counts.
3. **Step 3 — Units Generation**: Configure numbering rules, preview generated units, and stage them.
4. **Step 4 — Gates & Staffing**: Configure gate names, codes, types, and shift staffing requirements.
5. **Step 5 — Review & Commit**: Inspect configuration totals and commit in an atomic database transaction.

---

## 4. Gatekeeper Staffing Semantics

Staffing metrics explicitly decouple policy requirements from actual user records:

| Metric | Source | Description |
| :--- | :--- | :--- |
| **Required Headcount** | `society_staffing_requirements.total_required_headcount` | Total gatekeepers needed by the society. |
| **Required Staff Per Gate** | `SUM(society_gates.required_staff_per_shift)` | Total guard posts active per shift across all gates. |
| **Registered Staff** | `COUNT(staff)` where role=`GATEKEEPER` | Actual registered user accounts (0 in S05, deferred to S08). |
| **Assigned Staff** | `COUNT(shifts)` | Actual staff currently scheduled to guard shifts (0 in S05). |

---

## 5. Related Documentation
- [[../decisions/017-society-structure-and-formation|ADR 017: Society Structure, Formation Workflow, and Unit Generation Model]]
- [[../database/05-society-structure-schema|05 — Society Structure Database Schema]]
- [[../api/06-society-structure-api|06 — Society Structure API Specification]]
- [[../security/05-society-structure-security|05 — Society Structure Security Specification]]
- [[../sprints/S05-society-structure|S05 — Society Structure]]
