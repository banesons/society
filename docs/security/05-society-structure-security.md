# 05 — Society Structure Security Specification

## Overview

This document specifies the authorization, multi-tenant boundaries, and data integrity rules governing society structure and physical resource configuration.

---

## 1. Authorization Matrix

| Endpoint | `SUPER_ADMIN` | `SOC_ADMIN` (Assigned) | `SOC_ADMIN` (Unassigned) | Platform `ADMIN` | Resident |
| :--- | :---: | :---: | :---: | :---: | :---: |
| `GET /societies/{id}/buildings` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/buildings` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/units/preview` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/units/generate` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |
| `POST /societies/{id}/gates` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |
| `PUT /societies/{id}/staffing` | ✅ | ✅ (Active Society) | ❌ 403 | ❌ 403 | ❌ 403 |

---

## 2. Threat Vectors & Mitigations

### 2.1 Cross-Society Parent Substitution
- **Threat**: Attacker supplies `building_id` belonging to Society A when creating a unit in Society B.
- **Mitigation**: Foreign key integrity checked server-side fail-closed; SQL query verifies `WHERE id = $building_id AND society_id = $verified_society_id`.

### 2.2 Massive Bulk Generation Denial of Service
- **Threat**: Request requesting 100,000 units exhausts server memory and locks database tables.
- **Mitigation**: Capped at maximum 1,000 units per request. Total floors capped at 100, units per floor capped at 50.

### 2.3 Unit Identifier Spoofing & Collisions
- **Threat**: Duplicate unit numbers causing billing, ownership, or access mix-ups.
- **Mitigation**: Database-enforced `UNIQUE(society_id, unit_identifier)` constraint.

---

## 3. Related Documentation
- [[../decisions/017-society-structure-and-formation|ADR 017: Society Structure, Formation Workflow, and Unit Generation Model]]
- [[../architecture/09-society-structure|09 — Society Structure Architecture]]
- [[../database/05-society-structure-schema|05 — Society Structure Database Schema]]
- [[../api/06-society-structure-api|06 — Society Structure API Specification]]
- [[../sprints/S05-society-structure|S05 — Society Structure]]
