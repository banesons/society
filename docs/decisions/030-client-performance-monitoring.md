# ADR 030: Client-Side Performance Monitoring (Firebase Performance)

## Status
Accepted

## Context
Cooperative Housing Society (CHS) operations require fast, responsive user experiences across varied physical environments:
- **Security Gate Desktops & Tablets**: Must process visitor check-ins and vehicle scans in sub-second time without client UI stutter or hung network calls.
- **Resident Mobile Browsers**: Must quickly load announcement feeds, gate approval modals, and maintenance invoices over fluctuating mobile networks.
- **Society & Platform Administration**: High-density rosters, audit logs, and analytics dashboards need continuous monitoring for performance regressions and network bottlenecks.

While backend HTTP handler latency is observed server-side, client-side Real User Monitoring (RUM)—including First Contentful Paint (FCP), Largest Contentful Paint (LCP), DOM Interactive, network request durations, and custom workflow execution latency—was previously uninstrumented in the SvelteKit frontend.

## Decision

1. **Integrated Modular Firebase Performance SDK**:
   - Utilize the official modular Firebase Web SDK (`firebase/app` and `firebase/performance`).
   - Package configuration: `firebase@^11` in `frontend/package.json`.
   - Client imports:
     ```typescript
     import { initializeApp, getApps, getApp } from "firebase/app";
     import { getPerformance } from "firebase/performance";
     ```

2. **Isomorphic & Safe Browser-Only Execution**:
   - In SvelteKit, pages and components execute in both Node.js (Server-Side Rendering / SSR) and the client browser.
   - `firebase/performance` requires DOM and Navigation Timing APIs unavailable during server rendering.
   - All Performance Monitoring initialization is guarded by `typeof window !== 'undefined'` and executed exclusively inside client-side `onMount()` or dynamic client imports.
   - Server-side execution gracefully returns `null` without throwing errors or breaking SSR.

3. **Unified Firebase App Initialization**:
   - Consolidate client Firebase App initialization into a shared module (`src/lib/firebase.ts`).
   - Both Web Push Messaging (`src/lib/push.ts`) and Performance Monitoring (`src/lib/firebase.ts`) share the singleton `FirebaseApp` instance, avoiding redundant app initializations.
   - Configuration is supplied via client-safe environment variables (`PUBLIC_FIREBASE_*`).
   - If configuration is missing or incomplete (such as in offline development or test environments without Firebase credentials), initialization fails gracefully and logs a non-fatal warning without crashing the UI.

4. **Automated & Custom Traces**:
   - **Automated Out-of-the-Box Metrics**: Page load timing, network request latencies (via fetch/XHR interception), and Core Web Vitals.
   - **Custom Tracing Utility**: Provide a standardized helper `traceOperation<T>(traceName: string, operation: () => Promise<T>): Promise<T>` to trace critical multi-step frontend operations (e.g., visitor registration, bill generation, payment recording).

## Consequences

### Positive
- Real-time client performance data and network latency metrics visible in Firebase Console without custom backend collector infrastructure.
- Zero impact on SSR or server bundle execution.
- Shared singleton Firebase app prevents duplicate connections and SDK bloat.
- Graceful degradation when network connectivity is restricted or Firebase keys are unconfigured.

### Negative
- Increases client bundle size slightly (~25 kB gzipped for Firebase App + Performance).
- Browser tracking prevention or ad-blockers may block Firebase Performance endpoints (`firebaselogging.googleapis.com`), requiring fail-safe design.
