# ADR 038: Navigation Rail Architecture, Dedicated Notifications Screen, and Modular Society Sub-Routes

## Status
Accepted

## Context
As the CHS Neu authenticated application expanded, several navigation and architectural issues emerged across mobile and desktop viewports:
1. **Notification Popover vs. Dedicated Screen**:
   - Notifications were previously rendered solely inside an inline popover dropdown or floating card (`NotificationCenter`).
   - On mobile screens, this modal overlay cramped notification history, action buttons, and notice filters.
   - Modern residential workflows require a dedicated, distraction-free Notifications view (`/app/notifications`) with clear read states, category filters, and immediate access from primary navigation.
2. **Navigation Disparity (Web & Mobile)**:
   - Authenticated navigation lacked a consistent structural anchor across top-level domains (Dashboard, Societies, Notifications, Profile).
   - On web (desktop/tablet), users had to rely on a top header menu.
   - On mobile, users had no persistent navigation rail for fluid one-thumb switching.
3. **Mobile Drawer Clipping**:
   - The society application drawer (`SocietyDrawer`) suffered from viewport overflow and clipping on mobile devices: on screens under 400px width, `max-w-sm` and right-aligned flex positioning caused the drawer panel to shift 50% offscreen to the right, truncating titles and navigation links.
4. **Monolithic Society Route vs. Modular Sub-Routes**:
   - The society workspace (`/app/societies/[id]`) previously crammed 11 distinct operational modules (Overview, Members, Committee, Structure, Gate, Announcements, Maintenance, Documents, Audit, Resident Hub, Reports) into a single monolithic page with URL hash anchors (`#maintenance`, `#members`).
   - This concentrated excessive state, heavy component mounting, and cross-module re-renders into a single page rather than modular, deep-linkable URLs.

---

## Decision

### 1. Navigation Rail System (Reusing `screen.isMobile` / `DeviceType`)
The authenticated application shell introduces a dual-form Navigation Rail:
- **Desktop & Tablet (`!screen.isMobile` / `>= 768px`)**:
  - A slim, elegant Left Vertical Side Rail (`w-20` / `w-64` responsive) containing:
    1. Brand Logo
    2. Primary Destinations:
       - **Dashboard** (`/app/dashboard`)
       - **Societies** (`/app/societies`)
       - **Notifications** (`/app/notifications`) with real-time unread badge
       - **Platform Administration** (`/app/admin`, for platform administrators)
    3. Bottom Controls:
       - **Personal Profile** (`/app/profile`) with initials avatar
       - **Session Management / Logout**
- **Mobile (`screen.isMobile` / `< 768px`)**:
  - A persistent Bottom Navigation Rail anchored to the viewport bottom (`fixed inset-x-0 bottom-0 z-40 bg-white/95 backdrop-blur-md border-t border-neutral-200/80`):
    - **Dashboard**
    - **Societies**
    - **Notifications** (with unread indicator dot)
    - **Profile**
  - Includes safe-area inset bottom padding (`pb-[max(0.75rem,env(safe-area-inset-bottom))]`) for modern edge-to-edge mobile devices.

### 2. Dedicated Notifications Screen (`/app/notifications`)
- Created a top-level route `/app/notifications` providing:
  - Unread count indicator and "Mark All as Read" action.
  - Tabbed filtering: **All**, **Unread**, **Broadcasts**, **Visits**, **Billing**.
  - Direct integration with Firebase push and backend notification services.
  - Responsive layout for both mobile touch lists and desktop wide views.

### 3. Mobile Society Drawer Fix
- Restructured `SocietyDrawer` so that on mobile:
  - It spans full viewport width (`w-full max-w-full sm:max-w-md`) with proper right-anchored slide-in animation.
  - Removed fixed-width overflow that shifted the drawer off-screen on mobile viewports.
  - Added safe area padding and accessible dismiss behavior.

### 4. Modular Society Sub-Routes
Modularized `/app/societies/[id]` into dedicated, deep-linkable sub-pages sharing a common layout (`/app/societies/[id]/+layout.svelte`):
- `/app/societies/[id]` — Overview & Society Identity
- `/app/societies/[id]/members` — Members & Residents Roster
- `/app/societies/[id]/committee` — Managing Committee
- `/app/societies/[id]/structure` — Physical Structure & Unit Mapping
- `/app/societies/[id]/gate` — Gatekeeping Console
- `/app/societies/[id]/announcements` — Announcements Broadcast & Feed
- `/app/societies/[id]/maintenance` — Maintenance & Billing
- `/app/societies/[id]/documents` — Documents & Records
- `/app/societies/[id]/audit` — Society Audit Log
- `/app/societies/[id]/resident-hub` — Resident Visitor Hub
- `/app/societies/[id]/reports` — Operational Reports & Analytics

---

## Consequences
- Immediate, effortless navigation across key authenticated domains on both desktop and mobile.
- Mobile society drawer is 100% visible, fully responsive, and free of horizontal clipping.
- Clean separation of concerns with modular society sub-routes, improving load performance, maintainability, and deep-linking.
- Strict compliance with `/chs-documentation-first` principles.

## Related Documentation
- [[032-frontend-application-shell-and-ux-refactor|ADR 032: Frontend Application Shell, Dedicated Routing Boundaries, and Administrative UX Refactor]]
- [[033-role-based-society-navigation-and-functional-ux|ADR 033: Role-Based Society Navigation, Feature Gating, and Functional UX]]
- [[034-authenticated-application-architecture-and-route-separation|ADR 034: Authenticated Application Architecture and Route Separation]]
- [[../architecture/03-frontend|03 — Frontend Architecture]]
