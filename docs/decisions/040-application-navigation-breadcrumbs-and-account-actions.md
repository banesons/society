# ADR 040: Application Navigation Architecture, Contextual Society Submenu, Hierarchical Breadcrumbs, and Sign-Out Confirmation

## Status
Accepted

## Context
Following the implementation of the dual-form navigation rail (ADR 038) and modular society sub-routes (`/app/societies/[id]/*`), architectural review and user testing identified five UX and navigation structural issues:

1. **Missing Contextual Society Submenu**:
   - When entering a society workspace (`/app/societies/[id]/*`), the desktop navigation sidebar remained stuck in general workspace mode, rendering only *Dashboard*, *Societies*, and *Notifications*.
   - Users lacked contextual visibility into available modules (Overview, Members, Committee, Structure, Gate, Announcements, Maintenance, Resident Hub, Documents, Reports, Audit) without constantly opening the right-hand slide-in drawer.
2. **Missing & Incomplete Breadcrumbs**:
   - Society module pages relied on a crude `← Back to Dashboard` text link, failing to represent the actual navigation hierarchy (e.g. `Workspace / Greenview Residency / Announcements`).
   - Deep-linked routes offered no breadcrumb trail to navigate upward to society overview or workspace root.
3. **Missing Sign-Out Confirmation**:
   - Selecting "Sign Out" immediately terminated the session without confirmation.
   - While global session revocation ("Log Out All Devices") featured a dedicated confirmation modal (`LogoutAllDialog`), normal single-device sign-out lacked a protective confirmation step.
4. **Duplication of Navigation Destinations**:
   - *Notifications* appeared in the desktop sidebar, mobile bottom rail, header notification bell, and society drawer simultaneously.
   - *Profile* was duplicated in the desktop sidebar, mobile rail, account menu popover, and society drawer.
5. **Inconsistent Shell and Double Branding**:
   - Inside society routes, the desktop layout rendered a double brand logo (one in the navigation rail, and one in `SocietyShellHeader`).
   - The shell needed a unified, predictable structure across workspace, society modules, profile, and platform administration.

---

## Decision

### 1. Navigation Ownership Model
Clear boundaries are established across three primary surfaces:
- **Left Sidebar (Desktop / Tablet)**:
  - **State A — General Workspace (`/app/dashboard`, `/app/societies`, `/app/profile`, `/app/notifications`, `/app/admin`)**:
    - Primary destinations: *Dashboard*, *Societies*, and *Platform Administration* (if authorized).
    - Footer: Authenticated user identity and avatar shortcut.
  - **State B — Society Workspace (`/app/societies/[id]/*`)**:
    - Contextual Society Navigation:
      - Quick exit: `← All Societies` link returning to the societies directory.
      - Society Identity Header: Name, code, location, and verified role badge (`RESIDENT`, `SOC_ADMIN`, `GATEKEEPER`, `SUPER_ADMIN`).
      - Tenant Switcher shortcut when the user has multiple society assignments or is `SUPER_ADMIN`.
      - Modular Submenu grouped logically:
        - **OVERVIEW**: Society Overview (`/app/societies/${id}`)
        - **PEOPLE**: Members & Residents (`/members`), Managing Committee (`/committee`)
        - **OPERATIONS**: Structure (`/structure`), Gate Management (`/gate`), Announcements (`/announcements`), Maintenance & Billing (`/maintenance`), Resident Visitor Hub (`/resident-hub`), Documents & Records (`/documents`)
        - **INSIGHTS**: Reports & Analytics (`/reports`), Audit Logs (`/audit`)
      - Role Gating: Strict enforcement hiding administrative modules from residents and gatekeepers.
      - Exact route-aware active state: Deep-linked paths (e.g. `/announcements`) activate exclusively their respective module.
- **Top Header**:
  - Global Branding (on general workspace pages; deduplicated on society pages).
  - Breadcrumbs Container.
  - Notification Bell with real-time unread badge and popover center.
  - Account Menu (Desktop avatar trigger toggling user identity, profile link, and session management redirect; Mobile avatar is a direct anchor to `/app/profile` without popup overlays).
- **Mobile Bottom Navigation (`screen.isMobile`)**:
  - 4 primary thumb destinations: *Dashboard*, *Societies*, *Alerts*, *Profile*.
  - Society modules remain accessible via the contextual society drawer/menu button.
  - Eliminates redundant floating popup cards on mobile screens; tapping the top avatar directly navigates to `/app/profile`.

### 2. Hierarchical Breadcrumbs System
- A shared, accessible `<Breadcrumbs />` component rendered in the content header below the global header:
  - Semantic structure: `<nav aria-label="Breadcrumb">` with `<ol>` and `aria-current="page"`.
  - Derives hierarchy dynamically from the URL route and active entity:
    - Workspace: `Workspace / {Entity}`
    - Society modules: `Workspace / {Society Name} / {Module Label}`
    - Administrative: `Platform Administration / Societies`
  - Society name resolved from society profile or active assignments, with graceful fallback to society code during loading states (never raw UUIDs).
  - Intermediate crumbs are interactive links (`href`); the leaf crumb represents current page and is text-only.
  - Replaces and eliminates the redundant `← Back to Dashboard` button.

### 3. Sign-Out & Session Controls Anchored in Personal Profile
- Standard sign-out ("Sign Out") and global logout ("Log Out All Devices") are anchored in a dedicated *Session & Device Actions* card inside the Personal Profile (`/app/profile`).
- Removes redundant destructive actions from the ephemeral header dropdown and centralizes session lifecycle controls in the user's authoritative profile view.
- Standard sign-out requires confirmation via `SignOutDialog` before invoking `auth.logout()`.
- Built on `ResponsiveModal` adhering to ADR 036:
  - **Desktop**: Centered horizontally, exactly 180px from viewport top.
  - **Mobile**: Bottom sheet with safe-area padding.
- Copy:
  - Title: *"Sign out?"*
  - Description: *"You will be signed out of this device and will need to authenticate again to access your account."*
  - Actions: *Cancel* (dismisses modal), *Sign Out* (invokes `auth.logout()`, shows spinner, redirects to `/login`).
- Prevents duplicate submission while logout is in progress.
- Distinct from "Log Out All Devices" which uses `LogoutAllDialog` for global session revocation.

### 4. Role-Gated Physical Structure Summary
- Queries to `/api/v1/societies/:id/formation-summary` are role-gated on client to administrator principals (`isSocAdmin`). Resident and gatekeeper users do not request formation summary, preventing spurious 403 Forbidden errors.

---

## Consequences
- Navigating between society modules is immediate and persistent on desktop without returning to the dashboard.
- Users always understand their current position in the application hierarchy via clear breadcrumbs.
- Accidental session termination is eliminated via responsive sign-out confirmation.
- Redundant navigation links and double-logo rendering are eradicated.
- Visual design maintains CHS Neu editorial Swiss-minimalist tokens (`Hedvig Letters Sans` / `Serif`, `#f3f3f1` canvas, hairline borders).

## Related Documentation
- [[032-frontend-application-shell-and-ux-refactor|ADR 032: Frontend Application Shell, Dedicated Routing Boundaries, and Administrative UX Refactor]]
- [[033-role-based-society-navigation-and-functional-ux|ADR 033: Role-Based Society Navigation, Feature Gating, and Functional UX]]
- [[036-shared-responsive-modal-system|ADR 036: Shared Responsive Modal System and Adaptive Presentation]]
- [[038-navigation-rail-and-modular-society-routes|ADR 038: Navigation Rail Architecture, Dedicated Notifications Screen, and Modular Society Sub-Routes]]
- [[../architecture/03-frontend|03 — Frontend Architecture]]
