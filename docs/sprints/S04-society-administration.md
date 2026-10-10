# S04 — Society Administration

## Objective
Establish society-level administration by mapping platform Admins or designated users to one or multiple societies as `SOC_ADMIN` with scoped administrative privileges.

## Scope
### Backend
- Admin → Society mapping data model and management APIs.
- Assignment and revocation of `SOC_ADMIN` role.
- Society-scoped authorization middleware ensuring access is restricted to assigned societies.
- Support for an Admin managing multiple distinct societies.

### Frontend
- Society switcher UI allowing multi-society admins to alternate active society context.
- Society-level administration console shell.
- Platform admin interface for assigning and managing Society Admins.
- Society context provider embedded in frontend state.

## Dependencies
- [[S03-society-onboarding]]

## Deliverables
- Multi-society admin mapping capability with strict tenant isolation.
- Society-scoped administrative authorization checks.
- Functional society switcher and administrative interface shell.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 1 & 4: Society Administration)]]

## Status
Planned
