# ADR 041: Gatekeeper Experience, Unified Visitor Identity, Cross-Gate Lifecycle, and Personal Activity Timeline

## Status
Accepted

## Context
CHS Neu manages cooperative housing societies across multiple distinct user personas: platform administrators, society administrators, residents, visitors, and gatekeepers. An architectural review identified four structural gaps in the gate operations and identity subsystem:

1. **Gatekeeper Authentication & Dedicated Console**:
   - Gatekeepers lacked a dedicated login entry route (`/gate/login`), having to use generic `/login` which did not automatically route directly to their assigned operational gate console.
   - The gate console (`/app/societies/[id]/gate`) mixed operational tasks (visitor check-in, checkout, approvals) with administrative configurations (gate creation, shift definitions, gatekeeper assignments), resulting in 403 Forbidden errors when gatekeepers attempted to access admin-only datasets.
2. **Conflation of Visitor Records & Identity**:
   - A person, a registered user account, a visitor identity, and an individual visit were conflated. Unregistered visitors were stored merely as mutable strings in `society_visits`, with no first-class `visitor_identity` entity.
   - When an unregistered visitor subsequently registered on CHS Neu, their past visit history across societies could not be securely reconciled with their authenticated profile.
3. **Gatekeeper Privacy Boundaries (Data Minimization)**:
   - Gatekeepers need to verify visitor identity and destination without browsing residents' or visitors' full phone numbers, email addresses, or unrestricted profiles.
   - The API previously returned full unmasked phone numbers in visitor list payloads, relying on frontend display masking.
4. **Missing Chronological Activity Timeline**:
   - While `platform_audit_logs` recorded backend mutations, users had no visibility into their personal history (logins, session rotations, visits created, arrivals, departures, pass generation) within their Personal Profile (`/app/profile`).
5. **Messaging Invariants**:
   - Strict product requirement: No SMS invitations, no messaging inboxes, and no automated promotional outreach to visitors.

---

## Decision

### 1. Unified Authentication & Role-Based Routing
- Maintain `/app/login` (and `/login`) as the single, authoritative authentication entrance for all personas:
  - Reuses the existing phone OTP authentication infrastructure (`auth.sendOtp`, `auth.verifyOtp`, JWT authority, FIFO 2-device session limit).
  - Resolves authenticated roles and society assignments via `resolvePostLoginRoute(user)`.
  - Single-assignment gatekeepers navigate automatically and directly to `/app/societies/:id/gate`.
  - Multi-assignment users receive an explicit workspace selector modal (`<SocietySelectorDialog />`) without privilege leakage.
  - No separate `/gate/login` route or divergent authentication system is introduced.

### 2. Dedicated Gatekeeper Operational Workspace
- Decouple operational gatekeeping from society administration inside `GateConsole`:
  - **Gatekeeper View**:
    - **Active Operations**: Fast search, Walk-in Registration, Active Visits / Checkout, Expected Visitors / Approvals Queue, Authorized Activity History.
    - **Contextual Summary**: Society identity, assigned gate name, operational status, active inside count, pending approvals count.
    - **Mobile-First UX**: Large touch targets, one-handed mobile search, and simplified checkout flows via `screen.isMobile`.
    - **Admin Exclusion**: Hides and forbids gate configuration, shifts configuration, and gatekeeper roster assignments.
  - **Society Administrator View**:
    - Retains operational consoles and adds structural gate configuration and staff assignment tabs.

### 3. Unified Identity Architecture (`society_visitor_identities`) & Ambiguity Protection
- Introduce a first-class `society_visitor_identities` table:
  - `(id, society_id, phone, full_name, user_id, linked_at, created_at, updated_at)`
  - Composite uniqueness on `(society_id, phone)` guarantees that visitor identities remain strictly isolated within society tenant boundaries.
  - Foreign key `user_id UUID NULL REFERENCES users(id)` enables linking a visitor identity to an authenticated account once the phone number is verified via OTP.
  - `society_visits` adds `visitor_identity_id UUID NULL REFERENCES society_visitor_identities(id)`.
  - Safe Identity Linking (`SafeLinkVisitorIdentities`):
    - Matches candidate visitor identities by verified phone number upon OTP completion.
    - Ambiguity Safeguard: When multiple distinct names exist for a shared family phone number, identities are linked only if the candidate name matches the verified profile name (`user_profiles.first_name`), preventing one family member from claiming another's visit records.
    - Preserves historical visit records without altering historical timestamps.
    - Strictly no SMS invitations or account activation messaging.

### 4. Cross-Gate Entry and Exit
- Formally support visitor entry through Gate 1 and exit through Gate 2:
  - `society_visits.entry_gate_id` records the gate of ingress; `society_visits.exit_gate_id` records the gate of egress.
  - `CheckOutVisitor` accepts any active egress-capable gate in the society where the gatekeeper holds an active session.
  - `society_gate_events` immutably logs the respective entry and exit gates, gatekeeper attributions, and timestamps.
  - Concurrency control: Atomic row-level lock (`FOR UPDATE`) prevents duplicate concurrent checkouts, returning HTTP 409 Conflict.

### 5. Universal Backend Privacy Enforcement
- Enforce strict server-side data minimization across all visitor endpoints:
  - When the calling principal holds role `GATEKEEPER`, the backend zeroes/masks `visitor_phone` (`visitor_phone = visitor_phone_masked`) across all serialized responses:
    - `GET /visitors` (Visits list)
    - `GET /visitors/active` (Active inside list)
    - `GET /visitors/identities/search` (Phone search)
    - `GET /visitors/:id` (Visit detail)
    - `POST /visitors/walk-in` (Walk-in creation)
    - `POST /visitors/:id/check-in` (Perimeter entry)
    - `POST /visitors/:id/check-out` (Perimeter exit)
    - `POST /visitors/passes/verify` (Pass verification)
  - Raw phone numbers, emails, and unrelated society memberships never leave the server for gatekeeper callers.

### 6. Personal Profile Activity Timeline (`/app/profile`)
- Introduce an authorized, paginated Activity Timeline on `/app/profile`:
  - Endpoint: `GET /api/v1/auth/profile/activity?category=ALL&limit=20&cursor=...`
  - Categories: `ALL`, `ACCOUNT`, `VISITOR`, `SOCIETY`, `SECURITY`.
  - Aggregates real events from:
    - `platform_audit_logs` where `actor_id = authUser.ID` OR `target_user_id = authUser.ID`.
    - Visitor events where the user is linked as visitor (`society_visitor_identities.user_id = authUser.ID`) or destination resident (`society_visits.resident_id`).
  - Strict projection: Converts internal audit events into human-readable timeline items with title, summary, timestamp, category, and society context.
  - Excludes raw tokens, database mutations, and internal security traces.
  - Stable pagination using cursor (`created_at DESC, id DESC`).

---

## Consequences
- Gatekeepers have a streamlined, fast operational workflow accessible on mobile at perimeter gates without administrative clutter.
- Personal identity and visitor records remain distinct yet securely linkable upon verified account registration.
- Visitors can freely enter through one gate and exit through another.
- Visitor privacy is mathematically protected by withholding raw phone numbers from gatekeeper API responses.
- Users gain an intuitive, chronological activity audit log of their personal identity events directly in their profile.
- Zero SMS or spam invitations are generated.

## Related Documentation
- [[020-gate-foundation-and-staffing-operations|ADR 020: Gate Foundation and Staffing Operations]]
- [[023-visitor-lifecycle-and-gate-operations|ADR 023: Visitor Lifecycle and Gate Operations]]
- [[035-personal-profile-experience-and-identity|ADR 035: Personal Profile Experience and Identity]]
- [[037-canonical-user-profiles-and-identity-resolution|ADR 037: Canonical User Profiles and Identity Resolution]]
- [[040-application-navigation-breadcrumbs-and-account-actions|ADR 040: Application Navigation Architecture, Contextual Society Submenu, Hierarchical Breadcrumbs, and Sign-Out Confirmation]]
- [[../architecture/14-gate-visitor-operations|14 — Gate Visitor Operations Architecture]]
- [[../database/09-gate-visitor-operations-schema|09 — Gate Visitor Operations Database Schema]]
- [[../api/10-gate-visitor-operations-api|10 — Gate Visitor Operations API]]
