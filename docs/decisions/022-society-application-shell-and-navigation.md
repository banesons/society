# ADR 022: Society Application Shell and Navigation Architecture

## Status
Accepted

## Context
Prior to Sprint S09, authenticated society views at `/society/:id` were rendered under the global marketing layout shell. This exposed irrelevant public marketing navigation links (`societies`, `governance`, `members`, `gatekeeping`, `compliance`, `about`) inside operational society management sessions, lacked a unified mobile app bar, and offered no side-accessible drawer for multi-society tenant switching and session management.

Furthermore, post-login routing did not evaluate the authenticated user's accessible society assignments, defaulting all non-Super-Admin users to `/unauthorized`.

## Decision

1. **Route Boundaries & Layout Isolation**:
   - Establish three distinct interface zones:
     - **Public Marketing** (`/`, `/login`, etc.): Retains global marketing header and footer.
     - **Platform Administration** (`/admin`): Retains Super Admin governance shell.
     - **Society Application** (`/society/:id`, `/societies/:id`): Excluded from global marketing header and footer in root layout; wrapped by a dedicated `+layout.svelte` Society Application Shell.
   - Retain `/society/:id` as the canonical route; `/societies/:id` provides an HTTP 308 permanent compatibility redirect preserving deep links.

2. **Deliberate Desktop vs Mobile Header Layouts**:
   - **Desktop (`>= 640px`)**:
     - Minimalist CHS wordmark (`Logo`).
     - Compact society badge (`{name} · {society_code}`) and single menu trigger button (`aria-expanded`).
     - Dedicated contextual back-navigation row below the header: `[← Return to Admin Console / Return to Home]`.
   - **Mobile (`< 640px`)**:
     - Conventional compact app bar with comfortable touch targets (`>= 44x44px`).
     - Left to right: Leading `[← Back]` button, followed by `Logo`.
     - Right: Accessible `[Menu button]`.
     - Omit duplicate back-navigation row below the header on mobile.

3. **Side Dropdown / Drawer (`SocietyDrawer`)**:
   - Reusable responsive sheet: right-anchored on desktop, full-width sheet on mobile.
   - Organized into four structured groups:
     - **Group A (Current Society)**: Society name, code (`font-mono`), location, status, verified operational role.
     - **Group B (Society Navigation)**: Mapped strictly to operational sections (`Overview`, `Members & Residents`, `Managing Committee`, `Physical Structure`, `Gatekeeping`) with active indicators. Unimplemented future modules (`Visitors`, `Maintenance`, `Documents`, `Communications`, `Reports`, `Settings`) are clearly marked as disabled "Upcoming" items without dead links. Sprint labels are suppressed.
     - **Group C (Switch Society)**: Available when user holds multiple accessible assignments. For `SUPER_ADMIN`, discovers all active registered societies via `/platform/societies` without requiring prior `SOC_ADMIN` assignment. For Platform `ADMIN`, remains strictly restricted to authenticated `society_assignments` (zero implicit access). Opens `SocietySelectorDialog`.
     - **Group D (Account & Sessions)**: Formatted phone (`+91 99670 89101`), role badge, single-device logout, and multi-device revocation.

4. **Post-Login Routing Policy**:
   - Evaluated strictly against authenticated backend data from `GET /api/v1/auth/me`:
     - `SUPER_ADMIN` / `ADMIN`: Routes to `/admin` by default. No implicit society access for Platform Admin.
     - 0 Accessible Societies: Routes to `/unauthorized`.
     - 1 Accessible Society: Routes directly to `/society/${id}`.
     - >1 Accessible Societies: Displays mandatory `SocietySelectorDialog`.

5. **Tenant Isolation & Cache Partitioning**:
   - Switching societies triggers TanStack Query purge: `queryClient.removeQueries({ queryKey: ['society', previousId] })` and query invalidation to eliminate cross-tenant data leaks.

## Consequences

- Clean separation between platform, marketing, and society tenant experiences.
- Intuitive, tactile mobile header with zero horizontal overflow and comfortable touch targets.
- Preserved deep linking and backwards compatibility.
- Guaranteed multi-tenant data isolation across switching events.

## Related Documentation
- [[../architecture/03-frontend|03 — Frontend Architecture]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[../api/02-authentication-api|02 — Authentication API Specification]]
