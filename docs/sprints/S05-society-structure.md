# S05 — Society Formation & Structure

## Objective
Model and manage physical society infrastructure, including buildings, optional wings, floors, unit auto-generation with deterministic patterns, gate configuration, and honest gatekeeper staffing policy through a guided Society Formation workflow.

## Scope

### Backend
- `society_buildings`, `society_wings`, `society_units`, `society_gates`, and `society_staffing_requirements` schema.
- Deterministic unit auto-generation engine (pattern formula e.g. A101-A903 for 9 floors x 3 units; non-uniform floor overrides).
- Unit generation preview and transactional bulk generation APIs.
- Gate configuration with unique codes and shift staffing numbers.
- Formation summary calculation endpoint.
- Transactional audit logging for all physical structure mutations.
- Society-scoped authorization (`SUPER_ADMIN` or active assigned `SOC_ADMIN`).

### Frontend
- Multi-step guided Society Formation experience (Step 1 Society Review, Step 2 Buildings & Wings, Step 3 Unit Generation & Preview, Step 4 Gates & Staffing, Step 5 Summary & Commit).
- Interactive Unit Auto-Generation form with dynamic preview grid.
- Gate configuration and staffing requirement editor.
- Structure overview and editing tabs in Society Console.

## Acceptance Criteria

- [x] Guided Society Formation workflow exists in frontend.
- [x] Buildings and optional wings can be created and managed.
- [x] Unit numbering patterns are deterministic (e.g. 9 floors, 3 units/floor generates A101 through A903 = 27 units).
- [x] Non-uniform floor layouts supported via overrides.
- [x] Unit identifiers enforced unique across society via database constraint.
- [x] Unit generation preview matches persisted records.
- [x] Bulk unit generation is transactional.
- [x] Gate configuration supported with unique codes per society.
- [x] Required gatekeeper staffing captured without fabricating fake accounts.
- [x] Society-scoped authorization enforced; cross-society access prevented.
- [x] Structure mutations audited transactionally.
- [x] Unit and integration tests pass for frontend and backend.

## Related Documentation
- [[../decisions/017-society-structure-and-formation|ADR 017: Society Structure, Formation Workflow, and Unit Generation Model]]
- [[../architecture/09-society-structure|09 — Society Structure Architecture]]
- [[../database/05-society-structure-schema|05 — Society Structure Database Schema]]
- [[../api/06-society-structure-api|06 — Society Structure API Specification]]
- [[../security/05-society-structure-security|05 — Society Structure Security Specification]]
- [[README|Master Sprint Index]]

## Status
- Architecture & Documentation: **COMPLETE**
- Implementation: **COMPLETE**
- Remote Migrations: **APPLIED (20261015000000)**
- Automated Tests: **100% PASSING (Go + Vitest)**
