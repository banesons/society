# ADR 006: Local Development Topology and Cloud Supabase

## Status
Accepted

## Context
Development environments often suffer from friction when developers are forced to run heavy local database containers, local Supabase emulators, or complex Docker compose stacks before writing a line of code. We need a lightweight, low-friction setup for local iteration.

## Decision
The CHS development environment is structured with a hybrid local-cloud topology:
- **Local Application Services**:
  - `backend/`: Go HTTP API server runs locally on a development port (e.g. `http://localhost:8080`).
  - `frontend/`: SvelteKit development server runs locally (e.g. `http://localhost:5173`).
- **Cloud Database & Services**:
  - Supabase Cloud (`ap-south-1`) hosts PostgreSQL, database extensions, and connection poolers.
  - Both local backend and local frontend connect directly to Supabase Cloud via secure credentials loaded from `.env`.
- **No Local Dockerized Supabase**:
  - We do not run local Dockerized Supabase or local PostgreSQL containers in S00 unless a future recorded ADR changes this.

## Consequences
- Fast developer onboarding: clone repo, install Go and Node dependencies, supply `.env`, and start servers.
- No divergence between local PostgreSQL quirks and remote Supabase features.
- Secrets must be protected: `.env` is ignored by git; `.env.example` provides sanitized template values.

## Related Documentation
- [[../architecture/01-system-overview|System Overview]]
- [[../architecture/04-supabase|Supabase Architecture]]
