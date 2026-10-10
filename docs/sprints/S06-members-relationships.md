# S06 — Members & Relationships

## Objective
Establish unit-level occupancy relationships, modeling Owners (Primary, Secondary, Family) and Tenants (Primary, Secondary, Family) with appropriate association workflows and PII masking.

## Scope
### Backend
- Unit-member association models supporting:
  - `PRIMARY_OWNER`
  - `SECONDARY_OWNER`
  - `OWNER_FAMILY`
  - `PRIMARY_TENANT`
  - `SECONDARY_TENANT`
  - `TENANT_FAMILY`
- Member onboarding, verification, and association APIs.
- Member relationship modification and removal flows.
- PII masking on member directory endpoints following project standards.

### Frontend
- Member management directory for Society Admins.
- Member invitation and unit relationship assignment forms.
- Resident profile screen displaying associated unit(s) and household members.
- PII-safe resident contact views.

## Dependencies
- [[S05-society-structure]]

## Deliverables
- Comprehensive unit-occupancy relationship management system.
- Association workflows for all six supported resident relationship types.
- Masked resident directory ensuring data minimization.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 1, 4 & 8: Relationships & PII Policy)]]

## Status
Planned
