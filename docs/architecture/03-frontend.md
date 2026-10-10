# 03 — Frontend Architecture

## Design Overview

The CHS frontend is built with **SvelteKit** and **Tailwind CSS**. It follows modern web application design practices with server-state management powered by **TanStack Query** (`@tanstack/svelte-query`).

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
│   │   │   └── client.ts      # Central HTTP client (headers, baseURL, error handling)
│   │   └── components/
│   │       └── ui/            # Reusable primitive UI components
│   └── routes/
│       ├── +layout.svelte     # Root application shell & TanStack Query provider
│       └── +page.svelte       # Initial system status / landing view
├── .env.example               # Template environment configuration
├── package.json
├── svelte.config.js
├── tailwind.config.ts
└── vite.config.ts
```

## Central API Client

The centralized client in `src/lib/api/client.ts` encapsulates:
1. Base API URL injection from public environment variables (`PUBLIC_API_URL`).
2. Inclusion of standard headers (e.g. `Content-Type: application/json`).
3. Outbound request ID propagation and inbound request ID logging.
4. Parsing and unwrapping of standard CHS response envelopes (`{ success, data, meta }` or `{ success, error, request_id }`).
5. Uniform error creation with human-readable error messages.

## Scope Boundaries for S00

S00 creates only the foundational application shell:
- No authentication screens (login, OTP) — deferred to S01.
- No society dashboards, gatekeeper screens, or admin panels — deferred to future sprints.
- Validates end-to-end communication with the Go backend via `/api/v1/health`.

## Related Documentation
- [[01-system-overview|System Overview]]
- [[../api/01-api-overview|API Overview]]
- [[../decisions/001-go-sveltekit-supabase|ADR 001: Core Stack]]
