# 043 — Desktop Sidebar Collapsibility, Branding De-duplication, and Gatekeeper Workspace Routing

## Context and Problem Statement

Following the personal profile refinement and gate operations rollout, three user-experience and role-alignment gaps were identified:

1. **Duplicate Brand Logo**: On desktop viewports, both the side navigation rail (`NavigationRail`) and the top application shell header (`AppShellHeader`) rendered the `chs.` brand logo simultaneously, creating visual repetition and cluttering the header.
2. **Fixed Sidebar Width & Workspace Real Estate**: The desktop navigation rail had a fixed width (`w-56` in general workspace, `w-64` in society workspaces) with no user-controlled option to collapse into a streamlined, icon-only rail for higher content focus and wider dashboards.
3. **Mismatched Gatekeeper Dashboard Routing**: When a user authenticated as or assigned the `GATEKEEPER` role selected a society from the Workspace Dashboard (`/app/dashboard`) or Societies Directory (`/app/societies`), the application routed them to the administrative Society Overview page (`/app/societies/[id]`). Gatekeepers on duty do not manage society formation, building structure, committee rosters, or financial billing; their primary operational workspace is the Gate Dashboard (`/app/societies/[id]/gate`).

---

## Decisions

### 1. Branding De-duplication on Desktop Surfaces
- In [`frontend/src/lib/components/nav/app-shell-header.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/lib/components/nav/app-shell-header.svelte), the `chs.` brand logo is now hidden on desktop viewports (`md:hidden`) where the desktop sidebar is permanently rendered.
- On mobile viewports (`< 768px`), where the desktop navigation rail is hidden and replaced by the mobile bottom navigation bar, the top header maintains brand visibility.
- On desktop viewports, the left section of `AppShellHeader` now presents the sidebar collapse/expand toggle button alongside the subtle context label (`platform governance`, `administration`, or `workspace`).

### 2. Collapsible Desktop Navigation Rail
- **Reactive State & Persistence**: Created [`frontend/src/lib/sidebar.svelte.ts`](file:///Users/shubham/Developer/chs_neu/frontend/src/lib/sidebar.svelte.ts) utilizing Svelte 5 `$state` runes and `localStorage` key `chs_sidebar_collapsed` to persist user sidebar layout preferences across route transitions and page refreshes.
- **Visual Presentation**:
  - **Expanded State**: Full widths (`w-56` in general workspace, `w-64` in society workspace) with text labels, section categories, society identity cards, and collapse button (`PanelLeftClose`).
  - **Collapsed State**: Compact rail (`w-18`), centered navigation icons with accessible HTML `title` and `aria-label` attributes for hover tooltips, compact tenant badge, and expand button (`PanelLeftOpen`).
- Dual toggle points are provided: one in the sidebar header and one in the top `AppShellHeader`.

### 3. Dedicated Gatekeeper Workspace Routing
- **Direct Workspace Navigation**:
  - In [`frontend/src/routes/app/dashboard/+page.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/routes/app/dashboard/+page.svelte):
    - For societies where the user holds `role: GATEKEEPER`, the primary CTA is dynamically transformed from *"Enter Society Workspace"* to *"Enter Gate Operations"*, linking directly to `/app/societies/[id]/gate`.
    - Quick links for single-society gatekeepers are updated from resident billing/announcements to gatekeeper-relevant shortcuts: *Gate Console & Duty*, *Active Visits & Check-out*, and *Society Announcements*.
    - Multiple society cards for gatekeepers link directly to the Gate Console with label *"Open Gate Console"*.
  - In [`frontend/src/routes/app/societies/+page.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/routes/app/societies/+page.svelte), cards for gatekeepers link to `/app/societies/[id]/gate`.
  - In [`frontend/src/lib/components/nav/society-selector-dialog.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/lib/components/nav/society-selector-dialog.svelte) and [`frontend/src/routes/app/societies/[id]/+layout.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/routes/app/societies/[id]/+layout.svelte), switching to a society where the user is a gatekeeper opens `/gate` directly.
- **Overview Page Guard**:
  - In [`frontend/src/routes/app/societies/[id]/+page.svelte`](file:///Users/shubham/Developer/chs_neu/frontend/src/routes/app/societies/[id]/+page.svelte), if a gatekeeper directly accesses the root society overview URL, an `$effect` automatically replaces the URL with `/app/societies/[id]/gate`, preventing gatekeepers from viewing administrative society cards.
- **Navigation Streamlining**:
  - In [`frontend/src/lib/navigation.ts`](file:///Users/shubham/Developer/chs_neu/frontend/src/lib/navigation.ts), the redundant administrative `OVERVIEW` section is excluded for the `GATEKEEPER` role in `getGroupedSocietyNavItems`. Gatekeepers only receive their dedicated `OPERATIONS` group (*Gate Console* and *Announcements*).
