# ADR 032: Frontend Application Shell, Dedicated Routing Boundaries, and Administrative UX Refactor

## Status
Accepted

## Context
Prior to this architectural refactor:
1. **Public/Application Shell Mixing**:
   - The root SvelteKit layout (`src/routes/+layout.svelte`) rendered a global marketing header (with links to `societies`, `governance`, `members`, `gatekeeping`, `compliance`, `about`) and a marketing footer (with promotional copy and internal implementation status badges) on top-level authenticated administrative pages such as `/admin`.
   - The platform lacked a clear boundary separating the public marketing website (`/`) from the authenticated application surface (`/app`, `/app/admin`, `/society/:id`).
2. **Ambiguous Dashboard Destination**:
   - Platform Administration displayed `"Return to Main Dashboard"` pointing to `/` (the public marketing landing page) instead of an authenticated application home.
   - `getContextualBackDestination` in `src/lib/routing.ts` returned `{ href: '/', label: 'Return to Home' }` as fallback.
3. **Intrusive Session Replacement Feedback**:
   - When a user established a session that triggered FIFO session replacement under the two-device policy ([[011-two-device-limit|ADR 011]]), the application displayed a persistent, full-width top banner across the screen.
4. **Header Clutter & Security Leaks**:
   - The header exposed raw administrative pills (such as `Platform Admin`, verified phone numbers, and individual text links for `logout` and `logout all`).
   - The administrative screen displayed internal metadata badges (`Security Domain: Platform Layer (S02)`), exposing internal architectural labels directly to the UI.

---

## Decision

### 1. Separation of Public Marketing Surface from Authenticated Application Namespace
- **Public Marketing Surface**:
  - `/`: Public marketing homepage with editorial headlines, product value propositions, and public legal/company footer.
  - `/login`: Standalone authentication card without distracting marketing or application navigation.
  - `/developer`: Diagnostic utility interface for API health and token inspection.
- **Canonical Authenticated Application Namespace (`/app`)**:
  - `/app`: Canonical entry point and unified dashboard for authenticated users.
    - For `SUPER_ADMIN` and `ADMIN`: Offers quick access to Platform Administration, ecosystem health metrics, and accessible societies.
    - For `SOC_ADMIN` and residents: Displays accessible society cards, quick switcher, and active tenant status.
    - Unauthenticated requests to `/app` redirect cleanly to `/login`.
  - `/app/admin`: Platform Administration Console (bootstrap, society roster, platform administrators, immutable audit trail, active device session revocation).
  - `/admin`: Preserved as a transparent compatibility redirect to `/app/admin`, maintaining bookmarks and deep links.
  - `/society/:id`: Dedicated society operational management console wrapped by its dedicated layout shell per [[022-society-application-shell-and-navigation|ADR 022]].

### 2. Dedicated Authenticated Application Shell (`AppShellHeader`)
- **Isolation from Marketing Navigation**:
  - Marketing navigation links (`societies`, `governance`, `members`, `gatekeeping`, `compliance`, `about`) are strictly excluded from all authenticated application views.
- **Left Identity**:
  - Clean `chs.` wordmark using `<Logo />`.
  - Contextual location badge or back button when nested inside sub-modules.
- **Right Header Actions (Strict Persistent Icons)**:
  - Exactly two persistent interactive controls are displayed on the right:
    1. **Notification Center (`NotificationCenter`)**: Bell icon with unread count badge, notification panel, push registration toggle, and empty state.
    2. **Profile / Account Menu**: Accessible avatar/user button triggering an editorial popover with:
       - Formatted phone number (`+91 99670 89101`).
       - Platform and society role badges (`SUPER_ADMIN`, `SOC_ADMIN`).
       - Quick navigation to Dashboard (`/app`) and Platform Administration (`/app/admin` when permitted).
       - Single-device "Sign Out" action (`auth.logout()`).
       - Destructive multi-device "Log out all devices" action triggering `LogoutAllDialog`.
  - Persistent phone number pills and standalone `Platform Admin` buttons are removed from the global top header.

### 3. Reusable Common Toast Notification System (`src/lib/toast.svelte.ts`)
- Replaces intrusive full-width layout banners with a non-obstructive toast system:
  - **Methods**: `toast.success(message, options?)`, `toast.info(message, options?)`, `toast.warning(message, options?)`, `toast.error(message, options?)`.
  - **Fixed Viewport Overlay**: Positioned at top-right on desktop (`top-4 right-4`) and top-center on mobile, preventing any layout shifts.
  - **Auto-Dismissal**: Informational and success toasts dismiss automatically after 5000ms; warnings and errors remain visible longer (8000ms) or until manually dismissed.
  - **Accessibility**: Uses standard ARIA live-region behavior (`role="status"` with `aria-live="polite"` for non-urgent notifications; `role="alert"` with `aria-live="assertive"` for critical errors). Respects `prefers-reduced-motion`.
  - **Timer Cleanup**: Auto-dismiss timers are cleared when toasts are manually closed or when components unmount.

### 4. Session-Replacement Toast Feedback
- Per [[011-two-device-limit|ADR 011]], CHS enforces a strict maximum of two active sessions per user using FIFO replacement.
- When `POST /api/v1/auth/otp/verify` returns `session_replaced: true`:
  - The frontend triggers a single, concise informational toast:
    > *"Signed in successfully. Your oldest session was signed out."*
  - The full-width amber banner is completely removed.
  - The notification is displayed once and does not reappear across subsequent page navigations or refreshes unless a new replacement event occurs.

### 5. Canonical Dashboard Navigation & "Back to Dashboard"
- Centralized route constants are defined in `src/lib/routing.ts` (`ROUTES.APP = '/app'`, `ROUTES.ADMIN = '/app/admin'`, `ROUTES.LOGIN = '/login'`, `ROUTES.HOME = '/'`).
- The link on Platform Administration is renamed to **`Back to Dashboard`** and points directly to `/app`.
- `getContextualBackDestination` fallback points to `/app` for authenticated users.

### 6. Application Footer Policy
- Authenticated application views (`/app`, `/app/admin`, `/society/:id`) omit marketing footers completely, allocating full viewport height to workflows.
- The marketing footer on `/` removes internal architectural claims (`"Documentation-first engineering."`, `"Two-device strict session limit enabled."`) and presents a clean, minimalist footer.

### 7. Removal of Internal Security Labels
- The decorative badge `"Security Domain: Platform Layer (S02)"` is removed from user-facing screens.
- All underlying security enforcement (RBAC, platform authorization, Ed25519 signatures, audit logging) remains active in Go backend middleware and frontend route guards.

---

## Consequences
- Clean separation between public acquisition surfaces and authenticated management spaces.
- Polished, enterprise-grade administrative UX with zero visual clutter and intuitive profile/session controls.
- Unobtrusive, accessible toast feedback for transient security events.
- Full preservation of existing authorization, FIFO session revocation, and multi-tenant isolation.

## Related Documentation
- [[../architecture/03-frontend|03 — Frontend Architecture]]
- [[011-two-device-limit|ADR 011: Two-Device Limit with Automatic FIFO Replacement]]
- [[022-society-application-shell-and-navigation|ADR 022: Society Application Shell and Navigation Architecture]]
- [[012-platform-role-separation|ADR 012: Platform Role Separation]]
