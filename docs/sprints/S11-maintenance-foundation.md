# S11 — Maintenance Foundation

## Objective
Establish the financial structure for society maintenance: charge definitions, calculation rules, billing cycle schedules, and unit-level fee configuration.

## Scope
### Backend
- Charge head definitions (e.g., fixed rate, per-sqft rate, parking charges, sinking fund).
- Billing period and cycle configuration (monthly, quarterly, annual).
- Unit-level charge mapping and calculation engine.
- Maintenance configuration CRUD APIs for Society Admins and Treasurers.

### Frontend
- Maintenance charge setup wizard in Society Admin dashboard.
- Billing cycle and due-date configuration interface.
- Unit fee assignment and exception handling UI.
- Summary preview of projected bill amounts prior to billing cycles.

## Dependencies
- [[S07-society-roles-designations]]

## Deliverables
- Configurable maintenance charge engine supporting fixed and formula-based fees.
- Billing period scheduler and unit fee mapping registry.
- Financial configuration interface for society administrators.

## Related Documentation
- [[README|Master Sprint Index]]
- [[../eagle_view|Eagle View Architecture (Section 5: Maintenance & Charges)]]

## Status
Planned
