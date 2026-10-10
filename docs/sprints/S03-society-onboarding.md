# S03 — Society Onboarding

## Objective
Implement the initial end-to-end business workflow for society registration, including user submission into `PENDING` state, platform admin review, approval/rejection, and activation.

## Scope
### Backend
- Society creation API with initial `PENDING` state and creator association.
- Platform review APIs for approval, rejection, and activation state transitions.
- Society lifecycle validation rules and status transition guards.
- Basic society profile data model.

### Frontend
- End-user society registration form and onboarding wizard.
- Onboarding status tracker view for the society creator.
- Platform admin review console for evaluating pending societies.
- Society approval and rejection action interfaces.

## Dependencies
- [[S02-platform-administration]]

## Deliverables
- Complete society submission, review, approval, and activation workflow.
- Verified society state lifecycle transitions (`PENDING` → `APPROVED` → `ACTIVE` or `REJECTED`).
- Society creator relationship linking the applicant to the created society.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 2: Society Onboarding)]]

## Status
Planned
