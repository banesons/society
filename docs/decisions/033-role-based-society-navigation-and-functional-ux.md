# ADR 033: Role-Based Society Navigation, Feature Gating, and Functional UX

## Status
Accepted

## Context
Prior to this architectural change, the Society Application Drawer (`SocietyDrawer`) rendered an identical, monolithic list of twelve navigation items regardless of the authenticated user's role and permissions in the society:
1. **Resident Cognitive Overload & Dead Links**:
   - A resident logging into Greenview Residency or Shree Ram CHS was presented with 12 menu options, including administrative governance tools (`Members & Residents`, `Managing Committee`, `Physical Structure`, `Gatekeeping`, `Audit & Activity`, `Dashboard & Reports`).
   - Clicking these administrative links resulted in silent failures: the drawer closed and the URL hash updated, but no section was visible because administrative components are strictly gated behind `isSocAdmin` in `src/routes/society/[id]/+page.svelte`.
   - Clicking `Maintenance` or `Documents` jumped to admin section IDs (`#maintenance`, `#documents`) rather than resident dues and documents (`#resident-maintenance`, `#resident-documents`), producing a broken user experience.
2. **Advertising Unimplemented Capabilities**:
   - The drawer contained a non-functional item `"Settings — Upcoming"` marked as disabled, giving the impression of an unfinished prototype rather than an operational product.
3. **Mixed Administrative and Resident Boundaries**:
   - Frontline perimeter operations (`Gatekeeping`), administrative management tools, and resident-facing daily workflows were co-mingled in a single unorganized menu.
4. **Security Boundary Discipline**:
   - Navigation visibility must never serve as the security perimeter. The Go backend remains authoritative for all tenant isolation and role enforcement (`RequireSocietyAccess`, `RequireAnySocietyRoleAccess`, `RequireSocietyMemberOrAdminAccess`). Navigation must accurately project effective capabilities without leaking or promising unauthorized features.

---

## Decision

### 1. Role-Specific Society Navigation Capabilities
The application distinguishes three operational society personas plus platform administration:
- **`RESIDENT`**:
  - Focuses strictly on everyday residential workflows.
  - Exactly 5 purposeful, functional navigation destinations in logical priority:
    1. **Home / Overview** (`/society/:id#overview`): Society identity, code, location, and verified resident context.
    2. **Announcements** (`/society/:id#announcements`): Official society broadcast feed, notices, and unread tracking.
    3. **Maintenance & Dues** (`/society/:id#resident-maintenance`): Unit maintenance dues, bills, payment history, and downloadable receipts.
    4. **Society Documents** (`/society/:id#resident-documents`): Official society bylaws, circulars, audit summaries, and shared records.
    5. **Resident Visitor Hub** (`/society/:id#resident-hub`): Expected visitor pre-approvals, digital entry passes, and unit gate approval policy.
- **`SOC_ADMIN` / `SUPER_ADMIN`**:
  - Retains comprehensive society management tooling:
    1. **Overview** (`#overview`)
    2. **Dashboard & Reports** (`#reports`): Operational KPIs, financial collections, occupancy, and gate traffic.
    3. **Members & Residents** (`#members`): Member roster, unit occupancies, relationships, and departure workflows.
    4. **Managing Committee** (`#committee`): Committee appointments, designations, and governance terms.
    5. **Physical Structure** (`#structure`): Buildings, wings, units, floors, gates, and formation summary.
    6. **Gatekeeping Console** (`#gatekeeping`): Gate shifts, duty sessions, and perimeter logs.
    7. **Announcements** (`#announcements`): Resident feed and broadcast management console.
    8. **Maintenance & Billing** (`#maintenance`): Charge categories, formulas, billing cycles, bill generation, and payments.
    9. **Documents & Records** (`#documents`): Category administration, file uploads, versions, and classification access control.
    10. **Audit & Activity** (`#audit`): Immutable society audit event log with actor and action filters.
    11. **Resident Visitor Hub** (`#resident-hub`): Occupant visitor workflows.
- **`GATEKEEPER`**:
  - Restricted strictly to gate security and urgent broadcasts:
    1. **Overview** (`#overview`)
    2. **Perimeter & Gatekeeping** (`#gatekeeping`): Duty check-in/out, visitor walk-in logging, pass verification, and visitor logbook.
    3. **Announcements** (`#announcements`): Broadcast communication feed.
- **`PLATFORM_ADMINISTRATOR` (`SUPER_ADMIN` / `ADMIN`)**:
  - Platform administration remains an isolated security domain at `/app/admin`.
  - Inside a society drawer, platform administrators operate with verified tenant context without cross-polluting global and local domains.

### 2. Elimination of Unimplemented Links
- **Settings**: Removed completely from active navigation. No non-functional or "Upcoming" placeholder items may appear in primary navigation drawers.
- Planned future features must be tracked in product backlogs, not displayed as dead or disabled links in operational menus.

### 3. Centralized Typed Navigation Registry (`src/lib/navigation.ts`)
- Navigation items are generated through a centralized, strictly typed registry function:
  ```typescript
  export interface SocietyNavItem {
    id: string;
    label: string;
    href: string;
    icon: ComponentType;
    requiredRole: 'ANY' | 'RESIDENT' | 'SOC_ADMIN' | 'GATEKEEPER';
  }
  ```
- Evaluates the authenticated user's effective role for the current society tenant (`SUPER_ADMIN`, `ADMIN`, `SOC_ADMIN`, `GATEKEEPER`, `RESIDENT`).
- Guarantees every returned link points to an implemented, rendered section with working APIs.

### 4. Active Section Synchronization & Drawer UX
- The drawer dynamically synchronizes active section state using `page.url.hash` (e.g. `#resident-maintenance`).
- Accessible keyboard dismissal via `Escape`.
- Backdrop dismiss without accidental navigation.
- Accessible ARIA labeling (`role="dialog"`, `aria-modal="true"`).
- Separation of structural groups:
  - **Group A (Society Context)**: Society name, code (`font-mono`), location, and verified operational role badge.
  - **Group B (Navigation)**: Role-specific links with crisp Lucide icons and active dot indicators.
  - **Group C (Tenant Switcher)**: Visible only when the user holds assignments in multiple societies or is `SUPER_ADMIN`.
  - **Group D (Account & Session)**: Formatted phone number (`+91 98220 00109`), role description, single-device sign out, and multi-device revocation modal.

---

## Consequences

- Residents experience an uncluttered, intuitive, and 100% functional navigation menu with zero dead links.
- Society administrators retain seamless, immediate access to all 11 operational management modules.
- Gatekeepers receive a focused, distraction-free perimeter interface.
- Complete fidelity with CHS documentation-first principles and multi-tenant security architecture.

## Related Documentation
- [[../architecture/03-frontend|03 — Frontend Architecture]]
- [[../architecture/08-society-administration|08 — Society Administration Architecture]]
- [[022-society-application-shell-and-navigation|ADR 022: Society Application Shell and Navigation Architecture]]
- [[032-frontend-application-shell-and-ux-refactor|ADR 032: Frontend Application Shell, Dedicated Routing Boundaries, and Administrative UX Refactor]]
