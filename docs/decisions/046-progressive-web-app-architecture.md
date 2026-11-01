# ADR 046: Progressive Web Application (PWA) Architecture, Manifest Specification, Offline Caching, and Service Worker Lifecycle

## Status
Accepted

## Context
CHS is designed as an operational platform for housing societies, gated communities, and apartment complexes. In production environments, users operate across varied form factors and network conditions:
1. **Security Guards & Gatekeepers**: Operate at physical boundary checkpoints, basement gates, or barrier posts where Wi-Fi and 4G connectivity can experience transient latency drops or packet loss.
2. **Society Residents & Committee Members**: Rely on quick mobile access to view passes, receive gate arrival notifications, pay dues, and review notices without navigating browser address bars or encountering blank error screens during momentary network drops.
3. **Desktop Administrators**: Manage large society topologies and accounting records and benefit from standalone windowing and dock integration.

Prior to this decision, the frontend lacked a standard Web App Manifest (`manifest.webmanifest`), home-screen icons, standalone viewport configuration, and service worker registration on initial boot. Service worker registration was deferred until push notification opt-in, leaving unauthenticated users and visitors without offline fallback protection and preventing browser install prompts.

---

## Decision

### 1. Web App Manifest Specification
A canonical web application manifest is deployed at `/manifest.webmanifest` adhering to the W3C Web App Manifest specification:
- **Identity**:
  - `name`: "Banesons Society"
  - `short_name`: "Society"
  - `id`: "/"
  - `start_url`: "/app/dashboard"
  - `scope`: "/"
  - `display`: "standalone"
  - `orientation`: "portrait-primary"
- **Visual Design & Theming**:
  - `background_color`: `#f3f3f1` (warm studio neutral canvas conforming to CHS Swiss-minimalist tokens)
  - `theme_color`: `#121212` (deep jet black matching headers and status bars)
- **Application Shortcuts**:
  - Dedicated shortcuts for quick operational entry:
    - *Operations Hub* (`/app/dashboard`)
    - *Gate Console* (`/app/gate`)
    - *Visitor Pass* (`/visit`)
    - *Announcements* (`/app/announcements`)
- **Icon Assets**:
  - High-resolution SVG, 192x192 PNG, and 512x512 PNG icons in standard and maskable formats (`purpose: "any maskable"`).

### 2. Service Worker Lifecycle & Offline Cache Strategy
The root service worker (`/firebase-messaging-sw.js`) is unified and initialized on application boot:
- **Unified Role**: Serves both as the Firebase Cloud Messaging (FCM) background push processor and as the PWA offline resilience worker.
- **Cache Strategy**:
  - **Static Shell Assets** (`CACHE_NAME = 'chs-shell-v1'`): Pre-caches core brand icons, manifest, and `/offline.html` during the `install` phase.
  - **Stale-While-Revalidate for Static Resources**: Fonts (`fonts.googleapis.com`, `fonts.gstatic.com`) and static media are served from cache while updating in the background.
  - **Network-First for Navigations with Offline Fallback**: HTML document navigations prioritize live network data to ensure immediate authorization updates, but gracefully fallback to a branded `/offline.html` screen if offline or disconnected.
  - **Cache Pruning on Activate**: Automatically purges stale cache versions during worker activation using `clients.claim()`.

### 3. Immediate Boot Registration & Reactive Install Experience
- The root layout (`frontend/src/routes/+layout.svelte`) registers the service worker on mount during browser hydration.
- An install controller store (`frontend/src/lib/pwa.svelte.ts`) captures the browser `beforeinstallprompt` event and tracks standalone display mode (`window.matchMedia('(display-mode: standalone)')`).
- The application exposes an accessible "Install App" action within profile menus and banner surfaces when installation is eligible.

---

## Consequences

### Positive
- Fully installable on iOS (Safari "Add to Home Screen"), Android (Chrome "Install Application"), macOS, and Windows.
- Clean standalone execution without browser URL bars, providing a native application feel for guards and residents.
- Instant offline feedback with a branded fallback screen instead of browser "No Internet" errors.
- Cohesive integration of Firebase Cloud Messaging and PWA offline capabilities in a single unified service worker without script conflicts.

### Negative / Trade-offs
- Static asset caching requires careful cache version bumping during major releases to avoid stale client bundles (mitigated by network-first strategy on document requests).
