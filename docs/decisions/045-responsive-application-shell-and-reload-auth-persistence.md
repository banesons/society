# ADR 045: Responsive Application Shell, Operations Hub Redesign, and Reload Auth Persistence

## Status
Accepted

## Context
Production testing of CHS Neu identified three critical UX and reliability issues:

1. **Responsive Shell & Alignment Defects**:
   - At mobile widths (320px–375px), the desktop sidebar remained visible as an awkward vertical rail, pushing page content offscreen and creating horizontal clipping.
   - Breakpoint mismatch between components (`sm` at 640px vs `md` at 768px).
   - Component rendering relied on client-side JS runtime state (`screen.isMobile`) causing hydration flashes and race conditions during viewport emulation.
   - Fixed desktop widths and unconstrained flex containers (`min-width: auto`) caused tables and cards to overflow the screen.
   - Desktop hover tooltips triggered on mobile tap and remained persistently stuck on screen.

2. **Generic Dashboard Without Operational Purpose**:
   - `/app/societies/[id]` previously rendered 14 uncoordinated administrative consoles simultaneously on a single page, initiating over 200 HTTP calls on load.
   - Layout lacked role awareness, operational hierarchy, and Swiss-minimalist design token adherence.

3. **Authentication Failure on Reload**:
   - Users reloading any authenticated route were frequently logged out and redirected to `/login`.
   - On page reload, multiple protected components fired parallel API queries.
   - Each query received a 401 when the access token was being refreshed.
   - The backend enforces single-use refresh token rotation (`internal/auth/service.go`).
   - Concurrent calls to `POST /auth/refresh` raced: the first rotated the token, and the second was rejected as `ErrInvalidToken`, causing `clearSession()` to wipe the user's active session.

---

## Decision

### 1. Responsive Application Shell Unification
- **CSS-First Viewport Control**: Primary navigation visibility transitions are handled declaratively via CSS utility classes (`hidden md:flex` for desktop sidebar and `flex md:hidden` for mobile bottom navigation), eliminating hydration flash and client-side breakpoint race conditions.
- **Unified Breakpoint**: Standardized all responsive shell logic across `SocietyShellHeader`, `NavigationRail`, and modal shells to `768px` (`md`).
- **Zero Reserved Space on Mobile**: The mobile viewport does not reserve margin or column widths for the desktop rail. The main content container occupies full width (`min-w-0 w-full overflow-x-hidden`).
- **Touch-Safe Tooltips**: Sidebar tooltips are suppressed on mobile views and rendered exclusively when the desktop sidebar is in collapsed rail mode (`sidebar.isCollapsed`).

### 2. Purpose-Built Society Operations Hub
- **Role-Aware Information Architecture**:
  - `SOC_ADMIN` / `SUPER_ADMIN`: High-level operational scorecards via `GET /societies/{id}/dashboard`, actionable callout banners for pending visitor self-registrations (`pendingQRTotal > 0`), an operations directory routing directly to dedicated sub-routes, and real recent announcements.
  - `RESIDENT`: Focused resident experience prioritizing pass management, visitor checkouts, and dues history without administrative clutter.
  - `GATEKEEPER`: Immediate redirect and deep integration with `/app/societies/[id]/gate`.
- **Swiss-Minimalist Design Language**: Strictly adheres to `#f3f3f1` canvas, `#e2e2de` hairline borders, `#121212` primary text/actions, and Hedvig Letters typography. Eliminates bloated multi-console stacking.

### 3. In-Flight Refresh Token Deduplication & Offline Resilience
- **In-Flight Refresh Promise Lock**:
  ```typescript
  // frontend/src/lib/auth.svelte.ts
  if (this.refreshPromise) {
    return this.refreshPromise;
  }
  this.refreshPromise = this.executeRefresh();
  ```
  All concurrent 401 interceptors share the identical in-flight token rotation exchange, eliminating race conditions against the backend single-use rotation invariant.
- **Differentiated Error Semantics**:
  - `401 Unauthorized`: Distinguishes between legitimate token invalidation/revocation (`SESSION_REVOKED`, where `sessionReplaced` is preserved) and transient errors.
  - Network Failure / 5xx: Preserves local session tokens and sets `auth.networkError = true`. The application shell presents a non-destructive offline state with a `Retry Connection` action rather than logging the user out.
- **Tab-Scoped Secure Storage**: Refresh tokens remain stored in `sessionStorage` (preventing persistent token theft via `localStorage` while retaining active tab reloads), strictly obeying ADR 010 and ADR 011.

---

## Consequences

### Positive
- Reliable page reloads across all authenticated routes without unexpected logouts.
- Full viewport compliance on mobile (320px–375px) without horizontal clipping or lingering tooltips.
- Fast, cohesive society dashboard load with < 5 API requests instead of > 200.
- Strict preservation of the 2-device FIFO session model.

### Negative / Trade-offs
- Refresh tokens in `sessionStorage` are scoped to the current browser tab. Opening a new tab requires signing in or sharing credentials via an intentional cross-tab channel (consistent with existing security architecture).
