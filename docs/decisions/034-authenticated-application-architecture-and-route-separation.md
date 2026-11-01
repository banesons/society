# ADR 034: Authenticated Application Architecture and Route Separation

## Status
Accepted

## Context
The CHS application requires a strict, production-grade boundary between:
1. **Public Marketing Website**: Public informational pages (`/`, marketing sections, SEO-indexed content) that must never display protected application state, private society data, authenticated navigation, or account controls.
2. **Authenticated Product Application**: All post-authentication user workflows must live under the canonical `/app/*` route hierarchy:
   - `/app`: Canonical authenticated entry point.
   - `/app/dashboard`: Canonical authenticated workspace landing page, displaying accessible societies, verified roles, notifications, and real platform actions without fake statistics or empty widgets.
   - `/app/societies`: Dedicated accessible society listing and context-selection interface.
   - `/app/societies/[id]`: Canonical society workspace route, preserving role-aware navigation, section deep links, and tenant isolation.
   - `/app/admin`: Canonical platform administration console for `SUPER_ADMIN` and `ADMIN`.
   - `/app/login`: Deliberate unauthenticated entry point within `/app`, supporting return destinations and preventing redirect loops.

### Legacy Route Migration & Fragment Preservation
Previously, society workspaces were hosted at `/society/[id]` and platform administration at `/admin`. Crucially, role-based navigation uses URL fragments (`#resident-maintenance`, `#announcements`, `#gatekeeping`, `#documents`, `#audit`).
HTTP server-side redirects (HTTP 308/301/302) do NOT transmit URL fragments to the server. Therefore:
- Server-side redirects alone would discard user deep links like `/society/:id#resident-maintenance`.
- Legacy compatibility routes (`/society/[id]`, `/societies/[id]`, `/login`, `/admin`) must execute client-side redirects that capture `window.location.hash` and safely forward the user to `/app/societies/[id]#...` with `{ replaceState: true }`.

### Session Restoration Architecture
CHS uses client-side session tokens stored in `sessionStorage` (`chs_refresh_token`) refreshed via `/auth/refresh`. Tokens are not stored in HTTP cookies on initial SSR requests.
Consequently, route protection cannot rely on synchronous server-side `load` redirects. Instead:
- Client-side layout guards (`/app/+layout.svelte`) manage authentication state.
- While `auth.isLoading` is true, a clean minimal loading skeleton is displayed, completely preventing flashes of unauthorized or unauthenticated content.
- When unauthenticated, the user is redirected to `/app/login?returnTo=...`.

---

## Decision

### 1. Canonical Route Topology
We establish the following authoritative route structure:

| Canonical Route | Access Boundary | Purpose & Behaviour |
| :--- | :--- | :--- |
| `/` | Public | Public marketing and product overview. Zero authenticated leaks. |
| `/app` | Authenticated Entry | Redirects to `/app/dashboard` if authenticated, or `/app/login` if unauthenticated. |
| `/app/dashboard` | Protected Shell | Canonical landing page. Lists accessible societies, verified roles, platform shortcuts, session management. |
| `/app/societies` | Protected Shell | Dedicated society selector displaying only tenants accessible to the identity. |
| `/app/societies/[id]` | Protected Society Shell | Canonical society workspace. Preserves full role-aware navigation, sections, and tenant isolation. |
| `/app/admin` | Protected Platform Shell | Restricted platform governance console (`SUPER_ADMIN` / `ADMIN`). |
| `/app/login` | Deliberate Auth Entry | Phone OTP login form. Accepts safe internal `returnTo` parameters. |

### 2. Backward Compatibility & Fragment Preservation
To prevent broken bookmarks and deep links:
- `/login`: Redirects to `/app/login` (preserving query parameters like `returnTo`).
- `/admin`: Redirects to `/app/admin`.
- `/society/[id]` & `/societies/[id]`: Preserved as client-side compatibility pages that read `window.location.hash` and redirect to `/app/societies/${id}${hash}` via SvelteKit `goto(..., { replaceState: true })`.

### 3. Central Application Shell (`src/routes/app/+layout.svelte`)
All routes under `/app/*` are wrapped by a dedicated layout shell:
- **Authentication Guard**:
  - If `page.url.pathname === '/app/login'`, access is granted unconditionally.
  - For all other `/app/*` routes, if `auth.isLoading`, renders a distraction-free loading skeleton.
  - If `!auth.isAuthenticated`, redirects to `/app/login?returnTo=${encodeURIComponent(page.url.pathname + page.url.search + page.url.hash)}`.
- **Shell Header**:
  - Standard `/app` and `/app/admin` pages render `AppShellHeader` (Branding, Notifications, Profile Menu).
  - Society workspaces (`/app/societies/[id]`) provide their own dedicated `SocietyShellHeader` with contextual back-navigation, tenant boundary badge, and society drawer.
- **Strict SEO Isolation**:
  - All `/app/*` routes enforce `<meta name="robots" content="noindex, nofollow" />` in their `<svelte:head>` to ensure search engines never index application or tenant surfaces.

### 4. Typed Route Registry (`src/lib/routing.ts`)
The route registry is updated to export typed route builders and canonical constants:
```typescript
export const ROUTES = {
  HOME: '/',
  APP: '/app',
  DASHBOARD: '/app/dashboard',
  SOCIETIES: '/app/societies',
  LOGIN: '/app/login',
  ADMIN: '/app/admin',
  DEVELOPER: '/developer',
  UNAUTHORIZED: '/unauthorized'
} as const;

export const appRoutes = {
  root: ROUTES.APP,
  dashboard: ROUTES.DASHBOARD,
  societies: ROUTES.SOCIETIES,
  society: (id: string) => `/app/societies/${id}`,
  login: ROUTES.LOGIN,
  admin: ROUTES.ADMIN
};
```

`resolvePostLoginRoute` directs single-society users to `/app/societies/${id}`, platform admins to `/app/admin`, and multi-society users to `/app/societies` (or prompt selection).
`getSocietyNavItems` targets `/app/societies/${societyId}#...`.

### 5. Backend Authorization Authority
Frontend route guards provide UX transitions and prevent unauthorized content flashes. The Go backend remains the authoritative security boundary:
- Middleware (`RequireSocietyAccess`, `RequireAnySocietyRoleAccess`, `RequirePlatformAdmin`, `RequireSuperAdmin`) continues to strictly reject unauthorized requests with HTTP 401/403.
- No frontend routing modification alters backend authorization rules.

---

## Consequences

### Positive
- Strict separation of public marketing concerns from authenticated application surfaces.
- Zero leakage of private society or account state onto public pages.
- Canonical, intuitive URL hierarchy under `/app/*`.
- Seamless backward compatibility for legacy bookmarks and fragment links (`#resident-maintenance`, etc.).
- Robust client-side guard prevents redirect loops and flashes of unauthorized content.

### Negative / Trade-offs
- Client-side hash preservation requires keeping lightweight compatibility components at `/society/[id]` and `/societies/[id]` until full deprecation.
