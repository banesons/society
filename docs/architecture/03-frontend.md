# 03 — Frontend Architecture

## Design Overview

The CHS frontend is built with **SvelteKit** and **Tailwind CSS**. It follows modern web application design practices with server-state management powered by **TanStack Query** (`@tanstack/svelte-query`).

The visual language follows an **Editorial Swiss / Minimalist Publication** design system:
- **Palette**: Warm neutral studio canvas (`#f3f3f1`), high-contrast jet-black typography (`#121212`), muted secondary tones (`#666663`), and crisp hairline borders (`#e2e2de`).
- **Typography Standard**: Strictly standardized to **`Hedvig Letters Sans`** and **`Hedvig Letters Serif`** across the entire application.
- **Hero Composition**: High-impact multi-line grotesque headline, asymmetric three-column layout (topic exploration link, architectural imagery anchor, editorial narrative paragraph).
- **Navigation**: Clean lowercase branding (`chs.`), centered domain navigation menus, and right-aligned authentication status.

## Core Responsibilities & Boundaries

- **Local UI State**: Managed via native Svelte reactivity (`$state`, stores, or props).
- **Server State**: Managed exclusively through TanStack Query queries and mutations. Components do not cache or hold raw server responses in ad-hoc global variables.
- **Central API Client**: All HTTP traffic flows through `src/lib/api/client.ts`. Direct, unconfigured `fetch()` calls inside UI components are forbidden.

## Project Structure

```text
frontend/
├── src/
│   ├── lib/
│   │   ├── api/
│   │   │   ├── client.ts          # Central HTTP client (headers, baseURL, error handling)
│   │   │   └── platform.ts        # Platform administration API caller module
│   │   ├── components/
│   │   │   ├── admin/             # Platform admin components (roster, audit viewer, dialogs)
│   │   │   ├── auth/              # Auth dialogs & session management (logout-all)
│   │   │   ├── nav/               # Navigation components (society-switcher)
│   │   │   └── ui/                # Reusable primitive UI components (button, card, badge)
│   │   ├── auth.svelte.ts         # Reactive Svelte 5 AuthStore & session handling
│   │   ├── firebase.ts            # Shared Firebase App & Performance Monitoring initialization
│   │   └── push.ts                # Web Push notification registration & FCM token lifecycle
│   └── routes/
│       ├── +layout.svelte         # Root application shell, editorial navbar & query provider
│       ├── +page.svelte           # Editorial product landing page (asymmetric grid, metrics)
│       ├── login/+page.svelte     # Single authentication entry point (role-based redirect)
│       ├── admin/+page.svelte     # Protected Super Admin console (roster, audit, sessions)
│       ├── developer/+page.svelte # Developer diagnostics & API testing console
│       └── unauthorized/+page.svelte # Access-denied / unassigned role view
├── static/
│   └── hero-architecture.jpg      # High-resolution architectural photography asset
├── package.json
└── vite.config.ts
```

## Responsive & Adaptive Navigation System

The navigation and shell layout in `src/routes/+layout.svelte` is designed to be fully responsive and adaptive:

1. **Brand Identity (Left)**:
   - Reusable `Logo` component (`src/lib/components/ui/logo.svelte`) rendering lowercase bold branding `chs.` with an interactive accent underline bar animating on hover (`group-hover:w-full`).
2. **Domain Navigation Menus (Center / Inline)**:
   - Random/domain menus: `societies`, `governance`, `members`, `gatekeeping`, `compliance`, `about`.
   - Desktop view: Spaced horizontal text links in lowercase with soft hover transitions.
3. **Authentication Controls (Right)**:
   - Placed consistently on the far right of the navigation bar.
   - **Unauthenticated**: Sleek editorial link with persistent underline (`Login`) that navigates to `/login`.
   - **Authenticated**: Renders society switcher, verified mobile phone badge, Super Admin console shortcut, and device logout controls.
4. **Adaptive Mobile Viewport**:
   - On screens `< 768px` (mobile/tablet), domain links collapse into an adaptive animated drawer triggered by a minimalist hamburger button.
   - Full touch targets, developer diagnostics link, and authentication actions remain accessible in the drawer without layout shifts or horizontal overflows.
5. **Standalone Routes Exemption**:
   - Focused, dedicated flows such as `/login` conditionally omit the global navigation bar and footer to provide a distraction-free, full-height card experience.

## UI Primitive Design Tokens

Reusable components in `src/lib/components/ui` embody the editorial design principles:

- **Logo (`logo.svelte`)**:
  - Single source of truth for the platform wordmark.
  - Features an interactive animated underline (`h-[2.5px] w-6 bg-neutral-950 transition-all duration-300 group-hover:w-full`).
  - Standardized sizing options (`sm`, `md`, `lg`). Used identically across layout headers, footers, and standalone auth screens. Any branding modification in `logo.svelte` automatically updates the entire application.
- **Button (`button.svelte`)**:
  - `primary`: Jet-black pill (`bg-neutral-950 text-white rounded-full hover:bg-neutral-800`).
  - `secondary`: Architectural neutral (`bg-neutral-200 text-neutral-900 rounded-full`).
  - `outline`: Crisp hairline border (`border-neutral-300 bg-white/60 text-neutral-900 rounded-full hover:bg-neutral-100`).
  - `ghost`: Transparent pill with subtle hover state (`hover:bg-neutral-100 text-neutral-800 rounded-full`).
  - `link`: Editorial inline link with animated underline.
- **Card (`card.svelte`)**:
  - Pure white background (`bg-white/90`), hairline border (`border-neutral-200/90`), gentle curvature (`rounded-2xl` to `rounded-3xl`), and subtle elevation (`shadow-xs`).
- **Badge (`badge.svelte`)**:
  - Refined rounded-full pills in monochrome or soft semantic tints (`default`, `success`, `warning`, `destructive`, `outline`).

---

## Route Boundaries & Shell Architecture

Per [[../decisions/032-frontend-application-shell-and-ux-refactor|ADR 032]] and [[../decisions/034-authenticated-application-architecture-and-route-separation|ADR 034]], the CHS frontend strictly separates the public marketing website from the authenticated application surface:

1. **Public Marketing Surface (`/`, `/developer`)**:
   - `/`: Renders public marketing content with editorial typography, value propositions, and a clean public footer. No authenticated administrative links or application state leak into the public header.
   - `/developer`: Dedicated diagnostics console for platform developers.
2. **Canonical Authenticated Application Namespace (`/app/*`)**:
   - `/app`: Canonical authenticated entry point. Redirects authenticated users to `/app/dashboard` and unauthenticated users to `/app/login`.
   - `/app/dashboard`: Canonical authenticated workspace landing page, displaying accessible societies, assigned roles, notifications, and next actions with real data.
   - `/app/societies`: Dedicated accessible society selector.
   - `/app/societies/:id`: Canonical society operational management console wrapped by its layout shell per [[../decisions/022-society-application-shell-and-navigation|ADR 022]] and role-based navigation per [[../decisions/033-role-based-society-navigation-and-functional-ux|ADR 033]].
   - `/app/admin`: Platform Administration Console (bootstrap, society roster, platform administrators, immutable audit trail, active device session revocation).
   - `/app/login`: Deliberate unauthenticated entry point supporting safe internal `returnTo` parameters.
3. **Legacy Route Compatibility & Fragment Preservation**:
   - `/login` client-redirects to `/app/login`.
   - `/admin` client-redirects to `/app/admin`.
   - `/society/:id` and `/societies/:id` client-redirect to `/app/societies/:id`, preserving URL hash fragments (`#resident-maintenance`, etc.).
   - All `/app/*` routes enforce `noindex, nofollow` to prevent indexing of protected application content.
3. **Dedicated Authenticated Application Shell (`AppShellHeader`)**:
   - Excludes all marketing domain links (`societies`, `governance`, `members`, etc.) from the application header.
   - Left: Minimalist CHS wordmark (`<Logo size="md" />`).
   - Right: Strictly two persistent interactive controls:
     1. **Notification Center (`NotificationCenter`)**: Bell icon with real-time unread count, popover drawer, and push registration toggle.
     2. **Profile / Account Menu**: Desktop avatar button opening an editorial popover with formatted phone, verified role badges, and session management redirect; on mobile viewports, the avatar directly navigates to `/app/profile` without popup overlays.
     3. **Session & Device Actions**: Single-device sign-out (confirmed via `SignOutDialog`) and multi-device revocation (`LogoutAllDialog`) are anchored inside the Personal Profile view (`/app/profile`).
   - Removed persistent phone number pills and standalone `Platform Admin` buttons from top header.
4. **Common Toast Notification System (`src/lib/toast.svelte.ts`)**:
   - Replaces intrusive full-width banners with a non-layout-shifting toast overlay (`toast.success`, `toast.info`, `toast.warning`, `toast.error`).
   - Auto-dismisses with configurable timeout, manual close buttons, and ARIA live-region polite/assertive announcements.
   - When FIFO session replacement occurs ([[../decisions/011-two-device-limit|ADR 011]]), triggers a single concise toast: *"Signed in successfully. Your oldest session was signed out."*
5. **Application Footer Policy**:
   - Authenticated application views (`/app`, `/app/admin`, `/society/:id`) completely omit marketing footers, allocating 100% of screen real estate to administrative workflows.
   - Public marketing footer removes internal architectural labels (`"Documentation-first engineering."`, `"Two-device strict session limit enabled."`).

## Documented Post-Login Routing Resolution Policy

Post-login destination resolution (`src/lib/routing.ts`) evaluates authenticated backend data from `GET /api/v1/auth/me`:

1. **Platform Super Admin (`SUPER_ADMIN`)**:
   - Routes to `/admin` by default.
   - Entry into society tenants occurs exclusively through explicitly permitted governance inspection actions.
2. **Platform Admin (`ADMIN`)**:
   - Routes to `/admin` by default.
   - Receives zero implicit access to society tenants; cross-society operations return HTTP 403.
3. **Zero Accessible Societies**:
   - Ordinary authenticated user with empty `society_assignments` routes to `/unauthorized` with clear civic access status.
4. **Exactly One Accessible Society**:
   - Routes directly to `/society/${assignment.society_id}`. User is not forced through an unnecessary single-option selection.
5. **Multiple Accessible Societies**:
   - Opens mandatory `SocietySelectorDialog`. Displays only authenticated societies with name, code, location, and user's role. Non-dismissible until a valid tenant is chosen.

## Tenant Isolation & Cache Partitioning Rules

When switching societies or revoking tenant access:
1. **Query Purge**: TanStack Query cache is purged via `queryClient.removeQueries({ queryKey: ['society', previousId] })` and invalidated to prevent cross-tenant data bleed.
2. **Path & Context Isolation**: All mutation and query payloads use the verified URL parameter `societyId`.
3. **Fail-Closed Handling**: HTTP 401/403 errors immediately clear tenant data and display the authorized status banner.

---

## Administration & Internal Society Console Standards

All administrative dashboards (`/admin`) and internal society management views (`/society/[id]`) strictly adhere to the same Editorial Swiss publication language:

### 1. Platform Administration Console (`/admin`)
- **Header Banner**: Studio canvas card with clean title, super admin identity indicator, and tabbed sub-navigation.
- **Tabbed Views**:
  - `Societies`: Society roster table, status filters (All, Active, Pending, Suspended), quick search, registration dialog, society details, and activation/suspension flows.
  - `Administrators`: Platform admin assignment, role toggling, creation dialogs, and credential inspection.
  - `Audit Trail`: Immutable audit ledger table with event categorisation, actor attribution, timestamping, and payload inspection.
  - `Account & Sessions`: Multi-device active session viewer with Ed25519 device signatures and global revocation ("Logout All").

### 2. Internal Society Management Console (`/society/[id]`)
- **Header & Metrics**: Pure white editorial cards displaying registration codes, verified city/state metadata, and high-level unit/resident counts.
- **Society Sub-Modules**:
  - **Member Directory (`member-roster.svelte`)**: Resident directory with ownership type badges (Owner, Tenant, Family), contact information, and registration modal.
  - **Unit Relationships (`unit-relationship-dialog.svelte`)**: Assigning members to specific flats with unit relationship tagging.
  - **Departure Management (`departure-dialog.svelte`)**: Departure manifest with impact calculations, outstanding obligations check, and historical archival.
  - **Committee Governance (`committee-roster.svelte`)**: Managing office-bearers (Chairman, Secretary, Treasurer), appointments, terms, and custom designation catalogs.
  - **Physical Formation Wizard (`formation-wizard.svelte`)**: Guided 5-step stepper for defining buildings, wings, floors, auto-generating unit identifiers, physical gates, and staffing headcount requirements.

---

## Design System Specifications for New Development

Every new page, modal, or component created in CHS **must** adhere to the following principles:

### A. Typography & Font Standard (Strict Enforcement)
- **Only Two Fonts in the Entire Project**: Strictly **`'Hedvig Letters Sans'`** and **`'Hedvig Letters Serif'`**.
- **Absolute Prohibition**: No other font families (e.g. Inter, Roboto, Plus Jakarta Sans, Arial, system fallbacks) may be imported or specified anywhere in the application.
- **No Monospace for UI Text**: Monospace (`font-mono`) is strictly restricted to raw numbers/identifiers (phone digits, OTP inputs, UUIDs, technical codes). Section kickers, labels, badges, and headers must use `font-sans` (`Hedvig Letters Sans`).
- **Casing Rule (No All-Caps Strings)**: All text elements, headlines, tags, section kickers, and buttons must use either **PascalCase** (Title Case) or **all small (lowercase)**. ALL CAPS strings (e.g., `WELCOME TO CHS.`, `PLATFORM AUTHENTICATION`) are strictly prohibited across all screens.
- **Usage Mapping**:
  - Headlines & Major Section Titles: `font-serif tracking-tight text-neutral-950` or `font-sans font-bold tracking-tight text-neutral-950`.
  - Body Text, Controls, & Inputs: `font-sans text-neutral-800` or `text-neutral-600`.
  - Codes, Phone Numbers, UUIDs: `font-mono text-neutral-900`.

### B. Color Tokens & Theme
- **Canvas / Page Background**: Off-white studio warm neutral (`#f3f3f1` / `bg-[#f3f3f1]`).
- **Surface / Card Background**: Pure white translucent (`bg-white/90` or `bg-white`), bordered by hairline border (`border-neutral-200`).
- **Dark Containers**: **Forbidden on general layout surfaces**. All modals, cards, tabs, and tables must be light and airy. Only developer code/JSON inspectors in `/developer` may use high-contrast dark monospace viewports.
- **Accent & Primary**: Jet black (`#121212` / `bg-neutral-950 text-white`).
- **Muted & Meta**: `#666663` (`text-neutral-500` / `text-neutral-400`).
- **Semantic Accents**:
  - Success / Active: `bg-emerald-50 text-emerald-800 border-emerald-200`
  - Warning / Pending: `bg-amber-50 text-amber-800 border-amber-200`
  - Danger / Error / Suspended: `bg-rose-50 text-rose-800 border-rose-200`

### C. Modal Dialog Pattern
All modal dialogs must follow this unified markup structure:
```svelte
<div class="fixed inset-0 z-50 flex items-center justify-center bg-neutral-950/40 p-4 backdrop-blur-xs">
  <div class="relative w-full max-w-lg rounded-3xl border border-neutral-200 bg-white p-6 shadow-2xl text-neutral-900">
    <!-- Header -->
    <div class="flex items-center justify-between border-b border-neutral-200 pb-4">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-900">
          <IconComponent class="h-5 w-5" />
        </div>
        <div>
          <h3 class="font-serif text-lg font-medium tracking-tight text-neutral-950">Dialog Title</h3>
          <p class="text-xs text-neutral-500">Descriptive subtitle explaining the civic or administrative action</p>
        </div>
      </div>
      <button onclick={onClose} class="rounded-full p-2 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors">
        <X class="h-5 w-5" />
      </button>
    </div>

    <!-- Body / Form Fields -->
    <form class="mt-5 space-y-4">
      <div>
        <label class="block text-xs font-medium text-neutral-700">Field Label <span class="text-rose-500">*</span></label>
        <input class="mt-1.5 w-full rounded-xl border border-neutral-300 bg-white px-3.5 py-2.5 text-xs text-neutral-900 placeholder-neutral-400 transition-colors focus:border-neutral-950 focus:ring-1 focus:ring-neutral-950 focus:outline-hidden" />
      </div>

      <!-- Footer Actions -->
      <div class="mt-6 flex items-center justify-end gap-3 border-t border-neutral-200 pt-4">
        <Button variant="ghost" size="sm" onclick={onClose}>Cancel</Button>
        <Button variant="primary" size="sm" type="submit">Confirm Action</Button>
      </div>
    </form>
  </div>
</div>
```

### D. Table & Roster Pattern
All lists and rosters must use clean light tables:
- **Wrapper**: `overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-xs`
- **Header**: `bg-neutral-50/60 text-[11px] font-medium uppercase tracking-wider text-neutral-500`
- **Rows**: `divide-y divide-neutral-200 text-xs text-neutral-800`
- **Hover State**: `hover:bg-neutral-50/50 transition-colors`
### E. Component & Element Reusability (Mandatory Principle)
- **Always Reuse Elements as Much as Possible**: Do NOT duplicate layout frames, visual cards, branding blocks, or interaction patterns across screens or workflow steps.
- **Visual Card Reuse**: High-fidelity visual columns (such as `AuthVisualCard` in `src/lib/components/auth/auth-visual-card.svelte`) must be encapsulated and shared across multi-step flows (e.g., Phone Entry and OTP Verification) to maintain a rock-solid, persistent visual shell while only switching interactive form state.
- **Branding Mark Reuse**: The `chs.` wordmark must always use `<Logo />` (`src/lib/components/ui/logo.svelte`) rather than raw text so animation behaviors and stylistic tweaks remain uniform across the entire application.

### F. Viewport & Device Detection Helper (`src/lib/screen.svelte.ts`)
- **Centralized Screen Store**: Provides a unified, reactive Svelte 5 store based on viewport width:
  - `DeviceType` enum: `Mobile` (< 768px), `Tablet` (768px - 1023px), `Desktop` (>= 1024px).
  - Getters: `screen.isMobile`, `screen.isTablet`, `screen.isDesktop`.
- **Consistency**: Components must consume `screen` instead of creating ad-hoc `window.innerWidth` listeners or inconsistent media query thresholds.

### G. Mobile Fullscreen & Layout Adaptability
- **Mobile Fullscreen Shell**: On mobile screens (`< 768px`), authentication pages and standalone dialog shells render **full bleed** (`rounded-none`, `border-0`, `min-h-screen`, `p-4`) instead of desktop floating cards with outer margins. This prevents double scrollbars and maximizes mobile form readability.
- **Tactile Navigation**: Dismiss or back actions (e.g. `← back to home`) must remain prominent, tactile pills with directional hover animations (`group-hover:-translate-x-1`).

### H. Input Constraints & Display Formatting
- **Phone Number Field Constraints**:
  - Max length: strictly 10 digits (`maxlength="10"`).
  - Filter: integer digits only (`[0-9]`), stripping non-numeric characters on keypress/input.
  - Visual prefix: non-editable `+91` adornment.
- **Indian National Display Standard**:
  - Full phone formatting via `formatFullPhone()` in `src/lib/phone.ts` strictly outputs 5-space-5 national spacing: `+91 99123 21321`.
  - Used consistently across OTP verification prompts, audit logs, and member rosters.

### I. Typography Scale & Micro-Animations
- **Massive Headline Scale**: Standalone authentication screens use high-impact headlines (`text-5xl sm:text-6xl lg:text-7xl font-extrabold tracking-tight leading-[0.9]`) with the signature animated underline on the inline `chs.` brand mark.
- **Interactive Directional Indicators**: Action buttons with directional icons animate on hover (`group-hover:translate-x-1.5` on right arrow).
- **Graceful Timers**: Countdown badges (such as OTP `Resend in 55s`) feature elegant animated ping radar indicators (`animate-ping`) rather than plain static text.

---

## Developer Checklist for New Features

Before committing any frontend feature or page, verify each item:

- [ ] **Branding & Logo**: Use `<Logo />` from `src/lib/components/ui/logo.svelte` for any representation of the `chs.` brand mark to preserve animated hover consistency across the application.
- [ ] **Two-Font Rule**: Fonts use **strictly `'Hedvig Letters Sans'` and `'Hedvig Letters Serif'`**. No third-party font imports or fallbacks. Monospace is restricted to raw numbers/codes.
- [ ] **Casing Policy**: All text, titles, buttons, tags, and kickers are in **PascalCase** or **all small (lowercase)**. No all-caps strings anywhere.
- [ ] **Background & Colors**: Canvas uses `#f3f3f1`; cards and modals use `bg-white` and `border-neutral-200`. No `bg-slate-900`, `bg-slate-950`, or dark container backgrounds.
- [ ] **Buttons**: Primary actions use `variant="primary"` (pill rounded-full black). Auxiliary/dismiss actions use `variant="ghost"` or `variant="outline"`.
- [ ] **Forms**: All inputs have `rounded-xl border border-neutral-300 bg-white text-neutral-900 focus:border-neutral-950`.
- [ ] **Modals**: Backdrops use `bg-neutral-950/40 backdrop-blur-xs`, containers use `rounded-3xl border border-neutral-200 bg-white p-6 shadow-2xl`.
- [ ] **Element Reusability**: Prioritize element and component reuse. Never duplicate layout shells or visual columns (`AuthVisualCard`, `Logo`).
- [ ] **Viewport & Mobile Shell**: Leverage `src/lib/screen.svelte.ts` (`isMobile`, `isDesktop`) for responsive logic. Ensure mobile viewports render full bleed (`rounded-none`, `border-0`, `min-h-screen`) without awkward double padding.
- [ ] **Input Constraints**: Constrain Indian mobile inputs strictly to 10 digits and numbers only; format with 5-space-5 spacing (`+91 99123 21321`).
- [ ] **API Access**: All API requests pass through `src/lib/api/` using typed methods on `platformApi` or `societyApi`.
- [ ] **Documentation**: Any newly introduced routes, workflows, or roles are documented in `/docs` per `chs-documentation-first`.

---

## Central API Client

The centralized client in `src/lib/api/client.ts` encapsulates:
1. Base API URL injection from public environment variables (`PUBLIC_API_URL`).
2. Inclusion of standard headers (e.g. `Content-Type: application/json`).
3. Outbound request ID propagation and inbound request ID logging.
4. Parsing and unwrapping of standard CHS response envelopes (`{ success, data, meta }` or `{ success, error, request_id }`).
5. Uniform error creation with human-readable error messages.

## Authentication & FIFO Session Management Flow

1. **Authentication Flow (`/login`)**:
   - Dedicated standalone split-card page (`max-w-6xl` rounded container) without distracting global navbar or footer.
   - Reusable `Logo` component (`src/lib/components/ui/logo.svelte`) with interactive underline hover animation used consistently across layouts.
   - **Left Column**: Top `← back to home` link, `Logo`, large grotesque headline (`Welcome to chs.` / `Verify your passcode.`), phone & OTP forms with "Continue", and "Don't have an account? Sign up" text.
   - **Right Column**: Dark aesthetic card with layered community metrics and friendly non-technical trust indicators ("Verified & secure", "Safe community access").
   - **Copy & Casing Policy**: Strictly PascalCase or lowercase text; all-caps strings are forbidden.
   - Automatically navigates post-login based on platform privileges (`SUPER_ADMIN` → `/admin`, others → `/unauthorized`).
2. **Automatic FIFO Session Replacement UX**:
   - Eliminates third-login rejection barriers (`SESSION_LIMIT_REACHED`).
   - When the backend performs FIFO replacement (`session_replaced: true`), the frontend records `auth.sessionReplaced = true`.
   - `+layout.svelte` renders an informational dismissible banner: *"Signed in successfully. Your oldest session was signed out to keep your account limited to two active devices."*
   - Normal role-based routing proceeds without requiring manual user intervention or confirmation modals.

## Client-Side Firebase Performance Monitoring & SDK Lifecycle

Per [[../decisions/030-client-performance-monitoring|ADR 030]], the frontend integrates Firebase Performance Monitoring to capture real-user performance metrics, Core Web Vitals, and network request latencies:

1. **Modular SDK Configuration**:
   - Uses the official `firebase@^11` modular packages:
     ```typescript
     import { initializeApp, getApps, getApp } from "firebase/app";
     import { getPerformance } from "firebase/performance";
     ```
   - Configuration is consolidated in `src/lib/firebase.ts`, driven by browser-safe environment variables (`PUBLIC_FIREBASE_*`).

2. **Isomorphic Browser Guard**:
   - SvelteKit executes layout and component logic during both Server-Side Rendering (SSR) and client navigation.
   - `getPerformance()` requires DOM and Navigation Timing APIs. All Performance Monitoring initialization is guarded strictly by `typeof window !== 'undefined'` and invoked from `onMount()` in `src/routes/+layout.svelte`.
   - SSR code paths return `null` safely without throwing runtime errors.

3. **Unified App Singleton**:
   - `getFirebaseApp()` ensures that only one `FirebaseApp` instance exists across the client lifecycle.
   - Both Firebase Cloud Messaging (`src/lib/push.ts`) and Firebase Performance (`src/lib/firebase.ts`) share this single instance, preventing redundant network connections.

4. **Custom Trace Instrumentation**:
   - `traceOperation<T>(traceName: string, operation: () => Promise<T>): Promise<T>` provides standardized custom tracing for critical operations (such as gate check-ins, maintenance bill generation, and payment allocations).

## Shared Responsive Modal System

Per [[../decisions/036-shared-responsive-modal-system|ADR 036]], CHS Neu standardizes modal dialogs and confirmation sheets on a single reusable component (`src/lib/components/ui/responsive-modal.svelte`):

1. **Desktop & Tablet Contract**:
   - Fixed top offset: strictly `180px` from the viewport top (`top: 180px;`). It is not vertically centered.
   - Horizontal centering: `left: 50%; transform: translateX(-50%);`.
   - Max-width bounding: `w-[min(520px, calc(100vw - 48px))]` with sizes `sm`, `md`, `lg`, and `xl`.
   - Max-height bounding: `calc(100vh - 220px)` with internal scroll area.
2. **Mobile Bottom Sheet Contract**:
   - Fixed bottom anchor: `position: fixed; inset-x: 0; bottom: 0;`.
   - Safe-area padding: `pb-[max(1.25rem, env(safe-area-inset-bottom))]`.
   - Max-height bounding: `max-h-[85vh]` with top rounded corners (`rounded-t-[2rem]`) and subtle drag handle.
3. **Responsive State Management**:
   - Driven reactively by `screen.isMobile` (`src/lib/screen.svelte.ts`, breakpoint `< 768px`).
4. **Accessibility Guarantees**:
   - Full keyboard focus trapping, initial auto-focus, and focus restoration to the triggering element upon closure.
   - Body scroll locking (`document.body.style.overflow = 'hidden'`) with guaranteed cleanup on close or unmount.
   - Backdrop dismiss and Escape dismiss when `isDismissible` is active.

---

## Application Navigation Architecture (ADR 040)

Per [[../decisions/040-application-navigation-breadcrumbs-and-account-actions|ADR 040]], CHS Neu organizes navigation responsibilities into three distinct architectural tiers:

1. **Desktop Navigation Rail (`src/lib/components/nav/navigation-rail.svelte`)**:
   - **General Workspace Mode**: Displays primary workspace links (*Dashboard*, *Societies*, and *Platform Administration* when authorized), keeping personal profile in the sidebar footer identity card.
   - **Contextual Society Mode (`/app/societies/[id]/*`)**: Automatically expands into the selected tenant's contextual navigation. Displays:
     - Return link (`← All Societies`).
     - Society Identity banner (name, code, location, and verified role badge).
     - Grouped submodules (*Overview*, *People*, *Operations*, *Insights*) with role-based feature gating and exact route-aware active state highlighting.
2. **Hierarchical Breadcrumbs (`src/lib/components/nav/breadcrumbs.svelte`)**:
   - Placed below the top header and above primary content (`Workspace / {Society Name} / {Module Name}`).
   - Dynamically resolves society names from entity state, avoiding raw UUIDs.
   - Intermediate crumbs are interactive navigation links; the leaf item represents the active page with `aria-current="page"`.
3. **Sign-Out Confirmation (`src/lib/components/auth/sign-out-dialog.svelte`)**:
   - Normal single-device sign-out is protected by an explicit confirmation dialog built on `ResponsiveModal`.
   - Distinct from global session termination (`LogoutAllDialog`).

## Related Documentation
- [[01-system-overview|System Overview]]
- [[../api/01-api-overview|API Overview]]
- [[../decisions/001-go-sveltekit-supabase|ADR 001: Core Stack]]
- [[../decisions/011-two-device-limit|ADR 011: Two-Device Limit with Automatic FIFO Replacement]]
- [[../decisions/029-notifications|ADR 029: Notification System Architecture]]
- [[../decisions/030-client-performance-monitoring|ADR 030: Client-Side Performance Monitoring]]
- [[../decisions/032-frontend-application-shell-and-ux-refactor|ADR 032: Frontend Application Shell and Administrative UX Refactor]]
- [[../decisions/036-shared-responsive-modal-system|ADR 036: Shared Responsive Modal System and Adaptive Presentation]]
- [[../decisions/038-navigation-rail-and-modular-society-routes|ADR 038: Navigation Rail Architecture, Dedicated Notifications Screen, and Modular Society Sub-Routes]]
- [[../decisions/040-application-navigation-breadcrumbs-and-account-actions|ADR 040: Application Navigation Architecture, Contextual Society Submenu, Hierarchical Breadcrumbs, and Sign-Out Confirmation]]
- [[../decisions/045-responsive-application-shell-and-reload-auth-persistence|ADR 045: Responsive Application Shell, Operations Hub Redesign, and Reload Auth Persistence]]

---

## Responsive Application Shell & In-Flight Auth Refresh Deduplication (ADR 045)

Per [[../decisions/045-responsive-application-shell-and-reload-auth-persistence|ADR 045]]:
1. **CSS-First Shell Visibility**: Desktop navigation rail uses `hidden md:flex`, while mobile bottom navigation uses `flex md:hidden`. Mobile viewports (< 768px) do not reserve any layout margin or column space for desktop rails, allowing the primary content area to occupy the complete viewport (`min-w-0 w-full overflow-x-hidden`).
2. **In-Flight Refresh Promise Deduplication**: When page reload triggers parallel authenticated queries, the client intercepts concurrent 401s using a single deduplicated `refreshPromise` in `AuthStore`. This prevents race conditions against backend single-use rotating refresh tokens, ensuring reload persistence without accidental session eviction.
3. **Operations Hub Information Architecture**: Society dashboard `/app/societies/[id]` is restructured as an editorial operations hub displaying real operational statistics (`GET /societies/{id}/dashboard`), urgent pending registration alerts, modular navigation links, and recent announcements without multi-console bloat.

---

## Progressive Web Application (PWA) & Offline Shell (ADR 046)

Per [[../decisions/046-progressive-web-app-architecture|ADR 046]]:
1. **Web App Manifest**: Full compliance with W3C Web App Manifest at `/manifest.webmanifest` specifying standalone display mode, theme colors (`#121212`), background colors (`#f3f3f1`), operational shortcuts (Gate, Passes, Announcements), and high-resolution icons (192px, 512px, maskable).
2. **Unified Service Worker**: `/firebase-messaging-sw.js` is enhanced to handle both Firebase Cloud Messaging (FCM HTTP v1) push events and PWA caching lifecycle:
   - Pre-caches core brand and offline shell assets.
   - Network-first navigation with graceful fallback to branded `/offline.html`.
   - Stale-while-revalidate caching for fonts, icons, and static assets.
3. **Install Experience**: Boot-level service worker registration in `+layout.svelte` and reactive installation tracking (`pwa.svelte.ts`) allowing users on mobile and desktop browsers to install CHS to their home screen or dock.

