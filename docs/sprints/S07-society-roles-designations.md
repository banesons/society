# S07 — Society Roles & Designations

## Objective
Support society-level committee designations (Secretary, Treasurer, Chairman, etc.) alongside resident relationships without conflating them with global platform roles.

## Scope
### Backend
- Predefined society designation registry (e.g., Secretary, Treasurer, Chairman, Committee Member).
- Designation assignment, term management, and revocation APIs for Society Admins.
- Designation-scoped authorization checks for designation-specific operational tasks.
- Validation ensuring designations remain scoped to the respective society.

### Frontend
- Committee and designation management console in the Society Admin dashboard.
- Assignment workflow linking eligible members to designated roles.
- Designation badges and metadata display on resident profiles and notices.

## Dependencies
- [[S06-members-relationships]]

## Deliverables
- Functional society designation assignment and management workflows.
- Scoped permissions model for committee designations.
- Resident UI indicating active committee members and designations.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 1 & 4: Society Designations)]]

## Status
Planned
